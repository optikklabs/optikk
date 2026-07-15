package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/optikklabs/optikk/internal/queryclient"
	"github.com/spf13/cobra"
)

func newInfraCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "infra",
		Short: "Inspect host, node, and pod infrastructure",
		Long:  "Query the infrastructure API — hosts, Kubernetes nodes, pods, CPU, and memory.",
	}
	cmd.AddCommand(
		newInfraHostsCmd(app),
		newInfraNodesCmd(app),
		newInfraPodsCmd(app),
		newInfraCPUCmd(app),
		newInfraMemoryCmd(app),
	)
	return cmd
}

func newInfraHostsCmd(app *App) *cobra.Command {
	var from, to, service string
	cmd := &cobra.Command{
		Use:     "hosts",
		Short:   "List host-level saturation and RED metrics",
		Example: `  optikk infra hosts --from 1h`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, out, startMs, endMs, err := setupRange(cmd, app, from, to)
			if err != nil {
				return err
			}
			rows, err := client.ListHosts(cmd.Context(), startMs, endMs, service)
			if err != nil {
				return err
			}
			headers := []string{"HOST", "CPU%", "MEM%", "DISK%", "SATURATION", "STATUS"}
			items := make([]any, len(rows))
			for i := range rows {
				items[i] = rows[i]
			}
			return out.WriteItems(headers, items, func(v any) []string {
				h := v.(queryclient.Host)
				return []string{
					h.Host,
					fmt.Sprintf("%.1f", h.CPU),
					fmt.Sprintf("%.1f", h.Mem),
					fmt.Sprintf("%.1f", h.Disk),
					fmt.Sprintf("%.2f", h.Saturation),
					h.Status,
				}
			})
		},
	}
	addRangeFlags(cmd, &from, &to)
	cmd.Flags().StringVar(&service, "service", "", "filter to hosts running one service")
	return cmd
}

func newInfraNodesCmd(app *App) *cobra.Command {
	var from, to string
	cmd := &cobra.Command{
		Use:     "nodes",
		Short:   "List Kubernetes node-level aggregates",
		Example: `  optikk infra nodes --from 1h`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, out, startMs, endMs, err := setupRange(cmd, app, from, to)
			if err != nil {
				return err
			}
			rows, err := client.ListNodes(cmd.Context(), startMs, endMs)
			if err != nil {
				return err
			}
			headers := []string{"HOST", "PODS", "REQUESTS", "ERRORS", "ERR%", "P95 ms"}
			items := make([]any, len(rows))
			for i := range rows {
				items[i] = rows[i]
			}
			return out.WriteItems(headers, items, func(v any) []string {
				n := v.(queryclient.InfraNode)
				return []string{
					n.Host,
					fmt.Sprintf("%d", n.PodCount),
					fmt.Sprintf("%d", n.RequestCount),
					fmt.Sprintf("%d", n.ErrorCount),
					fmt.Sprintf("%.2f", n.ErrorRate),
					fmt.Sprintf("%.1f", n.P95LatencyMs),
				}
			})
		},
	}
	addRangeFlags(cmd, &from, &to)
	return cmd
}

func newInfraPodsCmd(app *App) *cobra.Command {
	var from, to string
	cmd := &cobra.Command{
		Use:     "pods",
		Short:   "List pod-level RED aggregates",
		Example: `  optikk infra pods --from 1h`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, out, startMs, endMs, err := setupRange(cmd, app, from, to)
			if err != nil {
				return err
			}
			rows, err := client.ListFleetPods(cmd.Context(), startMs, endMs)
			if err != nil {
				return err
			}
			headers := []string{"POD", "HOST", "SERVICES", "REQUESTS", "ERRORS", "P95 ms"}
			items := make([]any, len(rows))
			for i := range rows {
				items[i] = rows[i]
			}
			return out.WriteItems(headers, items, func(v any) []string {
				p := v.(queryclient.FleetPod)
				return []string{
					p.PodName,
					p.Host,
					strings.Join(p.Services, ","),
					fmt.Sprintf("%d", p.RequestCount),
					fmt.Sprintf("%d", p.ErrorCount),
					fmt.Sprintf("%.1f", p.P95LatencyMs),
				}
			})
		},
	}
	addRangeFlags(cmd, &from, &to)
	return cmd
}

func newInfraCPUCmd(app *App) *cobra.Command {
	return newInstanceMetricCmd(app, "cpu", "Show per-instance CPU utilization", (*queryclient.Client).CPUByInstance)
}

func newInfraMemoryCmd(app *App) *cobra.Command {
	return newInstanceMetricCmd(app, "memory", "Show per-instance memory utilization", (*queryclient.Client).MemoryByInstance)
}

func newInstanceMetricCmd(app *App, use, short string, fn func(*queryclient.Client, context.Context, int64, int64) ([]queryclient.InstanceMetric, error)) *cobra.Command {
	var from, to string
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, out, startMs, endMs, err := setupRange(cmd, app, from, to)
			if err != nil {
				return err
			}
			rows, err := fn(client, cmd.Context(), startMs, endMs)
			if err != nil {
				return err
			}
			headers := []string{"HOST", "POD", "CONTAINER", "SERVICE", "VALUE"}
			items := make([]any, len(rows))
			for i := range rows {
				items[i] = rows[i]
			}
			return out.WriteItems(headers, items, func(v any) []string {
				m := v.(queryclient.InstanceMetric)
				val := "-"
				if m.Value != nil {
					val = fmt.Sprintf("%.2f", *m.Value)
				}
				return []string{m.Host, m.Pod, m.Container, m.ServiceName, val}
			})
		},
	}
	addRangeFlags(cmd, &from, &to)
	return cmd
}
