package cmd

import (
	"fmt"
	"io"
	"time"

	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/optikklabs/optikk/internal/clierr"
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
			// Every section reports independently: status is the command you run
			// when something is wrong, so one broken check must not hide the rest.
			doc := statusDoc{
				API:     checkAPI(cmd, app),
				Session: currentSession(app).doc(),
				Version: checkVersion(cmd),
			}
			return writeResult(cmd, app, doc, func(w io.Writer) {
				fmt.Fprintf(w, "API\n")
				printAPIStatus(w, doc.API)
				fmt.Fprintf(w, "\nSession\n")
				printSession(w, app, "  ")
				fmt.Fprintf(w, "\nVersion\n")
				printVersionStatus(w, doc.Version)
			})
		},
	}
}

// statusDoc is the machine-readable status report.
type statusDoc struct {
	API     apiStatus     `json:"api"`
	Session sessionDoc    `json:"session"`
	Version versionStatus `json:"version"`
}

type apiStatus struct {
	URL       string `json:"url,omitempty"`
	Reachable bool   `json:"reachable"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
	Error     string `json:"error,omitempty"`
	Hint      string `json:"hint,omitempty"`
}

type versionStatus struct {
	Current         string `json:"current"`
	Latest          string `json:"latest,omitempty"`
	UpdateAvailable bool   `json:"update_available"`
	DevBuild        bool   `json:"dev_build,omitempty"`
	Error           string `json:"error,omitempty"`
}

// checkAPI reports whether the API answers its liveness probe.
func checkAPI(cmd *cobra.Command, app *App) apiStatus {
	apiBase, err := app.API()
	if err != nil {
		return apiStatus{Error: err.Error()}
	}
	st := apiStatus{URL: apiBase}
	start := time.Now()
	if err := apiclient.Ping(cmd.Context(), apiBase); err != nil {
		st.Error = err.Error()
		st.Hint = clierr.Hint(err)
		return st
	}
	st.Reachable = true
	st.LatencyMs = time.Since(start).Milliseconds()
	return st
}

func printAPIStatus(out io.Writer, st apiStatus) {
	switch {
	case st.URL == "":
		fmt.Fprintf(out, "  ✗ no usable API configured\n    %s\n", st.Error)
	case !st.Reachable:
		fmt.Fprintf(out, "  ✗ %s unreachable\n    %s\n", st.URL, st.Error)
		if st.Hint != "" {
			fmt.Fprintf(out, "    %s\n", st.Hint)
		}
	default:
		fmt.Fprintf(out, "  ✓ %s (%dms)\n", st.URL, st.LatencyMs)
	}
}

// checkVersion reports the running version and whether a newer release
// exists. A lookup failure is informational, not fatal: `status` should still
// report the API and session when GitHub is unreachable or rate-limiting.
func checkVersion(cmd *cobra.Command) versionStatus {
	st := versionStatus{Current: version}
	if selfupdate.IsDevBuild(version) {
		st.DevBuild = true
		return st
	}
	rel, err := selfupdate.New().Latest(cmd.Context())
	if err != nil {
		st.Error = err.Error()
		return st
	}
	st.Latest = rel.Version
	st.UpdateAvailable = selfupdate.IsNewer(version, rel.Version)
	return st
}

func printVersionStatus(out io.Writer, st versionStatus) {
	fmt.Fprintf(out, "  optikk %s\n", st.Current)
	switch {
	case st.DevBuild:
		fmt.Fprintf(out, "  development build — update checks do not apply\n")
	case st.Error != "":
		fmt.Fprintf(out, "  ? could not check for updates: %s\n", st.Error)
	case st.UpdateAvailable:
		fmt.Fprintf(out, "  ↑ %s is available — install it with: optikk update\n", st.Latest)
	default:
		fmt.Fprintf(out, "  ✓ up to date\n")
	}
}
