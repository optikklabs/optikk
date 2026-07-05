package cmd

import (
	"fmt"

	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/spf13/cobra"
)

func newConfigCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "config",
		Aliases:     []string{"cfg"},
		Short:       "Inspect and switch CLI contexts",
		Annotations: map[string]string{annotationSkipDeploy: "true"},
	}
	cmd.AddCommand(
		newConfigShowCmd(app),
		newConfigGetContextsCmd(),
		newConfigSetContextCmd(),
		newConfigUseContextCmd(),
	)
	return cmd
}

func newConfigShowCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:     "show",
		Aliases: []string{"view"},
		Short:   "Print the active context",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, err := apiclient.CurrentContext()
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "api_url:       %s\n", ctx.APIURL)
			fmt.Fprintf(w, "tenant_id:       %d\n", ctx.TenantID)
			fmt.Fprintf(w, "authenticated: %t\n", ctx.Token != "")
			return nil
		},
	}
}

func newConfigGetContextsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get-contexts",
		Aliases: []string{"contexts"},
		Short:   "List all contexts (current marked with *)",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			names, contexts, current, err := apiclient.ListContexts()
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if len(names) == 0 {
				fmt.Fprintln(w, "no contexts yet; create one with: optikk config set-context <name> --api-url <url>")
				return nil
			}
			fmt.Fprintf(w, "%-2s %-12s %-28s %-8s %s\n", "", "NAME", "API_URL", "TENANT_ID", "AUTH")
			for _, name := range names {
				marker := ""
				if name == current {
					marker = "*"
				}
				c := contexts[name]
				fmt.Fprintf(w, "%-2s %-12s %-28s %-8d %t\n", marker, name, c.APIURL, c.TenantID, c.Token != "")
			}
			return nil
		},
	}
}

func newConfigSetContextCmd() *cobra.Command {
	var apiURL string
	var tenantID int64
	cmd := &cobra.Command{
		Use:   "set-context <name>",
		Short: "Create or update a context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := apiclient.SetContext(args[0], apiURL, tenantID); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "context %q saved. Switch with: optikk config use-context %s\n", args[0], args[0])
			return nil
		},
	}
	cmd.Flags().StringVar(&apiURL, "api-url", "", "query API base URL for this context")
	cmd.Flags().Int64Var(&tenantID, "tenant-id", 0, "default tenant for X-Tenant-Id in this context")
	return cmd
}

func newConfigUseContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use-context <name>",
		Short: "Switch the active context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := apiclient.UseContext(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "switched to context %q\n", args[0])
			return nil
		},
	}
}
