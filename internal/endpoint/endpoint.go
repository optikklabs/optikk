// Package endpoint resolves the Optikk service URLs the CLI talks to.
// Optikk is a hosted service: the endpoints below are the only ones the CLI
// targets, and every one of them is HTTPS.
package endpoint

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const (
	// APIURL is the hosted query API. It must stay in sync with the web app's
	// VITE_API_BASE_URL (web/.env.production) — both call the same surface.
	APIURL = "https://api.optikk.in"

	// AppURL is the Optikk web app: dashboards, traces, and settings.
	AppURL = "https://app.optikk.in"

	// SiteURL is the marketing site, which hosts support and legal pages.
	SiteURL = "https://optikk.in"

	// DocsURL is the documentation site. It is a separate host from SiteURL —
	// optikk.in/docs does not exist and would silently render the landing page.
	DocsURL = "https://docs.optikk.dev"
)

// ErrInsecure is returned for any API URL that is not HTTPS.
var ErrInsecure = errors.New("api url must be https")

// Resolve returns the API base URL in priority order:
//  1. explicit (--api-url flag or OPTIKK_API_URL)
//  2. contextURL (the active context's api_url)
//  3. APIURL (the hosted API)
//
// The result is always an HTTPS URL with no trailing slash; anything else is
// rejected rather than silently downgraded, so credentials and telemetry can
// never leave over plaintext.
func Resolve(explicit, contextURL string) (string, error) {
	for _, candidate := range []string{explicit, contextURL} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		return validate(candidate)
	}
	return APIURL, nil
}

// validate normalises a URL and enforces the HTTPS-only rule.
func validate(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("%q is not a valid url: %w", raw, err)
	}
	if u.Scheme != "https" || u.Host == "" {
		return "", fmt.Errorf("%w, got %q\n  Optikk is a hosted service — use %s, or set a context with:\n    optikk config set-context <name> --api-url https://…",
			ErrInsecure, raw, APIURL)
	}
	return strings.TrimRight(u.String(), "/"), nil
}

