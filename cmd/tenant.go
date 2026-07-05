package cmd

import (
	"github.com/spf13/cobra"
)

func newTenantCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tenant",
		Aliases: []string{"tenants"},
		Short:   "Manage tenant members",
		Example: "  optikk tenant member add user@example.com --tenant 1 --password 'Secret123!'",
	}
	cmd.AddCommand(newTenantMemberCmd(app))
	return cmd
}
