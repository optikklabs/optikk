// Package target resolves the live connection details for a deployment target.
package target

import (
	"fmt"

	"github.com/optikklabs/optikk/internal/config"
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
		return nil, fmt.Errorf("target gcp is not implemented yet (M3)")
	}
	return nil, fmt.Errorf("unknown target %q", cfg.Target)
}
