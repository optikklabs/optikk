package selfupdate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
)

func TestIsNewer(t *testing.T) {
	tests := []struct {
		current, candidate string
		want               bool
	}{
		{current: "0.3.0", candidate: "0.4.0", want: true},
		{current: "v0.3.0", candidate: "v0.4.0", want: true},
		{current: "0.4.0", candidate: "0.4.0", want: false},
		{current: "0.4.0", candidate: "0.3.0", want: false},
		{current: "0.4.0", candidate: "0.4.1", want: true},
		{current: "0.9.0", candidate: "0.10.0", want: true}, // not a string compare
		{current: "1.0.0", candidate: "0.9.9", want: false},
		{current: "0.4.0-rc.1", candidate: "0.4.0", want: true}, // a prerelease precedes its release
		{current: "0.4.0", candidate: "0.4.0-rc.1", want: false},
		{current: DevVersion, candidate: "0.1.0", want: true}, // a dev build is always outdated
		{current: "garbage", candidate: "0.1.0", want: true},  // unparseable sorts as oldest
	}
	for _, tt := range tests {
		if got := IsNewer(tt.current, tt.candidate); got != tt.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.current, tt.candidate, got, tt.want)
		}
	}
}

func TestReleaseAssetNames(t *testing.T) {
	u := New()
	rel := u.release("v0.4.0")

	if rel.Version != "0.4.0" {
		t.Errorf("Version = %q, want 0.4.0", rel.Version)
	}
	// Must match .goreleaser.yaml's name_template and install.sh.
	want := "optikk_0.4.0_" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz"
	if rel.ArchiveName != want {
		t.Errorf("ArchiveName = %q, want %q", rel.ArchiveName, want)
	}
	if rel.SignatureURL != rel.ChecksumsURL+".sig" {
		t.Errorf("SignatureURL = %q, want %q", rel.SignatureURL, rel.ChecksumsURL+".sig")
	}
	for _, url := range []string{rel.ArchiveURL, rel.ChecksumsURL, rel.SignatureURL} {
		if got := url[:8]; got != "https://" {
			t.Errorf("release asset URL %q is not https", url)
		}
	}
}

func TestAtTagToleratesMissingVPrefix(t *testing.T) {
	var requested string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = r.URL.Path
		_, _ = w.Write([]byte(`{"tag_name":"v0.4.0"}`))
	}))
	defer srv.Close()

	u := New()
	u.apiBase, u.releaseBase = srv.URL, srv.URL
	if _, err := u.AtTag(context.Background(), "0.4.0"); err != nil {
		t.Fatalf("AtTag: %v", err)
	}
	if want := "/repos/optikklabs/optikk/releases/tags/v0.4.0"; requested != want {
		t.Errorf("requested %q, want %q", requested, want)
	}
}

func TestFetchReleaseErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{name: "not found", status: http.StatusNotFound, body: "{}", want: "no such release"},
		{name: "rate limited", status: http.StatusForbidden, body: "{}", want: "rate-limited"},
		{name: "empty tag", status: http.StatusOK, body: `{"tag_name":""}`, want: "no tag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			u := New()
			u.apiBase, u.releaseBase = srv.URL, srv.URL
			_, err := u.Latest(context.Background())
			if err == nil {
				t.Fatalf("Latest succeeded, want an error containing %q", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Latest error = %q, want it to mention %q", err, tt.want)
			}
		})
	}
}
