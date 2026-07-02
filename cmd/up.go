package cmd

import (
	"time"

	"github.com/optikklabs/optikk/internal/provision"
	"github.com/spf13/cobra"
)

func newUpCmd(app *App) *cobra.Command {
	var (
		managePodman    bool
		loadLocalImages bool
		timeout         time.Duration
	)
	cmd := &cobra.Command{
		Use:     "up",
		Aliases: []string{"start", "deploy"},
		Short:   "Provision infra and deploy the full stack",
		Example: "  optikk up\n  optikk up --manage-podman\n  optikk up --load-local-images --timeout 15m",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p := provision.NewLocal(provision.LocalOptions{
				DeployDir:       app.DeployDir,
				ManagePodman:    managePodman,
				LoadLocalImages: loadLocalImages,
				Timeout:         timeout,
				Out:             cmd.OutOrStdout(),
			})
			return p.Up(cmd.Context())
		},
	}
	cmd.Flags().BoolVar(&managePodman, "manage-podman", false, "let the CLI resize/start the Podman machine instead of instructing")
	cmd.Flags().BoolVar(&loadLocalImages, "load-local-images", false, "build ingest/query and load them into kind (private ghcr packages)")
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Minute, "rollout/create timeout")
	return cmd
}
