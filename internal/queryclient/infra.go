package queryclient

import (
	"context"
	"fmt"
)

// Host mirrors query/internal/modules/infrastructure/hosts.Host.
type Host struct {
	Host       string   `json:"host"`
	Subsystem  string   `json:"subsystem"`
	CPU        float64  `json:"cpu"`
	Mem        float64  `json:"mem"`
	Disk       float64  `json:"disk"`
	Saturation float64  `json:"saturation"`
	RPS        *float64 `json:"rps,omitempty"`
	ErrorRate  *float64 `json:"error_rate,omitempty"`
	P99Ms      *float64 `json:"p99_ms,omitempty"`
	Status     string   `json:"status,omitempty"`
	LastSeen   string   `json:"last_seen,omitempty"`
}

// InfraNode mirrors query/internal/modules/infrastructure/nodes.InfrastructureNode.
type InfraNode struct {
	Host           string   `json:"host"`
	PodCount       int64    `json:"pod_count"`
	ContainerCount int64    `json:"container_count"`
	Services       []string `json:"services"`
	RequestCount   int64    `json:"request_count"`
	ErrorCount     int64    `json:"error_count"`
	ErrorRate      float64  `json:"error_rate"`
	AvgLatencyMs   float64  `json:"avg_latency_ms"`
	P95LatencyMs   float64  `json:"p95_latency_ms"`
	LastSeen       string   `json:"last_seen"`
}

// FleetPod mirrors query/internal/modules/infrastructure/fleet.FleetPod.
type FleetPod struct {
	PodName      string   `json:"pod_name"`
	Host         string   `json:"host"`
	Services     []string `json:"services"`
	RequestCount int64    `json:"request_count"`
	ErrorCount   int64    `json:"error_count"`
	ErrorRate    float64  `json:"error_rate"`
	AvgLatencyMs float64  `json:"avg_latency_ms"`
	P95LatencyMs float64  `json:"p95_latency_ms"`
	LastSeen     string   `json:"last_seen"`
}

// InstanceMetric mirrors the CPU/memory by-instance response rows.
type InstanceMetric struct {
	Host        string   `json:"host"`
	Pod         string   `json:"pod"`
	Container   string   `json:"container"`
	ServiceName string   `json:"service_name"`
	Value       *float64 `json:"value"`
}

// ListHosts returns host-level saturation and RED metrics.
func (c *Client) ListHosts(ctx context.Context, startMs, endMs int64, service string) ([]Host, error) {
	path := fmt.Sprintf("/v1/infrastructure/hosts?startTime=%d&endTime=%d", startMs, endMs)
	if service != "" {
		path += "&service=" + service
	}
	var resp []Host
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// ListNodes returns Kubernetes node-level aggregates.
func (c *Client) ListNodes(ctx context.Context, startMs, endMs int64) ([]InfraNode, error) {
	path := fmt.Sprintf("/v1/infrastructure/nodes?startTime=%d&endTime=%d", startMs, endMs)
	var resp []InfraNode
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// ListFleetPods returns pod-level RED aggregates.
func (c *Client) ListFleetPods(ctx context.Context, startMs, endMs int64) ([]FleetPod, error) {
	path := fmt.Sprintf("/v1/infrastructure/fleet/pods?startTime=%d&endTime=%d", startMs, endMs)
	var resp []FleetPod
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// CPUByInstance returns per-instance CPU utilization.
func (c *Client) CPUByInstance(ctx context.Context, startMs, endMs int64) ([]InstanceMetric, error) {
	return c.instanceMetric(ctx, "cpu", startMs, endMs)
}

// MemoryByInstance returns per-instance memory utilization.
func (c *Client) MemoryByInstance(ctx context.Context, startMs, endMs int64) ([]InstanceMetric, error) {
	return c.instanceMetric(ctx, "memory", startMs, endMs)
}

func (c *Client) instanceMetric(ctx context.Context, kind string, startMs, endMs int64) ([]InstanceMetric, error) {
	path := fmt.Sprintf("/v1/infrastructure/%s/by-instance?startTime=%d&endTime=%d", kind, startMs, endMs)
	var resp []InstanceMetric
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}
