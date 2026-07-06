package cmd

import (
	"fmt"
	"time"

	"github.com/optikklabs/optikk/internal/clitime"
	"github.com/optikklabs/optikk/internal/queryclient"
	"github.com/spf13/cobra"
)

func newServicesCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "services",
		Short:       "Inspect the service fleet and dependency topology",
		Long:        "Query the services API — RED metrics, dependency topology, and error groups.",
	}
	cmd.AddCommand(
		newServicesListCmd(app),
		newServicesTopologyCmd(app),
		newServicesErrorsCmd(app),
		newServicesSummaryCmd(app),
		newServicesTopEndpointsCmd(app),
	)
	return cmd
}

func newServicesListCmd(app *App) *cobra.Command {
	var from, to string
	cmd := &cobra.Command{
		Use:         "list",
		Short:       "List RED metrics for every service in the fleet",
		Example:     `  optikk services list --from 1h`,
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
			rows, err := client.ListFleetServices(cmd.Context(), startMs, endMs)
			if err != nil {
				return err
			}
			headers := []string{"SERVICE", "REQUESTS", "ERRORS", "AVG ms", "P95 ms", "P99 ms"}
			items := make([]any, len(rows))
			for i := range rows {
				items[i] = rows[i]
			}
			return out.WriteItems(headers, items, func(v any) []string {
				s := v.(queryclient.ServiceREDMetric)
				return []string{
					s.ServiceName,
					fmt.Sprintf("%d", s.RequestCount),
					fmt.Sprintf("%d", s.ErrorCount),
					fmt.Sprintf("%.1f", s.AvgLatency),
					fmt.Sprintf("%.1f", s.P95Latency),
					fmt.Sprintf("%.1f", s.P99Latency),
				}
			})
		},
	}
	addRangeFlags(cmd, &from, &to)
	return cmd
}

func newServicesTopologyCmd(app *App) *cobra.Command {
	var from, to, service string
	cmd := &cobra.Command{
		Use:         "topology",
		Short:       "Show the service dependency graph",
		Example:     `  optikk services topology --service api --from 1h`,
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
			resp, err := client.GetTopology(cmd.Context(), startMs, endMs, service)
			if err != nil {
				return err
			}
			return out.WriteJSON(resp)
		},
	}
	addRangeFlags(cmd, &from, &to)
	cmd.Flags().StringVar(&service, "service", "", "focus the graph on one service")
	return cmd
}

func newServicesErrorsCmd(app *App) *cobra.Command {
	var from, to, service string
	var limit int
	cmd := &cobra.Command{
		Use:         "errors",
		Short:       "List aggregated error groups",
		Example:     `  optikk services errors --service api --from 1h`,
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
			resp, err := client.ListErrorGroups(cmd.Context(), startMs, endMs, service, limit)
			if err != nil {
				return err
			}
			headers := []string{"SERVICE", "OPERATION", "STATUS", "MESSAGE", "COUNT", "SAMPLE TRACE"}
			items := make([]any, len(resp.Results))
			for i := range resp.Results {
				items[i] = resp.Results[i]
			}
			return out.WriteItems(headers, items, func(v any) []string {
				g := v.(queryclient.ErrorGroup)
				msg := g.StatusMessage
				if len(msg) > 60 {
					msg = msg[:57] + "..."
				}
				return []string{
					g.ServiceName,
					g.OperationName,
					fmt.Sprintf("%d", g.HTTPStatusCode),
					msg,
					fmt.Sprintf("%d", g.ErrorCount),
					g.SampleTraceID,
				}
			})
		},
	}
	addRangeFlags(cmd, &from, &to)
	cmd.Flags().StringVar(&service, "service", "", "filter to one service")
	cmd.Flags().IntVar(&limit, "limit", 20, "max groups")
	return cmd
}

func newServicesSummaryCmd(app *App) *cobra.Command {
	var from, to, service string
	cmd := &cobra.Command{
		Use:         "summary",
		Short:       "Show the RED summary for one service",
		Example:     `  optikk services summary --service api --from 1h`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if service == "" {
				return fmt.Errorf("--service is required")
			}
			client, err := resolveClient(app)
			if err != nil {
				return err
			}
			out := resolveOutput(cmd, app)
			startMs, endMs, err := clitime.ParseRange(from, to, time.Now())
			if err != nil {
				return err
			}
			resp, err := client.ServiceSummary(cmd.Context(), startMs, endMs, service)
			if err != nil {
				return err
			}
			return out.WriteJSON(resp)
		},
	}
	addRangeFlags(cmd, &from, &to)
	cmd.Flags().StringVar(&service, "service", "", "service name (required)")
	return cmd
}

func newServicesTopEndpointsCmd(app *App) *cobra.Command {
	var from, to, service string
	cmd := &cobra.Command{
		Use:         "top-endpoints",
		Short:       "Show the busiest endpoints across the fleet or one service",
		Example:     `  optikk services top-endpoints --service api --from 1h`,
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
			resp, err := client.TopEndpoints(cmd.Context(), startMs, endMs, service)
			if err != nil {
				return err
			}
			return out.WriteJSON(resp)
		},
	}
	addRangeFlags(cmd, &from, &to)
	cmd.Flags().StringVar(&service, "service", "", "filter to one service")
	return cmd
}
