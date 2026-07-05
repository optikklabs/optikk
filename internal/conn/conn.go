// Package conn resolves the query API URL without requiring kubectl or a
// running cluster. Data commands use this instead of target.Resolve.
package conn

import "strings"

// DefaultAPIURL is the localhost endpoint exposed by kind's Traefik ingress.
const DefaultAPIURL = "http://localhost:18040"

// ManagedAPIURL is the team-hosted Optikk API (the `optikk cloud` subtree).
const ManagedAPIURL = "https://api.optikk.in"

// Resolve returns the API base URL in priority order:
//  1. apiURL argument (usually from --api-url or OPTIKK_API_URL)
//  2. config context api_url
//  3. DefaultAPIURL (localhost:18040 — local kind cluster)
//
// Unlike target.Resolve this never shells out to kubectl and cannot fail.
func Resolve(apiURL string) string {
	if apiURL != "" {
		return strings.TrimRight(apiURL, "/")
	}
	return DefaultAPIURL
}
