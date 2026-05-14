package core

import (
	"context"
	"testing"

	"github.com/HnyBadger/nightcrawler/pkg/api"
)

// stubPlugin is a no-op plugin used to exercise the DAG builder without
// pulling any real scanner logic.
type stubPlugin struct {
	name string
	deps []string
}

func (s stubPlugin) Manifest() api.Manifest {
	return api.Manifest{Name: s.name, DependsOn: s.deps}
}
func (s stubPlugin) Init(_ context.Context, _ api.Deps) error                 { return nil }
func (s stubPlugin) Run(_ context.Context, _ api.Target, _ api.Emitter) error { return nil }

func TestBuildDAG_NoDependencies(t *testing.T) {
	plugins := []api.Plugin{
		stubPlugin{name: "dns"},
		stubPlugin{name: "tls"},
		stubPlugin{name: "headers"},
	}
	layers := buildDAG(plugins)
	if len(layers) != 1 {
		t.Fatalf("expected 1 layer for independent plugins, got %d", len(layers))
	}
	if len(layers[0]) != 3 {
		t.Fatalf("expected 3 plugins in layer 0, got %d", len(layers[0]))
	}
}

func TestBuildDAG_LinearChain(t *testing.T) {
	// A → B → C
	plugins := []api.Plugin{
		stubPlugin{name: "C", deps: []string{"B"}},
		stubPlugin{name: "B", deps: []string{"A"}},
		stubPlugin{name: "A"},
	}
	layers := buildDAG(plugins)
	if len(layers) != 3 {
		t.Fatalf("expected 3 layers for A→B→C chain, got %d", len(layers))
	}
	if layers[0][0].Manifest().Name != "A" {
		t.Errorf("layer 0 should start with A, got %s", layers[0][0].Manifest().Name)
	}
	if layers[1][0].Manifest().Name != "B" {
		t.Errorf("layer 1 should be B, got %s", layers[1][0].Manifest().Name)
	}
	if layers[2][0].Manifest().Name != "C" {
		t.Errorf("layer 2 should be C, got %s", layers[2][0].Manifest().Name)
	}
}

func TestBuildDAG_DiamondDependency(t *testing.T) {
	//   A
	//  / \
	// B   C
	//  \ /
	//   D
	plugins := []api.Plugin{
		stubPlugin{name: "A"},
		stubPlugin{name: "B", deps: []string{"A"}},
		stubPlugin{name: "C", deps: []string{"A"}},
		stubPlugin{name: "D", deps: []string{"B", "C"}},
	}
	layers := buildDAG(plugins)
	if len(layers) != 3 {
		t.Fatalf("expected 3 layers for diamond, got %d", len(layers))
	}
	if len(layers[1]) != 2 {
		t.Errorf("middle layer should contain B and C (parallel), got %d plugins", len(layers[1]))
	}
}

func TestBuildDAG_MissingDependencyDegrades(t *testing.T) {
	// "ghost" depends on a plugin we did not provide. The DAG builder
	// must still produce a valid plan — silently degrading rather than
	// rejecting the input — per the comment in dag.go.
	plugins := []api.Plugin{
		stubPlugin{name: "ghost", deps: []string{"nonexistent"}},
	}
	layers := buildDAG(plugins)
	if len(layers) != 1 || layers[0][0].Manifest().Name != "ghost" {
		t.Fatalf("expected ghost to be scheduled despite missing dep, got %v", layers)
	}
}

func TestBuildDAG_CycleStillSchedules(t *testing.T) {
	// A → B → A. Pathological input; DAG builder must not deadlock —
	// must still emit a best-effort layer containing the cycle so the
	// scan can finish.
	plugins := []api.Plugin{
		stubPlugin{name: "A", deps: []string{"B"}},
		stubPlugin{name: "B", deps: []string{"A"}},
	}
	layers := buildDAG(plugins)
	scheduled := 0
	for _, l := range layers {
		scheduled += len(l)
	}
	if scheduled != 2 {
		t.Fatalf("expected both A and B to be scheduled despite cycle, got %d", scheduled)
	}
}
