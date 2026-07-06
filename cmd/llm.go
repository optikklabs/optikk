package cmd

import (
	"fmt"

	"github.com/optikklabs/optikk/internal/queryclient"
	"github.com/spf13/cobra"
)

func newLLMCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "llm",
		Short:       "Inspect LLM/AI observability data",
		Long:        "Query the LLM API — apps, cost, timeseries, and traces for AI workloads.",
	}
	cmd.AddCommand(
		newLLMAppsCmd(app),
		newLLMCostCmd(app),
		newLLMTimeseriesCmd(app),
		newLLMTracesCmd(app),
		newLLMTraceGetCmd(app),
	)
	return cmd
}

func newLLMAppsCmd(app *App) *cobra.Command {
	var from, to string
	cmd := &cobra.Command{
		Use:         "apps",
		Short:       "List per-service LLM usage summaries",
		Example:     `  optikk llm apps --from 1h`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, out, startMs, endMs, err := setupRange(cmd, app, from, to)
			if err != nil {
				return err
			}
			resp, err := client.ListLLMApps(cmd.Context(), startMs, endMs)
			if err != nil {
				return err
			}
			headers := []string{"SERVICE", "VENDOR", "MODEL", "SPANS", "ERR%", "P95 ms", "IN TOK", "OUT TOK", "COST $"}
			items := make([]any, len(resp.Apps))
			for i := range resp.Apps {
				items[i] = resp.Apps[i]
			}
			return out.WriteItems(headers, items, func(v any) []string {
				a := v.(queryclient.LLMApp)
				return []string{
					a.Service,
					a.Vendor,
					a.PrimaryModel,
					fmt.Sprintf("%d", a.TotalSpans),
					fmt.Sprintf("%.2f", a.ErrorRate),
					fmt.Sprintf("%.1f", a.P95Ms),
					fmt.Sprintf("%d", a.InputTokens),
					fmt.Sprintf("%d", a.OutputTokens),
					fmt.Sprintf("%.4f", a.Cost),
				}
			})
		},
	}
	addRangeFlags(cmd, &from, &to)
	return cmd
}

func newLLMCostCmd(app *App) *cobra.Command {
	var from, to, groupBy string
	cmd := &cobra.Command{
		Use:         "cost",
		Short:       "Show LLM cost broken down by a dimension",
		Example:     `  optikk llm cost --group-by model --from 1h`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, out, startMs, endMs, err := setupRange(cmd, app, from, to)
			if err != nil {
				return err
			}
			resp, err := client.LLMCostBreakdown(cmd.Context(), startMs, endMs, groupBy)
			if err != nil {
				return err
			}
			return out.WriteJSON(resp)
		},
	}
	addRangeFlags(cmd, &from, &to)
	cmd.Flags().StringVar(&groupBy, "group-by", "", "breakdown dimension (e.g. model, service, vendor)")
	return cmd
}

func newLLMTimeseriesCmd(app *App) *cobra.Command {
	var from, to, metric string
	cmd := &cobra.Command{
		Use:         "timeseries",
		Short:       "Show an LLM metric timeseries",
		Example:     `  optikk llm timeseries --metric spend --from 1h`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, out, startMs, endMs, err := setupRange(cmd, app, from, to)
			if err != nil {
				return err
			}
			resp, err := client.LLMTimeseries(cmd.Context(), startMs, endMs, metric)
			if err != nil {
				return err
			}
			return out.WriteJSON(resp)
		},
	}
	addRangeFlags(cmd, &from, &to)
	cmd.Flags().StringVar(&metric, "metric", "spend", "metric: spend|latency|tokens_by_vendor")
	return cmd
}

func newLLMTracesCmd(app *App) *cobra.Command {
	var from, to string
	var limit int
	cmd := &cobra.Command{
		Use:         "traces",
		Short:       "Search LLM traces",
		Example:     `  optikk llm traces --from 1h --limit 20`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, out, startMs, endMs, err := setupRange(cmd, app, from, to)
			if err != nil {
				return err
			}
			resp, err := client.LLMTracesQuery(cmd.Context(), startMs, endMs, limit)
			if err != nil {
				return err
			}
			return out.WriteJSON(resp)
		},
	}
	addRangeFlags(cmd, &from, &to)
	cmd.Flags().IntVar(&limit, "limit", 20, "max results")
	return cmd
}

func newLLMTraceGetCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:         "trace <traceId>",
		Short:       "Get full detail for one LLM trace",
		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := resolveClient(app)
			if err != nil {
				return err
			}
			out := resolveOutput(cmd, app)
			resp, err := client.LLMTraceDetail(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return out.WriteJSON(resp)
		},
	}
}
