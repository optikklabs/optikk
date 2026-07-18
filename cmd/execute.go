package cmd

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/optikklabs/optikk/internal/clierr"
	"github.com/spf13/cobra"
)

// Execute runs the CLI and returns the process exit code. Failures are
// rendered on stderr — prose with an indented hint for humans, a single JSON
// envelope in agent mode — and the exit code follows the clierr taxonomy.
func Execute() int {
	app := &App{}
	root := newRootCmd(app)
	err := root.Execute()
	if err == nil {
		return 0
	}

	// A silent exit carries a status, not a failure; the command has already
	// said what it needed to on stdout.
	var silent SilentExitError
	if errors.As(err, &silent) {
		return silent.Code
	}

	if agentModeForErrors(app) {
		clierr.RenderJSON(os.Stderr, err)
	} else {
		fmt.Fprintln(os.Stderr, "error:", err)
		if hint := clierr.Hint(err); hint != "" {
			fmt.Fprintln(os.Stderr, " ", hint)
		}
	}
	return clierr.ExitCode(err)
}

// agentModeForErrors decides how to render a failure. App.AgentMode is
// authoritative once PersistentPreRunE has run, but flag-parse failures occur
// before that, so fall back to scanning the raw args and environment.
func agentModeForErrors(app *App) bool {
	if app.AgentMode {
		return true
	}
	if v, err := strconv.ParseBool(strings.TrimSpace(os.Getenv("OPTIKK_AGENT"))); err == nil && v {
		return true
	}
	return slices.Contains(os.Args[1:], "--agent")
}

// wireUsageErrors classifies cobra's own flag failures as usage errors so
// they exit 2 like every other bad invocation.
func wireUsageErrors(root *cobra.Command) {
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return clierr.New(clierr.Usage, err.Error(), "run: "+cmd.CommandPath()+" --help")
	})
}
