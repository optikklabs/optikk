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
}

// persistent flag values, resolved into App.Cfg in PersistentPreRunE.
type rootFlags struct {
	configFile string
	target     string
	deployDir  string
	verbose    bool
}

// NewRootCmd builds the root command and registers every subcommand.
func NewRootCmd() *cobra.Command {
	app := &App{}
	f := &rootFlags{}

	root := &cobra.Command{
		Use:           "optikk",
		Short:         "Provision and operate the Optikk stack on local (kind) or GCP (GKE)",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return app.load(cmd, f)
		},
	}

	pf := root.PersistentFlags()
	pf.StringVar(&f.configFile, "config", "", "config file (default: ./optikk.yaml or ~/.optikk/config)")
	pf.StringVar(&f.target, "target", "", "deployment target: local|gcp (default local)")
	pf.StringVar(&f.deployDir, "deploy-dir", "", "path to the deploy/ kustomize tree (auto-detected)")
	pf.BoolVarP(&f.verbose, "verbose", "v", false, "verbose output")

	// Registering a command is one line here; see cmd/*.go for definitions.
	root.AddCommand(
		newUpCmd(app),
		newDownCmd(app),
		newStatusCmd(app),
		newVerifyCmd(app),
		newTenantCmd(app),
		newAdminCmd(app),
		newTeamCmd(app),
		newConfigCmd(app),
		newVersionCmd(app),
	)
	return root
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

	switch cfg.Target {
	case config.TargetLocal, config.TargetGCP:
	default:
		return fmt.Errorf("invalid --target %q (want local or gcp)", cfg.Target)
	}

	dir, err := deploypath.Resolve(cfg.DeployDir)
	if err != nil {
		return err
	}
	a.Cfg = cfg
	a.DeployDir = dir
	return nil
}
