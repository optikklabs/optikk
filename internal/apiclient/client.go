// Package apiclient talks to the query admin API (login, tenants, users).
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

	"github.com/optikklabs/optikk/internal/clierr"
	"github.com/optikklabs/optikk/internal/httpx"
)

// Client is a thin HTTP client for the query API under /api.
type Client struct {
	base  string
	token string
	http  *http.Client
}

// New builds a client for the query API. apiBase is the API host
// (e.g. https://api.optikk.in); the API lives under /api.
func New(apiBase string) *Client {
	return &Client{
		base: strings.TrimRight(apiBase, "/") + "/api",
		http: httpx.Client(30 * time.Second),
	}
}

// SetToken sets the bearer token used for admin-gated calls.
func (c *Client) SetToken(token string) { c.token = token }

// envelope is the query API's standard response wrapper.
type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
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
		return clierr.Unreachable(c.base, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("%s %s: %s: %s", method, path, resp.Status, strings.TrimSpace(string(raw)))
	}
	if resp.StatusCode/100 != 2 || !env.Success {
		code, msg := "", ""
		if env.Error != nil {
			code, msg = env.Error.Code, env.Error.Message
		}
		return clierr.FromAPI(method, path, resp.StatusCode, code, msg)
	}
	if out != nil {
		return json.Unmarshal(env.Data, out)
	}
	return nil
}
