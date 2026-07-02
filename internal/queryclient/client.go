// Package queryclient is a typed HTTP client for the Optikk query API.
// It generalises apiclient.Client by adding X-Team-Id header injection and
// typed methods for traces, logs, metrics, dashboards, and monitors.
package queryclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client talks to the query API through /api.
type Client struct {
	base   string // e.g. http://localhost:8080/api
	token  string
	teamID int64
	http   *http.Client
}

// New builds a query client. apiBase is the Traefik surface
// (e.g. http://localhost:8080); the API lives under /api.
func New(apiBase, token string, teamID int64) *Client {
	return &Client{
		base:   strings.TrimRight(apiBase, "/") + "/api",
		token:  token,
		teamID: teamID,
		http:   &http.Client{Timeout: 30 * time.Second},
	}
}

// envelope is the query API's standard response wrapper.
type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// do sends a JSON request and unmarshals the envelope data into out (may be nil).
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if c.teamID > 0 {
		req.Header.Set("X-Team-Id", fmt.Sprintf("%d", c.teamID))
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("connection refused — is the API running at %s? %w", c.base, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("%s %s: %s: %s", method, path, resp.Status, strings.TrimSpace(string(raw)))
	}
	if resp.StatusCode/100 != 2 || !env.Success {
		msg := resp.Status
		if env.Error != nil && env.Error.Message != "" {
			msg = env.Error.Message
		}
		return fmt.Errorf("%s %s: %s", method, path, msg)
	}
	if out != nil {
		return json.Unmarshal(env.Data, out)
	}
	return nil
}

// doRaw is like do but returns the raw response body without envelope unwrapping.
func (c *Client) doRaw(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if c.teamID > 0 {
		req.Header.Set("X-Team-Id", fmt.Sprintf("%d", c.teamID))
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connection refused — is the API running at %s? %w", c.base, err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
