// Package target resolves the live connection details for a deployment target.
package target

import (
	"context"
	"fmt"

	"github.com/optikklabs/optikk/internal/config"
	"github.com/optikklabs/optikk/internal/kubectl"
	"github.com/optikklabs/optikk/internal/localcluster"
	"github.com/optikklabs/optikk/internal/prereq"
)

// Conn holds everything commands need to reach a running stack.
type Conn struct {
	Kube     kubectl.Kube // Kubernetes access to the cluster
	APIBase  string       // query API through Traefik
	OTLPBase string       // OTLP/HTTP ingest through Traefik
}

// Resolve returns the connection for the configured target.
func Resolve(cfg config.Config) (*Conn, error) {
	switch cfg.Target {
	case config.TargetLocal:
		if err := prereq.Check(prereq.Kubectl); err != nil {
			return nil, err
		}
		k := kubectl.Kube{Context: localcluster.New(config.ClusterName).Context()}
		if _, err := k.Run(context.Background(), "get", "--raw", "/readyz", "--request-timeout=10s"); err != nil {
			return nil, fmt.Errorf("local cluster not reachable (is it up?): %w", err)
		}
		// kind maps host 8080 -> Traefik web, 4318 -> OTLP/HTTP.
		return &Conn{Kube: k, APIBase: "http://localhost:8080", OTLPBase: "http://localhost:4318"}, nil
	}
	return nil, fmt.Errorf("unknown target %q", cfg.Target)
}
