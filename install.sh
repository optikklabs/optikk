#!/bin/sh
# Optikk CLI installer.
#
#   curl -fsSL https://optikk.in/install.sh | sh
#
# Environment overrides:
#   OPTIKK_VERSION      install a specific version (e.g. "v0.3.1"); default: latest release
#   OPTIKK_INSTALL_DIR  install directory; default: /usr/local/bin
set -eu

REPO="optikklabs/optikk"
INSTALL_DIR="${OPTIKK_INSTALL_DIR:-/usr/local/bin}"

fail() {
  echo "error: $1" >&2
  echo "Manual downloads: https://github.com/${REPO}/releases" >&2
  exit 1
}

command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v tar >/dev/null 2>&1 || fail "tar is required"

case "$(uname -s)" in
  Darwin) OS="darwin" ;;
  Linux) OS="linux" ;;
  *) fail "unsupported OS: $(uname -s) (darwin and linux only)" ;;
esac

case "$(uname -m)" in
  x86_64 | amd64) ARCH="amd64" ;;
  arm64 | aarch64) ARCH="arm64" ;;
  *) fail "unsupported architecture: $(uname -m) (amd64 and arm64 only)" ;;
esac

TAG="${OPTIKK_VERSION:-}"
if [ -z "$TAG" ]; then
  TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" |
    grep '"tag_name"' | head -n 1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
  [ -n "$TAG" ] || fail "could not resolve the latest release tag"
fi
VERSION="${TAG#v}"

ARCHIVE="optikk_${VERSION}_${OS}_${ARCH}.tar.gz"
CHECKSUMS="optikk_${VERSION}_checksums.txt"
BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading optikk ${TAG} (${OS}/${ARCH})..."
curl -fsSL -o "${TMP_DIR}/${ARCHIVE}" "${BASE_URL}/${ARCHIVE}" ||
  fail "download failed: ${BASE_URL}/${ARCHIVE}"
curl -fsSL -o "${TMP_DIR}/${CHECKSUMS}" "${BASE_URL}/${CHECKSUMS}" ||
  fail "download failed: ${BASE_URL}/${CHECKSUMS}"

# Verify the checksums file's signature when cosign is available. The signature
# is what makes the checksums trustworthy; without cosign we can still detect a
# corrupt download below, but not a substituted one. `optikk update` always
# verifies, because it has the public key compiled in.
if command -v cosign >/dev/null 2>&1; then
  if curl -fsSL -o "${TMP_DIR}/${CHECKSUMS}.sig" "${BASE_URL}/${CHECKSUMS}.sig" &&
    curl -fsSL -o "${TMP_DIR}/cosign.pub" "https://raw.githubusercontent.com/${REPO}/${TAG}/internal/selfupdate/cosign.pub"; then
    cosign verify-blob \
      --key "${TMP_DIR}/cosign.pub" \
      --signature "${TMP_DIR}/${CHECKSUMS}.sig" \
      "${TMP_DIR}/${CHECKSUMS}" >/dev/null 2>&1 ||
      fail "signature verification failed for ${CHECKSUMS} — refusing to install"
    echo "Verified release signature."
  else
    echo "note: ${TAG} has no published signature; falling back to checksum only" >&2
  fi
else
  echo "note: cosign not found — verifying the checksum only." >&2
  echo "      Install cosign to verify the release signature: https://docs.sigstore.dev/cosign/installation/" >&2
fi

if command -v sha256sum >/dev/null 2>&1; then
  SHA_CMD="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
  SHA_CMD="shasum -a 256"
else
  fail "sha256sum or shasum is required to verify the download"
fi
(cd "$TMP_DIR" && grep " ${ARCHIVE}\$" "$CHECKSUMS" | $SHA_CMD -c - >/dev/null) ||
  fail "checksum verification failed for ${ARCHIVE}"

tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "$TMP_DIR" optikk

if [ -w "$INSTALL_DIR" ]; then
  install -m 755 "${TMP_DIR}/optikk" "${INSTALL_DIR}/optikk"
else
  echo "Escalating to sudo to write to ${INSTALL_DIR}..."
  sudo install -m 755 "${TMP_DIR}/optikk" "${INSTALL_DIR}/optikk"
fi

echo "Installed optikk ${TAG} to ${INSTALL_DIR}/optikk"
"${INSTALL_DIR}/optikk" version
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *) echo "note: ${INSTALL_DIR} is not on your PATH" ;;
esac
