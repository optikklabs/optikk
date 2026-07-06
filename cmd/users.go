package cmd

import (
	"fmt"

	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/spf13/cobra"
)

// adminClient builds an API client from the cached admin session.
func adminClient() (*apiclient.Client, error) {
	apiBase, token, err := apiclient.LoadToken()
	if err != nil {
		return nil, err
	}
	c := apiclient.New(apiBase)
	c.SetToken(token)
	return c, nil
}

func newUsersCmd(_ *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "users",
		Aliases: []string{"user", "member", "members"},
		Short:   "Manage users",
	}

	var (
		tenantID int64
		password string
		name     string
		role     string
	)
	add := &cobra.Command{
		Use:     "add <email>",
		Aliases: []string{"create"},
		Short:   "Add a user to a tenant",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := adminClient()
			if err != nil {
				return err
			}
			email := args[0]
			if name == "" {
				name = email
			}
			user, err := client.CreateUser(cmd.Context(), apiclient.CreateUserRequest{
				Email: email, Name: name, Password: password, Role: role, TenantID: tenantID,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "user_id: %d  email: %s  tenant: %d\n", user.ID, user.Email, tenantID)
			return nil
		},
	}
	add.Flags().Int64Var(&tenantID, "tenant", 0, "tenant id to add the user to (required)")
	add.Flags().StringVar(&password, "password", "", "user password (required)")
	add.Flags().StringVar(&name, "name", "", "user display name (default: email)")
	add.Flags().StringVar(&role, "role", "", "user role")
	add.MarkFlagRequired("tenant")
	add.MarkFlagRequired("password")
	cmd.AddCommand(add)
	return cmd
}
