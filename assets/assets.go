// Package assets embeds the deploy/ kustomize tree so the CLI is self-contained.
package assets

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed all:deploy
var deployFS embed.FS

// Materialize writes the embedded deploy/ tree to a stable cache dir and returns
// its path. It re-extracts each call so the tree always matches this binary.
func Materialize() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	root := filepath.Join(base, "optikk", "deploy")
	if err := os.RemoveAll(root); err != nil {
		return "", err
	}
	err = fs.WalkDir(deployFS, "deploy", func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		dst := filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(p, "deploy")))
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		data, err := deployFS.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0o644)
	})
	if err != nil {
		return "", err
	}
	return root, nil
}
