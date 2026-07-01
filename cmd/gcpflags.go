package cmd

import (
	"os/exec"
	"strings"
	"time"

	"github.com/optikklabs/optikk/internal/provision"
	"github.com/spf13/cobra"
)

// gcpFlags collects the gcp-target flags shared by up and down.
type gcpFlags struct {
	project     string
	region      string
	nodes       int32
	minNodes    int32
	maxNodes    int32
	machineType string
	mqBucket    string
	chBucket    string
	hmacSA      string
	keepBuckets bool
}

// register binds the gcp flags to a command.
func (g *gcpFlags) register(cmd *cobra.Command) {
	f := cmd.Flags()
	f.StringVar(&g.project, "project", "", "GCP project (default: gcloud active config)")
	f.StringVar(&g.region, "region", "", "GCP region")
	f.Int32Var(&g.nodes, "nodes", 1, "initial nodes per zone")
	f.Int32Var(&g.minNodes, "min-nodes", 3, "autoscaling minimum nodes")
	f.Int32Var(&g.maxNodes, "max-nodes", 6, "autoscaling maximum nodes")
	f.StringVar(&g.machineType, "machine-type", "e2-standard-4", "node machine type")
	f.StringVar(&g.mqBucket, "mq-bucket", "", "GCS bucket for mq tiered storage")
	f.StringVar(&g.chBucket, "ch-bucket", "", "GCS bucket for ClickHouse cold tier")
	f.StringVar(&g.hmacSA, "hmac-sa", "", "service account email for GCS HMAC keys")
	f.BoolVar(&g.keepBuckets, "keep-buckets", false, "(down) keep the GCS buckets")
}

// options builds provision.GCPOptions, merging config and gcloud fallback.
func (g *gcpFlags) options(app *App, timeout time.Duration) provision.GCPOptions {
	project := firstNonEmpty(g.project, app.Cfg.GCP.Project, gcloudProject())
	region := firstNonEmpty(g.region, app.Cfg.GCP.Region)
	return provision.GCPOptions{
		DeployDir:      app.DeployDir,
		Project:        project,
		Region:         region,
		Nodes:          g.nodes,
		MinNodes:       g.minNodes,
		MaxNodes:       g.maxNodes,
		MachineType:    firstNonEmpty(g.machineType, app.Cfg.GCP.MachineType),
		MQBucket:       firstNonEmpty(g.mqBucket, app.Cfg.GCP.MQBucket),
		CHBucket:       firstNonEmpty(g.chBucket, app.Cfg.GCP.CHBucket),
		HMACServiceAcc: firstNonEmpty(g.hmacSA, app.Cfg.GCP.HMACServiceAcc),
		Timeout:        timeout,
		KeepBuckets:    g.keepBuckets,
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// gcloudProject reads the active gcloud project, ignoring errors.
func gcloudProject() string {
	out, err := exec.Command("gcloud", "config", "get-value", "project", "-q").Output()
	if err != nil {
		return ""
	}
	p := strings.TrimSpace(string(out))
	if p == "(unset)" {
		return ""
	}
	return p
}
