package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// notImplemented marks a command whose behavior lands in a later milestone.
func notImplemented(milestone string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		return fmt.Errorf("%q is not implemented yet (%s)", cmd.CommandPath(), milestone)
	}
}

func newTenantCmd(_ *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tenant",
		Short: "Manage per-tenant otel-collectors",
	}
	cmd.AddCommand(
		&cobra.Command{Use: "onboard <slug>", Short: "Provision a per-tenant otel-collector", RunE: notImplemented("M4")},
		&cobra.Command{Use: "offboard <slug>", Short: "Remove a per-tenant otel-collector", RunE: notImplemented("M4")},
	)
	return cmd
}

func newAdminCmd(_ *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Platform super-admin setup and login",
	}
	cmd.AddCommand(
		&cobra.Command{Use: "setup", Short: "Seed the platform super-admin on the query deployment", RunE: notImplemented("M5")},
		&cobra.Command{Use: "login", Short: "Obtain and cache an admin JWT", RunE: notImplemented("M5")},
	)
	return cmd
}

func newTeamCmd(_ *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "team",
		Short: "Create teams and add members",
	}
	member := &cobra.Command{Use: "member", Short: "Manage team members"}
	member.AddCommand(&cobra.Command{Use: "add <email>", Short: "Add a member to a team", RunE: notImplemented("M5")})
	cmd.AddCommand(
		&cobra.Command{Use: "create <name>", Short: "Create a team (returns team_id + api_key)", RunE: notImplemented("M5")},
		member,
	)
	return cmd
}
