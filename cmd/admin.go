package cmd

import (
	"fmt"

	"github.com/optikklabs/optikk/internal/adminseed"
	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/optikklabs/optikk/internal/config"
	"github.com/optikklabs/optikk/internal/target"
	"github.com/spf13/cobra"
)

// default seeded super-admin (deploy query-secret defaults).
const (
	defaultAdminEmail    = "admin@optikk.dev"
	defaultAdminPassword = "Password123!"
)

func newAdminCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "admin",
		Aliases: []string{"auth"},
		Short:   "Set up and log in as platform admin",
		Example: "  optikk admin setup\n  optikk admin login\n  optikk admin login --email admin@optikk.dev --password 'Password123!'",
	}
	cmd.AddCommand(newAdminSetupCmd(app), newAdminLoginCmd(app))
	return cmd
}

func newAdminSetupCmd(app *App) *cobra.Command {
	var email, password string
	cmd := &cobra.Command{
		Use:     "setup",
		Aliases: []string{"seed"},
		Short:   "Set query super-admin credentials and reseed",
		RunE: func(cmd *cobra.Command, _ []string) error {
			conn, err := target.Resolve(app.Cfg)
			if err != nil {
				return err
			}
			if err := adminseed.SetCredentials(cmd.Context(), conn.Kube, config.Namespace, email, password); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "query-secret updated and query restarted for %s\n", email)
			fmt.Fprintln(cmd.OutOrStdout(), "note: query seeds create-if-absent — an existing admin's password is unchanged")
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", defaultAdminEmail, "super-admin email")
	cmd.Flags().StringVar(&password, "password", defaultAdminPassword, "super-admin password")
	return cmd
}

func newAdminLoginCmd(app *App) *cobra.Command {
	var email, password string
	cmd := &cobra.Command{
		Use:     "login",
		Aliases: []string{"signin"},
		Short:   "Cache an admin session for team commands",
		RunE: func(cmd *cobra.Command, _ []string) error {
			conn, err := target.Resolve(app.Cfg)
			if err != nil {
				return err
			}
			client := apiclient.New(conn.APIBase)
			token, err := client.Login(cmd.Context(), email, password)
			if err != nil {
				return err
			}
			if err := apiclient.SaveToken(conn.APIBase, token); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "admin session cached for %s\n", email)
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", defaultAdminEmail, "admin email")
	cmd.Flags().StringVar(&password, "password", defaultAdminPassword, "admin password")
	return cmd
}
