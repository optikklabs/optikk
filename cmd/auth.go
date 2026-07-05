package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/optikklabs/optikk/internal/conn"
	"github.com/spf13/cobra"
)

func newAuthCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "auth",
		Short:       "Authenticate with the Optikk API",
		Long:        "Manage authentication sessions for data commands (traces, logs, metrics, dashboards, monitors).",
		Example:     "  optikk auth login\n  optikk auth login --email user@example.com --password '...'\n  optikk auth status\n  optikk auth logout",
		Annotations: map[string]string{annotationSkipDeploy: "true"},
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
		Long:  "Logs in via POST /api/v1/auth/login, caches the JWT at ~/.optikk/token.json.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			apiBase := conn.Resolve(app.Cfg.ApiURL)
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
			fmt.Fprintf(cmd.OutOrStdout(), "  Token cached at ~/.optikk/token.json\n")
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", defaultAdminEmail, "account email")
	cmd.Flags().StringVar(&password, "password", defaultAdminPassword, "account password")
	return cmd
}

func newAuthStatusCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Try OPTIKK_TOKEN env first, then cached file.
			token := app.Cfg.Token
			apiBase := conn.Resolve(app.Cfg.ApiURL)
			source := "OPTIKK_TOKEN"

			if token == "" {
				base, tok, err := apiclient.LoadToken()
				if err != nil {
					fmt.Fprintln(cmd.OutOrStdout(), "✗ Not authenticated")
					fmt.Fprintln(cmd.OutOrStdout(), "  Run: optikk auth login")
					return nil
				}
				token = tok
				apiBase = base
				source = "~/.optikk/token.json"
			}

			// Decode JWT payload (no verification — just display).
			claims := decodeJWTClaims(token)
			email, _ := claims["email"].(string)
			exp, _ := claims["exp"].(float64)

			fmt.Fprintf(cmd.OutOrStdout(), "✓ Authenticated\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  Source: %s\n", source)
			fmt.Fprintf(cmd.OutOrStdout(), "  API:    %s\n", apiBase)
			if email != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  Email:  %s\n", email)
			}
			if exp > 0 {
				expTime := time.Unix(int64(exp), 0)
				if time.Now().After(expTime) {
					fmt.Fprintf(cmd.OutOrStdout(), "  Expiry: %s (EXPIRED)\n", expTime.Format(time.RFC3339))
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "  Expiry: %s\n", expTime.Format(time.RFC3339))
				}
			}
			if app.Cfg.TenantID > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "  Tenant:   %d\n", app.Cfg.TenantID)
			}
			return nil
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear cached authentication",
		RunE: func(cmd *cobra.Command, _ []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			path := home + "/.optikk/token.json"
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
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
