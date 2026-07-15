package cmd

import (
	"fmt"

	"github.com/optikklabs/optikk/internal/browser"
	"github.com/optikklabs/optikk/internal/endpoint"
	"github.com/spf13/cobra"
)

// link is a command that just opens a URL. They are declared as data rather
// than as a file of near-identical commands.
type link struct {
	name  string
	short string
	url   string
}

var links = []link{
	{name: "docs", short: "Open the Optikk documentation", url: endpoint.DocsURL},
	{name: "support", short: "Open Optikk support", url: endpoint.SiteURL + "/support"},
	{name: "feedback", short: "Report a bug or request a feature", url: "https://github.com/optikklabs/optikk/issues/new"},
}

// newLinkCmds builds one command per link.
func newLinkCmds() []*cobra.Command {
	cmds := make([]*cobra.Command, 0, len(links))
	for _, l := range links {
		cmds = append(cmds, newLinkCmd(l))
	}
	return cmds
}

func newLinkCmd(l link) *cobra.Command {
	return &cobra.Command{
		Use:   l.name,
		Short: l.short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Print before opening: on a headless machine the link is the output.
			fmt.Fprintln(cmd.OutOrStdout(), l.url)
			browser.Open(l.url)
			return nil
		},
	}
}
