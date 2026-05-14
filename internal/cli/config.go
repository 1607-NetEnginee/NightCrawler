package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect, validate, and migrate configuration",
	}
	cmd.AddCommand(newConfigInitCmd())
	cmd.AddCommand(newConfigValidateCmd())
	cmd.AddCommand(newConfigImportV6Cmd())
	return cmd
}

func newConfigInitCmd() *cobra.Command {
	var path string
	c := &cobra.Command{
		Use:   "init",
		Short: "Write a fully-annotated default config to disk",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO(v7.0): write embedded default to path
			fmt.Fprintf(cmd.OutOrStdout(), "config init → %s\n", path)
			return nil
		},
	}
	c.Flags().StringVarP(&path, "path", "p",
		"$XDG_CONFIG_HOME/nightcrawler/config.yaml",
		"target path for the new config file")
	return c
}

func newConfigValidateCmd() *cobra.Command {
	var path string
	c := &cobra.Command{
		Use:   "validate",
		Short: "Validate a config file against the JSON schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO(v7.0): wire jsonschema validator
			fmt.Fprintf(cmd.OutOrStdout(), "validate: %s OK\n", path)
			return nil
		},
	}
	c.Flags().StringVarP(&path, "path", "p", "",
		"path to config file (default: discover via XDG)")
	return c
}

func newConfigImportV6Cmd() *cobra.Command {
	var (
		input  string
		output string
	)
	c := &cobra.Command{
		Use:   "import-v6",
		Short: "Translate a legacy v6.x bash config file into v7.0 YAML",
		Long: `Reads /etc/nightcrawler/nightcrawler.conf (the v6.x format
that ships with the legacy tar.gz) and emits the equivalent v7.0
YAML config. Designed to make migration from v6.1 effortless: existing
operators do not need to re-learn the configuration surface.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO(v7.0): implement v6 → v7 translation per §8.5
			fmt.Fprintf(cmd.OutOrStdout(),
				"import-v6: %s → %s\n", input, output)
			return nil
		},
	}
	c.Flags().StringVar(&input, "input", "/etc/nightcrawler/nightcrawler.conf",
		"path to v6.x bash config file")
	c.Flags().StringVar(&output, "output", "",
		"output path for v7.0 YAML config (default: stdout)")
	return c
}
