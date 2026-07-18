package cmd

import (
	"fmt"
	"io"

	"github.com/optikklabs/optikk/internal/clierr"
	"github.com/spf13/cobra"
)

// keyCacheNote warns that ingest caches keys briefly after a change.
const keyCacheNote = "the old key may keep working for up to 5m until the ingest cache expires"

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

// keyDoc is the machine-readable result of keys rotate/revoke.
type keyDoc struct {
	Status string `json:"status"`
	APIKey string `json:"api_key,omitempty"`
	Note   string `json:"note"`
}

func newKeysRotateCmd(app *App) *cobra.Command {
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
			doc := keyDoc{Status: "rotated", APIKey: tenant.APIKey, Note: keyCacheNote}
			return writeResult(cmd, app, doc, func(w io.Writer) {
				fmt.Fprintf(w, "rotated. new api_key: %s\n", doc.APIKey)
				fmt.Fprintln(w, "update OTEL_EXPORTER_OTLP_HEADERS=x-api-key=<new key> on your services.")
				fmt.Fprintf(w, "note: %s.\n", keyCacheNote)
			})
		},
	}
}

func newKeysRevokeCmd(app *App) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "revoke",
		Short: "Disable ingest for your tenant until you rotate a new key",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !yes {
				return clierr.New(clierr.Usage,
					"revoke disables ingest for your whole tenant",
					"re-run with --yes to confirm")
			}
			client, err := adminClient()
			if err != nil {
				return err
			}
			if _, err := client.RevokeAPIKey(cmd.Context()); err != nil {
				return err
			}
			doc := keyDoc{Status: "revoked", Note: keyCacheNote}
			return writeResult(cmd, app, doc, func(w io.Writer) {
				fmt.Fprintln(w, "revoked. ingest is disabled; run `optikk keys rotate` to issue a new key.")
				fmt.Fprintf(w, "note: %s.\n", keyCacheNote)
			})
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm revocation (required)")
	return cmd
}
