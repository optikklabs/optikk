package cmd

import (
	"fmt"
	"time"

	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/optikklabs/optikk/internal/clitime"
	"github.com/optikklabs/optikk/internal/output"
	"github.com/optikklabs/optikk/internal/queryclient"
	"github.com/spf13/cobra"
)

// resolveClient builds a queryclient.Client from app config, resolving the
// token from OPTIKK_TOKEN env or ~/.optikk/config.json.
func resolveClient(app *App) (*queryclient.Client, error) {
	apiBase, err := app.API()
	if err != nil {
		return nil, err
	}
	token, err := resolveToken(app)
	if err != nil {
		return nil, err
	}
	return queryclient.New(apiBase, token, app.Cfg.TenantID), nil
}

// resolveToken returns the session JWT from OPTIKK_TOKEN or the active context.
func resolveToken(app *App) (string, error) {
	if app.Cfg.Token != "" {
		return app.Cfg.Token, nil
	}
	ctx, err := apiclient.CurrentContext()
	if err != nil || ctx.Token == "" {
		return "", fmt.Errorf("not authenticated — run: optikk login\n  (or set OPTIKK_TOKEN env var)")
	}
	return ctx.Token, nil
}

// resolveOutput returns an output.Writer for the current command.
func resolveOutput(cmd *cobra.Command, app *App) *output.Writer {
	format := output.Resolve(app.Cfg.Output, app.AgentMode)
	return output.New(format, cmd.OutOrStdout())
}

// addRangeFlags registers the shared --from/--to time-range flags.
func addRangeFlags(cmd *cobra.Command, from, to *string) {
	cmd.Flags().StringVar(from, "from", "1h", "start time (1h, 15m, 7d, ISO8601, epoch-ms)")
	cmd.Flags().StringVar(to, "to", "now", "end time")
}

// setupRange resolves the client, output writer, and parsed time range for a
// range-scoped data command in one call.
func setupRange(cmd *cobra.Command, app *App, from, to string) (*queryclient.Client, *output.Writer, int64, int64, error) {
	client, err := resolveClient(app)
	if err != nil {
		return nil, nil, 0, 0, err
	}
	startMs, endMs, err := clitime.ParseRange(from, to, time.Now())
	if err != nil {
		return nil, nil, 0, 0, err
	}
	return client, resolveOutput(cmd, app), startMs, endMs, nil
}
