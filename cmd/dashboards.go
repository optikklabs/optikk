package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/optikklabs/optikk/internal/conn"
	"github.com/optikklabs/optikk/internal/queryclient"
	"github.com/spf13/cobra"
)

func newDashboardsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dashboards",
		Aliases: []string{"dash"},
		Short:   "Manage dashboards",
		Long:    "CRUD operations for dashboard pages — list, get, create, update, delete, export, import.",
		Annotations: map[string]string{annotationSkipDeploy: "true"},
	}
	cmd.AddCommand(
		newDashboardsListCmd(app),
		newDashboardsGetCmd(app),
		newDashboardsCreateCmd(app),
		newDashboardsUpdateCmd(app),
		newDashboardsDeleteCmd(app),
		newDashboardsExportCmd(app),
		newDashboardsImportCmd(app),
		newDashboardsURLCmd(app),
	)
	return cmd
}

func newDashboardsListCmd(app *App) *cobra.Command {
	var search, tag string
	var limit, offset int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List dashboard pages",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := resolveClient(app)
			if err != nil {
				return err
			}
			out := resolveOutput(cmd, app)

			pages, err := client.ListDashboards(cmd.Context(), search, tag, limit, offset)
			if err != nil {
				return err
			}

			headers := []string{"ID", "NAME", "TAGS", "UPDATED"}
			items := make([]any, len(pages))
			for i := range pages {
				items[i] = pages[i]
			}
			return out.WriteItems(headers, items, func(v any) []string {
				p := v.(queryclient.DashboardPage)
				return []string{
					fmt.Sprintf("%d", p.ID),
					p.Name,
					fmt.Sprintf("%v", p.Tags),
					p.UpdatedAt.Format("2006-01-02 15:04"),
				}
			})
		},
	}

	cmd.Flags().StringVar(&search, "search", "", "filter by name")
	cmd.Flags().StringVar(&tag, "tag", "", "filter by tag")
	cmd.Flags().IntVar(&limit, "limit", 50, "max results")
	cmd.Flags().IntVar(&offset, "offset", 0, "pagination offset")
	return cmd
}

func newDashboardsGetCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get dashboard page details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := resolveClient(app)
			if err != nil {
				return err
			}
			out := resolveOutput(cmd, app)
			id := mustParseInt64(args[0])
			page, err := client.GetDashboard(cmd.Context(), id)
			if err != nil {
				return err
			}
			return out.WriteJSON(page)
		},
	}
}

func newDashboardsCreateCmd(app *App) *cobra.Command {
	var name, description, icon, iconColor string
	var tags []string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a new dashboard page",
		Example: `  optikk dashboards create --name "API Overview" --tags api,production`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := resolveClient(app)
			if err != nil {
				return err
			}
			out := resolveOutput(cmd, app)
			page, err := client.CreateDashboard(cmd.Context(), queryclient.CreatePageRequest{
				Name:        name,
				Description: description,
				Icon:        icon,
				IconColor:   iconColor,
				Tags:        tags,
			})
			if err != nil {
				return err
			}
			out.Msg("✓ Created dashboard %d: %s", page.ID, page.Name)
			return out.WriteJSON(page)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "dashboard name (required)")
	cmd.Flags().StringVar(&description, "description", "", "dashboard description")
	cmd.Flags().StringVar(&icon, "icon", "", "dashboard icon")
	cmd.Flags().StringVar(&iconColor, "icon-color", "", "dashboard icon color")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "dashboard tags")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newDashboardsUpdateCmd(app *App) *cobra.Command {
	var name, description string
	var tags []string

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a dashboard page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := resolveClient(app)
			if err != nil {
				return err
			}
			out := resolveOutput(cmd, app)
			id := mustParseInt64(args[0])
			page, err := client.UpdateDashboard(cmd.Context(), id, queryclient.CreatePageRequest{
				Name:        name,
				Description: description,
				Tags:        tags,
			})
			if err != nil {
				return err
			}
			out.Msg("✓ Updated dashboard %d", page.ID)
			return out.WriteJSON(page)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "new name")
	cmd.Flags().StringVar(&description, "description", "", "new description")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "new tags")
	return cmd
}

func newDashboardsDeleteCmd(app *App) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a dashboard page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := mustParseInt64(args[0])
			if !yes && !app.AgentMode {
				return fmt.Errorf("refusing to delete without --yes flag (dashboard %d)", id)
			}
			client, err := resolveClient(app)
			if err != nil {
				return err
			}
			out := resolveOutput(cmd, app)
			if err := client.DeleteDashboard(cmd.Context(), id); err != nil {
				return err
			}
			out.Msg("✓ Deleted dashboard %d", id)
			return nil
		},
	}

	cmd.Flags().BoolVar(&yes, "yes", false, "confirm deletion")
	return cmd
}

func newDashboardsExportCmd(app *App) *cobra.Command {
	var outputFile string

	cmd := &cobra.Command{
		Use:     "export <id>",
		Short:   "Export a dashboard as JSON",
		Args:    cobra.ExactArgs(1),
		Example: `  optikk dashboards export 3 -o dashboard.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := resolveClient(app)
			if err != nil {
				return err
			}
			id := mustParseInt64(args[0])
			export, err := client.ExportDashboard(cmd.Context(), id)
			if err != nil {
				return err
			}
			data, err := json.MarshalIndent(export, "", "  ")
			if err != nil {
				return err
			}
			if outputFile != "" {
				if err := os.WriteFile(outputFile, data, 0o644); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "✓ Exported dashboard %d to %s\n", id, outputFile)
				return nil
			}
			_, err = cmd.OutOrStdout().Write(append(data, '\n'))
			return err
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output-file", "f", "", "write to file instead of stdout")
	return cmd
}

func newDashboardsImportCmd(app *App) *cobra.Command {
	var file, name string

	cmd := &cobra.Command{
		Use:     "import",
		Short:   "Import a dashboard from JSON",
		Example: `  optikk dashboards import -f dashboard.json --name "API Overview Copy"`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			data, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("reading import file: %w", err)
			}
			var export queryclient.DashboardExport
			if err := json.Unmarshal(data, &export); err != nil {
				return fmt.Errorf("parsing import file: %w", err)
			}
			client, err := resolveClient(app)
			if err != nil {
				return err
			}
			out := resolveOutput(cmd, app)
			page, err := client.ImportDashboard(cmd.Context(), export, name)
			if err != nil {
				return err
			}
			out.Msg("✓ Imported dashboard %d: %s", page.ID, page.Name)
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "JSON file to import (required)")
	cmd.Flags().StringVar(&name, "name", "", "override dashboard name")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}

func newDashboardsURLCmd(app *App) *cobra.Command {
	var from string

	cmd := &cobra.Command{
		Use:     "url <id>",
		Short:   "Print the web UI URL for a dashboard",
		Args:    cobra.ExactArgs(1),
		Example: `  optikk dashboards url 3 --from 1h`,
		RunE: func(cmd *cobra.Command, args []string) error {
			apiBase := conn.Resolve(app.Cfg.ApiURL)
			id := args[0]
			url := fmt.Sprintf("%s/dashboards/%s", apiBase, id)
			if from != "" {
				url += "?from=" + from
			}
			fmt.Fprintln(cmd.OutOrStdout(), url)
			return nil
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "time range to embed in URL")
	return cmd
}

func mustParseInt64(s string) int64 {
	var n int64
	fmt.Sscanf(s, "%d", &n)
	return n
}
