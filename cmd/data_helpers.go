package cmd

import (
	"encoding/json"
	"io"
	"time"

	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/optikklabs/optikk/internal/clierr"
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
		return "", clierr.New(clierr.Auth, "not authenticated",
			"run: optikk login (or set OPTIKK_TOKEN)")
	}
	return ctx.Token, nil
}

// resolveOutput returns an output.Writer for the current command.
func resolveOutput(cmd *cobra.Command, app *App) *output.Writer {
	format := output.Resolve(app.Cfg.Output, app.AgentMode)
	return output.New(format, cmd.OutOrStdout())
}

// writeResult renders a typed result as JSON/YAML per the resolved format, or
// calls human for table/interactive output. Lifecycle commands use it to keep
// their human text unchanged while giving agents a parseable document.
func writeResult(cmd *cobra.Command, app *App, v any, human func(w io.Writer)) error {
	ow := resolveOutput(cmd, app)
	switch ow.Format {
	case output.FormatJSON:
		return ow.WriteJSON(v)
	case output.FormatYAML:
		return ow.WriteYAML(v)
	default:
		human(ow.Out)
		return nil
	}
}

// writeNDJSON emits one compact JSON object per line, for commands that
// report progress before their final result (e.g. the device login flow).
func writeNDJSON(w io.Writer, v any) {
	_ = json.NewEncoder(w).Encode(v)
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
