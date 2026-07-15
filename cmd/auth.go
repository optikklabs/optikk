package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/spf13/cobra"
)

func newAuthCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "auth",
		Short:   "Authenticate with the Optikk API",
		Long:    "Manage authentication sessions for data commands (traces, logs, metrics, dashboards, monitors).",
		Example: "  optikk auth login\n  optikk auth login --email user@example.com --password '...'\n  optikk auth status\n  optikk auth logout",
	}
	cmd.AddCommand(
		newAuthLoginCmd(app),
		newAuthStatusCmd(app),
		newAuthLogoutCmd(),
	)
	return cmd
}

func newAuthLoginCmd(app *App) *cobra.Command {
	var email, password string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate and cache a session JWT",
		Long:  "Logs in via POST /api/v1/auth/login, caches the JWT at ~/.optikk/config.json.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			apiBase, err := app.API()
			if err != nil {
				return err
			}
			client := apiclient.New(apiBase)
			token, err := client.Login(cmd.Context(), email, password)
			if err != nil {
				return fmt.Errorf("login failed: %w\n\nIs the API running at %s?", err, apiBase)
			}
			if err := apiclient.SaveToken(apiBase, token); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Authenticated as %s\n", email)
			fmt.Fprintf(cmd.OutOrStdout(), "  API: %s\n", apiBase)
			fmt.Fprintf(cmd.OutOrStdout(), "  Token cached at ~/.optikk/config.json\n")
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "account email")
	cmd.Flags().StringVar(&password, "password", "", "account password")
	return cmd
}

func newAuthStatusCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			printSession(cmd.OutOrStdout(), app, "")
			return nil
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear cached authentication",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := apiclient.ClearToken(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "✓ Logged out (token cleared)")
			return nil
		},
	}
}

// decodeJWTClaims decodes the payload of a JWT without verification.
func decodeJWTClaims(token string) map[string]any {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil
	}
	payload := parts[1]
	// Add padding if needed.
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}
	b, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil
	}
	var claims map[string]any
	json.Unmarshal(b, &claims)
	return claims
}
