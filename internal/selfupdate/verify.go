package selfupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// ErrVerification is returned whenever an artifact fails to verify. Callers
// must treat it as fatal: a failed check means the download is not trusted.
var ErrVerification = errors.New("release verification failed")

// verifyChecksum checks archive's SHA-256 against its entry in the release's
// checksums file. This catches a corrupt or truncated download. It is not an
// authenticity check: the manifest comes from the same origin as the archive,
// so TLS is what establishes that both are genuinely GitHub's. See the package
// doc for the full trust model.
func verifyChecksum(archive []byte, checksums []byte, name string) error {
	want, err := lookupChecksum(checksums, name)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(archive)
	got := hex.EncodeToString(sum[:])
	if got != want {
		return fmt.Errorf("%w: %s has checksum %s, but the release manifest says %s.\n"+
			"  The download is corrupt — it has not been installed.",
			ErrVerification, name, got, want)
	}
	return nil
}

// lookupChecksum finds name's expected digest in a `<sha256>  <filename>`
// manifest, the format sha256sum and goreleaser both emit.
func lookupChecksum(checksums []byte, name string) (string, error) {
	for _, line := range strings.Split(strings.TrimSpace(string(checksums)), "\n") {
		digest, file, found := strings.Cut(strings.TrimSpace(line), " ")
		if !found {
			continue
		}
		if strings.TrimSpace(file) == name {
			if len(digest) != hex.EncodedLen(sha256.Size) {
				return "", fmt.Errorf("%w: malformed checksum for %s", ErrVerification, name)
			}
			return digest, nil
		}
	}
	return "", fmt.Errorf("%w: %s is not listed in the release checksums", ErrVerification, name)
}
