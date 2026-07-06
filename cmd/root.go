// Package cmd wires the optikk CLI commands onto a shared App context.
package cmd

import (
	"github.com/optikklabs/optikk/internal/config"
	"github.com/spf13/cobra"
)

// App is the shared context injected into every command. Commands depend on
// this, not on globals, so new commands and targets slot in without edits.
type App struct {
	Cfg config.Config

	// Data-CLI state (populated from persistent flags).
	AgentMode bool
}

// persistent flag values, resolved into App.Cfg in PersistentPreRunE.
type rootFlags struct {
	configFile string
	verbose    bool

	// Data-CLI flags.
	apiURL    string
	tenantID  int64
	output    string
	agentMode bool
}

// NewRootCmd builds the root command and registers every subcommand.
func NewRootCmd() *cobra.Command {
	app := &App{}
	f := &rootFlags{}

	root := &cobra.Command{
		Use:   "optikk",
		Short: "Query the Optikk observability stack",
		Long:  "optikk provides Datadog Pup-style data commands for traces, logs, metrics, dashboards, and monitors.",
		Example: `  # Data commands (Datadog Pup-style)
  optikk auth login
  optikk traces search --query "service:api" --from 1h
  optikk logs search --query "severity_text:ERROR" --from 15m
  optikk metrics list --from 1h
  optikk dashboards list
  optikk monitors list --status triggered`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return app.load(cmd, f)
		},
	}

	pf := root.PersistentFlags()
	pf.StringVar(&f.configFile, "config", "", "config file (default: ./optikk.yaml or ~/.optikk/config)")
	pf.BoolVarP(&f.verbose, "verbose", "v", false, "verbose output")

	// Data-CLI persistent flags (Datadog Pup-style).
	pf.StringVar(&f.apiURL, "api-url", "", "query API base URL (OPTIKK_API_URL, default http://localhost:19090)")
	pf.Int64Var(&f.tenantID, "tenant-id", 0, "tenant context for X-Tenant-Id header (OPTIKK_TENANT_ID)")
	pf.StringVarP(&f.output, "output", "o", "", "output format: table|json|yaml (auto-detected from TTY)")
	pf.BoolVar(&f.agentMode, "agent", false, "agent mode: JSON output, skip confirmations (FORCE_AGENT_MODE)")

	root.AddCommand(
		newConfigCmd(app),
		newCompletionCmd(root),
		newVersionCmd(app),
	)

	// Data commands (Datadog Pup-style).
	root.AddCommand(
		newAuthCmd(app),
		newLoginCmd(app),
		newSignupCmd(app),
		newOnboardCmd(app),
		newKeysCmd(app),
		newTracesCmd(app),
		newLogsCmd(app),
		newMetricsCmd(app),
		newServicesCmd(app),
		newInfraCmd(app),
		newLLMCmd(app),
		newSaturationCmd(app),
		newDashboardsCmd(app),
		newMonitorsCmd(app),
		newAgentCmd(),
		newUsersCmd(app), // Re-parented from tenant member
	)

	return root
}

// load merges config file, env, and flags into app.Cfg.
func (a *App) load(cmd *cobra.Command, f *rootFlags) error {
	cfg, err := config.Load(f.configFile)
	if err != nil {
		return err
	}

	// Flags override file/env only when explicitly set.
	if cmd.Flags().Changed("verbose") {
		cfg.Verbose = f.verbose
	}
	if cmd.Flags().Changed("api-url") {
		cfg.ApiURL = f.apiURL
	}
	if cmd.Flags().Changed("tenant-id") {
		cfg.TenantID = f.tenantID
	}
	if cmd.Flags().Changed("output") {
		cfg.Output = f.output
	}
	a.AgentMode = f.agentMode

	a.Cfg = cfg
	return nil
}
