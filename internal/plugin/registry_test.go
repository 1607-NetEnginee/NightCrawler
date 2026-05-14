package plugin

import (
	"context"
	"testing"

	"github.com/1607-NetEnginee/NightCrawler/pkg/api"
)

type fakePlugin struct{ name string }

func (f fakePlugin) Manifest() api.Manifest                                   { return api.Manifest{Name: f.name} }
func (f fakePlugin) Init(_ context.Context, _ api.Deps) error                 { return nil }
func (f fakePlugin) Run(_ context.Context, _ api.Target, _ api.Emitter) error { return nil }

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := &PluginRegistry{plugins: map[string]api.Plugin{}}
	p := fakePlugin{name: "alpha"}
	r.Register(p)

	got, ok := r.Get("alpha")
	if !ok {
		t.Fatal("Get returned !ok for registered plugin")
	}
	if got.Manifest().Name != "alpha" {
		t.Fatalf("got name %q, want alpha", got.Manifest().Name)
	}
}

func TestRegistry_All_SortedByName(t *testing.T) {
	r := &PluginRegistry{plugins: map[string]api.Plugin{}}
	r.Register(fakePlugin{name: "gamma"})
	r.Register(fakePlugin{name: "alpha"})
	r.Register(fakePlugin{name: "beta"})

	all := r.All()
	if len(all) != 3 {
		t.Fatalf("expected 3 plugins, got %d", len(all))
	}
	want := []string{"alpha", "beta", "gamma"}
	for i, name := range want {
		if all[i].Manifest().Name != name {
			t.Errorf("All()[%d] = %q, want %q", i, all[i].Manifest().Name, name)
		}
	}
}

func TestRegistry_DuplicatePanic(t *testing.T) {
	r := &PluginRegistry{plugins: map[string]api.Plugin{}}
	r.Register(fakePlugin{name: "dup"})
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on duplicate registration")
		}
	}()
	r.Register(fakePlugin{name: "dup"})
}

func TestRegistry_EmptyNamePanic(t *testing.T) {
	r := &PluginRegistry{plugins: map[string]api.Plugin{}}
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on empty plugin name")
		}
	}()
	r.Register(fakePlugin{name: ""})
}
