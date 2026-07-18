package cmd

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/optikklabs/optikk/internal/clierr"
	"github.com/optikklabs/optikk/internal/endpoint"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newSignupCmd(app *App) *cobra.Command {
	var si signupInput
	cmd := &cobra.Command{
		Use:   "signup",
		Short: "Create an Optikk account, tenant and ingest API key",
		Long: "Self-serve signup via POST /api/v1/auth/signup: creates your account and tenant,\n" +
			"prints the tenant's ingest API key, and caches the session JWT at ~/.optikk/config.json.",
		Example: "  optikk signup\n  optikk signup --email founder@startup.dev --org startup --name Founder",
		RunE: func(cmd *cobra.Command, _ []string) error {
			apiBase, err := app.API()
			if err != nil {
				return err
			}
			client := apiclient.New(apiBase)
			res, err := signupInteractive(cmd, client, apiBase, si, allowPrompt(cmd, app))
			if err != nil {
				return err
			}
			doc := newSignupDoc(res, apiBase)
			return writeResult(cmd, app, doc, func(w io.Writer) {
				if doc.Status == signupStatusPending {
					fmt.Fprintf(w, "✓ %s\n", doc.Message)
					fmt.Fprintf(w, "\nOnce verified, run `optikk login` to authenticate this device.\n")
					return
				}
				fmt.Fprintf(w, "✓ Account created\n")
				fmt.Fprintf(w, "  Tenant:    %s (id %d)\n", doc.Tenant.Name, doc.Tenant.ID)
				fmt.Fprintf(w, "  API key: %s\n", doc.APIKey)
				fmt.Fprintf(w, "  Token cached at ~/.optikk/config.json\n\n")
				printOTLPSnippet(w, doc.OTLP)
			})
		},
	}
	addSignupFlags(cmd, &si)
	return cmd
}

// addSignupFlags registers the account-creation flags shared by signup and
// onboard, so the two commands cannot drift apart.
func addSignupFlags(cmd *cobra.Command, si *signupInput) {
	cmd.Flags().StringVar(&si.Email, "email", "", "account email (prompted if omitted)")
	cmd.Flags().StringVar(&si.Password, "password", "", "account password (prompted if omitted)")
	cmd.Flags().StringVar(&si.Name, "name", "", "your full name (prompted if omitted)")
	cmd.Flags().StringVar(&si.Org, "org", "", "organization name, becomes your tenant (prompted if omitted)")
	cmd.Flags().BoolVar(&si.AcceptTerms, "accept-terms", false, "accept the Terms of Service and Privacy Policy (prompted if omitted)")
}

const (
	signupStatusCreated = "created"
	signupStatusPending = "verification_pending"
)

// signupDoc is the machine-readable signup result.
type signupDoc struct {
	Status      string      `json:"status"`
	Message     string      `json:"message,omitempty"`
	Next        string      `json:"next,omitempty"`
	Email       string      `json:"email,omitempty"`
	Tenant      *tenantInfo `json:"tenant,omitempty"`
	APIKey      string      `json:"api_key,omitempty"`
	TokenCached bool        `json:"token_cached,omitempty"`
	OTLP        *otlpInfo   `json:"otlp,omitempty"`
	DocsURL     string      `json:"docs_url,omitempty"`
}

type tenantInfo struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// otlpInfo is the env pair a service needs to export telemetry to Optikk.
type otlpInfo struct {
	Endpoint string `json:"endpoint"`
	Headers  string `json:"headers"`
}

func newSignupDoc(res apiclient.SignupResult, apiBase string) signupDoc {
	if res.Message != "" {
		return signupDoc{Status: signupStatusPending, Message: res.Message, Next: "optikk login"}
	}
	return signupDoc{
		Status:      signupStatusCreated,
		Tenant:      &tenantInfo{ID: res.Tenant.ID, Name: res.Tenant.Name},
		APIKey:      res.APIKey,
		TokenCached: res.AccessToken != "",
		OTLP:        newOTLPInfo(apiBase, res.APIKey),
		DocsURL:     endpoint.DocsURL,
	}
}

func newOTLPInfo(apiBase, apiKey string) *otlpInfo {
	return &otlpInfo{
		Endpoint: otlpEndpoint(apiBase),
		Headers:  "x-api-key=" + apiKey,
	}
}

// printOTLPSnippet renders the copy-pasteable SDK env pair for humans.
func printOTLPSnippet(w io.Writer, o *otlpInfo) {
	fmt.Fprintf(w, "Point your OpenTelemetry SDK at Optikk:\n")
	fmt.Fprintf(w, "  OTEL_EXPORTER_OTLP_ENDPOINT=%s\n", o.Endpoint)
	fmt.Fprintf(w, "  OTEL_EXPORTER_OTLP_HEADERS=%s\n", o.Headers)
}

