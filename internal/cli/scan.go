package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/1607-NetEnginee/NightCrawler/internal/core"
)

// newScanCmd builds the `scan` subcommand. The job of this function is
// to translate user intent (flags) into a core.ScanRequest and hand it
// to the orchestrator. No business logic lives here.
func newScanCmd() *cobra.Command {
	var (
		targets        []string
		targetsFile    string
		profile        string
		plugins        []string
		excludePlugins []string
		outputDir      string
		outputFormats  []string
		concurrency    int
		failOn         string
		operator       string
		client         string
		dryRun         bool
		timeout        time.Duration
	)

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Run a security scan against one or more targets",
		Long: `Run NIGHTCRAWLER against one or more targets.

By default the "default" profile runs 16 built-in plugins concurrently
per target. The scan engine respects rate limits, honors signals (SIGINT
flushes a partial report), and writes a canonical NDJSON event stream.

Examples:
  nightcrawler scan --target example.com
  nightcrawler scan --target example.com --target api.example.com --profile stealth
  nightcrawler scan --targets-file scope.txt --profile aggressive --fail-on high
  nightcrawler scan --target example.com --plugins dns,tls,headers --no-banner`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(targets) == 0 && targetsFile == "" {
				return fmt.Errorf("at least one --target or --targets-file is required")
			}

			req := core.ScanRequest{
				Targets:        targets,
				TargetsFile:    targetsFile,
				Profile:        profile,
				EnablePlugins:  plugins,
				DisablePlugins: excludePlugins,
				OutputDir:      outputDir,
				OutputFormats:  outputFormats,
				Concurrency:    concurrency,
				FailOn:         failOn,
				Operator:       operator,
				ClientName:     client,
				DryRun:         dryRun,
				Timeout:        timeout,
			}

			if err := req.Validate(); err != nil {
				return fmt.Errorf("invalid scan request: %w", err)
			}

			orch, err := core.NewOrchestrator(cmd.Context())
			if err != nil {
				return fmt.Errorf("init orchestrator: %w", err)
			}

			result, err := orch.Run(cmd.Context(), req)
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}

			// Exit codes are taken from the result by main(); we just
			// surface a one-line summary here.
			fmt.Fprintln(cmd.OutOrStdout(), summarize(result))
			return nil
		},
	}

	cmd.Flags().StringSliceVarP(&targets, "target", "t", nil,
		"target domain or URL (repeatable)")
	cmd.Flags().StringVar(&targetsFile, "targets-file", "",
		"path to file with one target per line")
	cmd.Flags().StringVarP(&profile, "profile", "p", "default",
		"profile: stealth | default | aggressive | quick | compliance")
	cmd.Flags().StringSliceVar(&plugins, "plugins", nil,
		"explicit plugin list, overrides profile (e.g. dns,tls,headers)")
	cmd.Flags().StringSliceVar(&excludePlugins, "exclude", nil,
		"plugin names to skip (e.g. nikto,sqli)")
	cmd.Flags().StringVarP(&outputDir, "output", "o", "",
		"output directory (default: ~/nightcrawler-reports/<auto>)")
	cmd.Flags().StringSliceVarP(&outputFormats, "format", "f",
		[]string{"ndjson", "html"},
		"output formats: ndjson, txt, html, sarif")
	cmd.Flags().IntVar(&concurrency, "concurrency", 0,
		"max concurrent targets (0 = auto = NumCPU)")
	cmd.Flags().StringVar(&failOn, "fail-on", "high",
		"exit non-zero if any finding ≥ this severity (low|medium|high|critical|none)")
	cmd.Flags().StringVar(&operator, "operator", "",
		"operator name for audit trail (defaults to $USER)")
	cmd.Flags().StringVar(&client, "client", "",
		"client/organization name (required in production)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false,
		"resolve plan and print DAG without sending probes")
	cmd.Flags().DurationVar(&timeout, "timeout", 0,
		"hard wall-clock limit for the entire scan (0 = unlimited)")

	return cmd
}

// summarize renders a one-line scan result for the operator. Detailed
// summary is written in NDJSON / HTML by the report writer.
func summarize(r *core.ScanResult) string {
	if r == nil {
		return ""
	}
	parts := []string{
		fmt.Sprintf("scan_id=%s", r.ScanID),
		fmt.Sprintf("targets=%d", r.TargetsScanned),
		fmt.Sprintf("findings=%d", r.TotalFindings),
		fmt.Sprintf("critical=%d", r.SeverityCounts.Critical),
		fmt.Sprintf("high=%d", r.SeverityCounts.High),
		fmt.Sprintf("medium=%d", r.SeverityCounts.Medium),
		fmt.Sprintf("low=%d", r.SeverityCounts.Low),
		fmt.Sprintf("duration=%s", r.Duration.Round(time.Second)),
		fmt.Sprintf("report=%s", r.OutputDir),
	}
	return strings.Join(parts, " ")
}
