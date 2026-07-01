package provision

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/optikklabs/optikk/internal/gcp"
	"github.com/optikklabs/optikk/internal/k8sapply"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
)

// GCP provisions the stack on GKE with GCS-backed mq/ClickHouse cold tier.
type GCP struct {
	opts GCPOptions
}

// NewGCP builds the GCP provisioner.
func NewGCP(opts GCPOptions) *GCP { return &GCP{opts: opts} }

func (g *GCP) spec() gcp.ClusterSpec {
	return gcp.ClusterSpec{
		Project:     g.opts.Project,
		Region:      g.opts.Region,
		Name:        ClusterName,
		Nodes:       g.opts.Nodes,
		MinNodes:    g.opts.MinNodes,
		MaxNodes:    g.opts.MaxNodes,
		MachineType: g.opts.MachineType,
	}
}

// Up creates the cluster, buckets, and secrets, then applies overlays/gcp.
func (g *GCP) Up(ctx context.Context) error {
	if err := g.validate(); err != nil {
		return err
	}
	w := g.opts.Out

	step(w, "creating GKE cluster %q in %s", ClusterName, g.opts.Region)
	if err := gcp.CreateGKE(ctx, g.spec()); err != nil {
		return err
	}
	cfg, err := gcp.RESTConfig(ctx, g.spec())
	if err != nil {
		return err
	}

	step(w, "creating GCS buckets %s, %s", g.opts.MQBucket, g.opts.CHBucket)
	if err := gcp.CreateBucket(ctx, g.opts.Project, g.opts.MQBucket, g.opts.Region); err != nil {
		return err
	}
	if err := gcp.CreateBucket(ctx, g.opts.Project, g.opts.CHBucket, g.opts.Region); err != nil {
		return err
	}

	step(w, "minting HMAC key for %s", g.opts.HMACServiceAcc)
	hmac, err := gcp.CreateHMACKey(ctx, g.opts.Project, g.opts.HMACServiceAcc)
	if err != nil {
		return err
	}

	step(w, "creating namespace and GCS secrets")
	if err := k8sapply.EnsureNamespace(ctx, cfg, Namespace); err != nil {
		return err
	}
	gcsData := map[string]string{"access-key": hmac.AccessID, "secret-key": hmac.Secret}
	for _, name := range []string{"mq-gcs-secret", "ch-gcs-secret"} {
		if err := k8sapply.EnsureSecret(ctx, cfg, Namespace, name, gcsData); err != nil {
			return err
		}
	}

	step(w, "applying overlays/gcp")
	replacer := strings.NewReplacer("REPLACE_MQ_BUCKET", g.opts.MQBucket, "REPLACE_CH_COLD_BUCKET", g.opts.CHBucket)
	objs, err := k8sapply.RenderSubst(filepath.Join(g.opts.DeployDir, "overlays", "gcp"), replacer)
	if err != nil {
		return err
	}
	applier, err := k8sapply.NewApplier(cfg)
	if err != nil {
		return err
	}
	if err := applier.Apply(ctx, objs); err != nil {
		return err
	}

	step(w, "waiting for rollouts (timeout %s)", g.opts.Timeout)
	if err := k8sapply.WaitRollouts(ctx, cfg, Namespace, g.opts.Timeout); err != nil {
		return fmt.Errorf("rollout wait: %w (%s)", err, k8sapply.PendingSummary(ctx, cfg, Namespace))
	}

	ip, err := waitForLB(ctx, cfg, g.opts.Timeout)
	if err != nil {
		return err
	}
	step(w, "gcp stack ready — query API at http://%s  OTLP at %s:4317/:4318", ip, ip)
	return nil
}

// Down deletes the cluster and (unless kept) the GCS buckets.
func (g *GCP) Down(ctx context.Context) error {
	if err := g.validate(); err != nil {
		return err
	}
	w := g.opts.Out

	step(w, "deleting GKE cluster %q", ClusterName)
	if err := gcp.DeleteGKE(ctx, g.spec()); err != nil {
		return err
	}
	if g.opts.KeepBuckets {
		return nil
	}
	step(w, "deleting GCS buckets")
	if err := gcp.DeleteBucket(ctx, g.opts.MQBucket); err != nil {
		return err
	}
	return gcp.DeleteBucket(ctx, g.opts.CHBucket)
}

func (g *GCP) validate() error {
	var missing []string
	if g.opts.Project == "" {
		missing = append(missing, "--project")
	}
	if g.opts.Region == "" {
		missing = append(missing, "--region")
	}
	if g.opts.MQBucket == "" {
		missing = append(missing, "--mq-bucket")
	}
	if g.opts.CHBucket == "" {
		missing = append(missing, "--ch-bucket")
	}
	if g.opts.HMACServiceAcc == "" {
		missing = append(missing, "--hmac-sa")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required gcp settings: %s", strings.Join(missing, ", "))
	}
	return nil
}

// waitForLB blocks until the Traefik Service has an external address.
func waitForLB(ctx context.Context, cfg *rest.Config, timeout time.Duration) (string, error) {
	var ip string
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		got, err := k8sapply.TraefikLoadBalancerIP(ctx, cfg, Namespace)
		if err != nil {
			return false, err
		}
		if got != "" {
			ip = got
			return true, nil
		}
		return false, nil
	})
	return ip, err
}
