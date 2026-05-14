package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sourcegraph/conc/pool"
	"golang.org/x/sync/semaphore"

	"github.com/1607-NetEnginee/NightCrawler/internal/plugin"
	"github.com/1607-NetEnginee/NightCrawler/pkg/api"
)

// Orchestrator owns a single scan. It is constructed via
// NewOrchestrator, executes via Run, and writes a ScanResult.
//
// Concurrency model (matches §13 of the design document):
//
//	Layer 1: target pool       — N targets concurrent
//	Layer 2: plugin DAG pool   — M plugins per target concurrent
//	Layer 3: per-plugin probes — managed inside each Plugin.Run
//
// The orchestrator owns layers 1 and 2; layer 3 is the plugin's
// responsibility (see internal/plugins/dns for the canonical pattern).
type Orchestrator struct {
	logger     *slog.Logger
	httpClient *http.Client
	registry   *plugin.PluginRegistry
}

// NewOrchestrator wires the orchestrator's dependencies. The context
// is currently unused but reserved for future init steps (config load,
// signature DB fetch) that may want to honor cancellation.
func NewOrchestrator(_ context.Context) (*Orchestrator, error) {
	return &Orchestrator{
		logger:     slog.Default(),
		httpClient: defaultHTTPClient(),
		registry:   plugin.Registry(),
	}, nil
}

// Run executes a single scan and blocks until completion or context
// cancellation. On cancellation it still attempts to finalize a partial
// report (this is the v6.1 `trap cleanup INT TERM` behavior preserved
// as a Go pattern).
func (o *Orchestrator) Run(ctx context.Context, req ScanRequest) (*ScanResult, error) {
	scanID := newScanID()
	startedAt := time.Now()

	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	// Resolve target list (CLI args + file lines, deduplicated).
	targets, err := resolveTargets(req)
	if err != nil {
		return nil, fmt.Errorf("resolve targets: %w", err)
	}

	// Resolve plugin list (profile + explicit enable + explicit exclude).
	plugins, err := o.resolvePlugins(req)
	if err != nil {
		return nil, fmt.Errorf("resolve plugins: %w", err)
	}

	if req.DryRun {
		return o.dryRun(scanID, req, targets, plugins, startedAt), nil
	}

	// Output directory.
	outDir, err := prepareOutputDir(req, scanID)
	if err != nil {
		return nil, fmt.Errorf("prepare output dir: %w", err)
	}

	// Initialize all plugins once (Init is called per-scan, not per-target).
	for _, p := range plugins {
		err := p.Init(ctx, api.Deps{
			HTTP:   o.httpClient,
			Logger: o.logger.With("plugin", p.Manifest().Name),
			ScanID: scanID,
		})
		if err != nil {
			return nil, fmt.Errorf("plugin %s init: %w",
				p.Manifest().Name, err)
		}
	}

	// Event bus: findings flow here from plugins → aggregator.
	findings := make(chan api.Finding, 1024)

	sc := &scanContext{
		scanID:    scanID,
		cfg:       &req,
		plugins:   plugins,
		cache:     make(map[string]*targetCache),
		findings:  findings,
		ctx:       ctx,
		startedAt: startedAt,
	}

	// Aggregator: single goroutine drains findings into the result.
	var (
		result     = &ScanResult{ScanID: scanID, OutputDir: outDir}
		aggDone    = make(chan struct{})
		findingsMu sync.Mutex
	)
	go func() {
		defer close(aggDone)
		for f := range findings {
			findingsMu.Lock()
			result.TotalFindings++
			result.SeverityCounts.Add(f.Level)
			result.RiskScore += f.Level.Score()
			findingsMu.Unlock()
			// TODO(v7.0): write NDJSON line via output.Writer
			o.logger.Info("finding",
				"plugin", f.Plugin,
				"level", f.Level,
				"title", f.Title,
				"resource", f.Resource.URL,
			)
		}
	}()

	// Layer 1: target pool.
	targetPool := pool.New().
		WithMaxGoroutines(req.Concurrency).
		WithContext(ctx)
	for _, t := range targets {
		t := t
		targetPool.Go(func(ctx context.Context) error {
			return o.runTarget(ctx, sc, t)
		})
	}
	_ = targetPool.Wait() // plugin errors are not fatal at scan level

	close(findings)
	<-aggDone

	result.TargetsScanned = len(targets)
	result.Duration = time.Since(startedAt)
	if result.RiskScore > 100 {
		result.RiskScore = 100
	}

	// TODO(v7.0): finalize HTML report, sarif export, notifications

	return result, nil
}

