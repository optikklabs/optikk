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

func newTracesCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "traces",
		Short:       "Search and inspect distributed traces",
		Long:        "Query the traces API — search by DSL, get trace details, view trends.",
	}
	cmd.AddCommand(
		newTracesSearchCmd(app),
		newTracesGetCmd(app),
		newTracesTrendCmd(app),
		newTraceAnalysisCmd(app, "critical-path", "Show the critical path through a trace", (*queryclient.Client).CriticalPath),
		newTraceAnalysisCmd(app, "error-path", "Show the error propagation path through a trace", (*queryclient.Client).ErrorPath),
		newTraceAnalysisCmd(app, "service-map", "Show the service dependency map for a trace", (*queryclient.Client).ServiceMap),
		newTracesRelatedCmd(app),
		newTraceAnalysisCmd(app, "errors", "Show the errors within a trace", (*queryclient.Client).TraceErrors),
	)
	return cmd
}

// newTraceAnalysisCmd builds a `traces <verb> <traceId>` command backed by a
// client method that returns analytical JSON for one trace.
func newTraceAnalysisCmd(app *App, use, short string, fn func(*queryclient.Client, context.Context, string) (any, error)) *cobra.Command {
	return &cobra.Command{
		Use:         use + " <traceId>",
		Short:       short,
		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := resolveClient(app)
			if err != nil {
				return err
			}
			out := resolveOutput(cmd, app)
			resp, err := fn(client, cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return out.WriteJSON(resp)
		},
	}
}

// newTracesRelatedCmd handles `traces related` which needs a time range in
// addition to the trace id.
func newTracesRelatedCmd(app *App) *cobra.Command {
	var from, to, service, operation string
	cmd := &cobra.Command{
		Use:         "related <traceId>",
		Short:       "Show traces sharing a service+operation with a trace",
		Args:        cobra.ExactArgs(1),
		Example:     `  optikk traces related <id> --service api --operation GET /users --from 1h`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if service == "" || operation == "" {
				return fmt.Errorf("--service and --operation are required")
			}
			client, out, startMs, endMs, err := setupRange(cmd, app, from, to)
			if err != nil {
				return err
			}
			resp, err := client.RelatedTraces(cmd.Context(), args[0], service, operation, startMs, endMs)
			if err != nil {
				return err
			}
			return out.WriteJSON(resp)
		},
	}
	addRangeFlags(cmd, &from, &to)
	cmd.Flags().StringVar(&service, "service", "", "service name (required)")
	cmd.Flags().StringVar(&operation, "operation", "", "operation name (required)")
	return cmd
}

func newTracesSearchCmd(app *App) *cobra.Command {
	var query, from, to, cursor string
	var limit int

	cmd := &cobra.Command{
		Use:     "search",
		Short:   "Search traces by query",
		Example: `  optikk traces search --query "service:api status:error" --from 1h --limit 10`,
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

			req := queryclient.TracesQueryRequest{
				StartTime: startMs,
				EndTime:   endMs,
				Limit:     limit,
				Cursor:    cursor,
			}

			if query != "" {
				result := dsl.Parse(query, dsl.TraceKnownFields)
				for _, e := range result.Errors {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s (at offset %d)\n", e.Message, e.Offset)
				}
				req.TraceFilters = dsl.MapToTraceFilters(result.Filters)
			}

			resp, err := client.SearchTraces(cmd.Context(), req)
			if err != nil {
				return err
			}

			headers := []string{"TRACE ID", "SERVICE", "OPERATION", "STATUS", "DURATION", "SPANS", "TIME"}
			items := make([]any, len(resp.Results))
			for i := range resp.Results {
				items[i] = resp.Results[i]
			}
			return out.WriteItems(headers, items, func(v any) []string {
				t := v.(queryclient.Trace)
				dur := time.Duration(t.DurationNs) * time.Nanosecond
				return []string{
					t.TraceID,
					t.RootService,
					t.RootOperation,
					t.RootStatus,
					dur.String(),
					fmt.Sprintf("%d", t.SpanCount),
					t.StartTime.Format("15:04:05"),
				}
			})
		},
	}

	cmd.Flags().StringVarP(&query, "query", "q", "", `Datadog-style DSL (e.g. "service:api has_error:true")`)
	cmd.Flags().StringVar(&from, "from", "1h", "start time (1h, 15m, 7d, ISO8601, epoch-ms)")
	cmd.Flags().StringVar(&to, "to", "now", "end time")
	cmd.Flags().IntVar(&limit, "limit", 20, "max results")
	cmd.Flags().StringVar(&cursor, "cursor", "", "pagination cursor from previous response")
	return cmd
}

func newTracesGetCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:         "get <traceId>",
		Short:       "Get full trace details",
		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := resolveClient(app)
			if err != nil {
				return err
			}
			out := resolveOutput(cmd, app)
			resp, err := client.GetTrace(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return out.WriteJSON(resp)
		},
	}
}

func newTracesTrendCmd(app *App) *cobra.Command {
	var query, from, to string

	cmd := &cobra.Command{
		Use:     "trend",
		Short:   "View trace volume and error trends",
		Example: `  optikk traces trend --query "service:api" --from 1h`,
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

			req := queryclient.TracesTrendRequest{
				StartTime: startMs,
				EndTime:   endMs,
			}
			if query != "" {
				result := dsl.Parse(query, dsl.TraceKnownFields)
				req.TraceFilters = dsl.MapToTraceFilters(result.Filters)
			}

			resp, err := client.TraceTrend(cmd.Context(), req)
			if err != nil {
				return err
			}
			return out.WriteJSON(resp)
		},
	}

	cmd.Flags().StringVarP(&query, "query", "q", "", "Datadog-style DSL filter")
	cmd.Flags().StringVar(&from, "from", "1h", "start time")
	cmd.Flags().StringVar(&to, "to", "now", "end time")
	return cmd
}
