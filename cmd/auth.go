package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/optikklabs/optikk/internal/clierr"
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
		newAuthLogoutCmd(app),
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
			if (email == "" || password == "") && !allowPrompt(cmd, app) {
				return clierr.New(clierr.Usage,
					"missing required flags for non-interactive login: --email --password",
					"pass both flags, or use the browser flow: optikk login")
			}
			apiBase, err := app.API()
			if err != nil {
				return err
			}
			client := apiclient.New(apiBase)
			token, err := client.Login(cmd.Context(), email, password)
			if err != nil {
				return fmt.Errorf("login failed: %w", err)
			}
			if err := apiclient.SaveToken(apiBase, token); err != nil {
				return err
			}
			doc := authDoc{Status: "authenticated", Email: email, API: apiBase, TokenPath: tokenPath}
			return writeResult(cmd, app, doc, func(w io.Writer) {
				fmt.Fprintf(w, "✓ Authenticated as %s\n", email)
				fmt.Fprintf(w, "  API: %s\n", apiBase)
				fmt.Fprintf(w, "  Token cached at %s\n", tokenPath)
			})
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "account email")
	cmd.Flags().StringVar(&password, "password", "", "account password")
	return cmd
}

// tokenPath is where the session JWT is cached, as shown to users and agents.
const tokenPath = "~/.optikk/config.json"

// authDoc is the machine-readable result of auth login/logout.
type authDoc struct {
	Status    string `json:"status"`
	Email     string `json:"email,omitempty"`
	API       string `json:"api,omitempty"`
	TokenPath string `json:"token_path,omitempty"`
}

func newAuthStatusCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return writeResult(cmd, app, currentSession(app).doc(), func(w io.Writer) {
				printSession(w, app, "")
			})
		},
	}
}

func newAuthLogoutCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear cached authentication",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := apiclient.ClearToken(); err != nil {
				return err
			}
			return writeResult(cmd, app, authDoc{Status: "logged_out"}, func(w io.Writer) {
				fmt.Fprintln(w, "✓ Logged out (token cleared)")
			})
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
