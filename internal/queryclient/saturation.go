package queryclient

import (
	"context"
	"fmt"
)

// DatastoreSystem mirrors query/internal/modules/saturation/database/explorer.DatastoreSystemRow.
type DatastoreSystem struct {
	System            string  `json:"system"`
	Category          string  `json:"category"`
	QueryCount        int64   `json:"query_count"`
	AvgLatencyMs      float64 `json:"avg_latency_ms"`
	P95LatencyMs      float64 `json:"p95_latency_ms"`
	ErrorRate         float64 `json:"error_rate"`
	ActiveConnections int64   `json:"active_connections"`
	LastSeen          string  `json:"last_seen"`
}

// ListDatastoreSystems returns per-system database saturation summaries.
func (c *Client) ListDatastoreSystems(ctx context.Context, startMs, endMs int64) ([]DatastoreSystem, error) {
	path := fmt.Sprintf("/v1/saturation/datastores/systems?startTime=%d&endTime=%d", startMs, endMs)
	var resp []DatastoreSystem
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// saturationGet GETs a saturation sub-resource over the range as raw JSON.
func (c *Client) saturationGet(ctx context.Context, sub string, startMs, endMs int64) (any, error) {
	path := fmt.Sprintf("/v1/saturation/%s?startTime=%d&endTime=%d", sub, startMs, endMs)
	var resp any
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// DBLatencyBySystem returns database latency per system (JSON).
func (c *Client) DBLatencyBySystem(ctx context.Context, startMs, endMs int64) (any, error) {
	return c.saturationGet(ctx, "database/latency/by-system", startMs, endMs)
}

// DBSlowQueries returns slow-query patterns (JSON).
func (c *Client) DBSlowQueries(ctx context.Context, startMs, endMs int64) (any, error) {
	return c.saturationGet(ctx, "database/slow-queries/patterns", startMs, endMs)
}

// DBOpsBySystem returns database operation volume per system (JSON).
func (c *Client) DBOpsBySystem(ctx context.Context, startMs, endMs int64) (any, error) {
	return c.saturationGet(ctx, "database/ops/by-system", startMs, endMs)
}

// KafkaTopology returns the Kafka producer/consumer topology (JSON).
func (c *Client) KafkaTopology(ctx context.Context, startMs, endMs int64) (any, error) {
	return c.saturationGet(ctx, "kafka/topology", startMs, endMs)
}

// KafkaThroughput returns per-topic Kafka throughput (JSON).
func (c *Client) KafkaThroughput(ctx context.Context, startMs, endMs int64) (any, error) {
	return c.saturationGet(ctx, "kafka/topics/throughput", startMs, endMs)
}

// KafkaGroups returns consumer-group partition assignments and lag (JSON).
func (c *Client) KafkaGroups(ctx context.Context, startMs, endMs int64) (any, error) {
	return c.saturationGet(ctx, "kafka/groups/partitions", startMs, endMs)
}
