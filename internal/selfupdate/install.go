package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// maxArchiveSize caps what we will buffer from a release download. The CLI
// archive is a few megabytes; this bounds memory if a URL serves something else.
const maxArchiveSize = 128 << 20 // 128 MiB

// binaryName is the executable inside the release archive.
const binaryName = "optikk"

// Install downloads rel, verifies it, and atomically replaces the executable at
// dest. It returns an error without touching dest unless every check passed.
//
// Order matters: the signature authenticates the checksums file, and the
// checksums file authenticates the archive. Neither check is skippable.
func (u *Updater) Install(ctx context.Context, rel Release, dest string) error {
	checksums, err := u.download(ctx, rel.ChecksumsURL)
	if err != nil {
		return fmt.Errorf("downloading checksums: %w", err)
	}
	sig, err := u.download(ctx, rel.SignatureURL)
	if err != nil {
		return fmt.Errorf("%w: %s publishes no signature, so it cannot be verified.\n"+
			"  Releases are signed from the first release cut after signing was enabled;\n"+
			"  to install an older, unsigned release, download it manually from\n"+
			"  https://github.com/%s/releases\n"+
			"  (%v)", ErrVerification, rel.Tag, repo, err)
	}
	if err := verifySignature(checksums, sig); err != nil {
		return err
	}

	archive, err := u.download(ctx, rel.ArchiveURL)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", rel.ArchiveName, err)
	}
	if err := verifyChecksum(archive, checksums, rel.ArchiveName); err != nil {
		return err
	}

	binary, err := extractBinary(archive)
	if err != nil {
		return err
	}
	return replaceExecutable(dest, binary)
}

// download fetches a URL into memory over the hardened client.
func (u *Updater) download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := u.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned %s", url, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxArchiveSize))
	if err != nil {
		return nil, err
	}
	return body, nil
}

// extractBinary pulls the optikk executable out of a verified tar.gz.
func extractBinary(archive []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("opening archive: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("archive does not contain %q", binaryName)
		}
		if err != nil {
			return nil, fmt.Errorf("reading archive: %w", err)
		}
		// Require the exact top-level name: never honour a path from the
		// archive, which could otherwise escape the extract target.
		if hdr.Typeflag != tar.TypeReg || hdr.Name != binaryName {
			continue
		}
		return io.ReadAll(io.LimitReader(tr, maxArchiveSize))
	}
}

// replaceExecutable atomically swaps dest for the new binary. It writes to a
// temp file beside dest so the final step is a same-filesystem rename, which
// is atomic and safe to perform while the old binary is still running.
func replaceExecutable(dest string, binary []byte) error {
	dir := filepath.Dir(dest)
	tmp, err := os.CreateTemp(dir, ".optikk-update-*")
	if err != nil {
		return permissionHint(dir, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	if _, err := tmp.Write(binary); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmpName, dest); err != nil {
		return permissionHint(dir, err)
	}
	return nil
}

// permissionHint turns a bare EACCES into the actionable next step. Escalation
// is never automatic: the user decides whether to grant root.
func permissionHint(dir string, err error) error {
	if !errors.Is(err, os.ErrPermission) {
		return err
	}
	return fmt.Errorf("no permission to write to %s.\n"+
		"  Re-run with elevated privileges:\n"+
		"    sudo optikk update", dir)
}

// ExecutablePath returns the real path of the running binary, following any
// symlinks so the update lands on the file itself rather than the link.
func ExecutablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locating the running binary: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolving %s: %w", exe, err)
	}
	return resolved, nil
}
