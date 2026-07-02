package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newConfigCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "config",
		Aliases: []string{"cfg"},
		Short:   "Inspect CLI configuration",
	}
	cmd.AddCommand(&cobra.Command{
		Use:     "show",
		Aliases: []string{"view"},
		Short:   "Print the merged config",
		RunE: func(cmd *cobra.Command, _ []string) error {
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "target:     %s\n", app.Cfg.Target)
			fmt.Fprintf(w, "deploy_dir: %s\n", app.DeployDir)
			fmt.Fprintf(w, "verbose:    %t\n", app.Cfg.Verbose)
			return nil
		},
	})
	return cmd
}
