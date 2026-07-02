// Package tenant onboards/offboards per-tenant otel-collectors from the
// deploy/tenants/_template kustomization.
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

// Onboard materializes deploy/tenants/<slug> from the template with the slug
// and key substituted, then renders and applies it.
func Onboard(ctx context.Context, k kubectl.Kube, deployDir, slug, key string, out io.Writer) error {
	dst := tenantDir(deployDir, slug)
	if err := materialize(deployDir, dst, slug, key); err != nil {
		return err
	}
	log(out, "materialized %s", dst)

	if err := k8sapply.ApplyKustomize(ctx, k, dst); err != nil {
		return err
	}
	log(out, "applied tenant %q (collector otel-collector-%s)", slug, slug)
	return nil
}

// Offboard renders the tenant's resources, deletes them, and removes the dir.
func Offboard(ctx context.Context, k kubectl.Kube, deployDir, slug string, out io.Writer) error {
	dst := tenantDir(deployDir, slug)
	if _, err := os.Stat(dst); err != nil {
		return fmt.Errorf("tenant %q not found at %s", slug, dst)
	}
	if err := k8sapply.DeleteKustomize(ctx, k, dst); err != nil {
		return err
	}
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	log(out, "offboarded tenant %q", slug)
	return nil
}

func tenantDir(deployDir, slug string) string {
	return filepath.Join(deployDir, "tenants", slug)
}

// materialize copies the template files into dst, substituting placeholders.
func materialize(deployDir, dst, slug, key string) error {
	src := filepath.Join(deployDir, "tenants", "_template")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	replacer := strings.NewReplacer("REPLACE_WITH_<TEAM_SLUG_KEY>", key, "<slug>", slug)
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
