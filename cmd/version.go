package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version is the CLI version, overridable at build time via -ldflags.
var version = "0.1.0-dev"

func newVersionCmd(_ *App) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the optikk CLI version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), version)
			return nil
		},
	}
}
