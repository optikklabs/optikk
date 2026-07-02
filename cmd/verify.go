package cmd

import (
	"path/filepath"

	"github.com/optikklabs/optikk/internal/config"
	"github.com/optikklabs/optikk/internal/target"
	"github.com/optikklabs/optikk/internal/verify"
	"github.com/spf13/cobra"
)

func newVerifyCmd(app *App) *cobra.Command {
	var (
		apiKey    string
		traceFile string
	)
	cmd := &cobra.Command{
		Use:     "verify",
		Aliases: []string{"smoke"},
		Short:   "Run health and trace roundtrip checks",
		Example: "  optikk verify\n  optikk verify --api-key <tenant-key>\n  optikk verify --trace-file ./trace.json",
		RunE: func(cmd *cobra.Command, _ []string) error {
			conn, err := target.Resolve(app.Cfg)
			if err != nil {
				return err
			}
			if traceFile == "" {
				traceFile = filepath.Join(app.DeployDir, "example-trace.json")
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
	return cmd
}
