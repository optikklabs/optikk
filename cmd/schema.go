package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// AgentSchema represents the CLI command structure for AI discoverability.
type AgentSchema struct {
	Version  string         `json:"version"`
	Commands []AgentCommand `json:"commands"`
}

type AgentCommand struct {
	Use         string      `json:"use"`
	Short       string      `json:"short"`
	Long        string      `json:"long,omitempty"`
	Example     string      `json:"example,omitempty"`
	Flags       []AgentFlag `json:"flags,omitempty"`
	Subcommands []string    `json:"subcommands,omitempty"` // For groups
}

type AgentFlag struct {
	Name      string `json:"name"`
	Shorthand string `json:"shorthand,omitempty"`
	Type      string `json:"type"`
	Default   string `json:"default,omitempty"`
	Usage     string `json:"usage"`
}

func newAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "agent",
		Short:       "AI agent integrations",
	}
	cmd.AddCommand(newAgentSchemaCmd())
	return cmd
}

func newAgentSchemaCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "schema",
		Short: "Emit the CLI command tree as JSON for AI agents",
		RunE: func(cmd *cobra.Command, _ []string) error {
			root := cmd.Root()
			schema := AgentSchema{
				Version:  "1.0",
				Commands: extractCommands(root, ""),
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(schema)
		},
	}
}

func extractCommands(c *cobra.Command, prefix string) []AgentCommand {
	var cmds []AgentCommand
	for _, child := range c.Commands() {
		// Skip hidden, completion, and the agent command itself to avoid noise.
		if child.Hidden || child.Name() == "completion" || child.Name() == "agent" || child.Name() == "help" {
			continue
		}

		use := child.Use
		if prefix != "" {
			use = prefix + " " + use
		}

		ac := AgentCommand{
			Use:     use,
			Short:   child.Short,
			Long:    child.Long,
			Example: child.Example,
		}

		child.LocalFlags().VisitAll(func(f *pflag.Flag) {
			ac.Flags = append(ac.Flags, AgentFlag{
				Name:      f.Name,
				Shorthand: f.Shorthand,
				Type:      f.Value.Type(),
				Default:   f.DefValue,
				Usage:     f.Usage,
			})
		})

		for _, sub := range child.Commands() {
			if !sub.Hidden {
				ac.Subcommands = append(ac.Subcommands, sub.Name())
			}
		}

		cmds = append(cmds, ac)
		cmds = append(cmds, extractCommands(child, use)...)
	}
	return cmds
}
