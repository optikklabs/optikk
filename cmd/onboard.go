package cmd

import (
	"fmt"

	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/spf13/cobra"
)

func newOnboardCmd(app *App) *cobra.Command {
	var email, password, name, org string
	cmd := &cobra.Command{
		Use:   "onboard",
		Short: "Sign up and get your API key + OTLP endpoint",
		Long: "Signs you up (or reuses your cached session), then prints your API key\n" +
			"and the OTEL_EXPORTER_OTLP_* snippet to point your SDK at.",
		Example: "  optikk onboard\n  optikk onboard --org \"Acme Platform\"",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			out := cmd.OutOrStdout()
			apiBase, err := app.API()
			if err != nil {
				return err
			}
			client := apiclient.New(apiBase)

			if cachedBase, tok, err := apiclient.LoadToken(); err == nil && cachedBase == apiBase {
				client.SetToken(tok)
				fmt.Fprintln(out, "Using your cached session (run `optikk auth login` to switch accounts).")
			} else {
				res, err := signupInteractive(cmd, client, apiBase, signupInput{
					Email: email, Password: password, Name: name, Org: org,
				})
				if err != nil {
					return err
				}
				fmt.Fprintf(out, "✓ Account created — tenant %s (id %d)\n", res.Tenant.Name, res.Tenant.ID)
			}

			// API key works for cached sessions too, unlike the signup result.
			st, err := client.GetOnboardingStatus(ctx)
			if err != nil {
				return fmt.Errorf("loading tenant: %w (if your session expired, run `optikk auth login` and retry)", err)
			}

			fmt.Fprintln(out, "Point your OpenTelemetry SDK at Optikk:")
			fmt.Fprintf(out, "  OTEL_EXPORTER_OTLP_ENDPOINT=%s\n", otlpEndpoint(apiBase))
			fmt.Fprintf(out, "  OTEL_EXPORTER_OTLP_HEADERS=x-api-key=%s\n", st.APIKey)
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "account email (prompted if omitted)")
	cmd.Flags().StringVar(&password, "password", "", "account password (prompted if omitted)")
	cmd.Flags().StringVar(&name, "name", "", "your full name (prompted if omitted)")
	cmd.Flags().StringVar(&org, "org", "", "organization name, becomes your tenant (prompted if omitted)")
	return cmd
}
