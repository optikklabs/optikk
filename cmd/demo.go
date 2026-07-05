package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/optikklabs/optikk/internal/conn"
	"github.com/optikklabs/optikk/internal/verify"
	"github.com/spf13/cobra"
)

func newDemoCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "demo",
		Short: "Send synthetic data so you can see Optikk work before instrumenting a service",
	}
	cmd.AddCommand(newDemoSendCmd(app))
	return cmd
}

func newDemoSendCmd(app *App) *cobra.Command {
	var apiKey string
	var count int
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Push demo traces to your ingest endpoint, then query them",
		Long: "Sends synthetic OTLP traces (fresh timestamps, unique IDs) with your ingest\n" +
			"api key — no cluster access and no instrumented service needed. Then run\n" +
			"`optikk traces search` to see them land.",
		Example:     "  optikk demo send\n  optikk demo send --api-key <key> --count 5",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{annotationSkipDeploy: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			out := cmd.OutOrStdout()
			apiBase := conn.Resolve(app.Cfg.ApiURL)
			otlpBase := otlpEndpoint(apiBase)

			key := apiKey
			if key == "" {
				client, err := adminClient()
				if err != nil {
					return fmt.Errorf("no --api-key given and no cached session: %w (run `optikk login` or pass --api-key)", err)
				}
				st, err := client.GetOnboardingStatus(ctx)
				if err != nil {
					return conn.HintUnreachable(apiBase, err)
				}
				if st.APIKey == "" {
					return fmt.Errorf("your session has no ingest api key yet; pass --api-key")
				}
				key = st.APIKey
			}

			for i := 0; i < count; i++ {
				if err := verify.PostTraceBytes(ctx, otlpBase, key, demoTrace()); err != nil {
					return conn.HintUnreachable(apiBase, err)
				}
			}
			fmt.Fprintf(out, "✓ sent %d demo trace(s) to %s\n", count, otlpBase)
			fmt.Fprintln(out, "  run `optikk traces search` to see them (ingestion takes a few seconds).")
			return nil
		},
	}
	cmd.Flags().StringVar(&apiKey, "api-key", "", "ingest api key (default: from your cached session)")
	cmd.Flags().IntVar(&count, "count", 1, "number of demo traces to send")
	return cmd
}

// demoTrace builds one OTLP/HTTP trace payload with a fresh timestamp and random
// trace/span IDs so it appears in a default `traces search --from 1h` window.
func demoTrace() []byte {
	end := time.Now()
	start := end.Add(-50 * time.Millisecond)
	return []byte(fmt.Sprintf(`{"resourceSpans":[{"resource":{"attributes":[`+
		`{"key":"service.name","value":{"stringValue":"optikk-demo"}}]},`+
		`"scopeSpans":[{"scope":{"name":"optikk-demo"},"spans":[{`+
		`"traceId":%q,"spanId":%q,"name":"demo-request","kind":2,`+
		`"startTimeUnixNano":%q,"endTimeUnixNano":%q,"status":{"code":1}}]}]}]}`,
		randHex(16), randHex(8),
		fmt.Sprint(start.UnixNano()), fmt.Sprint(end.UnixNano())))
}

// randHex returns n random bytes hex-encoded (OTLP IDs are hex strings).
func randHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