// signupInput carries signup fields; empty account fields are prompted when
// prompting is allowed. Org is the organization name, which becomes the
// tenant name.
type signupInput struct {
	Email, Password, Name, Org string
	AcceptTerms                bool
}

// missingFlags lists the flags a non-interactive signup still needs.
func (si signupInput) missingFlags() []string {
	var missing []string
	if si.Email == "" {
		missing = append(missing, "--email")
	}
	if si.Password == "" {
		missing = append(missing, "--password")
	}
	if si.Name == "" {
		missing = append(missing, "--name")
	}
	if si.Org == "" {
		missing = append(missing, "--org")
	}
	if !si.AcceptTerms {
		missing = append(missing, "--accept-terms")
	}
	return missing
}

// allowPrompt reports whether this invocation may ask the user questions:
// never in agent mode, and never without a terminal on stdin.
func allowPrompt(cmd *cobra.Command, app *App) bool {
	return !app.AgentMode && stdinIsTerminal(cmd)
}

// stdinIsTerminal reports whether the command's input is an interactive TTY.
func stdinIsTerminal(cmd *cobra.Command) bool {
	f, ok := cmd.InOrStdin().(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// signupInteractive completes any missing fields (by prompting, when allowed),
// signs up, and caches the JWT.
func signupInteractive(cmd *cobra.Command, client *apiclient.Client, apiBase string, si signupInput, allowPrompt bool) (apiclient.SignupResult, error) {
	if !allowPrompt {
		if missing := si.missingFlags(); len(missing) > 0 {
			return apiclient.SignupResult{}, clierr.New(clierr.Usage,
				"missing required flags for non-interactive signup: "+strings.Join(missing, " "),
				"pass every flag (--accept-terms records the user's consent to https://optikk.in/terms), or run interactively in a terminal")
		}
	} else if err := promptSignupInput(cmd, &si); err != nil {
		return apiclient.SignupResult{}, err
	}

	res, err := client.Signup(cmd.Context(), apiclient.SignupRequest{
		Email: si.Email, Password: si.Password, Name: si.Name, TenantName: si.Org,
		AcceptedTerms: si.AcceptTerms,
	})
	if err != nil {
		return apiclient.SignupResult{}, fmt.Errorf("signup failed: %w", err)
	}
	if res.AccessToken != "" {
		if err := apiclient.SaveToken(apiBase, res.AccessToken); err != nil {
			return apiclient.SignupResult{}, err
		}
	}
	return res, nil
}

// promptSignupInput fills empty fields from the terminal.
func promptSignupInput(cmd *cobra.Command, si *signupInput) error {
	in := bufio.NewReader(cmd.InOrStdin())
	var err error
	if si.Email, err = promptIfEmpty(cmd, in, "Email", si.Email); err != nil {
		return err
	}
	if si.Name, err = promptIfEmpty(cmd, in, "Your name", si.Name); err != nil {
		return err
	}
	if si.Org, err = promptIfEmpty(cmd, in, "Organization", si.Org); err != nil {
		return err
	}
	if si.Password == "" {
		if si.Password, err = promptPassword(cmd, in); err != nil {
			return err
		}
	}
	return confirmTerms(cmd, in, si)
}

// confirmTerms requires the user to accept the Terms and Privacy Policy. The
// --accept-terms flag records consent non-interactively; otherwise we prompt.
func confirmTerms(cmd *cobra.Command, in *bufio.Reader, si *signupInput) error {
	if si.AcceptTerms {
		return nil
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "By continuing you agree to the Terms of Service (https://optikk.in/terms)\n")
	fmt.Fprintf(out, "and Privacy Policy (https://optikk.in/privacy).\n")
	fmt.Fprint(out, "Accept? [y/N]: ")
	line, err := in.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read confirmation: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		si.AcceptTerms = true
		return nil
	default:
		return fmt.Errorf("signup requires accepting the Terms of Service and Privacy Policy")
	}
}

// otlpEndpoint maps the API base to Traefik's OTLP/HTTP entrypoint (:4318).
// A deploy that fronts OTLP on a separate `ingest.<domain>` host has its `api`
// label swapped to `ingest`; localhost/IPs keep their host. An explicit
// OTEL_EXPORTER_OTLP_ENDPOINT env wins over the guess.
//
// 4318 is the public OTLP/HTTP port on the Traefik load balancer. The ingest
// Service's own 18317/18318 are cluster-internal and not routable from an
// SDK, so they must not appear here.
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
