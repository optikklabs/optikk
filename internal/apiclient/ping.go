package apiclient

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/optikklabs/optikk/internal/httpx"
)

// Ping reports whether the API is reachable, using the query service's
// unauthenticated liveness probe at <apiBase>/health. It is deliberately not a
// Client method: /health sits at the root, outside the /api surface Client
// wraps, and it needs no session.
func Ping(ctx context.Context, apiBase string) error {
	url := strings.TrimRight(apiBase, "/") + "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := httpx.Client(10 * time.Second).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned %s", url, resp.Status)
	}
	return nil
}
