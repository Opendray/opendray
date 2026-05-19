#!/usr/bin/env bash
# opendray universal installer — dual-mode entry point.
#
# Mode 1: local checkout
#   Invoked as `bash scripts/install.sh` from inside a clone. We detect
#   the sibling `lib/common.sh` and exec the matching OS installer.
#
# Mode 2: curl | bash bootstrap
#   Invoked over a pipe with no on-disk script directory:
#     curl -fsSL https://raw.githubusercontent.com/Opendray/opendray_v2/main/scripts/install.sh | bash
#   We install git if missing, shallow-clone the repo to a working
#   directory, then re-execute scripts/install.sh from the clone.
#
# Either way the user ends up running install-linux.sh / install-macos.sh
# from a real checkout — the bootstrap path just adds one git clone in
# front.

# Must run under bash (not sh/dash/zsh) — we use BASH_SOURCE, arrays,
# [[ ]], and printf -v downstream. No version floor: bash 3.2 (macOS
# /bin/bash) is supported. This only catches `curl ... | sh` mistakes.
if [ -z "${BASH_VERSION:-}" ]; then
    echo "[!] opendray installer must be run with bash, not sh. Re-run with:" >&2
    echo "    curl -fsSL https://raw.githubusercontent.com/Opendray/opendray_v2/main/scripts/install.sh | bash" >&2
    exit 1
fi

set -euo pipefail

# ── Locate ourselves on disk (or detect we're piped) ──────────────────
SCRIPT_DIR=""
if [ -n "${BASH_SOURCE[0]:-}" ] && [ -f "${BASH_SOURCE[0]}" ]; then
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi

# ── Mode 1: local checkout ───────────────────────────────────────────
if [ -n "$SCRIPT_DIR" ] && [ -f "$SCRIPT_DIR/lib/common.sh" ]; then
    case "$(uname -s)" in
        Linux)
            exec bash "$SCRIPT_DIR/install-linux.sh" "$@"
            ;;
        Darwin)
            exec bash "$SCRIPT_DIR/install-macos.sh" "$@"
            ;;
        MINGW*|MSYS*|CYGWIN*)
            cat <<'EOF'
[!] Native Windows is not supported.
    Use WSL2 + Ubuntu, then rerun this installer inside the WSL distribution.
    See scripts/install-windows.ps1 for the WSL2 setup helper.
EOF
            exit 1
            ;;
        *)
            echo "[!] Unsupported OS: $(uname -s)"
            exit 1
            ;;
    esac
fi

# ── Mode 2: curl | bash bootstrap ────────────────────────────────────

# Colours only if the parent terminal is interactive (curl|bash usually is).
if [ -t 1 ]; then
    C_RED=$'\033[0;31m'; C_GRN=$'\033[0;32m'; C_YEL=$'\033[1;33m'
    C_BLU=$'\033[0;34m'; C_NC=$'\033[0m'
else
    C_RED=''; C_GRN=''; C_YEL=''; C_BLU=''; C_NC=''
fi
say()  { printf "${C_BLU}[*]${C_NC} %s\n" "$*"; }
ok()   { printf "${C_GRN}[✓]${C_NC} %s\n" "$*"; }
warn() { printf "${C_YEL}[!]${C_NC} %s\n" "$*"; }
die()  { printf "${C_RED}[✗]${C_NC} %s\n" "$*" >&2; exit 1; }

REPO_URL="${OPENDRAY_INSTALL_REPO:-https://github.com/Opendray/opendray_v2.git}"
REF="${OPENDRAY_INSTALL_REF:-main}"
INSTALL_DIR="${OPENDRAY_INSTALL_DIR:-${TMPDIR:-/tmp}/opendray-install-$$}"

cat <<EOF
${C_BLU}━━━ opendray installer — bootstrap ━━━${C_NC}

  Source:   ${REPO_URL}@${REF}
  Workdir:  ${INSTALL_DIR}

This bootstrap step:
  1. Ensures git is installed (apt / brew if missing — needs sudo).
  2. Shallow-clones opendray_v2 to the workdir above.
  3. Hands off to scripts/install-<os>.sh, which is the real wizard.

After the wizard completes, the workdir is left on disk for re-runs.
Delete it whenever you want — opendray itself lives elsewhere (under
/usr/local/bin, /etc/opendray, /var/lib/opendray on Linux, or
~/.opendray on macOS).

EOF

# ── Detect OS for git install fallback + downstream pivot ────────────
case "$(uname -s)" in
    Linux)
        OS_KIND="linux"
        ;;
    Darwin)
        OS_KIND="macos"
        ;;
    MINGW*|MSYS*|CYGWIN*)
        cat <<'EOF'
[!] Native Windows is not supported by opendray. Use WSL2:

    Open PowerShell as Administrator and run:
        wsl --install -d Ubuntu

    Reboot when prompted, finish Ubuntu's first-time setup, then
    re-run this curl | bash command inside the Ubuntu shell.

EOF
        exit 1
        ;;
    *)
        die "Unsupported OS: $(uname -s). Supported: Linux, macOS."
        ;;
esac
ok "OS detected: $OS_KIND"

# ── Install git if missing ────────────────────────────────────────────
if ! command -v git >/dev/null 2>&1; then
    warn "git is not installed."
    case "$OS_KIND" in
        linux)
            if ! command -v apt-get >/dev/null 2>&1; then
                die "git missing and this distro doesn't have apt-get. Install git manually then re-run."
            fi
            say "Installing git via apt..."
            if [ "$EUID" -eq 0 ]; then
                apt-get update -qq && apt-get install -y -qq git
            elif command -v sudo >/dev/null 2>&1; then
                sudo apt-get update -qq && sudo apt-get install -y -qq git
            else
                die "Need git, but no sudo. Install git manually then re-run."
            fi
            ;;
        macos)
            if command -v brew >/dev/null 2>&1; then
                say "Installing git via Homebrew..."
                brew install git
            else
                die "git missing. Install Homebrew (https://brew.sh) or run 'xcode-select --install', then re-run."
            fi
            ;;
    esac
    ok "git installed: $(git --version)"
fi

# ── Clone the repo into the workdir ───────────────────────────────────
if [ -d "$INSTALL_DIR/.git" ]; then
    say "Re-using existing clone at $INSTALL_DIR (fast-forwarding)..."
    git -C "$INSTALL_DIR" fetch --depth=1 origin "$REF"
    git -C "$INSTALL_DIR" reset --hard "origin/$REF"
else
    say "Cloning $REPO_URL (ref: $REF) → $INSTALL_DIR ..."
    rm -rf "$INSTALL_DIR"
    git clone --depth=1 --branch "$REF" "$REPO_URL" "$INSTALL_DIR" 2>&1 \
        | grep -vE '^Cloning into|^remote: (Enumerating|Counting|Compressing|Total)' || true
fi
ok "Clone ready at $INSTALL_DIR"

# ── Pivot to the OS-specific installer ────────────────────────────────
cd "$INSTALL_DIR"
case "$OS_KIND" in
    linux)
        exec bash scripts/install-linux.sh "$@"
        ;;
    macos)
        exec bash scripts/install-macos.sh "$@"
        ;;
esac
