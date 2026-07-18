package clierr

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestExitCodes(t *testing.T) {
	cases := []struct {
		kind Kind
		want int
	}{
		{Internal, 1}, {Usage, 2}, {Auth, 3}, {Network, 4}, {API, 5},
	}
	for _, tc := range cases {
		if got := ExitCode(New(tc.kind, "boom", "")); got != tc.want {
			t.Errorf("ExitCode(kind %d) = %d, want %d", tc.kind, got, tc.want)
		}
	}
	if got := ExitCode(errors.New("plain")); got != 1 {
		t.Errorf("ExitCode(plain error) = %d, want 1", got)
	}
}

func TestExitCodeSeesWrappedErrors(t *testing.T) {
	wrapped := fmt.Errorf("context: %w", New(Auth, "no token", "log in"))
	if got := ExitCode(wrapped); got != 3 {
		t.Errorf("ExitCode(wrapped auth) = %d, want 3", got)
	}
	if got := Hint(wrapped); got != "log in" {
		t.Errorf("Hint(wrapped) = %q, want %q", got, "log in")
	}
}

func TestRenderJSON(t *testing.T) {
	var b strings.Builder
	err := FromAPI("GET", "/v1/x", 404, "NOT_FOUND", "missing")
	RenderJSON(&b, err)

	var env map[string]any
	if uerr := json.Unmarshal([]byte(b.String()), &env); uerr != nil {
		t.Fatalf("envelope is not valid JSON: %v\n%s", uerr, b.String())
	}
	if env["code"] != "api" || env["api_code"] != "NOT_FOUND" || env["exit_code"] != float64(5) {
		t.Errorf("envelope = %v, want code=api api_code=NOT_FOUND exit_code=5", env)
	}
	if !strings.HasSuffix(b.String(), "\n") || strings.Count(b.String(), "\n") != 1 {
		t.Errorf("envelope must be exactly one line, got %q", b.String())
	}
}

func TestFromAPIClassifiesAuth(t *testing.T) {
	for _, status := range []int{401, 403} {
		if e := FromAPI("GET", "/v1/x", status, "", ""); e.Kind != Auth {
			t.Errorf("FromAPI(status %d).Kind = %d, want Auth", status, e.Kind)
		}
	}
	if e := FromAPI("GET", "/v1/x", 200, "UNAUTHORIZED", "no"); e.Kind != Auth {
		t.Error("FromAPI(code UNAUTHORIZED) should classify as Auth")
	}
	if e := FromAPI("GET", "/v1/x", 500, "INTERNAL", "boom"); e.Kind != API || e.APICode != "INTERNAL" {
		t.Errorf("FromAPI(500) = kind %d apiCode %q, want API/INTERNAL", e.Kind, e.APICode)
	}
	// Empty message falls back to the HTTP status text.
	if e := FromAPI("GET", "/v1/x", 404, "", ""); !strings.Contains(e.Msg, "Not Found") {
		t.Errorf("FromAPI empty msg = %q, want status text fallback", e.Msg)
	}
}

func TestUnreachable(t *testing.T) {
	e := Unreachable("https://api.example.com", errors.New("dial tcp: refused"))
	if e.Kind != Network || e.Hint == "" {
		t.Errorf("Unreachable = kind %d hint %q, want Network with a hint", e.Kind, e.Hint)
	}
}
