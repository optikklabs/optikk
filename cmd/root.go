// Package cmd wires the optikk CLI commands onto a shared App context.
package cmd

import (
	"fmt"

	"github.com/optikklabs/optikk/internal/config"
	"github.com/optikklabs/optikk/internal/deploypath"
	"github.com/spf13/cobra"
)

// App is the shared context injected into every command. Commands depend on
// this, not on globals, so new commands and targets slot in without edits.
type App struct {
	Cfg       config.Config
	DeployDir string

	// Data-CLI state (populated from persistent flags).
	AgentMode bool
}

const (
	annotationNoConfig  = "optikk/no-config"
	// annotationSkipDeploy marks commands that need config but not deploy/.
	// Auth and data commands use this — they only need API URL + token.
	annotationSkipDeploy = "optikk/skip-deploy"
)

// persistent flag values, resolved into App.Cfg in PersistentPreRunE.
type rootFlags struct {
	configFile string
	target     string
	deployDir  string
	verbose    bool

	// Data-CLI flags.
	apiURL    string
	teamID    int64
	output    string
	agentMode bool
}

// NewRootCmd builds the root command and registers every subcommand.
func NewRootCmd() *cobra.Command {
	app := &App{}
	f := &rootFlags{}

	root := &cobra.Command{
		Use:           "optikk",
		Short:         "Provision, operate, and query the Optikk observability stack",
		Long:          "optikk provisions and operates the full Optikk observability stack, and provides Datadog Pup-style data commands for traces, logs, metrics, dashboards, and monitors.",
		Example: `  # Ops commands
  optikk doctor
  optikk up
  optikk verify --api-key <api-key>
  optikk status
  optikk down

  # Data commands (Datadog Pup-style)
  optikk auth login
  optikk traces search --query "service:api" --from 1h
  optikk logs search --query "severity_text:ERROR" --from 15m
  optikk metrics list --from 1h
  optikk dashboards list
  optikk monitors list --status triggered`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if skipsConfig(cmd) {
				return nil
			}
			return app.load(cmd, f)
		},
	}

	pf := root.PersistentFlags()
	pf.StringVar(&f.configFile, "config", "", "config file (default: ./optikk.yaml or ~/.optikk/config)")
	pf.StringVar(&f.target, "target", "", "deployment target: local (default local)")
	pf.StringVar(&f.deployDir, "deploy-dir", "", "path to the deploy/ kustomize tree (auto-detected)")
	pf.BoolVarP(&f.verbose, "verbose", "v", false, "verbose output")

	// Data-CLI persistent flags (Datadog Pup-style).
	pf.StringVar(&f.apiURL, "api-url", "", "query API base URL (OPTIKK_API_URL, default http://localhost:8080)")
	pf.Int64Var(&f.teamID, "team-id", 0, "team context for X-Team-Id header (OPTIKK_TEAM_ID)")
	pf.StringVarP(&f.output, "output", "o", "", "output format: table|json|yaml (auto-detected from TTY)")
	pf.BoolVar(&f.agentMode, "agent", false, "agent mode: JSON output, skip confirmations (FORCE_AGENT_MODE)")

	// Ops commands.
	root.AddCommand(
		newUpCmd(app),
		newDownCmd(app),
		newStatusCmd(app),
		newVerifyCmd(app),
		newDoctorCmd(),
		newTenantCmd(app),
		newAdminCmd(app),
		newTeamCmd(app),
		newConfigCmd(app),
		newCompletionCmd(root),
		newVersionCmd(app),
	)

	// Data commands (Datadog Pup-style).
	root.AddCommand(
		newAuthCmd(app),
		newTracesCmd(app),
		newLogsCmd(app),
		newMetricsCmd(app),
		newDashboardsCmd(app),
		newMonitorsCmd(app),
		newAgentCmd(),
	)

	return root
}

func skipsConfig(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Annotations[annotationNoConfig] == "true" {
			return true
		}
	}
	return false
}

func skipsDeploy(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Annotations[annotationSkipDeploy] == "true" {
			return true
		}
	}
	return false
}

// load merges config file, env, and flags into app.Cfg and resolves deploy/.
func (a *App) load(cmd *cobra.Command, f *rootFlags) error {
	cfg, err := config.Load(f.configFile)
	if err != nil {
		return err
	}

	// Flags override file/env only when explicitly set.
	if cmd.Flags().Changed("target") {
		cfg.Target = config.Target(f.target)
	}
	if cmd.Flags().Changed("deploy-dir") {
		cfg.DeployDir = f.deployDir
	}
	if cmd.Flags().Changed("verbose") {
		cfg.Verbose = f.verbose
	}
	if cmd.Flags().Changed("api-url") {
		cfg.ApiURL = f.apiURL
	}
	if cmd.Flags().Changed("team-id") {
		cfg.TeamID = f.teamID
	}
	if cmd.Flags().Changed("output") {
		cfg.Output = f.output
	}
	a.AgentMode = f.agentMode

	// Data commands skip deploy path + target validation.
	if skipsDeploy(cmd) {
		a.Cfg = cfg
		return nil
	}

	switch cfg.Target {
	case config.TargetLocal:
	default:
		return fmt.Errorf("invalid --target %q (want local)", cfg.Target)
	}

	dir, err := deploypath.Resolve(cfg.DeployDir)
	if err != nil {
		return err
	}
	a.Cfg = cfg
	a.DeployDir = dir
	return nil
}
