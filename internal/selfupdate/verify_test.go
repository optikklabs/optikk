package selfupdate

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"testing"
)

// signingKey generates a throwaway release key and installs its public half as
// the embedded key for the duration of the test.
func signingKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("marshalling public key: %v", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})

	original := releasePublicKey
	releasePublicKey = pubPEM
	t.Cleanup(func() { releasePublicKey = original })
	return priv
}

// sign produces a signature in the same shape cosign's sign-blob emits:
// base64-encoded ASN.1 ECDSA over the SHA-256 digest of the blob.
func sign(t *testing.T, priv *ecdsa.PrivateKey, blob []byte) []byte {
	t.Helper()
	digest := sha256.Sum256(blob)
	sig, err := ecdsa.SignASN1(rand.Reader, priv, digest[:])
	if err != nil {
		t.Fatalf("signing: %v", err)
	}
	return []byte(base64.StdEncoding.EncodeToString(sig))
}

func TestVerifySignatureAcceptsGenuineSignature(t *testing.T) {
	priv := signingKey(t)
	checksums := []byte("abc123  optikk_1.0.0_darwin_arm64.tar.gz\n")
	if err := verifySignature(checksums, sign(t, priv, checksums)); err != nil {
		t.Errorf("verifySignature rejected a genuine signature: %v", err)
	}
}

func TestVerifySignatureRejectsTamperedChecksums(t *testing.T) {
	priv := signingKey(t)
	checksums := []byte("abc123  optikk_1.0.0_darwin_arm64.tar.gz\n")
	sig := sign(t, priv, checksums)

	// An attacker swaps in their own checksum but reuses the real signature.
	tampered := []byte("deadbeef  optikk_1.0.0_darwin_arm64.tar.gz\n")
	if err := verifySignature(tampered, sig); !errors.Is(err, ErrVerification) {
		t.Errorf("verifySignature accepted tampered checksums; err = %v", err)
	}
}

func TestVerifySignatureRejectsForeignKey(t *testing.T) {
	signingKey(t) // the embedded key
	attacker, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	checksums := []byte("abc123  optikk_1.0.0_darwin_arm64.tar.gz\n")

	// Signed with a valid key — just not ours.
	if err := verifySignature(checksums, sign(t, attacker, checksums)); !errors.Is(err, ErrVerification) {
		t.Errorf("verifySignature accepted a signature from a foreign key; err = %v", err)
	}
}

func TestVerifySignatureRejectsGarbage(t *testing.T) {
	signingKey(t)
	checksums := []byte("abc123  optikk.tar.gz\n")
	for _, sig := range [][]byte{[]byte("not base64!"), {}, []byte("aGVsbG8=")} {
		if err := verifySignature(checksums, sig); !errors.Is(err, ErrVerification) {
			t.Errorf("verifySignature(%q) = %v, want ErrVerification", sig, err)
		}
	}
}

// An unconfigured build must refuse to install rather than skip verification.
func TestVerifySignatureFailsClosedWithoutAKey(t *testing.T) {
	original := releasePublicKey
	releasePublicKey = []byte("PLACEHOLDER — no release signing key is configured yet.\n")
	t.Cleanup(func() { releasePublicKey = original })

	if err := verifySignature([]byte("x"), []byte("aGVsbG8=")); !errors.Is(err, ErrVerification) {
		t.Errorf("verifySignature with a placeholder key = %v, want ErrVerification", err)
	}
}

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
