package cmd

import (
	"context"
	"fmt"

	"github.com/optikklabs/optikk/internal/queryclient"
	"github.com/spf13/cobra"
)

func newSaturationCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "saturation",
		Short: "Inspect database and Kafka saturation",
		Long:  "Query the saturation API — datastore and Kafka throughput, latency, and lag.",
	}
	cmd.AddCommand(
		newSaturationDBSystemsCmd(app),
		newSaturationJSONCmd(app, "db-latency", "Show database latency by system", (*queryclient.Client).DBLatencyBySystem),
		newSaturationJSONCmd(app, "db-slow-queries", "Show slow-query patterns", (*queryclient.Client).DBSlowQueries),
		newSaturationJSONCmd(app, "db-ops", "Show database operation volume by system", (*queryclient.Client).DBOpsBySystem),
		newSaturationJSONCmd(app, "kafka-topology", "Show the Kafka producer/consumer topology", (*queryclient.Client).KafkaTopology),
		newSaturationJSONCmd(app, "kafka-throughput", "Show per-topic Kafka throughput", (*queryclient.Client).KafkaThroughput),
		newSaturationJSONCmd(app, "kafka-groups", "Show consumer-group partitions and lag", (*queryclient.Client).KafkaGroups),
	)
	return cmd
}

func newSaturationDBSystemsCmd(app *App) *cobra.Command {
	var from, to string
	cmd := &cobra.Command{
		Use:     "db-systems",
		Short:   "List per-system database saturation summaries",
		Example: `  optikk saturation db-systems --from 1h`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, out, startMs, endMs, err := setupRange(cmd, app, from, to)
			if err != nil {
				return err
			}
			rows, err := client.ListDatastoreSystems(cmd.Context(), startMs, endMs)
			if err != nil {
				return err
			}
			headers := []string{"SYSTEM", "CATEGORY", "QUERIES", "AVG ms", "P95 ms", "ERR%", "CONNS"}
			items := make([]any, len(rows))
			for i := range rows {
				items[i] = rows[i]
			}
			return out.WriteItems(headers, items, func(v any) []string {
				s := v.(queryclient.DatastoreSystem)
				return []string{
					s.System,
					s.Category,
					fmt.Sprintf("%d", s.QueryCount),
					fmt.Sprintf("%.1f", s.AvgLatencyMs),
					fmt.Sprintf("%.1f", s.P95LatencyMs),
					fmt.Sprintf("%.2f", s.ErrorRate),
					fmt.Sprintf("%d", s.ActiveConnections),
				}
			})
		},
	}
	addRangeFlags(cmd, &from, &to)
	return cmd
}

// newSaturationJSONCmd builds a range-scoped saturation command that emits raw
// analytical JSON from the given client method.
func newSaturationJSONCmd(app *App, use, short string, fn func(*queryclient.Client, context.Context, int64, int64) (any, error)) *cobra.Command {
	var from, to string
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, out, startMs, endMs, err := setupRange(cmd, app, from, to)
			if err != nil {
				return err
			}
			resp, err := fn(client, cmd.Context(), startMs, endMs)
			if err != nil {
				return err
			}
			return out.WriteJSON(resp)
		},
	}
	addRangeFlags(cmd, &from, &to)
	return cmd
}
