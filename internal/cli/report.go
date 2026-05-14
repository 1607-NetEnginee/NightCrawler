package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newReportCmd builds the `report` subcommand group. Important
// affordance: a report can be re-rendered from an existing NDJSON event
// stream without re-scanning. This makes report style upgrades possible
// for already-completed work, and supports the canonical-NDJSON
// philosophy described in §18 of the design document.
func newReportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Render or inspect scan reports",
	}
	cmd.AddCommand(newReportRenderCmd())
	cmd.AddCommand(newReportSummaryCmd())
	return cmd
}

func newReportRenderCmd() *cobra.Command {
	var (
		input  string
		format string
		output string
		locale string
	)
	c := &cobra.Command{
		Use:   "render",
		Short: "Render an HTML/SARIF/PDF report from a scan NDJSON file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if input == "" {
				return fmt.Errorf("--input is required")
			}
			// TODO(v7.0): wire report.Builder once internal/report is implemented.
			fmt.Fprintf(cmd.OutOrStdout(),
				"render: %s → %s (format=%s locale=%s)\n",
				input, output, format, locale)
			return nil
		},
	}
	c.Flags().StringVar(&input, "input", "", "path to scan NDJSON file")
	c.Flags().StringVar(&format, "format", "html", "html | sarif | pdf | txt")
	c.Flags().StringVarP(&output, "output", "o", "", "output file path")
	c.Flags().StringVar(&locale, "locale", "auto", "id | en | auto")
	return c
}

func newReportSummaryCmd() *cobra.Command {
	var input string
	c := &cobra.Command{
		Use:   "summary",
		Short: "Print a one-screen summary of a scan NDJSON file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if input == "" {
				return fmt.Errorf("--input is required")
			}
			// TODO(v7.0): aggregate findings, print TUI-style summary.
			fmt.Fprintf(cmd.OutOrStdout(), "summary: %s\n", input)
			return nil
		},
	}
	c.Flags().StringVar(&input, "input", "", "path to scan NDJSON file")
	return c
}
