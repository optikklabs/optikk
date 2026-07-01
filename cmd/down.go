package cmd

import (
	"time"

	"github.com/optikklabs/optikk/internal/config"
	"github.com/optikklabs/optikk/internal/provision"
	"github.com/spf13/cobra"
)

func newDownCmd(app *App) *cobra.Command {
	var (
		keepCluster bool
		g           gcpFlags
	)
	cmd := &cobra.Command{
		Use:   "down",
		Short: "Tear down the stack and the cluster it created for --target",
		RunE: func(cmd *cobra.Command, _ []string) error {
			switch app.Cfg.Target {
			case config.TargetLocal:
				p := provision.NewLocal(provision.LocalOptions{
					DeployDir:   app.DeployDir,
					KeepCluster: keepCluster,
					Out:         cmd.OutOrStdout(),
				})
				return p.Down(cmd.Context())
			case config.TargetGCP:
				opts := g.options(app, 20*time.Minute)
				opts.Out = cmd.OutOrStdout()
				return provision.NewGCP(opts).Down(cmd.Context())
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&keepCluster, "keep-cluster", false, "delete the stack but keep the kind cluster")
	g.register(cmd)
	return cmd
}
