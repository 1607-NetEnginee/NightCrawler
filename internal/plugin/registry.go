// Package plugin owns the plugin registry. Built-in plugins register
// themselves via Register() called from their package init(). Remote
// plugins (v7.1+, see §12.6 of the design document) will register
// through a separate gRPC-backed loader that lives in plugin/grpc.
package plugin

import (
	"fmt"
	"sort"
	"sync"

	"github.com/1607-NetEnginee/NightCrawler/pkg/api"
)

// PluginRegistry is the in-process plugin catalog. Goroutine-safe so
// init-order doesn't matter across packages and so tests can mutate
// the registry without races.
type PluginRegistry struct {
	mu      sync.RWMutex
	plugins map[string]api.Plugin
}

var globalRegistry = &PluginRegistry{
	plugins: make(map[string]api.Plugin),
}

// Registry returns the process-wide singleton. Callers should treat
// this as read-mostly; Register/Unregister are reserved for init and
// for tests.
func Registry() *PluginRegistry {
	return globalRegistry
}

// Register adds a plugin to the registry. Duplicate names panic at
// init time, which is the correct level of strictness: a duplicate
// plugin name is a build error, not a runtime condition.
func (r *PluginRegistry) Register(p api.Plugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := p.Manifest().Name
	if name == "" {
		panic("plugin: Register called with empty manifest name")
	}
	if _, exists := r.plugins[name]; exists {
		panic(fmt.Sprintf("plugin: %q registered twice", name))
	}
	r.plugins[name] = p
}

// Get looks up a plugin by manifest name.
func (r *PluginRegistry) Get(name string) (api.Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plugins[name]
	return p, ok
}

// All returns all registered plugins, sorted by manifest name for
// stable output across calls (tests, list command).
func (r *PluginRegistry) All() []api.Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]api.Plugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Manifest().Name < out[j].Manifest().Name
	})
	return out
}

// Register is the package-level shortcut used by built-in plugins.
//
//	func init() { plugin.Register(New()) }
func Register(p api.Plugin) {
	globalRegistry.Register(p)
}
