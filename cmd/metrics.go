package cmd

import (
	"fmt"
	"time"

	"github.com/optikklabs/optikk/internal/clitime"
	"github.com/optikklabs/optikk/internal/queryclient"
	"github.com/spf13/cobra"
)

func newMetricsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "List, query, and inspect metrics",
		Long:  "Query the metrics API — list names, run timeseries queries, list tags.",
	}
	cmd.AddCommand(
		newMetricsListCmd(app),
		newMetricsQueryCmd(app),
		newMetricsTagsCmd(app),
	)
	return cmd
}

func newMetricsListCmd(app *App) *cobra.Command {
	var from, to, search string

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List available metric names",
		Example: `  optikk metrics list --from 1h --search cpu`,
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

			resp, err := client.ListMetricNames(cmd.Context(), startMs, endMs, search)
			if err != nil {
				return err
			}

			headers := []string{"NAME", "TYPE", "UNIT", "DESCRIPTION"}
			items := make([]any, len(resp.Metrics))
			for i := range resp.Metrics {
				items[i] = resp.Metrics[i]
			}
			return out.WriteItems(headers, items, func(v any) []string {
				m := v.(queryclient.MetricNameEntry)
				return []string{m.Name, m.Type, m.Unit, m.Description}
			})
		},
	}

	cmd.Flags().StringVar(&from, "from", "1h", "start time")
	cmd.Flags().StringVar(&to, "to", "now", "end time")
	cmd.Flags().StringVar(&search, "search", "", "filter metric names by substring")
	return cmd
}

func newMetricsQueryCmd(app *App) *cobra.Command {
	var from, to, metric, aggregation, step string
	var groupBy []string

	cmd := &cobra.Command{
		Use:     "query",
		Short:   "Execute a metrics timeseries query",
		Example: `  optikk metrics query --metric system.cpu.usage --aggregation avg --from 1h`,
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

			req := queryclient.MetricsQueryRequest{
				StartTime: startMs,
				EndTime:   endMs,
				Step:      step,
				Queries: []queryclient.MetricQuery{
					{
						ID:          "a",
						Aggregation: aggregation,
						MetricName:  metric,
						GroupBy:     groupBy,
					},
				},
			}

			resp, err := client.QueryMetrics(cmd.Context(), req)
			if err != nil {
				return err
			}
			return out.WriteJSON(resp)
		},
	}

	cmd.Flags().StringVar(&metric, "metric", "", "metric name (required)")
	cmd.Flags().StringVar(&aggregation, "aggregation", "avg", "aggregation function (avg, sum, min, max, count, p50, p95, p99, rate)")
	cmd.Flags().StringVar(&from, "from", "1h", "start time")
	cmd.Flags().StringVar(&to, "to", "now", "end time")
	cmd.Flags().StringVar(&step, "step", "", "time bucket step (auto if empty)")
	cmd.Flags().StringSliceVar(&groupBy, "group-by", nil, "tag keys to group by")
	_ = cmd.MarkFlagRequired("metric")
	return cmd
}

func newMetricsTagsCmd(app *App) *cobra.Command {
	var from, to string

	cmd := &cobra.Command{
		Use:     "tags <metricName>",
		Short:   "List tags for a metric",
		Args:    cobra.ExactArgs(1),
		Example: `  optikk metrics tags system.cpu.usage --from 1h`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := resolveClient(app)
			if err != nil {
				return err
			}
			out := resolveOutput(cmd, app)

			startMs, endMs, err := clitime.ParseRange(from, to, time.Now())
			if err != nil {
				return err
			}

			resp, err := client.ListTags(cmd.Context(), args[0], startMs, endMs)
			if err != nil {
				return err
			}

			headers := []string{"KEY", "VALUES"}
			items := make([]any, len(resp.Tags))
			for i := range resp.Tags {
				items[i] = resp.Tags[i]
			}
			return out.WriteItems(headers, items, func(v any) []string {
				t := v.(queryclient.TagEntry)
				vals := fmt.Sprintf("%v", t.Values)
				if len(vals) > 80 {
					vals = vals[:77] + "..."
				}
				return []string{t.Key, vals}
			})
		},
	}

	cmd.Flags().StringVar(&from, "from", "1h", "start time")
	cmd.Flags().StringVar(&to, "to", "now", "end time")
	return cmd
}
