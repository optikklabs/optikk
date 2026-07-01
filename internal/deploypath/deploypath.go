// Package deploypath locates the deploy/ kustomize tree the CLI renders.
package deploypath

import (
	"fmt"
	"os"
	"path/filepath"
)

// marker is a path that must exist inside a valid deploy/ directory.
var marker = filepath.Join("overlays", "local", "kustomization.yaml")

// Resolve returns the absolute deploy/ path. An explicit override wins;
// otherwise it walks up from the cwd looking for a sibling deploy/ dir.
func Resolve(override string) (string, error) {
	if override != "" {
		return validate(override)
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, "deploy")
		if isDeployDir(candidate) {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("deploy/ not found above %q; pass --deploy-dir", mustWd())
		}
		dir = parent
	}
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

func mustWd() string {
	wd, _ := os.Getwd()
	return wd
}
