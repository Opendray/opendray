#!/bin/sh
# OpenDray — installer for macOS and Linux.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/install.sh | sh
#
# Environment variables (optional):
#   OPENDRAY_VERSION       pin a specific tag (e.g. v0.4.0). Default: latest.
#   OPENDRAY_INSTALL_DIR   binary destination. Default: $HOME/.local/bin.
#   OPENDRAY_NO_SETUP      set to any value to skip auto-launching `opendray setup`.
#   OPENDRAY_REPO          override "Opendray/opendray" (fork / mirror testing).
#
# The script:
#   1. detects OS + arch, refuses unsupported combos early
#   2. resolves the release tag (latest by default) via the GitHub API
#   3. downloads the raw binary + SHA256SUMS, verifies the checksum
#   4. installs to $OPENDRAY_INSTALL_DIR/opendray with +x
#   5. strips the macOS quarantine xattr so Gatekeeper doesn't block first run
#   6. reassigns stdin to the controlling TTY (curl | sh trick) and execs
#      `opendray setup` so the user lands directly in the wizard.
#
set -eu

REPO="${OPENDRAY_REPO:-Opendray/opendray}"
BIN_NAME="opendray"

# ── tiny output helpers ──────────────────────────────────────
# Stick to plain text. Some terminals (BusyBox in minimal containers,
# serial consoles, non-UTF-8 locales) render box-drawing + colors as
# mojibake and the user-hostile output defeats the point.
say()  { printf '%s\n' "$*"; }
info() { printf '    %s\n' "$*"; }
step() { printf '→   %s\n' "$*"; }
ok()   { printf '✓   %s\n' "$*"; }
die()  { printf '✗   %s\n' "$*" >&2; exit 1; }

say ""
say "OpenDray installer"
say "───────────────────────────────────────────────────────"
say ""

# ── detect platform ──────────────────────────────────────────
OS_RAW=$(uname -s)
OS=$(printf '%s' "$OS_RAW" | tr '[:upper:]' '[:lower:]')
case "$OS" in
    darwin|linux) ;;
    mingw*|msys*|cygwin*)
        die "Windows is not yet supported. OpenDray's core (agent CLI in a pseudo-terminal) needs ConPTY support, which is on the roadmap." ;;
    *)
        die "Unsupported OS: ${OS_RAW}. OpenDray supports macOS and Linux." ;;
esac

ARCH_RAW=$(uname -m)
case "$ARCH_RAW" in
    x86_64|amd64)  ARCH=amd64 ;;
    arm64|aarch64) ARCH=arm64 ;;
    *)
        die "Unsupported architecture: ${ARCH_RAW}. OpenDray supports amd64 and arm64." ;;
esac

ASSET="opendray-${OS}-${ARCH}"

step "Platform: ${OS}/${ARCH}"

