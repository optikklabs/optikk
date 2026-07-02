package provision

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/optikklabs/optikk/internal/hostexec"
	"github.com/optikklabs/optikk/internal/k8sapply"
	"github.com/optikklabs/optikk/internal/kubectl"
	"github.com/optikklabs/optikk/internal/localcluster"
	"github.com/optikklabs/optikk/internal/prereq"
)

// Local provisions the stack on a kind cluster backed by Podman.
type Local struct {
	opts LocalOptions
}

// NewLocal builds the local provisioner.
func NewLocal(opts LocalOptions) *Local { return &Local{opts: opts} }

// Up prechecks the required tools and Podman, creates the kind cluster,
// installs metrics-server, and applies the local overlay, waiting for every
// workload to become ready.
func (l *Local) Up(ctx context.Context) error {
	w := l.opts.Out

	if err := prereq.Check(prereq.Podman, prereq.Kind, prereq.Kubectl); err != nil {
		return err
	}

	step(w, "prechecking podman machine")
	if err := hostexec.PrecheckPodman(l.opts.ManagePodman); err != nil {
		return err
	}

	kc := localcluster.New(ClusterName)
	exists, err := kc.Exists()
	if err != nil {
		return err
	}
	if exists {
		step(w, "kind cluster %q already exists, reusing", ClusterName)
	} else {
		step(w, "creating kind cluster %q", ClusterName)
		if err := kc.Create(filepath.Join(l.opts.DeployDir, "kind", "kind-config.yaml"), l.opts.Timeout); err != nil {
			return err
		}
	}

	step(w, "lifting pids-limit on %s", kc.NodeContainer())
	if err := hostexec.SetPidsLimit(kc.NodeContainer()); err != nil {
		return err
	}

	if l.opts.LoadLocalImages {
		step(w, "loading local app images into kind")
		if err := l.loadLocalImages(ctx, kc); err != nil {
			return err
		}
	}

	k := kubectl.Kube{Context: kc.Context()}

	step(w, "installing metrics-server")
	if err := k8sapply.InstallMetricsServer(ctx, k); err != nil {
		return err
	}

	step(w, "applying overlays/local")
	if err := k8sapply.ApplyKustomize(ctx, k, filepath.Join(l.opts.DeployDir, "overlays", "local")); err != nil {
		return err
	}

	step(w, "waiting for rollouts (timeout %s)", l.opts.Timeout)
	if err := k8sapply.WaitRollouts(ctx, k, Namespace, l.opts.Timeout); err != nil {
		return fmt.Errorf("rollout wait: %w (%s)", err, k8sapply.PendingSummary(ctx, k, Namespace))
	}

	step(w, "local stack ready — query API at http://localhost:8080")
	return nil
}

// Down deletes the kind cluster, or just the stack when KeepCluster is set.
func (l *Local) Down(ctx context.Context) error {
	w := l.opts.Out

	if err := prereq.Check(prereq.Kind, prereq.Kubectl); err != nil {
		return err
	}
	kc := localcluster.New(ClusterName)

	if !l.opts.KeepCluster {
		step(w, "deleting kind cluster %q", ClusterName)
		return kc.Delete()
	}

	step(w, "deleting the optikk stack (keeping cluster)")
	k := kubectl.Kube{Context: kc.Context()}
	return k8sapply.DeleteKustomize(ctx, k, filepath.Join(l.opts.DeployDir, "overlays", "local"))
}

// localImages are the private app images kind must have loaded when the ghcr
// packages aren't public. clickhouse/mariadb/traefik/otel pull from public
// registries and are not listed here.
var localImages = []string{
	"ghcr.io/optikklabs/ingest:latest",
	"ghcr.io/optikklabs/query:latest",
	"ghcr.io/optikklabs/web:latest",
	"ghcr.io/ramantayal12/mq:latest",
}

// loadLocalImages saves the locally-present app images (podman, host-level)
// then imports the archive into the kind node.
func (l *Local) loadLocalImages(ctx context.Context, kc *localcluster.Cluster) error {
	archive := filepath.Join(os.TempDir(), "optikk-images.tar")
	defer os.Remove(archive)

	saveArgs := append([]string{"save", "-m", "-o", archive}, localImages...)
	if err := sh(ctx, "podman", saveArgs...); err != nil {
		return fmt.Errorf("podman save app images: %w", err)
	}
	return kc.LoadImageArchive(archive)
}

func sh(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
