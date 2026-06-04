#!/usr/bin/env bash
# Sign + notarize a macOS binary with an Apple Developer ID, using quill
# (github.com/anchore/quill) so it runs on the Linux release runner — no
# macOS runner or Xcode needed. Invoked by goreleaser as a per-build post
# hook (see .goreleaser.yml).
#
# Safe no-op in three cases, so source builds, Linux/Windows artifacts, and
# unsigned snapshot/PR builds keep working unchanged:
#   - the built target isn't darwin;
#   - no Developer ID cert is configured (QUILL_SIGN_P12 empty);
#   - no notary key is configured → sign only (skip notarization).
#
# Why this matters: a Developer ID-signed release binary keeps a STABLE TCC
# identity across versions, so a user's Full Disk Access grant survives
# `opendray update`; notarization additionally clears Gatekeeper for
# browser-downloaded release assets.
#
# Expected environment (set by the Release workflow from repo secrets):
#   QUILL_SIGN_P12        base64 of the Developer ID Application .p12 (or a path)
#   QUILL_SIGN_PASSWORD   the .p12 export password
#   MACOS_NOTARY_KEY_P8   App Store Connect API key (.p8) CONTENTS (optional)
#   QUILL_NOTARY_KEY_ID   ASC key id        (default below)
#   QUILL_NOTARY_ISSUER   ASC issuer id     (default below)
set -euo pipefail

BIN="${1:?usage: macos-sign.sh <binary-path> <goos>}"
OS="${2:-}"

if [ "$OS" != "darwin" ]; then
  exit 0
fi
if [ -z "${QUILL_SIGN_P12:-}" ]; then
  echo "macos-sign: no Developer ID cert (QUILL_SIGN_P12 unset) — leaving '$BIN' ad-hoc signed."
  exit 0
fi

if ! command -v quill >/dev/null 2>&1; then
  echo "macos-sign: signing requested but 'quill' is not installed — aborting." >&2
  exit 1
fi

# App Store Connect .p8 arrives as content; quill wants a file path.
if [ -n "${MACOS_NOTARY_KEY_P8:-}" ]; then
  keyfile="$(mktemp)"
  trap 'rm -f "$keyfile"' EXIT
  printf '%s' "$MACOS_NOTARY_KEY_P8" > "$keyfile"
  export QUILL_NOTARY_KEY="$keyfile"
fi
# Identifiers the maintainer supplied (not secrets — also embedded in the
# signed/notarized binary). Overridable via the environment.
: "${QUILL_NOTARY_KEY_ID:=BPL8QFJ8M2}"
: "${QUILL_NOTARY_ISSUER:=d2ec008d-9e90-49a5-82f5-d5dfbdeff726}"
export QUILL_NOTARY_KEY_ID QUILL_NOTARY_ISSUER

if [ -n "${QUILL_NOTARY_KEY:-}" ]; then
  echo "macos-sign: signing + notarizing '$BIN' with Developer ID via quill…"
  quill sign-and-notarize "$BIN"
else
  echo "macos-sign: signing '$BIN' with Developer ID via quill (no notary key → skipping notarization)…"
  quill sign "$BIN"
fi
echo "macos-sign: done — $BIN"
