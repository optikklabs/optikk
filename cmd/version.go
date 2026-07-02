package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version is the CLI version, overridable at build time via -ldflags.
var (
	version = "0.1.0-dev"
	commit  = "unknown"
	date    = "unknown"
)

func newVersionCmd(_ *App) *cobra.Command {
	return &cobra.Command{
		Use:         "version",
		Aliases:     []string{"v"},
		Short:       "Print build version information",
		Annotations: map[string]string{annotationNoConfig: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "optikk %s\ncommit %s\nbuilt  %s\n", version, commit, date)
			return nil
		},
	}
}
