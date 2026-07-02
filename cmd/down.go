package cmd

import (
	"github.com/optikklabs/optikk/internal/provision"
	"github.com/spf13/cobra"
)

func newDownCmd(app *App) *cobra.Command {
	var keepCluster bool
	cmd := &cobra.Command{
		Use:     "down",
		Aliases: []string{"stop", "destroy"},
		Short:   "Tear down the stack and local cluster",
		Example: "  optikk down\n  optikk down --keep-cluster",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p := provision.NewLocal(provision.LocalOptions{
				DeployDir:   app.DeployDir,
				KeepCluster: keepCluster,
				Out:         cmd.OutOrStdout(),
			})
			return p.Down(cmd.Context())
		},
	}
	cmd.Flags().BoolVar(&keepCluster, "keep-cluster", false, "delete the stack but keep the kind cluster")
	return cmd
}
