#!/bin/sh
# OpenDray installer.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/install.sh | sh
#
# Env vars:
#   OPENDRAY_VERSION       pin a specific release (e.g. v0.4.0). Default: latest.
#   OPENDRAY_INSTALL_DIR   where to drop the binary. Default: $HOME/.local/bin.
#   OPENDRAY_NO_LAUNCH     set to any value to skip the auto-launch prompt.
#
set -eu

REPO="Opendray/opendray"
BIN_NAME="opendray"

# ── detect platform ───────────────────────────────────────────
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    darwin|linux) ;;
    *) printf '✗ unsupported OS: %s\n  OpenDray supports macOS and Linux.\n' "$OS" >&2; exit 1 ;;
esac

ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)  ARCH=amd64 ;;
    arm64|aarch64) ARCH=arm64 ;;
    *) printf '✗ unsupported architecture: %s\n  OpenDray supports amd64 and arm64.\n' "$ARCH" >&2; exit 1 ;;
esac

ASSET="opendray-${OS}-${ARCH}"

# ── resolve release ──────────────────────────────────────────
# Use the GitHub API for `latest` resolution. For pinned versions skip
# the API call and jump straight to the tag URL.
if [ "${OPENDRAY_VERSION:-latest}" = "latest" ]; then
    API="https://api.github.com/repos/${REPO}/releases/latest"
    printf '→ looking up latest release\n'
    # `-L` follows redirects; `-f` fails on 4xx/5xx with a useful message.
    RELEASE_JSON=$(curl -fsSL "$API") || {
        printf '✗ could not reach GitHub API. Try again or set OPENDRAY_VERSION manually.\n' >&2
        exit 1
    }
    TAG=$(printf '%s' "$RELEASE_JSON" | grep -E '"tag_name"' | head -1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    if [ -z "$TAG" ]; then
        printf '✗ could not parse latest release tag.\n' >&2
        exit 1
    fi
else
    TAG="$OPENDRAY_VERSION"
fi

BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"
ASSET_URL="${BASE_URL}/${ASSET}"
SUMS_URL="${BASE_URL}/SHA256SUMS"

# ── install location ──────────────────────────────────────────
INSTALL_DIR="${OPENDRAY_INSTALL_DIR:-$HOME/.local/bin}"
TARGET="$INSTALL_DIR/$BIN_NAME"
mkdir -p "$INSTALL_DIR"

# ── download ─────────────────────────────────────────────────
TMP=$(mktemp -t opendray.XXXXXX)
trap 'rm -f "$TMP" "$TMP.sums"' EXIT

printf '→ downloading %s (%s)\n' "$ASSET" "$TAG"
curl -fsSL -o "$TMP" "$ASSET_URL" || {
    printf '✗ download failed: %s\n' "$ASSET_URL" >&2
    exit 1
}

# ── verify SHA256 ────────────────────────────────────────────
# Non-fatal if SHA256SUMS isn't present (older releases). If it IS present
# we verify properly — a mismatch aborts the install.
if curl -fsSL -o "$TMP.sums" "$SUMS_URL" 2>/dev/null; then
    EXPECTED=$(grep " $ASSET\$" "$TMP.sums" | awk '{print $1}')
    if [ -n "$EXPECTED" ]; then
        ACTUAL=""
        if command -v shasum >/dev/null 2>&1; then
            ACTUAL=$(shasum -a 256 "$TMP" | awk '{print $1}')
        elif command -v sha256sum >/dev/null 2>&1; then
            ACTUAL=$(sha256sum "$TMP" | awk '{print $1}')
        fi
        if [ -n "$ACTUAL" ]; then
            if [ "$EXPECTED" != "$ACTUAL" ]; then
                printf '✗ SHA256 mismatch for %s\n  expected: %s\n  got:      %s\n' \
                    "$ASSET" "$EXPECTED" "$ACTUAL" >&2
                exit 1
            fi
            printf '✓ SHA256 verified\n'
        fi
    fi
fi

# ── move into place ──────────────────────────────────────────
mv "$TMP" "$TARGET"
chmod +x "$TARGET"

# macOS: strip the "downloaded from the internet" quarantine so
# Gatekeeper doesn't block first-run. Silently continues if xattr
# isn't available or the attribute isn't set.
if [ "$OS" = "darwin" ] && command -v xattr >/dev/null 2>&1; then
    xattr -d com.apple.quarantine "$TARGET" 2>/dev/null || true
fi

printf '\n✓ OpenDray %s installed to %s\n' "$TAG" "$TARGET"

# ── PATH hint ────────────────────────────────────────────────
case ":$PATH:" in
    *":$INSTALL_DIR:"*)
        IN_PATH=1
        ;;
    *)
        IN_PATH=0
        printf '\n⚠  %s is not in your PATH.\n' "$INSTALL_DIR"
        printf '   Add this line to your shell rc (~/.zshrc or ~/.bashrc):\n\n'
        printf '       export PATH="%s:$PATH"\n' "$INSTALL_DIR"
        ;;
esac

# ── next steps ───────────────────────────────────────────────
printf '\nNext steps:\n'
if [ "$IN_PATH" = "1" ]; then
    printf '   opendray\n'
else
    printf '   %s\n' "$TARGET"
fi
printf '\nOn first run OpenDray starts a setup wizard in your browser.\n'
