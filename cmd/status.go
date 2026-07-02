package cmd

import (
	"github.com/optikklabs/optikk/internal/config"
	"github.com/optikklabs/optikk/internal/status"
	"github.com/optikklabs/optikk/internal/target"
	"github.com/spf13/cobra"
)

func newStatusCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:     "status",
		Aliases: []string{"ps", "pods"},
		Short:   "List Optikk pod readiness",
		Example: "  optikk status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			conn, err := target.Resolve(app.Cfg)
			if err != nil {
				return err
			}
			return status.Print(cmd.Context(), conn.Kube, config.Namespace, cmd.OutOrStdout())
		},
	}
}
