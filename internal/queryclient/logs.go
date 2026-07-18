package queryclient

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/optikklabs/optikk/internal/dsl"
)

// LogsQueryRequest mirrors query/internal/modules/logs/explorer.QueryRequest.
type LogsQueryRequest struct {
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
	Limit     int    `json:"limit"`
	Cursor    string `json:"cursor,omitempty"`

	dsl.LogFilters
}

// Log is a single log search result.
type Log struct {
	ID           string            `json:"id"`
	Timestamp    time.Time         `json:"timestamp"`
	ServiceName  string            `json:"serviceName"`
	SeverityText string            `json:"severityText"`
	Body         string            `json:"body"`
	TraceID      string            `json:"traceId,omitempty"`
	SpanID       string            `json:"spanId,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty"`
}

// LogsQueryResponse wraps log search results.
type LogsQueryResponse struct {
	Results  []Log    `json:"results"`
	PageInfo PageInfo `json:"pageInfo"`
}

// SearchLogs queries logs matching the given filters.
func (c *Client) SearchLogs(ctx context.Context, req LogsQueryRequest) (*LogsQueryResponse, error) {
	var resp LogsQueryResponse
	if err := c.do(ctx, "POST", "/v1/logs/query", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// LogsRangeRequest is the shared body for facets, summary, and trend.
type LogsRangeRequest struct {
	StartTime int64 `json:"startTime"`
	EndTime   int64 `json:"endTime"`

	dsl.LogFilters
}

// LogFacets returns facet counts over matching logs (JSON).
func (c *Client) LogFacets(ctx context.Context, req LogsRangeRequest) (any, error) {
	var resp any
	if err := c.do(ctx, "POST", "/v1/logs/facets", req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// LogSummary returns aggregate summary stats over matching logs (JSON).
func (c *Client) LogSummary(ctx context.Context, req LogsRangeRequest) (any, error) {
	var resp any
	if err := c.do(ctx, "POST", "/v1/logs/summary", req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// LogTrend returns the log volume trend over matching logs (JSON).
func (c *Client) LogTrend(ctx context.Context, req LogsRangeRequest) (any, error) {
	var resp any
	if err := c.do(ctx, "POST", "/v1/logs/trend", req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// LogsByTrace returns every log emitted within a trace, oldest first.
func (c *Client) LogsByTrace(ctx context.Context, traceID string, limit int) ([]Log, error) {
	var logs []Log
	path := fmt.Sprintf("/v1/logs/trace/%s?limit=%d", url.PathEscape(traceID), limit)
	if err := c.do(ctx, "GET", path, nil, &logs); err != nil {
		return nil, err
	}
	return logs, nil
}
