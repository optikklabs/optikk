package selfupdate

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
)

// releasePublicKey is the Optikk release signing key, compiled into the binary
// so a compromised download server cannot supply both a payload and the key
// that vouches for it. Its private half signs checksums.txt in CI
// (.goreleaser.yaml `signs`).
//
//go:embed cosign.pub
var releasePublicKey []byte

// ErrVerification is returned whenever an artifact fails to verify. Callers
// must treat it as fatal: a failed check means the download is not trusted.
var ErrVerification = errors.New("release verification failed")

// verifySignature checks that sig is a valid signature over checksums by the
// embedded release key. cosign's `sign-blob` writes a base64-encoded ECDSA
// signature over the SHA-256 digest of the blob, which is what we verify here.
func verifySignature(checksums, sig []byte) error {
	key, err := parsePublicKey(releasePublicKey)
	if err != nil {
		return fmt.Errorf("%w: this build has no usable release signing key embedded (%v).\n"+
			"  Refusing to install an unverifiable download. Install manually from\n"+
			"  https://github.com/%s/releases", ErrVerification, err, repo)
	}

	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sig)))
	if err != nil {
		return fmt.Errorf("%w: signature is not valid base64: %v", ErrVerification, err)
	}

	digest := sha256.Sum256(checksums)
	if !ecdsa.VerifyASN1(key, digest[:], raw) {
		return fmt.Errorf("%w: checksums.txt is not signed by the Optikk release key.\n"+
			"  This download may have been tampered with — it has not been installed.\n"+
			"  Report this at https://github.com/%s/issues", ErrVerification, repo)
	}
	return nil
}

// parsePublicKey decodes the PEM-encoded ECDSA public key cosign generates.
func parsePublicKey(pemBytes []byte) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("not PEM-encoded")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := parsed.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("want an ECDSA key, got %T", parsed)
	}
	return key, nil
}

// verifyChecksum checks archive's SHA-256 against its entry in a signed
// checksums file. Call it only after verifySignature has passed: the signature
// is what makes these checksums trustworthy.
func verifyChecksum(archive []byte, checksums []byte, name string) error {
	want, err := lookupChecksum(checksums, name)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(archive)
	got := hex.EncodeToString(sum[:])
	if got != want {
		return fmt.Errorf("%w: %s has checksum %s, but the signed manifest says %s.\n"+
			"  The download is corrupt or tampered with — it has not been installed.",
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
	return "", fmt.Errorf("%w: %s is not listed in the signed checksums", ErrVerification, name)
}
