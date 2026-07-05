// Package tenant onboards/offboards per-tenant otel-collectors from the
// deploy/tenants/_template kustomization, keyed by tenant id.
package tenant

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/optikklabs/optikk/internal/k8sapply"
	"github.com/optikklabs/optikk/internal/kubectl"
)

// templateFiles are the per-tenant kustomization files to materialize.
var templateFiles = []string{"kustomization.yaml", "ingressroute.yaml"}

// Onboard materializes deploy/tenants/<id> from the template with the tenant id
// and key substituted, then renders and applies it.
func Onboard(ctx context.Context, k kubectl.Kube, deployDir, id, key string, out io.Writer) error {
	dst := tenantDir(deployDir, id)
	if err := materialize(deployDir, dst, id, key); err != nil {
		return err
	}
	log(out, "materialized %s", dst)

	if err := k8sapply.ApplyKustomize(ctx, k, dst); err != nil {
		return err
	}
	log(out, "applied tenant %q (collector otel-collector-%s)", id, id)
	return nil
}

// Offboard renders the tenant's resources, deletes them, and removes the dir.
func Offboard(ctx context.Context, k kubectl.Kube, deployDir, id string, out io.Writer) error {
	dst := tenantDir(deployDir, id)
	if _, err := os.Stat(dst); err != nil {
		return fmt.Errorf("tenant %q not found at %s", id, dst)
	}
	if err := k8sapply.DeleteKustomize(ctx, k, dst); err != nil {
		return err
	}
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	log(out, "offboarded tenant %q", id)
	return nil
}

func tenantDir(deployDir, id string) string {
	return filepath.Join(deployDir, "tenants", id)
}

// materialize copies the template files into dst, substituting placeholders.
func materialize(deployDir, dst, id, key string) error {
	src := filepath.Join(deployDir, "tenants", "_template")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	replacer := strings.NewReplacer("REPLACE_WITH_<TENANT_KEY>", key, "<id>", id)
	for _, name := range templateFiles {
		content, err := os.ReadFile(filepath.Join(src, name))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dst, name), []byte(replacer.Replace(string(content))), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func log(w io.Writer, format string, args ...any) {
	if w != nil {
		fmt.Fprintf(w, format+"\n", args...)
	}
}
