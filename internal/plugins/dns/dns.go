// Package dns implements the DNS reconnaissance plugin. It is the
// canonical example of how a v7.0 plugin is structured: a small
// concrete type, a manifest that declares its dependencies (none, in
// this case — DNS is a Layer-0 plugin), and a Run method that emits
// findings via the supplied callback.
//
// The work this plugin does is the port of v6.1's scan_dns() and
// subdomain brute-force loop (lines ~1254 and ~1296 of the legacy
// shell script), with the following changes:
//
//   - Concurrent A-record lookups via a semaphore-bounded worker set,
//     replacing the sequential `for sub in "${list[@]}"; do dig …`.
//   - Wordlist loaded from the embedded signature pack rather than
//     hard-coded inline in the script.
//   - Findings emitted as structured api.Finding instead of free-form
//     log lines.
//   - Subdomain results stashed in the per-target cache so downstream
//     plugins (paths, headers) can iterate them without re-resolving.
package dns

import (
	"context"
	_ "embed"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/1607-NetEnginee/NightCrawler/internal/plugin"
	"github.com/1607-NetEnginee/NightCrawler/pkg/api"
	"golang.org/x/sync/semaphore"
)

const (
	pluginName    = "dns"
	pluginVersion = "1.0.0"

	maxConcurrentResolves = 50
	resolveTimeout        = 3 * time.Second

	// CacheKeySubdomains is the per-target cache key used by downstream
	// plugins to consume the list of resolved subdomains without
	// re-running the DNS work.
	CacheKeySubdomains = "dns.subdomains"
)

// Built-in wordlist for subdomain brute-force. A direct port of the
// 80-entry list from v6.1 nightcrawler line 1285. Kept inline here for
// the zero-config bootstrap; the production version loads
// signatures/dns/subdomains.yaml from the signature pack and falls
// back to this when offline.
//
//go:embed wordlist.txt
var defaultWordlist string

// Plugin is the DNS scanner. Held as a stateless value: Init populates
// only configuration, never per-target state, so Run is safe to invoke
// concurrently for many targets.
type Plugin struct {
	cfg      api.PluginConfig
	wordlist []string
}

// New constructs a fresh DNS plugin. Called from the package init()
// below; tests can call it directly to bypass the registry.
func New() *Plugin {
	return &Plugin{wordlist: parseWordlist(defaultWordlist)}
}

func init() {
	plugin.Register(New())
}

// Manifest is the static description used by the registry and the CLI
// `plugin info` command. The list of dependencies is empty: DNS is a
// Layer-0 plugin (nothing else needs to run before it).
func (p *Plugin) Manifest() api.Manifest {
	return api.Manifest{
		Name:        pluginName,
		Version:     pluginVersion,
		Author:      "HnyBadger",
		Description: "DNS recon: A/CNAME/TXT lookups, subdomain brute-force, basic anomaly flags.",
		Category:    api.CategoryRecon,
		Profile:     api.ProfileDefault,
		Tags:        []string{"dns", "recon", "stealth-safe", "builtin"},
		DependsOn:   nil,
		OutputFields: []string{
			"resource.host",
			"target.ip",
			"validation.notes",
		},
	}
}

// Init is called once per scan. The DNS plugin has nothing to cache or
// preload at this point, so it just records the config handle.
func (p *Plugin) Init(_ context.Context, deps api.Deps) error {
	p.cfg = deps.Config
	return nil
}

// Run resolves a base domain and enumerates subdomains via parallel
// brute-force. It writes discovered subdomains to the per-target cache
// under CacheKeySubdomains so downstream plugins can consume them.
func (p *Plugin) Run(ctx context.Context, target api.Target, emit api.Emitter) error {
	// 1. Resolve the apex.
	base := strings.TrimPrefix(target.Domain, "www.")
	ips, err := lookupA(ctx, base)
	if err != nil {
		return fmt.Errorf("apex resolve %s: %w", base, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("apex %s has no A records", base)
	}

	emit(api.Finding{
		Plugin:        pluginName,
		PluginVersion: pluginVersion,
		Level:         api.SeverityInfo,
		Title:         "Domain resolved",
		Resource:      api.Resource{Host: base},
		Target:        api.TargetRef{Domain: base, IP: ips[0]},
		Tags:          []string{"dns", "apex"},
	})

	// 2. Parallel subdomain brute-force, bounded by a weighted semaphore.
	sem := semaphore.NewWeighted(maxConcurrentResolves)
	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		found []SubdomainHit
	)
	for _, prefix := range p.wordlist {
		if ctx.Err() != nil {
			break
		}
		if err := sem.Acquire(ctx, 1); err != nil {
			break
		}
		wg.Add(1)
		go func(prefix string) {
			defer wg.Done()
			defer sem.Release(1)

			host := prefix + "." + base
			subIPs, err := lookupA(ctx, host)
			if err != nil || len(subIPs) == 0 {
				return
			}
			hit := SubdomainHit{Host: host, IPs: subIPs}
			mu.Lock()
			found = append(found, hit)
			mu.Unlock()

			emit(api.Finding{
				Plugin:        pluginName,
				PluginVersion: pluginVersion,
				Level:         api.SeverityInfo,
				Title:         "Subdomain discovered: " + host,
				Resource:      api.Resource{Host: host},
				Target:        api.TargetRef{Domain: host, IP: subIPs[0]},
				Tags:          []string{"dns", "subdomain", "brute"},
			})
		}(prefix)
	}
	wg.Wait()

	// 3. Publish results to the per-target cache for downstream plugins.
	target.Cache.Set(CacheKeySubdomains, found)

	return nil
}

// SubdomainHit is the unit emitted to the per-target cache. Downstream
// plugins type-assert their fetched cache value to []SubdomainHit.
type SubdomainHit struct {
	Host string
	IPs  []string
}

// lookupA wraps Go's stdlib resolver with a short, plugin-local
// timeout. Once internal/dns lands this will be replaced with the
// miekg/dns-backed resolver from api.Deps.
func lookupA(ctx context.Context, host string) ([]string, error) {
	subCtx, cancel := context.WithTimeout(ctx, resolveTimeout)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupHost(subCtx, host)
	if err != nil {
		return nil, err
	}
	return addrs, nil
}

func parseWordlist(raw string) []string {
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}
