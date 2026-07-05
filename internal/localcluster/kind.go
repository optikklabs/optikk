// Package localcluster manages the local kind cluster via the kind CLI.
package localcluster

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Cluster names one kind cluster driven through the kind binary on Podman.
type Cluster struct {
	Name string
}

// New binds a cluster name to the Podman-backed kind CLI.
func New(name string) *Cluster {
	return &Cluster{Name: name}
}

// Exists reports whether the kind cluster is already present.
func (c *Cluster) Exists() (bool, error) {
	out, err := c.capture("get", "clusters")
	if err != nil {
		return false, err
	}
	for _, n := range strings.Fields(out) {
		if n == c.Name {
			return true, nil
		}
	}
	return false, nil
}

// Create brings up the cluster from the given kind config, waiting for the
// control plane to become ready. kind's own progress goes to the terminal.
func (c *Cluster) Create(configFile string, wait time.Duration) error {
	return c.stream("create", "cluster",
		"--name", c.Name,
		"--config", configFile,
		"--wait", wait.String())
}

// ExportKubeconfig writes the kubeconfig for the cluster.
func (c *Cluster) ExportKubeconfig() error {
	return c.stream("export", "kubeconfig", "--name", c.Name)
}

// Delete removes the cluster.
func (c *Cluster) Delete() error {
	return c.stream("delete", "cluster", "--name", c.Name)
}

// Context is the kubeconfig context kind registers for this cluster.
func (c *Cluster) Context() string {
	return "kind-" + c.Name
}

// NodeContainer is the Podman container name of the control-plane node.
func (c *Cluster) NodeContainer() string {
	return fmt.Sprintf("%s-control-plane", c.Name)
}

// capture runs kind and returns stdout; stderr is folded into the error.
func (c *Cluster) capture(args ...string) (string, error) {
	cmd := c.cmd(args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("kind %s: %w (%s)",
			strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

// stream runs kind with its output attached to the terminal (progress bars).
func (c *Cluster) stream(args ...string) error {
	cmd := c.cmd(args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kind %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

func (c *Cluster) cmd(args ...string) *exec.Cmd {
	cmd := exec.Command("kind", args...)
	cmd.Env = append(os.Environ(), "KIND_EXPERIMENTAL_PROVIDER=podman")
	return cmd
}
