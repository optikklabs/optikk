#!/bin/sh
# Generate the Optikk release signing key.
#
# Run this once, on a trusted machine. It produces:
#
#   cosign.key  the PRIVATE key — goes into the COSIGN_PRIVATE_KEY repo secret,
#               then gets deleted. Never commit it.
#   cosign.pub  the PUBLIC key — commit it to internal/selfupdate/cosign.pub,
#               where it is embedded into every optikk binary and used to verify
#               downloads in `optikk update`.
#
# Rotating this key means older binaries can no longer verify new releases:
# they carry the old public key. Users on those versions must reinstall via
# install.sh rather than `optikk update`. Rotate only if the key is compromised.
set -eu

command -v cosign >/dev/null 2>&1 || {
  echo "error: cosign is required — https://docs.sigstore.dev/cosign/installation/" >&2
  exit 1
}

WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT

echo "Generating a release key pair."
echo "You will be asked for a password — save it; it becomes COSIGN_PASSWORD."
echo
(cd "$WORK_DIR" && cosign generate-key-pair)

DEST="$(git rev-parse --show-toplevel)/internal/selfupdate/cosign.pub"
cp "${WORK_DIR}/cosign.pub" "$DEST"

echo
echo "✓ Public key written to ${DEST}"
echo "  Commit it: git add internal/selfupdate/cosign.pub"
echo
echo "Now set the repo secrets (this prints the private key — do not log it):"
echo
echo "  gh secret set COSIGN_PRIVATE_KEY < ${WORK_DIR}/cosign.key"
echo "  gh secret set COSIGN_PASSWORD"
echo
echo "Run the gh commands now — ${WORK_DIR} is deleted when this script exits."
echo "Press enter once the secrets are set."
read -r _