# ── resolve release tag ──────────────────────────────────────
if [ "${OPENDRAY_VERSION:-latest}" = "latest" ]; then
    step "Resolving latest release"
    API="https://api.github.com/repos/${REPO}/releases/latest"
    RELEASE_JSON=$(curl -fsSL "$API" 2>/dev/null) || die "GitHub API unreachable ($API)
    Retry later, or pin a version: OPENDRAY_VERSION=v0.4.0 sh install.sh"
    TAG=$(printf '%s' "$RELEASE_JSON" \
        | grep -E '"tag_name"' \
        | head -1 \
        | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    [ -n "$TAG" ] || die "Could not parse release tag from GitHub API response."
else
    TAG="$OPENDRAY_VERSION"
fi

info "Version: ${TAG}"

BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"
ASSET_URL="${BASE_URL}/${ASSET}"
SUMS_URL="${BASE_URL}/SHA256SUMS"

# ── install location ─────────────────────────────────────────
INSTALL_DIR="${OPENDRAY_INSTALL_DIR:-$HOME/.local/bin}"
TARGET="$INSTALL_DIR/$BIN_NAME"
mkdir -p "$INSTALL_DIR"

# ── download ─────────────────────────────────────────────────
TMP=$(mktemp -t opendray.XXXXXX)
TMP_SUMS="${TMP}.sums"
trap 'rm -f "$TMP" "$TMP_SUMS"' EXIT

step "Downloading ${ASSET}"
curl -fsSL -o "$TMP" "$ASSET_URL" \
    || die "Download failed: $ASSET_URL
    The release might not yet have a ${OS}/${ARCH} build."

# ── verify SHA256 ────────────────────────────────────────────
step "Verifying checksum"
curl -fsSL -o "$TMP_SUMS" "$SUMS_URL" \
    || die "Could not download SHA256SUMS from $SUMS_URL
    Release is missing its checksum file. Refuse to install unverified binary."

EXPECTED=$(grep " $ASSET\$" "$TMP_SUMS" | awk '{print $1}' || true)
[ -n "$EXPECTED" ] || die "${ASSET} is not listed in SHA256SUMS.
    Either the release is incomplete or the asset name changed.
    Refusing to install unverified binary."

ACTUAL=""
if command -v shasum >/dev/null 2>&1; then
    ACTUAL=$(shasum -a 256 "$TMP" | awk '{print $1}')
elif command -v sha256sum >/dev/null 2>&1; then
    ACTUAL=$(sha256sum "$TMP" | awk '{print $1}')
else
    die "Neither 'shasum' nor 'sha256sum' is available.
    Install one and rerun, or skip verification at your own risk."
fi

if [ "$EXPECTED" != "$ACTUAL" ]; then
    die "SHA256 mismatch for ${ASSET}
    expected: ${EXPECTED}
    got:      ${ACTUAL}
    The downloaded binary is either corrupt or tampered with.
    Aborting install."
fi
ok "SHA256 verified"

# ── install ──────────────────────────────────────────────────
step "Installing to ${TARGET}"
mv "$TMP" "$TARGET"
chmod +x "$TARGET"
trap - EXIT

# macOS: strip the "downloaded from the internet" quarantine attribute.
# Without this, Gatekeeper halts first-run with a modal that requires
# the user to open System Settings → Privacy — hostile UX for a CLI.
if [ "$OS" = "darwin" ] && command -v xattr >/dev/null 2>&1; then
    xattr -d com.apple.quarantine "$TARGET" 2>/dev/null || true
fi

ok "Installed opendray ${TAG}"

# ── PATH hint ────────────────────────────────────────────────
IN_PATH=0
case ":$PATH:" in
    *":$INSTALL_DIR:"*) IN_PATH=1 ;;
esac

if [ "$IN_PATH" = "0" ]; then
    say ""
    say "NOTE: ${INSTALL_DIR} is not in your PATH."
    say "      Add this line to your shell rc (~/.zshrc or ~/.bashrc):"
    say ""
    say "          export PATH=\"${INSTALL_DIR}:\$PATH\""
    say ""
fi

# ── hand off to setup wizard ─────────────────────────────────
# Pipe installs (curl | sh) have stdin attached to the pipe, so we can't
# prompt the user directly. /dev/tty is the controlling terminal — if
# it's available we reassign stdin to it and exec the wizard, giving the
# user a seamless install → configure flow in the same session.
#
# On truly non-interactive runs (CI, Docker build RUN steps, stdin
# redirected from a file) /dev/tty doesn't exist — we detect that and
# just print the next-step command for the user to run manually.

if [ -n "${OPENDRAY_NO_SETUP:-}" ]; then
    say ""
    say "Skipping setup (OPENDRAY_NO_SETUP is set)."
    say "Run when ready:"
    say ""
    if [ "$IN_PATH" = "1" ]; then
        say "    opendray setup"
    else
        say "    ${TARGET} setup"
    fi
    say ""
    exit 0
fi

if [ ! -c /dev/tty ] || [ ! -r /dev/tty ]; then
    say ""
    say "No controlling terminal detected — skipping auto-setup."
    say "When you're back at an interactive shell, run:"
    say ""
    if [ "$IN_PATH" = "1" ]; then
        say "    opendray setup"
    else
        say "    ${TARGET} setup"
    fi
    say ""
    exit 0
fi

say ""
say "Starting setup wizard…"
say ""

# Reassign stdin to the TTY so the wizard can read keystrokes even
# though this script itself is being read from a curl pipe.
exec < /dev/tty
exec "$TARGET" setup
