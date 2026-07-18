package cmd

import (
	"fmt"
	"io"
	"time"

	"github.com/optikklabs/optikk/internal/clierr"
	"github.com/spf13/cobra"
)

func newVerifyCmd(app *App) *cobra.Command {
	var from, to string
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Check that telemetry is arriving for your tenant",
		Long: "Queries the ingestion summary for the window and reports per-signal record\n" +
			"counts. Exits 0 when data is flowing, 1 when the window is empty — loop on\n" +
			"it after onboarding until your first telemetry lands.",
		Example: "  optikk verify --from 15m",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, _, startMs, endMs, err := setupRange(cmd, app, from, to)
			if err != nil {
				return err
			}
			totals, err := client.IngestionSummary(cmd.Context(), startMs, endMs)
			if err != nil {
				return err
			}
			doc := verifyDoc{
				OK:   totals.Records > 0,
				From: time.UnixMilli(startMs).UTC().Format(time.RFC3339),
				To:   time.UnixMilli(endMs).UTC().Format(time.RFC3339),
				Records: verifyRecords{
					Spans: totals.Spans, Logs: totals.Logs,
					Metrics: totals.MetricDatapoints, Total: totals.Records,
				},
			}
			// The report always prints, pass or fail: the counts are the answer.
			if err := writeResult(cmd, app, doc, func(w io.Writer) {
				if doc.OK {
					fmt.Fprintf(w, "✓ Telemetry is flowing (%s → %s)\n", doc.From, doc.To)
				} else {
					fmt.Fprintf(w, "✗ No telemetry received (%s → %s)\n", doc.From, doc.To)
				}
				fmt.Fprintf(w, "  spans: %d  logs: %d  metric datapoints: %d\n",
					doc.Records.Spans, doc.Records.Logs, doc.Records.Metrics)
			}); err != nil {
				return err
			}
			if !doc.OK {
				return clierr.New(clierr.Internal, "no telemetry received in the window",
					"check OTEL_EXPORTER_OTLP_ENDPOINT and OTEL_EXPORTER_OTLP_HEADERS on your service, wait ~1m after first traffic, then re-run: optikk verify --from 15m")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&from, "from", "15m", "start time (15m, 1h, ISO8601, epoch-ms)")
	cmd.Flags().StringVar(&to, "to", "now", "end time")
	return cmd
}

// verifyDoc is the machine-readable verify report.
type verifyDoc struct {
	OK      bool          `json:"ok"`
	From    string        `json:"from"`
	To      string        `json:"to"`
	Records verifyRecords `json:"records"`
}

type verifyRecords struct {
	Spans   uint64 `json:"spans"`
	Logs    uint64 `json:"logs"`
	Metrics uint64 `json:"metrics"`
	Total   uint64 `json:"total"`
}
