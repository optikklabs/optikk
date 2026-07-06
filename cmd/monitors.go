package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/optikklabs/optikk/internal/clitime"
	"github.com/optikklabs/optikk/internal/queryclient"
	"github.com/spf13/cobra"
	"time"
)

func newMonitorsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "monitors",
		Aliases:     []string{"mon"},
		Short:       "Manage monitors and alerts",
		Long:        "CRUD operations for monitors — list, get, create, update, delete, mute, unmute, ack, test.",
	}
	cmd.AddCommand(
		newMonitorsListCmd(app),
		newMonitorsGetCmd(app),
		newMonitorsCreateCmd(app),
		newMonitorsUpdateCmd(app),
		newMonitorsDeleteCmd(app),
		newMonitorsMuteCmd(app),
		newMonitorsUnmuteCmd(app),
		newMonitorsAckCmd(app),
		newMonitorsTestCmd(app),
	)
	return cmd
}

func newMonitorsListCmd(app *App) *cobra.Command {
	var status, monType, priority, search string
	var limit, offset int

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List monitors",
		Example: `  optikk monitors list --status triggered`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := resolveClient(app)
			if err != nil {
				return err
			}
			out := resolveOutput(cmd, app)

			monitors, err := client.ListMonitors(cmd.Context(), status, monType, priority, search, limit, offset)
			if err != nil {
				return err
			}

			headers := []string{"ID", "NAME", "TYPE", "STATUS", "PRIORITY", "MUTED"}
			items := make([]any, len(monitors))
			for i := range monitors {
				items[i] = monitors[i]
			}
			return out.WriteItems(headers, items, func(v any) []string {
				m := v.(queryclient.Monitor)
				muted := "no"
				if m.Muted {
					muted = "yes"
				}
				return []string{
					fmt.Sprintf("%d", m.ID),
					m.Name,
					m.Type,
					m.Status,
					m.Priority,
					muted,
				}
			})
		},
	}

	cmd.Flags().StringVar(&status, "status", "", "filter by status (e.g. triggered, ok)")
	cmd.Flags().StringVar(&monType, "type", "", "filter by type")
	cmd.Flags().StringVar(&priority, "priority", "", "filter by priority")
	cmd.Flags().StringVarP(&search, "search", "q", "", "search monitors by name")
	cmd.Flags().IntVar(&limit, "limit", 50, "max results")
	cmd.Flags().IntVar(&offset, "offset", 0, "pagination offset")
	return cmd
}

func newMonitorsGetCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get monitor details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := resolveClient(app)
			if err != nil {
				return err
			}
			out := resolveOutput(cmd, app)
			id := mustParseInt64(args[0])
			mon, err := client.GetMonitor(cmd.Context(), id)
			if err != nil {
				return err
			}
			return out.WriteJSON(mon)
		},
	}
}

func newMonitorsCreateCmd(app *App) *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a monitor from a JSON file",
		Example: `  optikk monitors create --file monitor.json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			data, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}
			client, err := resolveClient(app)
			if err != nil {
				return err
			}
			out := resolveOutput(cmd, app)
			mon, err := client.CreateMonitor(cmd.Context(), json.RawMessage(data))
			if err != nil {
				return err
			}
			out.Msg("✓ Created monitor %d: %s", mon.ID, mon.Name)
			return out.WriteJSON(mon)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "JSON file with monitor definition (required)")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}

func newMonitorsUpdateCmd(app *App) *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a monitor from a JSON file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}
			client, err := resolveClient(app)
			if err != nil {
				return err
			}
			out := resolveOutput(cmd, app)
			id := mustParseInt64(args[0])
			mon, err := client.UpdateMonitor(cmd.Context(), id, json.RawMessage(data))
			if err != nil {
				return err
			}
			out.Msg("✓ Updated monitor %d: %s", mon.ID, mon.Name)
			return out.WriteJSON(mon)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "JSON file with monitor definition (required)")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}

func newMonitorsDeleteCmd(app *App) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a monitor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := mustParseInt64(args[0])
			if !yes && !app.AgentMode {
				return fmt.Errorf("refusing to delete without --yes flag (monitor %d)", id)
			}
			client, err := resolveClient(app)
			if err != nil {
				return err
			}
			out := resolveOutput(cmd, app)
			if err := client.DeleteMonitor(cmd.Context(), id); err != nil {
				return err
			}
			out.Msg("✓ Deleted monitor %d", id)
			return nil
		},
	}

	cmd.Flags().BoolVar(&yes, "yes", false, "confirm deletion")
	return cmd
}

func newMonitorsMuteCmd(app *App) *cobra.Command {
	var duration string

	cmd := &cobra.Command{
		Use:     "mute <id>",
		Short:   "Mute a monitor for a duration",
		Args:    cobra.ExactArgs(1),
		Example: `  optikk monitors mute 5 --duration 1h`,
		RunE: func(cmd *cobra.Command, args []string) error {
			durMs, err := clitime.Parse(duration, time.Now())
			if err != nil {
				return fmt.Errorf("invalid --duration: %w", err)
			}
			// Parse returns epoch-ms for "now - duration"; we need seconds.
			nowMs := time.Now().UnixMilli()
			durationSec := int((nowMs - durMs) / 1000)
			if durationSec <= 0 {
				return fmt.Errorf("--duration must be a positive relative time (e.g. 1h, 30m)")
			}

			client, err := resolveClient(app)
			if err != nil {
				return err
			}
			out := resolveOutput(cmd, app)
			id := mustParseInt64(args[0])
			if err := client.MuteMonitor(cmd.Context(), id, durationSec); err != nil {
				return err
			}
			out.Msg("✓ Muted monitor %d for %s", id, duration)
			return nil
		},
	}

	cmd.Flags().StringVar(&duration, "duration", "1h", "mute duration (1h, 30m, etc.)")
	return cmd
}

func newMonitorsUnmuteCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "unmute <id>",
		Short: "Unmute a monitor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := resolveClient(app)
			if err != nil {
				return err
			}
			out := resolveOutput(cmd, app)
			id := mustParseInt64(args[0])
			if err := client.UnmuteMonitor(cmd.Context(), id); err != nil {
				return err
			}
			out.Msg("✓ Unmuted monitor %d", id)
			return nil
		},
	}
}

func newMonitorsAckCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "ack <id>",
		Short: "Acknowledge a triggered monitor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := resolveClient(app)
			if err != nil {
				return err
			}
			out := resolveOutput(cmd, app)
			id := mustParseInt64(args[0])
			if err := client.AckMonitor(cmd.Context(), id); err != nil {
				return err
			}
			out.Msg("✓ Acknowledged monitor %d", id)
			return nil
		},
	}
}

func newMonitorsTestCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "test <id>",
		Short: "Run a test evaluation of a monitor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := resolveClient(app)
			if err != nil {
				return err
			}
			out := resolveOutput(cmd, app)
			id := mustParseInt64(args[0])
			result, err := client.TestMonitor(cmd.Context(), id)
			if err != nil {
				return err
			}
			return out.WriteJSON(result)
		},
	}
}
