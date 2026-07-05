// Package deploypath resolves the deploy/ kustomize tree the CLI renders.
package deploypath

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/optikklabs/optikk/assets"
)

// marker is a path that must exist inside a valid deploy/ directory: the root
// kustomization the CLI applies.
const marker = "kustomization.yaml"

// Resolve returns the absolute deploy/ path. An explicit override wins (dev
// against a working tree); otherwise the embedded tree is materialized to disk.
func Resolve(override string) (string, error) {
	if override != "" {
		return validate(override)
	}
	return assets.Materialize()
}

// validate confirms an explicit override actually points at a deploy/ tree.
func validate(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if !isDeployDir(abs) {
		return "", fmt.Errorf("--deploy-dir %q is not a deploy/ tree (missing %s)", abs, marker)
	}
	return abs, nil
}

func isDeployDir(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, marker))
	return err == nil
}
