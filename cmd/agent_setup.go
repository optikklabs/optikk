package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/optikklabs/optikk/internal/agentdocs"
	"github.com/spf13/cobra"
)

// skillRelPath is where Claude Code discovers project skills.
const skillRelPath = ".claude/skills/optikk/SKILL.md"

func newAgentSetupCmd(app *App) *cobra.Command {
	var dir string
	var agentsMD, printOnly bool
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Install the Optikk agent guide into a project",
		Long: "Writes the guide that teaches AI coding agents to operate this CLI.\n" +
			"By default it creates " + skillRelPath + " (Claude Code); --agents-md\n" +
			"maintains a marked section in AGENTS.md for other agents. Both are fully\n" +
			"generated — re-run after upgrading optikk to refresh them.",
		Example: "  optikk agent setup\n" +
			"  optikk agent setup --agents-md\n" +
			"  optikk agent setup --print",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if printOnly {
				guide, err := agentdocs.Guide(version)
				if err != nil {
					return err
				}
				fmt.Fprint(cmd.OutOrStdout(), guide)
				return nil
			}

			doc := agentSetupDoc{Written: []string{}, Updated: []string{}}
			skill, err := agentdocs.Skill(version)
			if err != nil {
				return err
			}
			skillPath := filepath.Join(dir, filepath.FromSlash(skillRelPath))
			if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(skillPath, []byte(skill), 0o644); err != nil {
				return err
			}
			doc.Written = append(doc.Written, skillPath)

			if agentsMD {
				agentsPath := filepath.Join(dir, "AGENTS.md")
				existing, err := os.ReadFile(agentsPath)
				if err != nil && !os.IsNotExist(err) {
					return err
				}
				updated, err := agentdocs.UpsertAgentsSection(string(existing), version)
				if err != nil {
					return err
				}
				if err := os.WriteFile(agentsPath, []byte(updated), 0o644); err != nil {
					return err
				}
				if len(existing) == 0 {
					doc.Written = append(doc.Written, agentsPath)
				} else {
					doc.Updated = append(doc.Updated, agentsPath)
				}
			}

			return writeResult(cmd, app, doc, func(w io.Writer) {
				for _, p := range doc.Written {
					fmt.Fprintf(w, "✓ wrote %s\n", p)
				}
				for _, p := range doc.Updated {
					fmt.Fprintf(w, "✓ updated %s\n", p)
				}
			})
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "project directory to install into")
	cmd.Flags().BoolVar(&agentsMD, "agents-md", false, "also maintain a marked section in AGENTS.md")
	cmd.Flags().BoolVar(&printOnly, "print", false, "print the guide to stdout without writing files")
	return cmd
}

// agentSetupDoc is the machine-readable setup result.
type agentSetupDoc struct {
	Written []string `json:"written"`
	Updated []string `json:"updated"`
}
