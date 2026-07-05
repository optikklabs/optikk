package cmd

import (
	"github.com/optikklabs/optikk/internal/target"
	"github.com/optikklabs/optikk/internal/tenant"
	"github.com/spf13/cobra"
)

func newTenantCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tenant",
		Aliases: []string{"tenants", "collector", "collectors"},
		Short:   "Manage per-tenant collectors and members",
		Example: "  optikk tenant onboard 1 --key <api-key>\n  optikk tenant offboard 1\n  optikk tenant member add user@example.com --tenant 1 --password 'Secret123!'",
	}
	cmd.AddCommand(newTenantOnboardCmd(app), newTenantOffboardCmd(app), newTenantMemberCmd(app))
	return cmd
}

func newTenantOnboardCmd(app *App) *cobra.Command {
	var key string
	cmd := &cobra.Command{
		Use:     "onboard <id>",
		Aliases: []string{"create", "add"},
		Short:   "Provision a per-tenant collector",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			conn, err := target.Resolve(app.Cfg)
			if err != nil {
				return err
			}
			return tenant.Onboard(cmd.Context(), conn.Kube, app.DeployDir, args[0], key, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&key, "key", "", "tenant api key (from tenant create) (required)")
	cmd.MarkFlagRequired("key")
	return cmd
}

func newTenantOffboardCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:     "offboard <id>",
		Aliases: []string{"delete", "remove", "rm"},
		Short:   "Remove a per-tenant collector",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			conn, err := target.Resolve(app.Cfg)
			if err != nil {
				return err
			}
			return tenant.Offboard(cmd.Context(), conn.Kube, app.DeployDir, args[0], cmd.OutOrStdout())
		},
	}
}
