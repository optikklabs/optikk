// Package k8sapply applies kustomize directories and waits on rollouts via kubectl.
package k8sapply

import (
	"context"
	"fmt"
	"time"

	"github.com/optikklabs/optikk/internal/kubectl"
)

// FieldManager identifies the CLI as the owner of applied fields (SSA).
const FieldManager = "optikk-cli"

// ApplyKustomize server-side-applies a kustomize dir. kubectl orders
// namespaces/CRDs first, but CRs can still race CRD registration, so the
// whole apply is retried until discovery catches up.
func ApplyKustomize(ctx context.Context, k kubectl.Kube, dir string) error {
	var lastErr error
	for attempt := 0; attempt < 6; attempt++ {
		_, lastErr = k.Run(ctx, "apply", "-k", dir,
			"--server-side", "--field-manager", FieldManager, "--force-conflicts")
		if lastErr == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return fmt.Errorf("apply -k %s: %w", dir, lastErr)
}

// DeleteKustomize removes a kustomize dir's resources, ignoring missing ones.
func DeleteKustomize(ctx context.Context, k kubectl.Kube, dir string) error {
	_, err := k.Run(ctx, "delete", "-k", dir, "--ignore-not-found=true", "--wait=false")
	return err
}
