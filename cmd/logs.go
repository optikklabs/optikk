package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/optikklabs/optikk/internal/clitime"
	"github.com/optikklabs/optikk/internal/dsl"
	"github.com/optikklabs/optikk/internal/queryclient"
	"github.com/spf13/cobra"
)

func newLogsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "logs",
		Short:       "Search and inspect logs",
		Long:        "Query the logs API — search by DSL, view log details.",
		Annotations: map[string]string{annotationSkipDeploy: "true"},
	}
	cmd.AddCommand(
		newLogsSearchCmd(app),
		newLogsAggCmd(app, "facets", "Show facet counts over matching logs", (*queryclient.Client).LogFacets),
		newLogsAggCmd(app, "summary", "Show aggregate summary stats over matching logs", (*queryclient.Client).LogSummary),
		newLogsAggCmd(app, "trend", "Show the log volume trend over matching logs", (*queryclient.Client).LogTrend),
	)
	return cmd
}

// newLogsAggCmd builds a DSL-filtered log aggregation command emitting raw JSON.
func newLogsAggCmd(app *App, use, short string, fn func(*queryclient.Client, context.Context, queryclient.LogsRangeRequest) (any, error)) *cobra.Command {
	var query, from, to string
	cmd := &cobra.Command{
		Use:         use,
		Short:       short,
		Annotations: map[string]string{annotationSkipDeploy: "true"},
		Example:     fmt.Sprintf(`  optikk logs %s --query "severity_text:ERROR" --from 1h`, use),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, out, startMs, endMs, err := setupRange(cmd, app, from, to)
			if err != nil {
				return err
			}
			req := queryclient.LogsRangeRequest{StartTime: startMs, EndTime: endMs}
			if query != "" {
				result := dsl.Parse(query, dsl.LogKnownFields)
				for _, e := range result.Errors {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s (at offset %d)\n", e.Message, e.Offset)
				}
				req.LogFilters = dsl.MapToLogFilters(result.Filters)
			}
			resp, err := fn(client, cmd.Context(), req)
			if err != nil {
				return err
			}
			return out.WriteJSON(resp)
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "Datadog-style DSL filter")
	addRangeFlags(cmd, &from, &to)
	return cmd
}

func newLogsSearchCmd(app *App) *cobra.Command {
	var query, from, to, cursor string
	var limit int

	cmd := &cobra.Command{
		Use:     "search",
		Short:   "Search logs by query",
		Example: `  optikk logs search --query "severity_text:ERROR service_name:api" --from 15m --limit 50`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := resolveClient(app)
			if err != nil {
				return err
			}
			out := resolveOutput(cmd, app)

			startMs, endMs, err := clitime.ParseRange(from, to, time.Now())
			if err != nil {
				return err
			}

			req := queryclient.LogsQueryRequest{
				StartTime: startMs,
				EndTime:   endMs,
				Limit:     limit,
				Cursor:    cursor,
			}

			if query != "" {
				result := dsl.Parse(query, dsl.LogKnownFields)
				for _, e := range result.Errors {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s (at offset %d)\n", e.Message, e.Offset)
				}
				req.LogFilters = dsl.MapToLogFilters(result.Filters)
			}

			resp, err := client.SearchLogs(cmd.Context(), req)
			if err != nil {
				return err
			}

			headers := []string{"TIMESTAMP", "SERVICE", "SEVERITY", "BODY"}
			items := make([]any, len(resp.Results))
			for i := range resp.Results {
				items[i] = resp.Results[i]
			}
			return out.WriteItems(headers, items, func(v any) []string {
				l := v.(queryclient.Log)
				body := l.Body
				if len(body) > 120 {
					body = body[:117] + "..."
				}
				return []string{
					l.Timestamp.Format("15:04:05.000"),
					l.ServiceName,
					l.SeverityText,
					body,
				}
			})
		},
	}

	cmd.Flags().StringVarP(&query, "query", "q", "", `Datadog-style DSL (e.g. "severity_text:ERROR")`)
	cmd.Flags().StringVar(&from, "from", "1h", "start time (1h, 15m, 7d, ISO8601, epoch-ms)")
	cmd.Flags().StringVar(&to, "to", "now", "end time")
	cmd.Flags().IntVar(&limit, "limit", 20, "max results")
	cmd.Flags().StringVar(&cursor, "cursor", "", "pagination cursor from previous response")
	return cmd
}
