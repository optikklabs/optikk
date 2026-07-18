package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/optikklabs/optikk/internal/queryclient"
	"github.com/spf13/cobra"
)

func newErrorsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "errors",
		Short: "Investigate error groups",
		Long: "Error groups cluster similar error spans. List them, then pivot into a\n" +
			"group's traces, timeseries, or latest occurrence (message + stacktrace).",
		Example: "  optikk errors list --service checkout --from 1h\n" +
			"  optikk errors latest <group-id>",
	}
	cmd.AddCommand(
		newErrorsListCmd(app),
		newErrorsGetCmd(app),
		newErrorsTracesCmd(app),
		newErrorsTimeseriesCmd(app),
		newErrorsLatestCmd(app),
	)
	return cmd
}

func newErrorsListCmd(app *App) *cobra.Command {
	var service, cursor, from, to string
	var limit int
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List error groups, most frequent first",
		Example: "  optikk errors list --service checkout --from 1h",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, out, startMs, endMs, err := setupRange(cmd, app, from, to)
			if err != nil {
				return err
			}
			resp, err := client.ListErrorGroups(cmd.Context(), startMs, endMs, service, limit, cursor)
			if err != nil {
				return err
			}
			items := make([]any, len(resp.Results))
			for i := range resp.Results {
				items[i] = resp.Results[i]
			}
			return out.WriteItems(
				[]string{"GROUP_ID", "SERVICE", "OPERATION", "COUNT", "LAST_SEEN", "SAMPLE_TRACE"},
				items, func(v any) []string {
					g := v.(queryclient.ErrorGroup)
					return []string{
						g.GroupID, g.ServiceName, g.OperationName,
						strconv.FormatInt(g.ErrorCount, 10),
						g.LastOccurrence.Format(time.RFC3339),
						g.SampleTraceID,
					}
				})
		},
	}
	cmd.Flags().StringVar(&service, "service", "", "filter to one service")
	cmd.Flags().IntVar(&limit, "limit", 100, "max groups per page")
	cmd.Flags().StringVar(&cursor, "cursor", "", "pagination cursor from previous response")
	addRangeFlags(cmd, &from, &to)
	return cmd
}

func newErrorsGetCmd(app *App) *cobra.Command {
	var from, to string
	cmd := &cobra.Command{
		Use:   "get <group-id>",
		Short: "Show one error group's aggregate detail",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, out, startMs, endMs, err := setupRange(cmd, app, from, to)
			if err != nil {
				return err
			}
			resp, err := client.GetErrorGroupDetail(cmd.Context(), startMs, endMs, args[0])
			if err != nil {
				return err
			}
			return out.WriteJSON(resp)
		},
	}
	addRangeFlags(cmd, &from, &to)
	return cmd
}

func newErrorsTracesCmd(app *App) *cobra.Command {
	var cursor, from, to string
	var limit int
	cmd := &cobra.Command{
		Use:   "traces <group-id>",
		Short: "List trace occurrences of an error group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, out, startMs, endMs, err := setupRange(cmd, app, from, to)
			if err != nil {
				return err
			}
			resp, err := client.GetErrorGroupTraces(cmd.Context(), startMs, endMs, args[0], limit, cursor)
			if err != nil {
				return err
			}
			items := make([]any, len(resp.Results))
			for i := range resp.Results {
				items[i] = resp.Results[i]
			}
			return out.WriteItems(
				[]string{"TRACE_ID", "SPAN_ID", "TIMESTAMP", "DURATION_MS", "STATUS"},
				items, func(v any) []string {
					t := v.(queryclient.ErrorGroupTrace)
					return []string{
						t.TraceID, t.SpanID,
						t.Timestamp.Format(time.RFC3339),
						fmt.Sprintf("%.1f", t.DurationMs),
						t.StatusCode,
					}
				})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "max traces per page (server caps at 20)")
	cmd.Flags().StringVar(&cursor, "cursor", "", "pagination cursor from previous response")
	addRangeFlags(cmd, &from, &to)
	return cmd
}

func newErrorsTimeseriesCmd(app *App) *cobra.Command {
	var from, to string
	cmd := &cobra.Command{
		Use:   "timeseries <group-id>",
		Short: "Show an error group's rate over time",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, out, startMs, endMs, err := setupRange(cmd, app, from, to)
			if err != nil {
				return err
			}
			resp, err := client.GetErrorGroupTimeseries(cmd.Context(), startMs, endMs, args[0])
			if err != nil {
				return err
			}
			return out.WriteJSON(resp)
		},
	}
	addRangeFlags(cmd, &from, &to)
	return cmd
}

func newErrorsLatestCmd(app *App) *cobra.Command {
	var from, to string
	cmd := &cobra.Command{
		Use:   "latest <group-id>",
		Short: "Show a group's most recent occurrence (message + stacktrace)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, out, startMs, endMs, err := setupRange(cmd, app, from, to)
			if err != nil {
				return err
			}
			resp, err := client.GetErrorGroupLatest(cmd.Context(), startMs, endMs, args[0])
			if err != nil {
				return err
			}
			return out.WriteJSON(resp)
		},
	}
	addRangeFlags(cmd, &from, &to)
	return cmd
}
