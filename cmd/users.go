package cmd

import (
	"fmt"
	"io"
	"strconv"

	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/optikklabs/optikk/internal/clierr"
	"github.com/spf13/cobra"
)

// adminClient builds an API client from the cached admin session.
func adminClient() (*apiclient.Client, error) {
	apiBase, token, err := apiclient.LoadToken()
	if err != nil {
		return nil, clierr.New(clierr.Auth, "not authenticated",
			"run: optikk login (or set OPTIKK_TOKEN)")
	}
	c := apiclient.New(apiBase)
	c.SetToken(token)
	return c, nil
}

func newUsersCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "users",
		Aliases: []string{"user", "member", "members"},
		Short:   "Manage your tenant's users",
	}
	cmd.AddCommand(newUsersAddCmd(app), newUsersListCmd(app))
	return cmd
}

// userDoc is the machine-readable result of `users add`.
type userDoc struct {
	UserID   int64  `json:"user_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Role     string `json:"role,omitempty"`
	TenantID int64  `json:"tenant_id"`
	// Invited is true when no password was set: the server emails the user a
	// set-password link instead.
	Invited bool `json:"invited"`
}

func newUsersAddCmd(app *App) *cobra.Command {
	var password, name, role string
	cmd := &cobra.Command{
		Use:     "add <email>",
		Aliases: []string{"create", "invite"},
		Short:   "Add a user to your tenant",
		Long: "Creates a user in your tenant (admin only). Without --password the server\n" +
			"emails the user a set-password link, making this an invite.",
		Args: cobra.ExactArgs(1),
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
				Email: email, Name: name, Password: password, Role: role,
			})
			if err != nil {
				return err
			}
			doc := userDoc{
				UserID: user.ID, Email: user.Email, Name: user.Name,
				Role: user.Role, TenantID: user.TenantID, Invited: password == "",
			}
			return writeResult(cmd, app, doc, func(w io.Writer) {
				fmt.Fprintf(w, "✓ Added %s (user id %d) to tenant %d\n", doc.Email, doc.UserID, doc.TenantID)
				if doc.Invited {
					fmt.Fprintln(w, "  They will receive an email with a set-password link.")
				}
			})
		},
	}
	cmd.Flags().StringVar(&password, "password", "", "user password (omit to email a set-password link)")
	cmd.Flags().StringVar(&name, "name", "", "user display name (default: email)")
	cmd.Flags().StringVar(&role, "role", "", "user role: admin|member (default: member)")
	return cmd
}

func newUsersListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List your tenant's users",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := adminClient()
			if err != nil {
				return err
			}
			users, err := client.ListUsers(cmd.Context())
			if err != nil {
				return err
			}
			items := make([]any, len(users))
			for i, u := range users {
				items[i] = u
			}
			return resolveOutput(cmd, app).WriteItems(
				[]string{"ID", "EMAIL", "NAME", "ROLE", "ACTIVE"}, items,
				func(v any) []string {
					u := v.(apiclient.User)
					return []string{strconv.FormatInt(u.ID, 10), u.Email, u.Name, u.Role, strconv.FormatBool(u.Active)}
				})
		},
	}
}
