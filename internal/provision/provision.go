// Package provision orchestrates end-to-end bring-up/teardown per target.
package provision

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/optikklabs/optikk/internal/config"
)

// ClusterName and Namespace are sourced from config (single definition).
const (
	ClusterName = config.ClusterName
	Namespace   = config.Namespace
)

// Provisioner brings a target's full stack up or tears it down. New targets
// implement this interface without touching the up/down commands (OCP).
type Provisioner interface {
	Up(ctx context.Context) error
	Down(ctx context.Context) error
}

// LocalOptions configures the kind-on-Podman target.
type LocalOptions struct {
	DeployDir       string
	ManagePodman    bool
	LoadLocalImages bool
	Timeout         time.Duration
	KeepCluster     bool
	Out             io.Writer
}

// GCPOptions configures the GKE + GCS target.
type GCPOptions struct {
	DeployDir      string
	Project        string
	Region         string
	Nodes          int32
	MinNodes       int32
	MaxNodes       int32
	MachineType    string
	MQBucket       string
	CHBucket       string
	HMACServiceAcc string
	Timeout        time.Duration
	KeepBuckets    bool
	Out            io.Writer
}

func step(w io.Writer, format string, args ...any) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, "==> "+format+"\n", args...)
}
