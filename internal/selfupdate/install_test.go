package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// tarGz packs contents as the release archive layout: a single top-level
// "optikk" executable.
func tarGz(t *testing.T, name string, contents []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{
		Name: name, Mode: 0o755, Size: int64(len(contents)), Typeflag: tar.TypeReg,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(contents); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// newFakeRelease serves a release and returns an Updater pointed at it.
// mutate can corrupt an asset before it is served, to exercise failure paths.
func newFakeRelease(t *testing.T, tag string, mutate func(assets map[string][]byte)) *Updater {
	t.Helper()
	version := strings.TrimPrefix(tag, "v")
	archiveName := "optikk_" + version + "_" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz"
	checksumsName := "optikk_" + version + "_checksums.txt"

	archive := tarGz(t, binaryName, []byte("#!/bin/sh\necho new binary\n"))
	sum := sha256.Sum256(archive)
	checksums := []byte(hex.EncodeToString(sum[:]) + "  " + archiveName + "\n")

	assets := map[string][]byte{
		archiveName:   archive,
		checksumsName: checksums,
	}
	if mutate != nil {
		mutate(assets)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Release API: report the tag.
		if strings.Contains(r.URL.Path, "/releases/latest") || strings.Contains(r.URL.Path, "/releases/tags/") {
			_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": tag})
			return
		}
		// Asset download.
		if body, ok := assets[filepath.Base(r.URL.Path)]; ok {
			_, _ = w.Write(body)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	u := New()
	u.apiBase, u.releaseBase = srv.URL, srv.URL
	return u
}

func TestInstallReplacesTheBinary(t *testing.T) {
	updater := newFakeRelease(t, "v9.9.9", nil)

	dest := filepath.Join(t.TempDir(), "optikk")
	if err := os.WriteFile(dest, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	rel, err := updater.Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if rel.Tag != "v9.9.9" || rel.Version != "9.9.9" {
		t.Fatalf("Latest = %+v, want tag v9.9.9 / version 9.9.9", rel)
	}
	if err := updater.Install(context.Background(), rel, dest); err != nil {
		t.Fatalf("Install: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "new binary") {
		t.Errorf("binary at %s was not replaced; got %q", dest, got)
	}
	info, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf("mode = %v, want 0755 (the replacement must stay executable)", info.Mode().Perm())
	}
}

// A tampered archive must leave the existing binary untouched.
func TestInstallRejectsTamperedArchiveAndLeavesBinaryIntact(t *testing.T) {
	updater := newFakeRelease(t, "v9.9.9", func(assets map[string][]byte) {
		for name := range assets {
			if strings.HasSuffix(name, ".tar.gz") {
				assets[name] = tarGz(t, binaryName, []byte("#!/bin/sh\necho PWNED\n"))
			}
		}
	})

	dest := filepath.Join(t.TempDir(), "optikk")
	if err := os.WriteFile(dest, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	rel, err := updater.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	err = updater.Install(context.Background(), rel, dest)
	if !errors.Is(err, ErrVerification) {
		t.Fatalf("Install error = %v, want ErrVerification", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old binary" {
		t.Errorf("a failed update modified the binary on disk: %q", got)
	}
	// The temp file must not be left behind either.
	entries, err := os.ReadDir(filepath.Dir(dest))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("failed update left %d files in the install dir, want 1", len(entries))
	}
}

func TestExtractBinaryIgnoresPathsOutsideTheArchiveRoot(t *testing.T) {
	// A malicious archive naming a traversal path must not be treated as the
	// binary, even though its basename matches.
	if _, err := extractBinary(tarGz(t, "../../etc/optikk", []byte("evil"))); err == nil {
		t.Error("extractBinary accepted a traversal path")
	}
	if _, err := extractBinary(tarGz(t, "NOTICE", []byte("not the binary"))); err == nil {
		t.Error("extractBinary accepted an archive with no optikk binary")
	}
}
