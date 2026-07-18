package cmd

import (
	"bufio"
	"fmt"
	"io"

	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/optikklabs/optikk/internal/endpoint"
	"github.com/spf13/cobra"
)

func newConfigCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "config",
		Aliases: []string{"cfg"},
		Short:   "Inspect and switch CLI contexts",
	}
	cmd.AddCommand(
		newConfigInitCmd(),
		newConfigShowCmd(app),
		newConfigGetContextsCmd(),
		newConfigSetContextCmd(),
		newConfigUseContextCmd(),
		newConfigCurrentContextCmd(),
		newConfigSetCmd(),
		newConfigUnsetCmd(),
		newConfigDeleteContextCmd(),
	)
	return cmd
}

// configKeys are the settable fields on a context, named as they appear on disk.
const configKeys = "api_url, tenant_id"

func newConfigCurrentContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "current-context",
		Short: "Print the active context's name",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			name, err := apiclient.CurrentContextName()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), name)
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	var context string
	cmd := &cobra.Command{
		Use:     "set <key> <value>",
		Short:   "Set one field on a context (" + configKeys + ")",
		Args:    cobra.ExactArgs(2),
		Example: "  optikk config set api_url https://api.optikk.in\n  optikk config set tenant_id 42",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := targetContext(context)
			if err != nil {
				return err
			}
			if err := apiclient.SetContextValue(name, args[0], args[1]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "context %q: %s = %s\n", name, args[0], args[1])
			return nil
		},
	}
	cmd.Flags().StringVar(&context, "context", "", "context to modify (default: the active one)")
	return cmd
}

func newConfigUnsetCmd() *cobra.Command {
	var context string
	cmd := &cobra.Command{
		Use:     "unset <key>",
		Short:   "Clear one field on a context (" + configKeys + ")",
		Args:    cobra.ExactArgs(1),
		Example: "  optikk config unset tenant_id",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := targetContext(context)
			if err != nil {
				return err
			}
			if err := apiclient.UnsetContextValue(name, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "context %q: %s cleared\n", name, args[0])
			return nil
		},
	}
	cmd.Flags().StringVar(&context, "context", "", "context to modify (default: the active one)")
	return cmd
}

func newConfigDeleteContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete-context <name>",
		Short:   "Delete a context and its cached session",
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"unset-context"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := apiclient.DeleteContext(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "deleted context %q\n", args[0])
			return nil
		},
	}
}

// targetContext returns the named context, or the active one when unnamed.
func targetContext(name string) (string, error) {
	if name != "" {
		return name, nil
	}
	return apiclient.CurrentContextName()
}

func newConfigInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize a new CLI context interactively",
		RunE: func(cmd *cobra.Command, _ []string) error {
			in := bufio.NewReader(cmd.InOrStdin())
			apiURL, err := promptIfEmpty(cmd, in, "API URL ["+endpoint.APIURL+"]", "")
			if err != nil {
				return err
			}
			if apiURL == "" {
				apiURL = endpoint.APIURL
			}
			// Reject an unusable URL now, not on the next command.
			if apiURL, err = endpoint.Resolve(apiURL, ""); err != nil {
				return err
			}

			name, err := promptIfEmpty(cmd, in, "Context name [default]", "")
			if err != nil {
				return err
			}
			if name == "" {
				name = "default"
			}

			if err := apiclient.SetContext(name, apiURL, 0); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Context %q saved. Run `optikk login` to authenticate.\n", name)
			return nil
		},
	}
}

// configShowDoc is the machine-readable active-context report.
type configShowDoc struct {
	Context       string `json:"context,omitempty"`
	APIURL        string `json:"api_url"`
	APIURLError   string `json:"api_url_error,omitempty"`
	TenantID      int64  `json:"tenant_id"`
	Authenticated bool   `json:"authenticated"`
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
			name, _ := apiclient.CurrentContextName()
			doc := configShowDoc{Context: name, TenantID: ctx.TenantID, Authenticated: ctx.Token != ""}
			// Report the effective API, but never fail: `config show` has to stay
			// usable precisely when the stored URL is the thing that is wrong.
			if apiBase, err := app.API(); err != nil {
				doc.APIURL = ctx.APIURL
				doc.APIURLError = firstLine(err)
			} else {
				doc.APIURL = apiBase
			}
			return writeResult(cmd, app, doc, func(w io.Writer) {
				apiLine := doc.APIURL
				if doc.APIURLError != "" {
					apiLine = fmt.Sprintf("%s (unusable — %s)", doc.APIURL, doc.APIURLError)
				}
				fmt.Fprintf(w, "api_url:       %s\n", apiLine)
				fmt.Fprintf(w, "tenant_id:     %d\n", doc.TenantID)
				fmt.Fprintf(w, "authenticated: %t\n", doc.Authenticated)
			})
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
