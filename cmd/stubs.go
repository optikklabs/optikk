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

