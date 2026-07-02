package queryclient

import (
	"context"
	"fmt"
)

// MetricNameEntry mirrors query/internal/modules/metrics/explorer.FEMetricNameEntry.
type MetricNameEntry struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Unit        string `json:"unit,omitempty"`
	Description string `json:"description,omitempty"`
}

// MetricNamesResponse wraps metric name listing results.
type MetricNamesResponse struct {
	Metrics []MetricNameEntry `json:"metrics"`
}

// MetricFilter mirrors FEFilter for metric queries.
type MetricFilter struct {
	Key      string `json:"key"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}

// MetricQuery is a single metric query within a multi-query request.
type MetricQuery struct {
	ID               string         `json:"id"`
	Aggregation      string         `json:"aggregation"`
	MetricName       string         `json:"metricName"`
	Where            []MetricFilter `json:"where,omitempty"`
	GroupBy          []string       `json:"groupBy,omitempty"`
	SpaceAggregation string         `json:"spaceAggregation,omitempty"`
}

// MetricsQueryRequest mirrors FEQueryRequest.
type MetricsQueryRequest struct {
	StartTime int64         `json:"startTime"`
	EndTime   int64         `json:"endTime"`
	Step      string        `json:"step,omitempty"`
	Queries   []MetricQuery `json:"queries"`
}

// MetricSeries is a single series in the query result.
type MetricSeries struct {
	Tags   map[string]string `json:"tags"`
	Values []*float64        `json:"values"`
}

// MetricQueryResult holds timestamps + series for one query ID.
type MetricQueryResult struct {
	Timestamps []int64        `json:"timestamps"`
	Series     []MetricSeries `json:"series"`
}

// MetricsQueryResponse wraps the full multi-query response.
type MetricsQueryResponse struct {
	Results map[string]MetricQueryResult `json:"results"`
}

// TagEntry is a tag key with its values.
type TagEntry struct {
	Key    string   `json:"key"`
	Values []string `json:"values"`
}

// TagsResponse wraps metric tag listing results.
type TagsResponse struct {
	Tags []TagEntry `json:"tags"`
}

// ListMetricNames lists available metric names.
func (c *Client) ListMetricNames(ctx context.Context, startMs, endMs int64, search string) (*MetricNamesResponse, error) {
	path := fmt.Sprintf("/v1/metrics/names?startTime=%d&endTime=%d", startMs, endMs)
	if search != "" {
		path += "&search=" + search
	}
	var resp MetricNamesResponse
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// QueryMetrics executes a metrics explorer query.
func (c *Client) QueryMetrics(ctx context.Context, req MetricsQueryRequest) (*MetricsQueryResponse, error) {
	var resp MetricsQueryResponse
	if err := c.do(ctx, "POST", "/v1/metrics/explorer/query", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListTags lists tags for a metric.
func (c *Client) ListTags(ctx context.Context, metricName string, startMs, endMs int64) (*TagsResponse, error) {
	path := fmt.Sprintf("/v1/metrics/%s/tags?startTime=%d&endTime=%d", metricName, startMs, endMs)
	var resp TagsResponse
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
