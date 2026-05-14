// NIGHTCRAWLER v7.0 — Offensive Security Framework
// Author: HnyBadger / Cyberoutcast
// License: Apache-2.0
//
// Entry point for the `nightcrawler` binary. The job of main() is small:
// wire signals, hand control to the CLI layer, exit with a meaningful
// code. Everything else lives in internal/cli and below.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/HnyBadger/nightcrawler/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	if err := cli.NewRoot().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "nightcrawler:", err)
		os.Exit(exitCodeFor(err))
	}
}

// exitCodeFor maps error kinds to Unix-friendly exit codes so CI
// pipelines can distinguish "found findings" from "scanner failed".
// 0 — clean, no findings above threshold
// 1 — findings above threshold (configurable via --fail-on)
// 2 — scanner error (config invalid, target unreachable, plugin crash)
// 130 — interrupted by user (signal)
func exitCodeFor(err error) int {
	// Wired up properly once cli.Errors define typed sentinels.
	// Placeholder defaults to generic error code.
	return 2
}
