#!/usr/bin/env bash
# opendray uninstaller — dual-mode entry point.
#
# Mode 1: local checkout
#   bash scripts/uninstall.sh [--purge]
#
# Mode 2: curl | bash
#   curl -fsSL https://raw.githubusercontent.com/Opendray/opendray_v2/main/scripts/uninstall.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/Opendray/opendray_v2/main/scripts/uninstall.sh | bash -s -- --purge
#
# Default (no --purge):
#   Stops + removes the running gateway (binary, systemd unit / launchd
#   plist) but KEEPS the database, config, and data directory. You can
#   re-install later and pick up where you left off.
#
# --purge:
#   Also drops the PostgreSQL database + role, deletes config, data
#   directory, logs, and the service user.
#   Asks for confirmation before each destructive step.

set -euo pipefail

SCRIPT_DIR=""
if [ -n "${BASH_SOURCE[0]:-}" ] && [ -f "${BASH_SOURCE[0]}" ]; then
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi

# ── Mode 1: local checkout ───────────────────────────────────────────
if [ -n "$SCRIPT_DIR" ] && [ -f "$SCRIPT_DIR/lib/common.sh" ]; then
    case "$(uname -s)" in
        Linux)  exec bash "$SCRIPT_DIR/uninstall-linux.sh" "$@" ;;
        Darwin) exec bash "$SCRIPT_DIR/uninstall-macos.sh" "$@" ;;
        *)      echo "[!] Unsupported OS: $(uname -s)"; exit 1 ;;
    esac
fi

# ── Mode 2: curl | bash bootstrap ────────────────────────────────────
if [ -t 1 ]; then
    C_RED=$'\033[0;31m'; C_GRN=$'\033[0;32m'; C_YEL=$'\033[1;33m'
    C_BLU=$'\033[0;34m'; C_NC=$'\033[0m'
else
    C_RED=''; C_GRN=''; C_YEL=''; C_BLU=''; C_NC=''
fi
say() { printf "${C_BLU}[*]${C_NC} %s\n" "$*"; }
die() { printf "${C_RED}[✗]${C_NC} %s\n" "$*" >&2; exit 1; }

REPO_URL="${OPENDRAY_INSTALL_REPO:-https://github.com/Opendray/opendray_v2.git}"
REF="${OPENDRAY_INSTALL_REF:-main}"
INSTALL_DIR="${OPENDRAY_INSTALL_DIR:-${TMPDIR:-/tmp}/opendray-uninstall-$$}"

cat <<EOF
${C_BLU}━━━ opendray uninstaller — bootstrap ━━━${C_NC}

  Source:   ${REPO_URL}@${REF}
  Workdir:  ${INSTALL_DIR}

Same as install: ensures git, shallow-clones the repo, then hands off
to scripts/uninstall-<os>.sh. The workdir is left on disk after the
uninstall so you can inspect or rerun.

EOF

case "$(uname -s)" in
    Linux)  OS_KIND="linux" ;;
    Darwin) OS_KIND="macos" ;;
    *) die "Unsupported OS: $(uname -s). Supported: Linux, macOS." ;;
esac

if ! command -v git >/dev/null 2>&1; then
    case "$OS_KIND" in
        linux)
            command -v apt-get >/dev/null 2>&1 || die "git missing, no apt-get. Install git first."
            say "Installing git via apt..."
            if [ "$EUID" -eq 0 ]; then
                apt-get update -qq && apt-get install -y -qq git
            elif command -v sudo >/dev/null 2>&1; then
                sudo apt-get update -qq && sudo apt-get install -y -qq git
            else
                die "git missing and no sudo. Install git, rerun."
            fi
            ;;
        macos)
            command -v brew >/dev/null 2>&1 || die "git missing. Install via brew or xcode-select --install."
            brew install git
            ;;
    esac
fi

if [ -d "$INSTALL_DIR/.git" ]; then
    say "Refreshing existing clone..."
    git -C "$INSTALL_DIR" fetch --depth=1 origin "$REF"
    git -C "$INSTALL_DIR" reset --hard "origin/$REF"
else
    say "Cloning $REPO_URL → $INSTALL_DIR ..."
    rm -rf "$INSTALL_DIR"
    git clone --depth=1 --branch "$REF" "$REPO_URL" "$INSTALL_DIR" 2>&1 \
        | grep -vE '^Cloning into|^remote: (Enumerating|Counting|Compressing|Total)' || true
fi

cd "$INSTALL_DIR"
case "$OS_KIND" in
    linux)  exec bash scripts/uninstall-linux.sh "$@" ;;
    macos)  exec bash scripts/uninstall-macos.sh "$@" ;;
esac
