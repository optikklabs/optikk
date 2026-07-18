package queryclient

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// ErrorGroup mirrors query/internal/modules/services/errors.ErrorGroup.
type ErrorGroup struct {
	GroupID         string    `json:"group_id"`
	ServiceName     string    `json:"service_name"`
	OperationName   string    `json:"operation_name"`
	StatusMessage   string    `json:"status_message"`
	HTTPStatusCode  int       `json:"http_status_code"`
	ErrorCount      int64     `json:"error_count"`
	LastOccurrence  time.Time `json:"last_occurrence"`
	FirstOccurrence time.Time `json:"first_occurrence"`
	SampleTraceID   string    `json:"sample_trace_id"`
}

// ErrorPageInfo is the errors module's cursor pagination (distinct from the
// traces module's PageInfo).
type ErrorPageInfo struct {
	HasMore    bool   `json:"hasMore"`
	NextCursor string `json:"nextCursor,omitempty"`
	Limit      int    `json:"limit"`
}

// PaginatedErrorGroups wraps an error-group listing page.
type PaginatedErrorGroups struct {
	Results  []ErrorGroup  `json:"results"`
	PageInfo ErrorPageInfo `json:"pageInfo"`
}

// ErrorGroupDetail mirrors the group-detail response.
type ErrorGroupDetail struct {
	GroupID         string    `json:"group_id"`
	ServiceName     string    `json:"service_name"`
	OperationName   string    `json:"operation_name"`
	HTTPStatusCode  int       `json:"http_status_code"`
	ErrorCount      int64     `json:"error_count"`
	LastOccurrence  time.Time `json:"last_occurrence"`
	FirstOccurrence time.Time `json:"first_occurrence"`
	ExceptionType   string    `json:"exception_type,omitempty"`
}

// ErrorGroupTrace is one trace occurrence within a group.
type ErrorGroupTrace struct {
	TraceID    string    `json:"trace_id"`
	SpanID     string    `json:"span_id"`
	Timestamp  time.Time `json:"timestamp"`
	DurationMs float64   `json:"duration_ms"`
	StatusCode string    `json:"status_code"`
}

// PaginatedErrorTraces wraps a group's trace listing page.
type PaginatedErrorTraces struct {
	Results  []ErrorGroupTrace `json:"results"`
	PageInfo ErrorPageInfo     `json:"pageInfo"`
}

// ErrorLatestOccurrence is the context of a group's most recent error span —
// message and stacktrace included, which is the payload an investigation
// pivots on.
type ErrorLatestOccurrence struct {
	TraceID        string    `json:"trace_id"`
	SpanID         string    `json:"span_id"`
	Timestamp      time.Time `json:"timestamp"`
	DurationMs     float64   `json:"duration_ms"`
	Message        string    `json:"message"`
	Stacktrace     string    `json:"stacktrace,omitempty"`
	HTTPMethod     string    `json:"http_method"`
	HTTPRoute      string    `json:"http_route"`
	HTTPStatusCode string    `json:"http_status_code"`
	ServiceVersion string    `json:"service_version"`
	Environment    string    `json:"environment"`
	Pod            string    `json:"pod"`
	Host           string    `json:"host"`
}

// ErrorTimeSeriesPoint is one bucket of a group's error-rate series.
type ErrorTimeSeriesPoint struct {
	ServiceName  string    `json:"service_name"`
	Timestamp    time.Time `json:"timestamp"`
	RequestCount int64     `json:"request_count"`
	ErrorCount   int64     `json:"error_count"`
	ErrorRate    float64   `json:"error_rate"`
	AvgLatency   float64   `json:"avg_latency"`
}

// ListErrorGroups returns the tenant's error groups for the range.
func (c *Client) ListErrorGroups(ctx context.Context, startMs, endMs int64, service string, limit int, cursor string) (*PaginatedErrorGroups, error) {
	path := fmt.Sprintf("/v1/errors/groups?startTime=%d&endTime=%d&limit=%d", startMs, endMs, limit)
	if service != "" {
		path += "&serviceName=" + url.QueryEscape(service)
	}
	if cursor != "" {
		path += "&cursor=" + url.QueryEscape(cursor)
	}
	var resp PaginatedErrorGroups
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetErrorGroupDetail returns one group's aggregate detail.
func (c *Client) GetErrorGroupDetail(ctx context.Context, startMs, endMs int64, groupID string) (*ErrorGroupDetail, error) {
	var resp ErrorGroupDetail
	path := fmt.Sprintf("/v1/errors/groups/%s?startTime=%d&endTime=%d", url.PathEscape(groupID), startMs, endMs)
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetErrorGroupTraces returns trace occurrences for a group.
func (c *Client) GetErrorGroupTraces(ctx context.Context, startMs, endMs int64, groupID string, limit int, cursor string) (*PaginatedErrorTraces, error) {
	path := fmt.Sprintf("/v1/errors/groups/%s/traces?startTime=%d&endTime=%d&limit=%d",
		url.PathEscape(groupID), startMs, endMs, limit)
	if cursor != "" {
		path += "&cursor=" + url.QueryEscape(cursor)
	}
	var resp PaginatedErrorTraces
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetErrorGroupTimeseries returns a group's error-rate series.
func (c *Client) GetErrorGroupTimeseries(ctx context.Context, startMs, endMs int64, groupID string) ([]ErrorTimeSeriesPoint, error) {
	var resp []ErrorTimeSeriesPoint
	path := fmt.Sprintf("/v1/errors/groups/%s/timeseries?startTime=%d&endTime=%d", url.PathEscape(groupID), startMs, endMs)
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetErrorGroupLatest returns the group's most recent occurrence, with
// message and stacktrace.
func (c *Client) GetErrorGroupLatest(ctx context.Context, startMs, endMs int64, groupID string) (*ErrorLatestOccurrence, error) {
	var resp ErrorLatestOccurrence
	path := fmt.Sprintf("/v1/errors/groups/%s/latest-occurrence?startTime=%d&endTime=%d", url.PathEscape(groupID), startMs, endMs)
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
