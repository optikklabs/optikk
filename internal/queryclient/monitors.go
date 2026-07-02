package queryclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Monitor mirrors query/internal/modules/alerting/monitors models.
type Monitor struct {
	ID        int64           `json:"id"`
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	Status    string          `json:"status"`
	Priority  string          `json:"priority"`
	Muted     bool            `json:"muted"`
	MutedAt   *time.Time      `json:"muted_at,omitempty"`
	MuteUntil *time.Time      `json:"mute_until,omitempty"`
	Tags      []string        `json:"tags,omitempty"`
	Query     json.RawMessage `json:"query,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// ListMonitors lists monitors with optional filters.
func (c *Client) ListMonitors(ctx context.Context, status, monType, priority, search string, limit, offset int) ([]Monitor, error) {
	path := fmt.Sprintf("/v1/monitors?limit=%d&offset=%d", limit, offset)
	if status != "" {
		path += "&status=" + status
	}
	if monType != "" {
		path += "&type=" + monType
	}
	if priority != "" {
		path += "&priority=" + priority
	}
	if search != "" {
		path += "&q=" + search
	}
	var resp struct {
		Items []Monitor `json:"items"`
	}
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

// GetMonitor returns a single monitor.
func (c *Client) GetMonitor(ctx context.Context, id int64) (*Monitor, error) {
	var mon Monitor
	if err := c.do(ctx, "GET", fmt.Sprintf("/v1/monitors/%d", id), nil, &mon); err != nil {
		return nil, err
	}
	return &mon, nil
}

// CreateMonitor creates a monitor from a JSON body.
func (c *Client) CreateMonitor(ctx context.Context, body json.RawMessage) (*Monitor, error) {
	var mon Monitor
	if err := c.do(ctx, "POST", "/v1/monitors", body, &mon); err != nil {
		return nil, err
	}
	return &mon, nil
}

// UpdateMonitor updates a monitor from a JSON body.
func (c *Client) UpdateMonitor(ctx context.Context, id int64, body json.RawMessage) (*Monitor, error) {
	var mon Monitor
	if err := c.do(ctx, "PUT", fmt.Sprintf("/v1/monitors/%d", id), body, &mon); err != nil {
		return nil, err
	}
	return &mon, nil
}

// DeleteMonitor deletes a monitor.
func (c *Client) DeleteMonitor(ctx context.Context, id int64) error {
	return c.do(ctx, "DELETE", fmt.Sprintf("/v1/monitors/%d", id), nil, nil)
}

// MuteMonitor mutes a monitor for the given duration in seconds.
func (c *Client) MuteMonitor(ctx context.Context, id int64, durationSec int) error {
	body := map[string]int{"duration_sec": durationSec}
	return c.do(ctx, "POST", fmt.Sprintf("/v1/monitors/%d/mute", id), body, nil)
}

// UnmuteMonitor unmutes a monitor.
func (c *Client) UnmuteMonitor(ctx context.Context, id int64) error {
	return c.do(ctx, "POST", fmt.Sprintf("/v1/monitors/%d/unmute", id), nil, nil)
}

// AckMonitor acknowledges a triggered monitor.
func (c *Client) AckMonitor(ctx context.Context, id int64) error {
	return c.do(ctx, "POST", fmt.Sprintf("/v1/monitors/%d/ack", id), nil, nil)
}

// TestMonitor runs a test evaluation of a monitor.
func (c *Client) TestMonitor(ctx context.Context, id int64) (any, error) {
	var result any
	if err := c.do(ctx, "POST", fmt.Sprintf("/v1/monitors/%d/test", id), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}
