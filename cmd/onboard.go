package cmd

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/optikklabs/optikk/internal/conn"
	"github.com/spf13/cobra"
)

func newOnboardCmd(app *App) *cobra.Command {
	var email, password, name, org string
	var managed, local bool
	cmd := &cobra.Command{
		Use:   "onboard",
		Short: "Sign up and get your API key + OTLP endpoint",
		Long: "Signs you up (or reuses your cached session) against the local cluster\n" +
			"(--local, default) or the hosted Optikk (--managed), then prints your API\n" +
			"key and the OTEL_EXPORTER_OTLP_* snippet to point your SDK at.\n\n" +
			"Bring the local stack up with `optikk init`; provision a per-tenant\n" +
			"collector with `optikk tenant onboard`.",
		Example:     "  optikk onboard\n  optikk onboard --managed\n  optikk onboard --org \"Acme Platform\"",
		Annotations: map[string]string{annotationSkipDeploy: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			out := cmd.OutOrStdout()
			apiBase := conn.Resolve(app.Cfg.ApiURL)
			if managed && app.Cfg.ApiURL == "" {
				apiBase = conn.ManagedAPIURL
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
	cmd.Flags().BoolVar(&managed, "managed", false, "sign up against the hosted Optikk ("+conn.ManagedAPIURL+")")
	cmd.Flags().BoolVar(&local, "local", false, "sign up against the local cluster ("+conn.DefaultAPIURL+", default)")
	cmd.MarkFlagsMutuallyExclusive("managed", "local")
	return cmd
}

// waitStatus polls the onboarding status every 3s until done(st) or timeout.
func waitStatus(ctx context.Context, out io.Writer, client *apiclient.Client, timeout time.Duration,
	done func(apiclient.OnboardingStatus) bool) (apiclient.OnboardingStatus, error) {
	return pollUntil(ctx, out, timeout, 3*time.Second, client.GetOnboardingStatus, done)
}

// pollUntil calls fetch every interval, printing a dot per attempt, until
// done(result), fetch error, timeout, or context cancellation.
func pollUntil(ctx context.Context, out io.Writer, timeout, interval time.Duration,
	fetch func(context.Context) (apiclient.OnboardingStatus, error),
	done func(apiclient.OnboardingStatus) bool) (apiclient.OnboardingStatus, error) {
	deadline := time.Now().Add(timeout)
	for {
		st, err := fetch(ctx)
		if err != nil {
			return st, err
		}
		if done(st) {
			return st, nil
		}
		if time.Now().After(deadline) {
			return st, fmt.Errorf("timed out after %s", timeout)
		}
		fmt.Fprint(out, ".")
		select {
		case <-ctx.Done():
			return st, ctx.Err()
		case <-time.After(interval):
		}
	}
}
