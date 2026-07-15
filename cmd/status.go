package cmd

import (
	"fmt"
	"io"
	"time"

	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/optikklabs/optikk/internal/endpoint"
	"github.com/optikklabs/optikk/internal/selfupdate"
	"github.com/spf13/cobra"
)

func newStatusCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check API reachability, your session, and available updates",
		Long: "A single view of whether optikk can do its job: whether the API answers,\n" +
			"whether you are signed in, and whether a newer release exists.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()

			// Every section reports independently: status is the command you run
			// when something is wrong, so one broken check must not hide the rest.
			fmt.Fprintf(out, "API\n")
			printAPIStatus(cmd, out, app)

			fmt.Fprintf(out, "\nSession\n")
			printSession(out, app, "  ")

			fmt.Fprintf(out, "\nVersion\n")
			printVersionStatus(cmd, out)
			return nil
		},
	}
}

// printAPIStatus reports whether the API answers its liveness probe.
func printAPIStatus(cmd *cobra.Command, out io.Writer, app *App) {
	apiBase, err := app.API()
	if err != nil {
		fmt.Fprintf(out, "  ✗ no usable API configured\n    %v\n", err)
		return
	}
	start := time.Now()
	if err := apiclient.Ping(cmd.Context(), apiBase); err != nil {
		fmt.Fprintf(out, "  ✗ %s unreachable\n", apiBase)
		fmt.Fprintf(out, "    %v\n", endpoint.HintUnreachable(apiBase, err))
		return
	}
	fmt.Fprintf(out, "  ✓ %s (%dms)\n", apiBase, time.Since(start).Milliseconds())
}

// printVersionStatus reports the running version and whether a newer release
// exists. A lookup failure is informational, not fatal: `status` should still
// report the API and session when GitHub is unreachable or rate-limiting.
func printVersionStatus(cmd *cobra.Command, out io.Writer) {
	fmt.Fprintf(out, "  optikk %s\n", version)
	if selfupdate.IsDevBuild(version) {
		fmt.Fprintf(out, "  development build — update checks do not apply\n")
		return
	}

	rel, err := selfupdate.New().Latest(cmd.Context())
	if err != nil {
		fmt.Fprintf(out, "  ? could not check for updates: %v\n", err)
		return
	}
	if selfupdate.IsNewer(version, rel.Version) {
		fmt.Fprintf(out, "  ↑ %s is available — install it with: optikk update\n", rel.Version)
		return
	}
	fmt.Fprintf(out, "  ✓ up to date\n")
}
