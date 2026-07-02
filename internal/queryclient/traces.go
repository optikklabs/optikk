package queryclient

import (
	"context"
	"fmt"
	"time"

	"github.com/optikklabs/optikk/internal/dsl"
)

// TracesQueryRequest mirrors query/internal/modules/traces/explorer.QueryRequest.
type TracesQueryRequest struct {
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
	Limit     int    `json:"limit"`
	Cursor    string `json:"cursor,omitempty"`

	dsl.TraceFilters
}

// Trace is a single trace search result.
type Trace struct {
	TraceID       string    `json:"traceId"`
	RootService   string    `json:"rootService"`
	RootOperation string    `json:"rootOperation"`
	RootStatus    string    `json:"rootStatus"`
	DurationNs    uint64    `json:"durationNs"`
	SpanCount     int       `json:"spanCount"`
	HasError      bool      `json:"hasError"`
	StartTime     time.Time `json:"startTime"`
	ServiceSet    []string  `json:"serviceSet,omitempty"`
}

// PageInfo carries cursor-based pagination state.
type PageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

// TracesQueryResponse wraps trace search results.
type TracesQueryResponse struct {
	Results  []Trace  `json:"results"`
	PageInfo PageInfo `json:"pageInfo"`
}

// TracesTrendRequest mirrors query/internal/modules/traces/explorer.TrendRequest.
type TracesTrendRequest struct {
	StartTime int64 `json:"startTime"`
	EndTime   int64 `json:"endTime"`

	dsl.TraceFilters
}

// SearchTraces queries traces matching the given filters.
func (c *Client) SearchTraces(ctx context.Context, req TracesQueryRequest) (*TracesQueryResponse, error) {
	var resp TracesQueryResponse
	if err := c.do(ctx, "POST", "/v1/traces/query", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetTrace returns the full span tree for a trace.
func (c *Client) GetTrace(ctx context.Context, traceID string) (any, error) {
	var resp any
	if err := c.do(ctx, "GET", fmt.Sprintf("/v1/traces/%s", traceID), nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// TraceTrend returns volume/error/duration trend data for matching traces.
func (c *Client) TraceTrend(ctx context.Context, req TracesTrendRequest) (any, error) {
	var resp any
	if err := c.do(ctx, "POST", "/v1/traces/trend", req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}
