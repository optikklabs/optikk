package httpx

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"
)

func TestClientEnforcesTLSPolicy(t *testing.T) {
	c := Client(5 * time.Second)
	if c.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want 5s", c.Timeout)
	}
	tr, ok := c.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Transport is %T, want *http.Transport", c.Transport)
	}
	if tr.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion = %x, want TLS 1.2 (%x)", tr.TLSClientConfig.MinVersion, tls.VersionTLS12)
	}
	if tr.TLSClientConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify is true; certificate verification must never be disabled")
	}
	if tr.TLSClientConfig.RootCAs != nil {
		t.Error("RootCAs is set; the system trust store must be used")
	}
}

func TestTransportIsNotShared(t *testing.T) {
	// Each call must return an independent transport, so mutating one client's
	// TLS config cannot weaken another's.
	a, b := Transport(), Transport()
	if a == b {
		t.Fatal("Transport returned the same instance twice")
	}
	if a.TLSClientConfig == b.TLSClientConfig {
		t.Error("transports share a TLS config")
	}
	if http.DefaultTransport.(*http.Transport).TLSClientConfig == a.TLSClientConfig {
		t.Error("transport mutated http.DefaultTransport")
	}
}
