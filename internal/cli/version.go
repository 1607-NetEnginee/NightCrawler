package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/1607-NetEnginee/NightCrawler/internal/version"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print detailed version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(),
				"NIGHTCRAWLER v%s\n"+
					"  commit:     %s\n"+
					"  built:      %s\n"+
					"  go:         %s\n"+
					"  author:     HnyBadger / Cyberoutcast\n"+
					"  license:    Apache-2.0\n",
				version.Version, version.Commit, version.BuildDate, version.GoVersion())
			return nil
		},
	}
}
