module github.com/1607-NetEnginee/NightCrawler

go 1.22

// Dependencies are added incrementally as Wave B/C modules land
// (see docs/ANALYSIS_AND_REDESIGN.md §8.1). The Wave A skeleton uses
// only what is needed to compile a working orchestrator + DNS plugin.
require (
	github.com/sourcegraph/conc v0.3.0
	github.com/spf13/cobra v1.8.0
	golang.org/x/sync v0.7.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
)
