package cmd

import (
	"time"

	"github.com/optikklabs/optikk/internal/config"
	"github.com/optikklabs/optikk/internal/provision"
	"github.com/spf13/cobra"
)

func newUpCmd(app *App) *cobra.Command {
	var (
		managePodman    bool
		loadLocalImages bool
		timeout         time.Duration
		g               gcpFlags
	)
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Provision infra from scratch and deploy the full stack for --target",
		RunE: func(cmd *cobra.Command, _ []string) error {
			switch app.Cfg.Target {
			case config.TargetLocal:
				p := provision.NewLocal(provision.LocalOptions{
					DeployDir:       app.DeployDir,
					ManagePodman:    managePodman,
					LoadLocalImages: loadLocalImages,
					Timeout:         timeout,
					Out:             cmd.OutOrStdout(),
				})
				return p.Up(cmd.Context())
			case config.TargetGCP:
				opts := g.options(app, timeout)
				opts.Out = cmd.OutOrStdout()
				return provision.NewGCP(opts).Up(cmd.Context())
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&managePodman, "manage-podman", false, "let the CLI resize/start the Podman machine instead of instructing")
	cmd.Flags().BoolVar(&loadLocalImages, "load-local-images", false, "build ingest/query and load them into kind (private ghcr packages)")
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Minute, "rollout/create timeout")
	g.register(cmd)
	return cmd
}
