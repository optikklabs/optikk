package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/optikklabs/optikk/internal/config"
	"github.com/optikklabs/optikk/internal/deploypath"
	"github.com/optikklabs/optikk/internal/target"
	"github.com/optikklabs/optikk/internal/verify"
	"github.com/spf13/cobra"
)

func newVerifyCmd(app *App) *cobra.Command {
	var (
		apiKey    string
		traceFile string
		remote    bool
	)
	cmd := &cobra.Command{
		Use:     "verify",
		Aliases: []string{"smoke"},
		Short:   "Run health and trace roundtrip checks",
		Example: "  optikk verify\n  optikk verify --remote\n  optikk verify --api-key <tenant-key>",
		// Skip deploy resolution: --remote needs no repo, and the kube path
		// resolves the deploy tree lazily only for its default trace file.
		Annotations: map[string]string{annotationSkipDeploy: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if remote {
				return runRemoteVerify(cmd)
			}
			conn, err := target.Resolve(app.Cfg)
			if err != nil {
				return err
			}
			if traceFile == "" {
				dir, err := deploypath.Resolve(app.Cfg.DeployDir)
				if err != nil {
					return err
				}
				traceFile = filepath.Join(dir, "example-trace.json")
			}
			return verify.Run(cmd.Context(), verify.Options{
				Kube:      conn.Kube,
				Namespace: config.Namespace,
				APIBase:   conn.APIBase,
				OTLPBase:  conn.OTLPBase,
				APIKey:    apiKey,
				TraceFile: traceFile,
				Out:       cmd.OutOrStdout(),
			})
		},
	}
	cmd.Flags().StringVar(&apiKey, "api-key", "c3448fae", "tenant api key for the OTLP trace")
	cmd.Flags().StringVar(&traceFile, "trace-file", "", "trace JSON to send (default: <deploy>/example-trace.json)")
	cmd.Flags().BoolVar(&remote, "remote", false, "verify via the query API only (no cluster access needed)")
	return cmd
}

// runRemoteVerify checks a customer's onboarding purely through the query API:
// first trace seen. No kubectl, no cluster access.
func runRemoteVerify(cmd *cobra.Command) error {
	client, err := adminClient()
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	ctx := cmd.Context()

	fmt.Fprint(out, "Checking for your first trace")
	st, err := waitStatus(ctx, out, client, 2*time.Minute,
		func(st apiclient.OnboardingStatus) bool { return st.FirstSpanAt != nil })
	fmt.Fprintln(out)
	if err != nil {
		return fmt.Errorf("no trace data seen yet: %w (send OTLP with your api key, then retry)", err)
	}
	fmt.Fprintf(out, "✓ First trace received at %s\n", st.FirstSpanAt.Local().Format(time.RFC3339))
	return nil
}
