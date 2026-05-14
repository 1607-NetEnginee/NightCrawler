package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/HnyBadger/nightcrawler/internal/plugin"
)

// newPluginCmd builds the `plugin` subcommand group: list, info,
// validate. Remote plugin install (v7.1+) is intentionally absent here
// to keep the v7.0 GA surface honest.
func newPluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Inspect and manage scanner plugins",
	}
	cmd.AddCommand(newPluginListCmd())
	cmd.AddCommand(newPluginInfoCmd())
	return cmd
}

func newPluginListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all built-in plugins with their categories",
		RunE: func(cmd *cobra.Command, args []string) error {
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			defer tw.Flush()

			fmt.Fprintln(tw, "NAME\tVERSION\tCATEGORY\tDEPENDS-ON\tDESCRIPTION")
			for _, p := range plugin.Registry().All() {
				m := p.Manifest()
				deps := "-"
				if len(m.DependsOn) > 0 {
					deps = fmt.Sprintf("%v", m.DependsOn)
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
					m.Name, m.Version, m.Category, deps, m.Description)
			}
			return nil
		},
	}
}

func newPluginInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <plugin-name>",
		Short: "Show full manifest of a single plugin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, ok := plugin.Registry().Get(args[0])
			if !ok {
				return fmt.Errorf("plugin %q not found", args[0])
			}
			m := p.Manifest()
			fmt.Fprintf(cmd.OutOrStdout(), "Name:        %s\n", m.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "Version:     %s\n", m.Version)
			fmt.Fprintf(cmd.OutOrStdout(), "Author:      %s\n", m.Author)
			fmt.Fprintf(cmd.OutOrStdout(), "Category:    %s\n", m.Category)
			fmt.Fprintf(cmd.OutOrStdout(), "Profile:     %s\n", m.Profile)
			fmt.Fprintf(cmd.OutOrStdout(), "Tags:        %v\n", m.Tags)
			fmt.Fprintf(cmd.OutOrStdout(), "Depends-on:  %v\n", m.DependsOn)
			fmt.Fprintf(cmd.OutOrStdout(), "CWE:         %v\n", m.CWE)
			fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", m.Description)
			return nil
		},
	}
}