// runTarget executes the plugin DAG against a single target. Layer 2
// concurrency lives here. This is also where the per-target cache is
// established and seeded.
func (o *Orchestrator) runTarget(ctx context.Context, sc *scanContext, domain string) error {
	cache := sc.cacheFor(domain)
	target := api.Target{
		Domain: domain,
		URL:    "https://" + domain,
		Scheme: "https",
		Cache:  cache,
	}

	emit := func(f api.Finding) {
		// Stamp the event with scan + target metadata before forwarding.
		if f.Schema == "" {
			f.Schema = "nightcrawler.io/v1/finding"
		}
		if f.Timestamp.IsZero() {
			f.Timestamp = time.Now().UTC()
		}
		f.ScanID = sc.scanID
		if f.Target.Domain == "" {
			f.Target.Domain = domain
		}
		select {
		case sc.findings <- f:
		case <-ctx.Done():
		}
	}

	// Build DAG: plugins grouped by topological layer.
	layers := buildDAG(sc.plugins)

	maxPlugins := max(4, sc.cfg.Concurrency/2)
	sem := semaphore.NewWeighted(int64(maxPlugins))

	for _, layer := range layers {
		var wg sync.WaitGroup
		for _, p := range layer {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err := sem.Acquire(ctx, 1); err != nil {
				return err
			}
			wg.Add(1)
			go func(p api.Plugin) {
				defer wg.Done()
				defer sem.Release(1)
				if err := p.Run(ctx, target, emit); err != nil {
					o.logger.Warn("plugin returned error",
						"plugin", p.Manifest().Name,
						"target", domain,
						"err", err,
					)
				}
			}(p)
		}
		wg.Wait()
	}
	return nil
}

// resolvePlugins applies profile + enable + exclude rules to the
// registry to compute the final plugin set for this scan.
func (o *Orchestrator) resolvePlugins(req ScanRequest) ([]api.Plugin, error) {
	all := o.registry.All()

	enabled := map[string]bool{}
	if len(req.EnablePlugins) > 0 {
		// Explicit enable list overrides profile.
		for _, name := range req.EnablePlugins {
			enabled[strings.TrimSpace(name)] = true
		}
	} else {
		for _, p := range all {
			enabled[p.Manifest().Name] = true
		}
	}
	for _, name := range req.DisablePlugins {
		delete(enabled, strings.TrimSpace(name))
	}

	var out []api.Plugin
	for _, p := range all {
		if enabled[p.Manifest().Name] {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("plugin selection resolved to empty set")
	}
	return out, nil
}

// dryRun emits a synthetic result describing what the scan WOULD do.
// Useful in CI to validate scope without sending probes.
func (o *Orchestrator) dryRun(scanID string, req ScanRequest, targets []string,
	plugins []api.Plugin, startedAt time.Time) *ScanResult {
	var names []string
	for _, p := range plugins {
		names = append(names, p.Manifest().Name)
	}
	o.logger.Info("dry-run plan",
		"scan_id", scanID,
		"targets", targets,
		"plugins", names,
		"profile", req.Profile,
	)
	return &ScanResult{
		ScanID:         scanID,
		TargetsScanned: len(targets),
		Duration:       time.Since(startedAt),
		OutputDir:      "(dry-run, no output written)",
	}
}

// newScanID produces a sortable, unique-enough scan identifier.
// Format: s-YYYYMMDD-HHmmss-<rand4>. Sortable lexicographically and
// readable in logs.
func newScanID() string {
	var b [2]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("s-%s-%s",
		time.Now().UTC().Format("20060102-150405"),
		hex.EncodeToString(b[:]),
	)
}

func resolveTargets(req ScanRequest) ([]string, error) {
	seen := map[string]struct{}{}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, t := range req.Targets {
		add(t)
	}
	if req.TargetsFile != "" {
		data, err := os.ReadFile(req.TargetsFile)
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			add(line)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no usable targets")
	}
	return out, nil
}

// prepareOutputDir resolves the per-scan output directory. Pattern
// follows the v6.1 layout but with a safer naming scheme that prevents
// the path-traversal vulnerability identified in §6.2.
func prepareOutputDir(req ScanRequest, scanID string) (string, error) {
	base := req.OutputDir
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, "nightcrawler-reports")
	}
	dir := filepath.Join(base, scanID)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}
	return dir, nil
}

// defaultHTTPClient returns a stealth-aware http.Client. Real
// implementation lives in internal/http; this is a placeholder so the
// orchestrator can be exercised end-to-end before that package lands.
func defaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
