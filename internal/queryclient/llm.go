package queryclient

import (
	"context"
	"fmt"
)

// LLMApp mirrors query/internal/modules/llm.App.
type LLMApp struct {
	Service      string  `json:"service"`
	Vendor       string  `json:"vendor"`
	PrimaryModel string  `json:"primaryModel"`
	TotalSpans   uint64  `json:"totalSpans"`
	ErrorRate    float64 `json:"errorRate"`
	P95Ms        float64 `json:"p95Ms"`
	InputTokens  uint64  `json:"inputTokens"`
	OutputTokens uint64  `json:"outputTokens"`
	Cost         float64 `json:"cost"`
}

// LLMAppsResponse wraps LLM app listing results.
type LLMAppsResponse struct {
	Apps []LLMApp `json:"apps"`
}

// ListLLMApps returns per-service LLM usage summaries.
func (c *Client) ListLLMApps(ctx context.Context, startMs, endMs int64) (*LLMAppsResponse, error) {
	path := fmt.Sprintf("/v1/llm/apps?startTime=%d&endTime=%d", startMs, endMs)
	var resp LLMAppsResponse
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// LLMTimeseries returns the LLM metric timeseries (analytical JSON).
func (c *Client) LLMTimeseries(ctx context.Context, startMs, endMs int64, metric string) (any, error) {
	path := fmt.Sprintf("/v1/llm/timeseries?startTime=%d&endTime=%d", startMs, endMs)
	if metric != "" {
		path += "&metric=" + metric
	}
	var resp any
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// LLMCostBreakdown returns cost broken down by the given dimension (JSON).
func (c *Client) LLMCostBreakdown(ctx context.Context, startMs, endMs int64, groupBy string) (any, error) {
	path := fmt.Sprintf("/v1/llm/cost/breakdown?startTime=%d&endTime=%d", startMs, endMs)
	if groupBy != "" {
		path += "&groupBy=" + groupBy
	}
	var resp any
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// LLMTracesQuery searches LLM traces within the range (JSON).
func (c *Client) LLMTracesQuery(ctx context.Context, startMs, endMs int64, limit int) (any, error) {
	body := map[string]any{"startTime": startMs, "endTime": endMs, "limit": limit}
	var resp any
	if err := c.do(ctx, "POST", "/v1/llm/traces/query", body, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// LLMTraceDetail returns the full detail of one LLM trace (JSON).
func (c *Client) LLMTraceDetail(ctx context.Context, traceID string) (any, error) {
	var resp any
	if err := c.do(ctx, "GET", fmt.Sprintf("/v1/llm/traces/%s", traceID), nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}
