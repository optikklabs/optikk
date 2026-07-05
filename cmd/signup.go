package cmd

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/optikklabs/optikk/internal/conn"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newSignupCmd(app *App) *cobra.Command {
	var email, password, name, org string
	cmd := &cobra.Command{
		Use:   "signup",
		Short: "Create an Optikk account, tenant and ingest API key",
		Long: "Self-serve signup via POST /api/v1/auth/signup: creates your account and tenant,\n" +
			"prints the tenant's ingest API key, and caches the session JWT at ~/.optikk/token.json.",
		Example:     "  optikk signup\n  optikk signup --email founder@startup.dev --org startup --name Founder",
		Annotations: map[string]string{annotationSkipDeploy: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			apiBase := conn.Resolve(app.Cfg.ApiURL)
			client := apiclient.New(apiBase)
			res, err := signupInteractive(cmd, client, apiBase, signupInput{
				Email: email, Password: password, Name: name, Org: org,
			})
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "✓ Account created\n")
			fmt.Fprintf(out, "  Tenant:    %s (id %d)\n", res.Tenant.Name, res.Tenant.ID)
			fmt.Fprintf(out, "  API key: %s\n", res.APIKey)
			fmt.Fprintf(out, "  Token cached at ~/.optikk/token.json\n\n")
			fmt.Fprintf(out, "Point your OpenTelemetry SDK at Optikk:\n")
			fmt.Fprintf(out, "  OTEL_EXPORTER_OTLP_ENDPOINT=%s\n", otlpEndpoint(apiBase))
			fmt.Fprintf(out, "  OTEL_EXPORTER_OTLP_HEADERS=x-api-key=%s\n", res.APIKey)
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "account email (prompted if omitted)")
	cmd.Flags().StringVar(&password, "password", "", "account password (prompted if omitted)")
	cmd.Flags().StringVar(&name, "name", "", "your full name (prompted if omitted)")
	cmd.Flags().StringVar(&org, "org", "", "organization name, becomes your tenant (prompted if omitted)")
	return cmd
}

// signupInput carries signup fields; empty account fields are prompted. Org is
// the organization name, which becomes the tenant name.
type signupInput struct {
	Email, Password, Name, Org string
}

// signupInteractive prompts for any missing fields, signs up, and caches the JWT.
func signupInteractive(cmd *cobra.Command, client *apiclient.Client, apiBase string, si signupInput) (apiclient.SignupResult, error) {
	in := bufio.NewReader(cmd.InOrStdin())
	var err error
	if si.Email, err = promptIfEmpty(cmd, in, "Email", si.Email); err != nil {
		return apiclient.SignupResult{}, err
	}
	if si.Name, err = promptIfEmpty(cmd, in, "Your name", si.Name); err != nil {
		return apiclient.SignupResult{}, err
	}
	if si.Org, err = promptIfEmpty(cmd, in, "Organization", si.Org); err != nil {
		return apiclient.SignupResult{}, err
	}
	if si.Password == "" {
		if si.Password, err = promptPassword(cmd, in); err != nil {
			return apiclient.SignupResult{}, err
		}
	}

	res, err := client.Signup(cmd.Context(), apiclient.SignupRequest{
		Email: si.Email, Password: si.Password, Name: si.Name, TenantName: si.Org,
	})
	if err != nil {
		return apiclient.SignupResult{}, conn.HintUnreachable(apiBase, fmt.Errorf("signup failed: %w", err))
	}
	if err := apiclient.SaveToken(apiBase, res.AccessToken); err != nil {
		return apiclient.SignupResult{}, err
	}
	return res, nil
}

// otlpEndpoint maps the API base to Traefik's OTLP/HTTP entrypoint (:4318).
// Hosted deploys front OTLP on a separate `ingest.<domain>` host (see
// overlays/prod), so the `api` label is swapped to `ingest`; localhost/IPs keep
// their host. An explicit OTEL_EXPORTER_OTLP_ENDPOINT env wins over the guess.
func otlpEndpoint(apiBase string) string {
	if v := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); v != "" {
		return v
	}
	u, err := url.Parse(apiBase)
	if err != nil || u.Host == "" {
		return apiBase
	}
	host := u.Hostname()
	if label, rest, found := strings.Cut(host, "."); found && label == "api" {
		host = "ingest." + rest
	}
	u.Host = host + ":4318"
	return u.String()
}

// promptIfEmpty returns value, or reads one line from in after printing label.
func promptIfEmpty(cmd *cobra.Command, in *bufio.Reader, label, value string) (string, error) {
	if value != "" {
		return value, nil
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s: ", label)
	line, err := in.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read %s: %w", strings.ToLower(label), err)
	}
	return strings.TrimSpace(line), nil
}

// promptPassword reads without echo on a TTY, falling back to line input.
func promptPassword(cmd *cobra.Command, in *bufio.Reader) (string, error) {
	fmt.Fprint(cmd.OutOrStdout(), "Password (min 8 chars): ")
	if f, ok := cmd.InOrStdin().(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		b, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(cmd.OutOrStdout())
		if err != nil {
			return "", fmt.Errorf("read password: %w", err)
		}
		return string(b), nil
	}
	line, err := in.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	return strings.TrimSpace(line), nil
}
