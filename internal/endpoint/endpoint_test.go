package endpoint

import (
	"errors"
	"testing"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name       string
		explicit   string
		contextURL string
		want       string
	}{
		{name: "defaults to the hosted api", want: APIURL},
		{name: "context wins over default", contextURL: "https://api.example.com", want: "https://api.example.com"},
		{name: "explicit wins over context", explicit: "https://flag.example.com", contextURL: "https://ctx.example.com", want: "https://flag.example.com"},
		{name: "blank explicit falls back to context", explicit: "   ", contextURL: "https://ctx.example.com", want: "https://ctx.example.com"},
		{name: "trailing slash trimmed", explicit: "https://api.example.com/", want: "https://api.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.explicit, tt.contextURL)
			if err != nil {
				t.Fatalf("Resolve(%q, %q) returned error: %v", tt.explicit, tt.contextURL, err)
			}
			if got != tt.want {
				t.Errorf("Resolve(%q, %q) = %q, want %q", tt.explicit, tt.contextURL, got, tt.want)
			}
		})
	}
}

func TestResolveRejectsInsecure(t *testing.T) {
	insecure := []string{
		"http://localhost:19090",
		"http://api.optikk.in",
		"localhost:19090",
		"ftp://api.optikk.in",
		"https://",
	}
	for _, raw := range insecure {
		t.Run(raw, func(t *testing.T) {
			if _, err := Resolve(raw, ""); !errors.Is(err, ErrInsecure) {
				t.Errorf("Resolve(%q) error = %v, want ErrInsecure", raw, err)
			}
		})
	}
}

func TestResolveRejectsInsecureContext(t *testing.T) {
	// A context saved before the https rule must not silently downgrade.
	if _, err := Resolve("", "http://localhost:19090"); !errors.Is(err, ErrInsecure) {
		t.Errorf("Resolve with insecure context error = %v, want ErrInsecure", err)
	}
}

func TestHintUnreachable(t *testing.T) {
	server := errors.New("401 unauthorized")
	if got := HintUnreachable(APIURL, server); got != server {
		t.Errorf("HintUnreachable passed through = %v, want the original error", got)
	}
	if got := HintUnreachable(APIURL, nil); got != nil {
		t.Errorf("HintUnreachable(nil) = %v, want nil", got)
	}
	dial := errors.New("dial tcp: connection refused")
	if got := HintUnreachable(APIURL, dial); got == dial {
		t.Error("HintUnreachable did not wrap a dial failure")
	}
}
