package cmd

import (
	"github.com/optikklabs/optikk/internal/conn"
	"github.com/spf13/cobra"
)

// newCloudCmd groups the account + data commands that run against the hosted
// Optikk (conn.ManagedAPIURL). It reuses the same command builders as the
// top-level (local) tree — the only difference is the default API base, which
// App.load flips to managed for anything under this command (see
// annotationManaged). Provisioning commands are intentionally absent: you can't
// provision the team-hosted stack from here.
func newCloudCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cloud",
		Short: "Account and data commands against the hosted Optikk (" + conn.ManagedAPIURL + ")",
		Long: "cloud runs the same account and data commands as the local CLI, but against\n" +
			"the team-hosted Optikk at " + conn.ManagedAPIURL + " (OTLP at ingest.optikk.in).\n" +
			"Point at a different deployment with --api-url. Provisioning commands\n" +
			"(init/down/status/verify/tenant/admin) are local-only and not available here.",
		Example: "  optikk cloud signup\n  optikk cloud onboard\n  optikk cloud traces search -q \"service:api\"",
		Annotations: map[string]string{
			annotationManaged:    "true",
			annotationSkipDeploy: "true",
		},
	}
	cmd.AddCommand(
		newSignupCmd(app),
		newLoginCmd(app),
		newAuthCmd(app),
		newOnboardCmd(app),
		newKeysCmd(app),
		newDemoCmd(app),
		newTracesCmd(app),
		newLogsCmd(app),
		newMetricsCmd(app),
		newServicesCmd(app),
		newInfraCmd(app),
		newLLMCmd(app),
		newSaturationCmd(app),
		newDashboardsCmd(app),
		newMonitorsCmd(app),
	)
	return cmd
}
