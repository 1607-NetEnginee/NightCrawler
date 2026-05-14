module github.com/HnyBadger/nightcrawler

go 1.22

// Dependencies are added incrementally as Wave B/C modules land
// (see docs/ANALYSIS_AND_REDESIGN.md §8.1). The Wave A skeleton uses
// only what is needed to compile a working orchestrator + DNS plugin.
require (
	github.com/sourcegraph/conc v0.3.0
	github.com/spf13/cobra v1.8.0
	golang.org/x/sync v0.7.0
)

