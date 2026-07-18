package cmd

import (
	"fmt"
	"io"

	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/spf13/cobra"
)

func newOnboardCmd(app *App) *cobra.Command {
	var si signupInput
	cmd := &cobra.Command{
		Use:   "onboard",
		Short: "Sign up and get your API key + OTLP endpoint",
		Long: "Signs you up (or reuses your cached session), then prints your API key\n" +
			"and the OTEL_EXPORTER_OTLP_* snippet to point your SDK at.",
		Example: "  optikk onboard\n" +
			"  optikk onboard --email founder@startup.dev --password '…' --name Founder --org \"Acme Platform\" --accept-terms",
		RunE: func(cmd *cobra.Command, _ []string) error {
			apiBase, err := app.API()
			if err != nil {
				return err
			}
			client := apiclient.New(apiBase)

			doc := onboardDoc{Status: "ready", Next: "optikk verify --from 15m"}
			if cachedBase, tok, err := apiclient.LoadToken(); err == nil && cachedBase == apiBase {
				client.SetToken(tok)
				// The raw API key is only ever returned at signup or rotate; an
				// existing session cannot fetch it again.
				doc.Account = "cached_session"
				doc.Hint = "existing session has no retrievable API key — run: optikk keys rotate (invalidates the old key)"
			} else {
				res, err := signupInteractive(cmd, client, apiBase, si, allowPrompt(cmd, app))
				if err != nil {
					return err
				}
				doc.Account = "created"
				doc.Tenant = &tenantInfo{ID: res.Tenant.ID, Name: res.Tenant.Name}
				doc.APIKey = res.APIKey
				doc.OTLP = newOTLPInfo(apiBase, res.APIKey)
			}

			return writeResult(cmd, app, doc, func(w io.Writer) {
				if doc.Account == "cached_session" {
					fmt.Fprintln(w, "Using your cached session (run `optikk auth login` to switch accounts).")
					fmt.Fprintln(w, "Your API key was shown at signup and cannot be re-read; to issue a fresh")
					fmt.Fprintln(w, "one (invalidating the old), run: optikk keys rotate")
					return
				}
				fmt.Fprintf(w, "✓ Account created — tenant %s (id %d)\n", doc.Tenant.Name, doc.Tenant.ID)
				printOTLPSnippet(w, doc.OTLP)
				fmt.Fprintln(w, "\nThen confirm data is flowing with: optikk verify --from 15m")
			})
		},
	}
	addSignupFlags(cmd, &si)
	return cmd
}

// onboardDoc is the machine-readable onboarding result.
type onboardDoc struct {
	Status  string      `json:"status"`
	Account string      `json:"account"` // created | cached_session
	Tenant  *tenantInfo `json:"tenant,omitempty"`
	APIKey  string      `json:"api_key,omitempty"`
	OTLP    *otlpInfo   `json:"otlp,omitempty"`
	Next    string      `json:"next"`
	Hint    string      `json:"hint,omitempty"`
}
