package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/optikklabs/optikk/internal/browser"
	"github.com/optikklabs/optikk/internal/endpoint"
	"github.com/spf13/cobra"
)

// appPages maps a shorthand to its path in the web app. Keep these in step
// with the web app's route tree (web/src/routeTree.gen.ts).
var appPages = map[string]string{
	"home":           "/",
	"overview":       "/overview",
	"traces":         "/traces",
	"logs":           "/logs",
	"metrics":        "/metrics",
	"services":       "/services",
	"infrastructure": "/infrastructure",
	"saturation":     "/saturation",
	"errors":         "/errors",
	"dashboards":     "/dashboards",
	"monitors":       "/monitors",
	"llm":            "/llm",
	"settings":       "/settings",
}

func newOpenCmd(_ *App) *cobra.Command {
	return &cobra.Command{
		Use:       "open [page]",
		Short:     "Open the Optikk web app in your browser",
		Long:      "Opens the web app at " + endpoint.AppURL + ", optionally jumping to a page.",
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: pageNames(),
		Example:   "  optikk open\n  optikk open traces\n  optikk open dashboards",
		RunE: func(cmd *cobra.Command, args []string) error {
			page := "home"
			if len(args) == 1 {
				page = strings.ToLower(args[0])
			}
			path, ok := appPages[page]
			if !ok {
				return fmt.Errorf("unknown page %q; try one of: %s", page, strings.Join(pageNames(), ", "))
			}

			url := strings.TrimSuffix(endpoint.AppURL+path, "/")
			// Print before opening: on a headless machine the link is the output.
			fmt.Fprintln(cmd.OutOrStdout(), url)
			browser.Open(url)
			return nil
		},
	}
}

// pageNames returns the shorthands accepted by `optikk open`, sorted so help
// text and completions are stable.
func pageNames() []string {
	names := make([]string, 0, len(appPages))
	for name := range appPages {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
