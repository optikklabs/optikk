// Package apiclient talks to the query admin API (login, teams, users).
package apiclient

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

// Client is a thin HTTP client for the query API under /api.
type Client struct {
	base  string
	token string
	http  *http.Client
}

// New builds a client for the query API. apiBase is the Traefik surface
// (e.g. http://localhost:8080); the API lives under /api.
func New(apiBase string) *Client {
	return &Client{
		base: strings.TrimRight(apiBase, "/") + "/api",
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

// SetToken sets the bearer token used for admin-gated calls.
func (c *Client) SetToken(token string) { c.token = token }

// envelope is the query API's standard response wrapper.
type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// do sends a JSON request and unmarshals data into out (may be nil).
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

	resp, err := c.http.Do(req)
	if err != nil {
		return err
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
