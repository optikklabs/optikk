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

command -v gh >/dev/null 2>&1 || {
  echo "error: gh is required to set the repo secrets — https://cli.github.com" >&2
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
echo

# Upload the private half straight from the temp dir. It is never printed and
# never copied elsewhere; the temp dir is wiped on exit, so once this finishes
# the only surviving copy is the one GitHub holds.
echo "Setting COSIGN_PRIVATE_KEY..."
gh secret set COSIGN_PRIVATE_KEY <"${WORK_DIR}/cosign.key"

echo "Setting COSIGN_PASSWORD — enter the same password you chose above:"
gh secret set COSIGN_PASSWORD

echo
echo "✓ Secrets set; the private key is now gone from this machine."
echo
echo "Commit the public key BEFORE tagging a release — install.sh fetches"
echo "cosign.pub at the tag, and the binary embeds it at build time:"
echo
echo "  git add internal/selfupdate/cosign.pub"
echo "  git commit -m 'chore: add release signing key'"
echo "  git push"
