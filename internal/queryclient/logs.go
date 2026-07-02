package queryclient

import (
	"context"
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
