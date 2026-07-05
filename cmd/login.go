package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/optikklabs/optikk/internal/conn"
	"github.com/spf13/cobra"
)

func newLoginCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Sign in via your browser (device authorization)",
		Long: "Starts an RFC 8628 device-authorization login: prints a short code, " +
			"opens the approval page in your browser, and polls until you confirm.",
		Annotations: map[string]string{annotationSkipDeploy: "true"},
		Args:        cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			apiBase := conn.Resolve(app.Cfg.ApiURL)
			client := apiclient.New(apiBase)

			code, err := client.StartDeviceAuth(cmd.Context())
			if err != nil {
				return fmt.Errorf("could not start login: %w\n\nIs the API running at %s?", err, apiBase)
			}

			verifyURL := fmt.Sprintf("%s/device?user_code=%s", apiBase, code.UserCode)
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "First, open this page in your browser:\n\n    %s\n\n", verifyURL)
			fmt.Fprintf(w, "and confirm this code:\n\n    %s\n\n", code.UserCode)
			openBrowser(verifyURL)
			fmt.Fprintln(w, "Waiting for approval…")

			token, err := pollDeviceToken(cmd.Context(), client, code)
			if err != nil {
				return err
			}
			if err := apiclient.SaveToken(apiBase, token); err != nil {
				return err
			}
			fmt.Fprintf(w, "\n✓ Signed in. Token cached at ~/.optikk/token.json\n")
			return nil
		},
	}
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

// openBrowser best-effort opens a URL; failure is non-fatal since the CLI
// already printed the link.
func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd, args = "open", []string{url}
	case "windows":
		cmd, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd, args = "xdg-open", []string{url}
	}
	_ = exec.Command(cmd, args...).Start()
}
