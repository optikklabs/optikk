// Package httpx builds the HTTP clients every Optikk network call goes through.
// It exists so the CLI's transport security is stated once, in one place,
// rather than re-derived (or forgotten) at each call site.
package httpx

import (
	"crypto/tls"
	"net/http"
	"time"
)

// Client returns an HTTP client that verifies TLS against the system root CAs
// and refuses to negotiate below TLS 1.2. Certificate verification is never
// disabled: there is deliberately no knob for it.
func Client(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: Transport(),
	}
}

// Transport returns the shared hardened transport. It clones Go's default so
// proxy support, connection pooling, and HTTP/2 keep their tuned defaults, and
// only replaces the TLS policy.
func Transport() *http.Transport {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		// RootCAs nil = system trust store.
	}
	return tr
}
