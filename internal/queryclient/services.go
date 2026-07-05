package queryclient

import (
	"context"
	"fmt"
)

// ServiceNode mirrors query/internal/modules/services/topology.ServiceNode.
type ServiceNode struct {
	Name         string  `json:"name"`
	RequestCount int64   `json:"request_count"`
	ErrorCount   int64   `json:"error_count"`
	ErrorRate    float64 `json:"error_rate"`
	P50LatencyMs float64 `json:"p50_latency_ms"`
	P95LatencyMs float64 `json:"p95_latency_ms"`
	P99LatencyMs float64 `json:"p99_latency_ms"`
	Health       string  `json:"health"`
}

// ServiceEdge is a directed call between two services.
type ServiceEdge struct {
	Source       string  `json:"source"`
	Target       string  `json:"target"`
	CallCount    int64   `json:"call_count"`
	ErrorCount   int64   `json:"error_count"`
	ErrorRate    float64 `json:"error_rate"`
	P50LatencyMs float64 `json:"p50_latency_ms"`
	P95LatencyMs float64 `json:"p95_latency_ms"`
}

// TopologyResponse wraps the service dependency graph.
type TopologyResponse struct {
	Nodes []ServiceNode `json:"nodes"`
	Edges []ServiceEdge `json:"edges"`
}

// ServiceREDMetric mirrors query/internal/modules/services/redfleet.ServiceREDMetric.
type ServiceREDMetric struct {
	ServiceName  string  `json:"service_name"`
	RequestCount int64   `json:"request_count"`
	ErrorCount   int64   `json:"error_count"`
	AvgLatency   float64 `json:"avg_latency"`
	P95Latency   float64 `json:"p95_latency"`
	P99Latency   float64 `json:"p99_latency"`
}

// GetTopology returns the service dependency graph for the range.
func (c *Client) GetTopology(ctx context.Context, startMs, endMs int64, service string) (*TopologyResponse, error) {
	path := fmt.Sprintf("/v1/services/topology?startTime=%d&endTime=%d", startMs, endMs)
	if service != "" {
		path += "&service=" + service
	}
	var resp TopologyResponse
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListFleetServices returns one RED row per service across the fleet.
func (c *Client) ListFleetServices(ctx context.Context, startMs, endMs int64) ([]ServiceREDMetric, error) {
	path := fmt.Sprintf("/v1/spans/red/services?startTime=%d&endTime=%d", startMs, endMs)
	var resp []ServiceREDMetric
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// ErrorGroup mirrors query/internal/modules/services/errors.ErrorGroup.
type ErrorGroup struct {
	GroupID        string `json:"group_id"`
	ServiceName    string `json:"service_name"`
	OperationName  string `json:"operation_name"`
	StatusMessage  string `json:"status_message"`
	HTTPStatusCode int    `json:"http_status_code"`
	ErrorCount     int64  `json:"error_count"`
	SampleTraceID  string `json:"sample_trace_id"`
}

// ErrorGroupsResponse wraps paginated error groups.
type ErrorGroupsResponse struct {
	Results  []ErrorGroup `json:"results"`
	PageInfo PageInfo     `json:"pageInfo"`
}

// ListErrorGroups returns aggregated error groups for the range.
func (c *Client) ListErrorGroups(ctx context.Context, startMs, endMs int64, service string, limit int) (*ErrorGroupsResponse, error) {
	path := fmt.Sprintf("/v1/errors/groups?startTime=%d&endTime=%d&limit=%d", startMs, endMs, limit)
	if service != "" {
		path += "&service=" + service
	}
	var resp ErrorGroupsResponse
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ServiceSummary returns the RED summary for one service (analytical JSON).
func (c *Client) ServiceSummary(ctx context.Context, startMs, endMs int64, service string) (any, error) {
	path := fmt.Sprintf("/v1/spans/red/summary?startTime=%d&endTime=%d&service=%s", startMs, endMs, service)
	var resp any
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// TopEndpoints returns the top endpoints for a service (analytical JSON).
func (c *Client) TopEndpoints(ctx context.Context, startMs, endMs int64, service string) (any, error) {
	path := fmt.Sprintf("/v1/spans/red/top-endpoints?startTime=%d&endTime=%d", startMs, endMs)
	if service != "" {
		path += "&serviceName=" + service
	}
	var resp any
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}
