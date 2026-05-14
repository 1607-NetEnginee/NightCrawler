// Package core contains the orchestrator: the part of NIGHTCRAWLER
// that owns a single scan's lifecycle from request validation through
// to report finalization. It is the only package that knows how to
// turn user intent into plugin invocations.
package core

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/HnyBadger/nightcrawler/pkg/api"
)

// ScanRequest is the typed translation of CLI flags. Validation
// happens in one place so the CLI layer stays thin.
type ScanRequest struct {
	Targets        []string
	TargetsFile    string
	Profile        string
	EnablePlugins  []string
	DisablePlugins []string
	OutputDir      string
	OutputFormats  []string
	Concurrency    int
	FailOn         string
	Operator       string
	ClientName     string
	DryRun         bool
	Timeout        time.Duration
}

// Validate enforces the contract documented in --help. It returns a
// joined error so a single call surfaces every problem the user must
// fix, rather than the usual run-fix-rerun loop.
func (r *ScanRequest) Validate() error {
	var problems []string

	if len(r.Targets) == 0 && r.TargetsFile == "" {
		problems = append(problems, "no targets provided")
	}
	if r.Profile == "" {
		r.Profile = "default"
	}
	if !isValidProfile(r.Profile) {
		problems = append(problems, fmt.Sprintf("unknown profile %q", r.Profile))
	}
	if r.Concurrency < 0 {
		problems = append(problems, "concurrency must be >= 0")
	}
	if r.Concurrency == 0 {
		r.Concurrency = runtime.NumCPU()
	}
	if r.FailOn == "" {
		r.FailOn = "high"
	}
	if !isValidSeverity(r.FailOn) && r.FailOn != "none" {
		problems = append(problems, fmt.Sprintf("invalid --fail-on %q", r.FailOn))
	}
	if len(r.OutputFormats) == 0 {
		r.OutputFormats = []string{"ndjson", "html"}
	}

	if len(problems) > 0 {
		return fmt.Errorf("%s", strings.Join(problems, "; "))
	}
	return nil
}

func isValidProfile(p string) bool {
	switch p {
	case "stealth", "default", "aggressive", "quick", "compliance":
		return true
	}
	return false
}

func isValidSeverity(s string) bool {
	switch s {
	case "info", "low", "medium", "high", "critical":
		return true
	}
	return false
}

// SeverityCounts is a histogram of finding levels emitted during a
// scan. Fields are explicit (not a map) so JSON output is stable and
// downstream consumers don't need to handle missing keys.
type SeverityCounts struct {
	Info     int `json:"info"`
	Low      int `json:"low"`
	Medium   int `json:"medium"`
	High     int `json:"high"`
	Critical int `json:"critical"`
}

// Add increments the counter for sev.
func (c *SeverityCounts) Add(sev api.Severity) {
	switch sev {
	case api.SeverityInfo:
		c.Info++
	case api.SeverityLow:
		c.Low++
	case api.SeverityMedium:
		c.Medium++
	case api.SeverityHigh:
		c.High++
	case api.SeverityCritical:
		c.Critical++
	}
}

// Total returns the sum of all severity counts.
func (c *SeverityCounts) Total() int {
	return c.Info + c.Low + c.Medium + c.High + c.Critical
}

// ScanResult is what main() inspects to decide exit code.
type ScanResult struct {
	ScanID         string
	TargetsScanned int
	TotalFindings  int
	SeverityCounts SeverityCounts
	Duration       time.Duration
	OutputDir      string
	RiskScore      float64
}

// targetCache is a goroutine-safe per-target cache implementing
// api.TargetCache. Populated by upstream plugins (e.g. dns, tech-profile)
// and consumed by downstream plugins (e.g. paths, webshell).
type targetCache struct {
	mu sync.RWMutex
	m  map[string]any
}

func newTargetCache() *targetCache {
	return &targetCache{m: make(map[string]any)}
}

func (c *targetCache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.m[key]
	return v, ok
}

func (c *targetCache) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = value
}

// scanContext bundles everything plugins or sub-routines need. We use
// composition over a giant struct so a plugin can hold a narrow view.
type scanContext struct {
	scanID    string
	cfg       *ScanRequest
	plugins   []api.Plugin
	cache     map[string]*targetCache // per-target
	cacheMu   sync.Mutex
	findings  chan api.Finding
	ctx       context.Context
	startedAt time.Time
}

func (s *scanContext) cacheFor(domain string) *targetCache {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if c, ok := s.cache[domain]; ok {
		return c
	}
	c := newTargetCache()
	s.cache[domain] = c
	return c
}
