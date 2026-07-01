// Package localcluster manages the local kind cluster via the kind Go library.
package localcluster

import (
	"fmt"
	"os"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cmd"
)

// Cluster wraps a kind provider bound to Podman for one named cluster.
type Cluster struct {
	Name     string
	provider *cluster.Provider
}

// New builds a kind provider using the Podman node backend.
func New(name string) *Cluster {
	os.Setenv("KIND_EXPERIMENTAL_PROVIDER", "podman")
	provider := cluster.NewProvider(
		cluster.ProviderWithLogger(cmd.NewLogger()),
		cluster.ProviderWithPodman(),
	)
	return &Cluster{Name: name, provider: provider}
}

// Exists reports whether the kind cluster is already present.
func (c *Cluster) Exists() (bool, error) {
	names, err := c.provider.List()
	if err != nil {
		return false, err
	}
	for _, n := range names {
		if n == c.Name {
			return true, nil
		}
	}
	return false, nil
}

// Create brings up the cluster from the given kind config, waiting for the
// control plane to become ready.
func (c *Cluster) Create(configFile string, wait time.Duration) error {
	return c.provider.Create(c.Name,
		cluster.CreateWithConfigFile(configFile),
		cluster.CreateWithWaitForReady(wait),
	)
}

// Delete removes the cluster.
func (c *Cluster) Delete() error {
	return c.provider.Delete(c.Name, "")
}

// RESTConfig returns a client-go config for the cluster's API server.
func (c *Cluster) RESTConfig() (*rest.Config, error) {
	kubeconfig, err := c.provider.KubeConfig(c.Name, false)
	if err != nil {
		return nil, err
	}
	return clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
}

// NodeContainer is the Podman container name of the control-plane node.
func (c *Cluster) NodeContainer() string {
	return fmt.Sprintf("%s-control-plane", c.Name)
}
