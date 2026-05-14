package core

import (
	"github.com/HnyBadger/nightcrawler/pkg/api"
)

// buildDAG performs a Kahn-style topological sort on the plugin list,
// using the DependsOn field of each manifest. It returns plugins
// grouped into "layers" where each layer can run fully in parallel,
// and layer N+1 may only start after layer N completes.
//
// Plugins whose dependencies are not present in the input set are
// treated as having no dependency (the missing prerequisite is silently
// degraded — see TODO below for the strict-mode option).
//
// This is the implementation of the topological execution model from
// §13.4 of the design document.
func buildDAG(plugins []api.Plugin) [][]api.Plugin {
	byName := make(map[string]api.Plugin, len(plugins))
	for _, p := range plugins {
		byName[p.Manifest().Name] = p
	}

	// Build adjacency: depCount[plugin] = number of prerequisites
	// still un-scheduled.
	depCount := make(map[string]int, len(plugins))
	for _, p := range plugins {
		count := 0
		for _, dep := range p.Manifest().DependsOn {
			if _, ok := byName[dep]; ok {
				count++
			}
			// TODO(v7.0): if strict, fail when dep not found.
		}
		depCount[p.Manifest().Name] = count
	}

	var layers [][]api.Plugin
	scheduled := make(map[string]bool, len(plugins))

	for len(scheduled) < len(plugins) {
		var layer []api.Plugin
		for _, p := range plugins {
			name := p.Manifest().Name
			if scheduled[name] {
				continue
			}
			if depCount[name] == 0 {
				layer = append(layer, p)
			}
		}
		if len(layer) == 0 {
			// Cycle or unsatisfiable dependency. Force-schedule whatever
			// is left in a single best-effort layer so the scan still
			// runs.
			for _, p := range plugins {
				if !scheduled[p.Manifest().Name] {
					layer = append(layer, p)
				}
			}
		}
		// Commit this layer.
		for _, p := range layer {
			scheduled[p.Manifest().Name] = true
		}
		// Decrement dep counts for downstream plugins.
		for _, p := range plugins {
			if scheduled[p.Manifest().Name] {
				continue
			}
			for _, dep := range p.Manifest().DependsOn {
				for _, done := range layer {
					if done.Manifest().Name == dep {
						depCount[p.Manifest().Name]--
					}
				}
			}
		}
		layers = append(layers, layer)
	}
	return layers
}
