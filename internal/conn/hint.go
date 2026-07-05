package conn

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
)

// HintUnreachable wraps a connection-refused error with an actionable next step,
// so a fresh machine sees how to start or target a stack instead of a raw dial
// error. Non-dial errors pass through unchanged.
func HintUnreachable(apiBase string, err error) error {
	if err == nil || !isConnRefused(err) {
		return err
	}
	if apiBase == DefaultAPIURL {
		return fmt.Errorf("nothing is running at %s.\n"+
			"Start a local stack with 'optikk up', or target a hosted API with '--api-url https://…'", apiBase)
	}
	return fmt.Errorf("can't reach %s.\n"+
		"Check the URL and your network, and that you ran 'optikk auth login'", apiBase)
}

// isConnRefused reports whether err (or anything it wraps) is a refused dial.
func isConnRefused(err error) bool {
	return errors.Is(err, syscall.ECONNREFUSED) || strings.Contains(err.Error(), "connection refused")
}
