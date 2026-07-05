package cmd

import (
	"time"

	"github.com/optikklabs/optikk/internal/provision"
	"github.com/spf13/cobra"
)

func newUpCmd(app *App) *cobra.Command {
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:     "init",
		Aliases: []string{"start", "deploy", "up"},
		Short:   "Provision infra and deploy the full stack",
		Example: "  optikk init\n  optikk init --timeout 15m",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p := provision.NewLocal(provision.LocalOptions{
				DeployDir: app.DeployDir,
				Timeout:   timeout,
				Out:       cmd.OutOrStdout(),
			})
			return p.Up(cmd.Context())
		},
	}
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Minute, "rollout/create timeout")
	return cmd
}
