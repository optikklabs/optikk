// Package conn resolves the query API URL without requiring kubectl or a
// running cluster. Data commands use this instead of target.Resolve.
package conn

import "strings"

// DefaultAPIURL is the localhost endpoint exposed by kind's Traefik ingress.
const DefaultAPIURL = "http://localhost:8080"

// Resolve returns the query API base URL. The precedence chain mirrors
// Datadog's DD_SITE resolution:
//
//  1. Config.ApiURL (set via --api-url flag or OPTIKK_API_URL env)
//  2. DefaultAPIURL (localhost:8080 — local kind cluster)
//
// Unlike target.Resolve this never shells out to kubectl and cannot fail.
func Resolve(apiURL string) string {
	if apiURL != "" {
		return strings.TrimRight(apiURL, "/")
	}
	return DefaultAPIURL
}
