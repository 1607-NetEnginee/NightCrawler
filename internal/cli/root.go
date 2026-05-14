// Package cli is the command-line surface of NIGHTCRAWLER v7.0.
// Subcommands live in sibling files. Root only handles:
//   - top-level flags (config path, log level, no-banner)
//   - banner printing (suppressed in non-TTY mode)
//   - subcommand registration
package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/1607-NetEnginee/NightCrawler/internal/version"
)

// NewRoot builds the root cobra command. Constructed per-call (no
// package globals) so tests can spin up isolated command trees.
func NewRoot() *cobra.Command {
	var (
		cfgPath  string
		logLevel string
		noBanner bool
		noColor  bool
	)

	root := &cobra.Command{
		Use:           "nightcrawler",
		Short:         "Offensive Security Framework — by HnyBadger / Cyberoutcast",
		Long:          longDescription,
		Version:       version.Full(),
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !noBanner && isTerminal(os.Stderr) {
				printBanner(os.Stderr, noColor)
			}
			return nil
		},
	}

	root.PersistentFlags().StringVar(&cfgPath, "config", "",
		"path to config file (default: $XDG_CONFIG_HOME/nightcrawler/config.yaml)")
	root.PersistentFlags().StringVar(&logLevel, "log-level", "info",
		"log level: debug | info | warn | error")
	root.PersistentFlags().BoolVar(&noBanner, "no-banner", false,
		"suppress ASCII banner")
	root.PersistentFlags().BoolVar(&noColor, "no-color", false,
		"disable ANSI color output")

	// Subcommands. Each lives in its own file.
	root.AddCommand(newScanCmd())
	root.AddCommand(newReportCmd())
	root.AddCommand(newPluginCmd())
	root.AddCommand(newConfigCmd())
	root.AddCommand(newVersionCmd())

	return root
}

const longDescription = `NIGHTCRAWLER v7.0 — modular offensive security platform.

A complete reverse-engineering and migration of the v6.1 Bash framework
to a single Go binary. Concurrent, plugin-driven, signature-database
backed, and ready for cloud, container, or air-gapped deployment.

"ignored, but critical"`

// printBanner renders the ASCII banner described in §16 of the design
// document. Variant A (geometric) is the default. ldflags-injected
// version is interpolated at print time so a single binary's banner
// always shows its real build version.
func printBanner(w io.Writer, plain bool) {
	body := fmt.Sprintf(bannerTemplate, version.Version)
	if plain {
		fmt.Fprintln(w, body)
		return
	}
	fmt.Fprintln(w, "\x1b[38;5;141m"+body+"\x1b[0m")
}

// bannerTemplate is the geometric (Variant A) banner from §16.1 of
// the design document, with a single %s for the version string.
const bannerTemplate = `
 ███╗   ██╗ ██████╗
 ████╗  ██║██╔════╝   NIGHTCRAWLER  v%s
 ██╔██╗ ██║██║        ─────────────────────────────────
 ██║╚██╗██║██║        Offensive Security Framework
 ██║ ╚████║╚██████╗   Author: HnyBadger · Cyberoutcast
 ╚═╝  ╚═══╝ ╚═════╝   ignored, but critical
`
