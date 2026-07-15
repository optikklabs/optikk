package selfupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
)

func TestVerifyChecksum(t *testing.T) {
	archive := []byte("pretend this is a tarball")
	sum := sha256.Sum256(archive)
	name := "optikk_1.0.0_linux_amd64.tar.gz"
	// goreleaser writes "<sha256>  <filename>", two spaces.
	manifest := []byte(
		"0000000000000000000000000000000000000000000000000000000000000000  other_file.tar.gz\n" +
			hex.EncodeToString(sum[:]) + "  " + name + "\n")

	if err := verifyChecksum(archive, manifest, name); err != nil {
		t.Errorf("verifyChecksum rejected a matching archive: %v", err)
	}

	if err := verifyChecksum([]byte("a different tarball"), manifest, name); !errors.Is(err, ErrVerification) {
		t.Errorf("verifyChecksum accepted a mismatched archive; err = %v", err)
	}

	if err := verifyChecksum(archive, manifest, "optikk_1.0.0_windows_amd64.tar.gz"); !errors.Is(err, ErrVerification) {
		t.Errorf("verifyChecksum accepted an unlisted archive; err = %v", err)
	}
}

func TestLookupChecksumRejectsMalformedDigest(t *testing.T) {
	manifest := []byte("tooshort  optikk_1.0.0_linux_amd64.tar.gz\n")
	if _, err := lookupChecksum(manifest, "optikk_1.0.0_linux_amd64.tar.gz"); !errors.Is(err, ErrVerification) {
		t.Errorf("lookupChecksum accepted a malformed digest; err = %v", err)
	}
}
