package queryclient

import (
	"context"
	"fmt"
)

// IngestionTotals is the per-signal record counts from /v1/ingestion/summary
// (subset we surface — the endpoint also reports bytes and other usage facts
// the CLI does not need).
type IngestionTotals struct {
	Logs             uint64 `json:"logs"`
	Spans            uint64 `json:"spans"`
	MetricDatapoints uint64 `json:"metricDatapoints"`
	Records          uint64 `json:"records"`
}

// IngestionSummary returns the tenant's ingest totals for the range. It backs
// `optikk verify`: nonzero totals prove telemetry is arriving end to end.
func (c *Client) IngestionSummary(ctx context.Context, startMs, endMs int64) (*IngestionTotals, error) {
	var resp struct {
		Totals IngestionTotals `json:"totals"`
	}
	path := fmt.Sprintf("/v1/ingestion/summary?startTime=%d&endTime=%d", startMs, endMs)
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Totals, nil
}
