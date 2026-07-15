package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/optikklabs/optikk/internal/selfupdate"
	"github.com/spf13/cobra"
)

// SilentExitError sets the process exit status without printing an error.
// It carries a status that is not a failure — `update --check` finding an
// update, for instance — so scripts can branch on the exit code while the
// human-readable message stays on stdout.
type SilentExitError struct{ Code int }

func (e SilentExitError) Error() string { return fmt.Sprintf("exit status %d", e.Code) }

func newUpdateCmd(app *App) *cobra.Command {
	var check, assumeYes bool
	var target string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update optikk to the latest release",
		Long: "Downloads the latest release from GitHub over HTTPS, checks it against the\n" +
			"release checksums, and replaces the running binary. A download that fails\n" +
			"the checksum is discarded, never installed.",
		Args: cobra.NoArgs,
		Example: "  optikk update\n" +
			"  optikk update --check\n" +
			"  optikk update --version v0.4.0",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUpdate(cmd, app, updateOptions{check: check, assumeYes: assumeYes, target: target})
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "report whether an update exists without installing it")
	cmd.Flags().BoolVarP(&assumeYes, "yes", "y", false, "install without confirmation")
	cmd.Flags().StringVar(&target, "version", "", "install a specific version (e.g. v0.4.0) instead of the latest")
	return cmd
}

type updateOptions struct {
	check     bool
	assumeYes bool
	target    string
}

func runUpdate(cmd *cobra.Command, app *App, opts updateOptions) error {
	out := cmd.OutOrStdout()
	updater := selfupdate.New()

	rel, err := resolveRelease(cmd.Context(), updater, opts.target)
	if err != nil {
		return err
	}

	if !selfupdate.IsNewer(version, rel.Version) {
		fmt.Fprintf(out, "optikk %s is already the latest release.\n", version)
		return nil
	}

	if opts.check {
		fmt.Fprintf(out, "Update available: %s → %s\n", version, rel.Version)
		fmt.Fprintf(out, "Install it with: optikk update\n")
		return SilentExitError{Code: 1}
	}

	// A dev build has no meaningful version to compare, so overwriting it with
	// a release is almost certainly not what the developer meant.
	if selfupdate.IsDevBuild(version) && opts.target == "" {
		return fmt.Errorf("this is a development build (%s), not a release.\n"+
			"  To install a release over it anyway, name the version:\n"+
			"    optikk update --version %s", version, rel.Tag)
	}

	dest, err := selfupdate.ExecutablePath()
	if err != nil {
		return err
	}

	if !opts.assumeYes && !app.AgentMode {
		ok, err := confirmUpdate(cmd, rel.Version, dest)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(out, "Cancelled.")
			return nil
		}
	}

	fmt.Fprintf(out, "Downloading optikk %s…\n", rel.Tag)
	if err := updater.Install(cmd.Context(), rel, dest); err != nil {
		return err
	}
	fmt.Fprintf(out, "✓ Updated optikk %s → %s (%s)\n", version, rel.Version, dest)
	return nil
}

// resolveRelease picks the release to install: an explicit version, or latest.
func resolveRelease(ctx context.Context, u *selfupdate.Updater, target string) (selfupdate.Release, error) {
	if target != "" {
		return u.AtTag(ctx, target)
	}
	return u.Latest(ctx)
}

// confirmUpdate asks before replacing the binary on disk.
func confirmUpdate(cmd *cobra.Command, newVersion, dest string) (bool, error) {
	fmt.Fprintf(cmd.OutOrStdout(), "Update optikk %s → %s at %s? [y/N]: ", version, newVersion, dest)
	line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return false, nil
		}
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}
