package cmd

import (
	"fmt"

	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/optikklabs/optikk/internal/conn"
	"github.com/optikklabs/optikk/internal/output"
	"github.com/optikklabs/optikk/internal/queryclient"
	"github.com/spf13/cobra"
)

// resolveClient builds a queryclient.Client from app config, resolving the
// token from OPTIKK_TOKEN env or ~/.optikk/token.json.
func resolveClient(app *App) (*queryclient.Client, error) {
	apiBase := conn.Resolve(app.Cfg.ApiURL)
	token := app.Cfg.Token
	teamID := app.Cfg.TeamID

	if token == "" {
		base, tok, err := apiclient.LoadToken()
		if err != nil {
			return nil, fmt.Errorf("not authenticated — run: optikk auth login\n  (or set OPTIKK_TOKEN env var)")
		}
		token = tok
		if app.Cfg.ApiURL == "" {
			apiBase = base
		}
	}

	return queryclient.New(apiBase, token, teamID), nil
}

// resolveOutput returns an output.Writer for the current command.
func resolveOutput(cmd *cobra.Command, app *App) *output.Writer {
	format := output.Resolve(app.Cfg.Output, app.AgentMode)
	return output.New(format, cmd.OutOrStdout())
}
