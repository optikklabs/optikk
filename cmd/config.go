package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newConfigCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect CLI configuration",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Print the merged config (flags + file + env)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "target:     %s\n", app.Cfg.Target)
			fmt.Fprintf(w, "deploy_dir: %s\n", app.DeployDir)
			fmt.Fprintf(w, "verbose:    %t\n", app.Cfg.Verbose)
			fmt.Fprintf(w, "gcp.project: %s\n", app.Cfg.GCP.Project)
			fmt.Fprintf(w, "gcp.region:  %s\n", app.Cfg.GCP.Region)
			fmt.Fprintf(w, "gcp.machine_type: %s\n", app.Cfg.GCP.MachineType)
			return nil
		},
	})
	return cmd
}
