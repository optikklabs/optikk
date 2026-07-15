package cmd

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/optikklabs/optikk/internal/browser"
	"github.com/optikklabs/optikk/internal/endpoint"
	"github.com/spf13/cobra"
)

func newLoginCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Sign in via your browser (device authorization)",
		Long: "Starts an RFC 8628 device-authorization login: prints a short code, " +
			"opens the approval page in your browser, and polls until you confirm.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			apiBase, err := app.API()
			if err != nil {
				return err
			}
			client := apiclient.New(apiBase)

			code, err := client.StartDeviceAuth(cmd.Context())
			if err != nil {
				return fmt.Errorf("could not start login: %w\n\nIs the API running at %s?", err, apiBase)
			}

			verifyURL := deviceVerifyURL(code.UserCode)
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "First, open this page in your browser:\n\n    %s\n\n", verifyURL)
			fmt.Fprintf(w, "and confirm this code:\n\n    %s\n\n", code.UserCode)
			browser.Open(verifyURL)
			fmt.Fprintln(w, "Waiting for approval…")

			token, err := pollDeviceToken(cmd.Context(), client, code)
			if err != nil {
				return err
			}
			if err := apiclient.SaveToken(apiBase, token); err != nil {
				return err
			}
			fmt.Fprintf(w, "\n✓ Signed in. Token cached at ~/.optikk/config.json\n")
			return nil
		},
	}
}

// deviceVerifyURL builds the browser page that approves a device login.
//
// The page is served by the web app, not the API: api.optikk.in exposes the
// device endpoints only as POST JSON, so deriving this from the API base — as
// this once did — hands the user a 404. It is deliberately not affected by
// --api-url: pointing the CLI at a different API does not move the web app.
func deviceVerifyURL(userCode string) string {
	return fmt.Sprintf("%s/device?user_code=%s", endpoint.AppURL, url.QueryEscape(userCode))
}

// pollDeviceToken loops at the server-advised interval until the user approves
// or the code expires, honoring RFC 8628 slow_down back-off.
func pollDeviceToken(ctx context.Context, client *apiclient.Client, code apiclient.DeviceCodeResult) (string, error) {
	interval := time.Duration(code.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(code.ExpiresIn) * time.Second)

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(interval):
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("login timed out; run `optikk login` again")
		}

		res, err := client.PollDeviceToken(ctx, code.DeviceCode)
		if err != nil {
			return "", err
		}
		switch res.Status {
		case "complete":
			if res.Session == nil || res.Session.AccessToken == "" {
				return "", fmt.Errorf("login completed but no token was returned")
			}
			return res.Session.AccessToken, nil
		case "slow_down":
			interval += 5 * time.Second
		case "expired_token":
			return "", fmt.Errorf("login code expired; run `optikk login` again")
		case "authorization_pending":
			// keep waiting
		}
	}
}
