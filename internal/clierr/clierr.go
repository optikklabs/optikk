// Package clierr classifies CLI failures so callers — human or agent — can
// tell why a command failed without parsing prose. Each Kind maps to a fixed
// process exit code, and agent mode renders errors as a single JSON envelope
// on stderr. It is a leaf package: everything above it may import it.
package clierr

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Kind is the failure category. The zero value is Internal so an
// unclassified error stays exit code 1.
type Kind int

const (
	Internal Kind = iota // unclassified failure — exit 1
	Usage                // bad arguments or missing flags — exit 2
	Auth                 // missing, expired, or rejected credentials — exit 3
	Network              // the API host could not be reached at all — exit 4
	API                  // the API answered with an error — exit 5
)

// exitCode maps a Kind to its process exit code.
func (k Kind) exitCode() int {
	switch k {
	case Usage:
		return 2
	case Auth:
		return 3
	case Network:
		return 4
	case API:
		return 5
	default:
		return 1
	}
}

// code is the stable string form used in the JSON envelope.
func (k Kind) code() string {
	switch k {
	case Usage:
		return "usage"
	case Auth:
		return "auth"
	case Network:
		return "network"
	case API:
		return "api"
	default:
		return "internal"
	}
}

// Error is a classified CLI failure. Msg says what went wrong; Hint is the
// one actionable next step; APICode carries the server's error code
// (UNAUTHORIZED, NOT_FOUND, …) when the API produced the failure.
type Error struct {
	Kind    Kind
	Msg     string
	Hint    string
	APICode string
}

func (e *Error) Error() string { return e.Msg }

// New builds a classified error. hint may be empty.
func New(kind Kind, msg, hint string) *Error {
	return &Error{Kind: kind, Msg: msg, Hint: hint}
}

// ExitCode returns the process exit code for err: the Kind's code when err
// is (or wraps) an *Error, otherwise 1.
func ExitCode(err error) int {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind.exitCode()
	}
	return 1
}

// Hint returns the actionable next step attached to err, if any.
func Hint(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Hint
	}
	return ""
}

// envelope is the agent-mode stderr document. exit_code duplicates the
// process exit status so agents that only capture streams still see it.
type envelope struct {
	Error    string `json:"error"`
	Code     string `json:"code"`
	APICode  string `json:"api_code,omitempty"`
	ExitCode int    `json:"exit_code"`
	Hint     string `json:"hint,omitempty"`
}

// RenderJSON writes err as the agent-mode envelope: one JSON object, one line.
func RenderJSON(w io.Writer, err error) {
	env := envelope{Error: err.Error(), Code: Internal.code(), ExitCode: 1}
	var e *Error
	if errors.As(err, &e) {
		env.Code = e.Kind.code()
		env.APICode = e.APICode
		env.ExitCode = e.Kind.exitCode()
		env.Hint = e.Hint
	}
	_ = json.NewEncoder(w).Encode(env)
}

// Unreachable classifies a transport failure: the host was never reached, so
// the next step is network- and config-checking, not credentials.
func Unreachable(base string, err error) *Error {
	return &Error{
		Kind: Network,
		Msg:  fmt.Sprintf("cannot reach %s: %v", base, err),
		Hint: "check your network connection, then retry; if you are pointing at a custom API, confirm the URL with: optikk config show",
	}
}

// FromAPI classifies a non-success API response. 401/403 (or their server
// error codes) become Auth with a re-login hint; everything else is API,
// carrying the server's error code for agents to branch on.
func FromAPI(method, path string, status int, apiCode, msg string) *Error {
	if msg == "" {
		msg = http.StatusText(status)
	}
	e := &Error{
		Kind:    API,
		Msg:     fmt.Sprintf("%s %s: %s", method, path, msg),
		APICode: apiCode,
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden ||
		apiCode == "UNAUTHORIZED" || apiCode == "FORBIDDEN" {
		e.Kind = Auth
		e.Hint = "run: optikk auth login (or set OPTIKK_TOKEN)"
	}
	return e
}
