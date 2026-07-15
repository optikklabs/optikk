package cmd

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/spf13/cobra"
)

func newWhoamiCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:     "whoami",
		Short:   "Show who you are signed in as",
		Long:    "Prints the account, tenant, and API the current session belongs to.",
		Args:    cobra.NoArgs,
		Aliases: []string{"me"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			printSession(cmd.OutOrStdout(), app, "")
			return nil
		},
	}
}

// firstLine reduces a multi-line error to its headline. The long form, with
// its remediation steps, belongs in a command that fails — not squeezed into
// a field of a status report.
func firstLine(err error) string {
	line, _, _ := strings.Cut(err.Error(), "\n")
	return line
}

// session is the resolved identity behind the current invocation.
type session struct {
	Authenticated bool
	Source        string // where the token came from
	APIBase       string
	Email         string
	TenantID      int64
	Expiry        time.Time
}

// currentSession resolves the active session without contacting the API. The
// JWT is decoded, not verified — the server is the authority on validity; this
// only reports what the CLI is holding.
func currentSession(app *App) session {
	s := session{APIBase: app.APIBase, TenantID: app.Cfg.TenantID}
	if _, err := app.API(); err != nil {
		s.APIBase = fmt.Sprintf("(unusable — %s)", firstLine(err))
	}

	token := app.Cfg.Token
	s.Source = "OPTIKK_TOKEN"
	if token == "" {
		ctx, err := apiclient.CurrentContext()
		if err != nil || ctx.Token == "" {
			return s
		}
		token, s.Source = ctx.Token, "~/.optikk/config.json"
	}

	s.Authenticated = true
	claims := decodeJWTClaims(token)
	s.Email, _ = claims["email"].(string)
	if exp, ok := claims["exp"].(float64); ok && exp > 0 {
		s.Expiry = time.Unix(int64(exp), 0)
	}
	return s
}

// Expired reports whether the cached token's own expiry has passed.
func (s session) Expired() bool {
	return !s.Expiry.IsZero() && time.Now().After(s.Expiry)
}

// printSession renders the shared identity block used by `whoami`, `status`,
// and `auth status`, so the three never drift apart. indent prefixes every
// line, letting `status` nest it under a heading.
func printSession(w io.Writer, app *App, indent string) {
	s := currentSession(app)
	if !s.Authenticated {
		fmt.Fprintf(w, "%s✗ Not authenticated\n", indent)
		fmt.Fprintf(w, "%s  Run: optikk login\n", indent)
		return
	}

	fmt.Fprintf(w, "%s✓ Authenticated\n", indent)
	fmt.Fprintf(w, "%s  Source: %s\n", indent, s.Source)
	fmt.Fprintf(w, "%s  API:    %s\n", indent, s.APIBase)
	if s.Email != "" {
		fmt.Fprintf(w, "%s  Email:  %s\n", indent, s.Email)
	}
	if s.TenantID > 0 {
		fmt.Fprintf(w, "%s  Tenant: %d\n", indent, s.TenantID)
	}
	if !s.Expiry.IsZero() {
		suffix := ""
		if s.Expired() {
			suffix = " (EXPIRED — run: optikk login)"
		}
		fmt.Fprintf(w, "%s  Expiry: %s%s\n", indent, s.Expiry.Format(time.RFC3339), suffix)
	}
}
