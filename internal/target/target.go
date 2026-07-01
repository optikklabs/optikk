// Package target resolves the live connection details for a deployment target.
package target

import (
	"context"
	"fmt"

	"github.com/optikklabs/optikk/internal/config"
	"github.com/optikklabs/optikk/internal/gcp"
	"github.com/optikklabs/optikk/internal/k8sapply"
	"github.com/optikklabs/optikk/internal/localcluster"
	"k8s.io/client-go/rest"
)

// Conn holds everything commands need to reach a running stack.
type Conn struct {
	REST     *rest.Config // Kubernetes API of the cluster
	APIBase  string       // query API through Traefik
	OTLPBase string       // OTLP/HTTP ingest through Traefik
}

// Resolve returns the connection for the configured target.
func Resolve(cfg config.Config) (*Conn, error) {
	switch cfg.Target {
	case config.TargetLocal:
		rc, err := localcluster.New(config.ClusterName).RESTConfig()
		if err != nil {
			return nil, fmt.Errorf("local cluster not reachable (is it up?): %w", err)
		}
		// kind maps host 8080 -> Traefik web, 4318 -> OTLP/HTTP.
		return &Conn{REST: rc, APIBase: "http://localhost:8080", OTLPBase: "http://localhost:4318"}, nil
	case config.TargetGCP:
		spec := gcp.ClusterSpec{Project: cfg.GCP.Project, Region: cfg.GCP.Region, Name: config.ClusterName}
		if spec.Project == "" || spec.Region == "" {
			return nil, fmt.Errorf("gcp target needs --project and --region (or optikk.yaml)")
		}
		rc, err := gcp.RESTConfig(context.Background(), spec)
		if err != nil {
			return nil, fmt.Errorf("gcp cluster not reachable: %w", err)
		}
		ip, err := k8sapply.TraefikLoadBalancerIP(context.Background(), rc, config.Namespace)
		if err != nil || ip == "" {
			return nil, fmt.Errorf("traefik load balancer IP not available yet: %w", err)
		}
		return &Conn{REST: rc, APIBase: "http://" + ip, OTLPBase: "http://" + ip + ":4318"}, nil
	}
	return nil, fmt.Errorf("unknown target %q", cfg.Target)
}
