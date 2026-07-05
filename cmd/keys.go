package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newKeysCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "keys",
		Aliases: []string{"key"},
		Short:   "Rotate or revoke your tenant's ingest API key",
		Example: "  optikk keys rotate\n  optikk keys revoke --yes",
	}
	cmd.AddCommand(newKeysRotateCmd(app), newKeysRevokeCmd(app))
	return cmd
}

func newKeysRotateCmd(_ *App) *cobra.Command {
	return &cobra.Command{
		Use:   "rotate",
		Short: "Issue a fresh API key; the previous key stops working",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := adminClient()
			if err != nil {
				return err
			}
			tenant, err := client.RotateAPIKey(cmd.Context())
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "rotated. new api_key: %s\n", tenant.APIKey)
			fmt.Fprintln(w, "update OTEL_EXPORTER_OTLP_HEADERS=x-api-key=<new key> on your services.")
			fmt.Fprintln(w, "note: the old key keeps working for up to 5m until the ingest cache expires.")
			return nil
		},
	}
}

func newKeysRevokeCmd(_ *App) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "revoke",
		Short: "Disable ingest for your tenant until you rotate a new key",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !yes {
				return fmt.Errorf("revoke disables ingest for your whole tenant; re-run with --yes to confirm")
			}
			client, err := adminClient()
			if err != nil {
				return err
			}
			if _, err := client.RevokeAPIKey(cmd.Context()); err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			fmt.Fprintln(w, "revoked. ingest is disabled; run `optikk keys rotate` to issue a new key.")
			fmt.Fprintln(w, "note: an already-cached key may keep working for up to 5m.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm revocation (required)")
	return cmd
}
