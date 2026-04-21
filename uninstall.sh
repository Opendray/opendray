#!/bin/sh
# OpenDray — nuclear uninstaller for macOS and Linux.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Opendray/opendray/main/uninstall.sh | sh
#
# Prefer `opendray uninstall` when the binary can still run — it's
# nicer (interactive confirmation, external-DB drop-script helper,
# graceful PG shutdown). This shell script is the fallback for:
#   - binary is corrupt / incompatible / missing
#   - config is so broken the Go uninstaller won't start
#   - running from CI / cloud-init cleanup
#
# Environment variables (optional):
#   OPENDRAY_YES=1            skip confirmation prompt
#   OPENDRAY_DRY_RUN=1        print plan, remove nothing
#   OPENDRAY_INSTALL_DIR      override the default binary location
#                             (default: $HOME/.local/bin)
#
# What this script touches:
#   - $HOME/.opendray/                   (data dir — PG cluster, plugins, cache)
#   - $XDG_CONFIG_HOME/opendray/         (config fallback path)
#   - $HOME/.config/opendray/            (config fallback path without XDG)
#   - $OPENDRAY_INSTALL_DIR/opendray     (binary)
#   - any process listening on the configured / default port (127.0.0.1:8640)
#
# What this script does NOT touch:
#   - any external PostgreSQL databases OpenDray was pointed at. Those
#     are user-managed and may share schema names with other apps.

set -eu

YES="${OPENDRAY_YES:-}"
DRY_RUN="${OPENDRAY_DRY_RUN:-}"
INSTALL_DIR="${OPENDRAY_INSTALL_DIR:-$HOME/.local/bin}"
BIN="${INSTALL_DIR}/opendray"
DATA_DIR="${HOME}/.opendray"
XDG_CFG="${XDG_CONFIG_HOME:-$HOME/.config}/opendray"

say()   { printf '%s\n' "$*"; }
info()  { printf '    %s\n' "$*"; }
step()  { printf '→   %s\n' "$*"; }
ok()    { printf '✓   %s\n' "$*"; }
warn()  { printf '⚠   %s\n' "$*"; }
err()   { printf '✗   %s\n' "$*" >&2; }

say ""
say "OpenDray uninstaller (nuclear)"
say "───────────────────────────────────────────────────────"
say ""

# ── detect platform just for early refusal ───────────────────
OS_RAW=$(uname -s)
OS=$(printf '%s' "$OS_RAW" | tr '[:upper:]' '[:lower:]')
case "$OS" in
    darwin|linux) ;;
    mingw*|msys*|cygwin*)
        err "This shell looks like Git Bash / MSYS on Windows.
    OpenDray's Windows uninstaller is a PowerShell script:

        irm https://raw.githubusercontent.com/Opendray/opendray/main/uninstall.ps1 | iex"
        exit 1 ;;
    *)
        err "Unsupported OS: ${OS_RAW}"
        exit 1 ;;
esac

# ── enumerate targets ────────────────────────────────────────
TARGETS=""
add_target() {
    if [ -e "$1" ]; then
        TARGETS="${TARGETS}${1}
"
    fi
}
add_target "$BIN"
add_target "$DATA_DIR"
add_target "$XDG_CFG"

if [ -z "$TARGETS" ]; then
    say "Nothing to remove — no OpenDray files found in the default locations:"
    info "  $BIN"
    info "  $DATA_DIR"
    info "  $XDG_CFG"
    say ""
    say "If you installed to a custom location, set OPENDRAY_INSTALL_DIR."
    exit 0
fi

# ── show plan ────────────────────────────────────────────────
say "Will remove:"
say ""
printf '%s' "$TARGETS" | while IFS= read -r t; do
    [ -z "$t" ] && continue
    info "  $t"
done
say ""
say "Will NOT touch:"
info "  any external PostgreSQL database you may have pointed OpenDray at."
info "  (its tables live in your own DB server and may share names with"
info "   other apps — drop them yourself with SQL if you need to.)"
say ""

# ── dry-run short-circuit ────────────────────────────────────
if [ -n "$DRY_RUN" ]; then
    warn "OPENDRAY_DRY_RUN set — nothing will be removed."
    exit 0
fi

# ── confirm ──────────────────────────────────────────────────
if [ -z "$YES" ]; then
    # Reattach stdin to the TTY when we're piped in (curl | sh).
    if [ -r /dev/tty ] && [ -c /dev/tty ]; then
        exec < /dev/tty
    fi
    printf 'Proceed with removal? [y/N] '
    read REPLY
    case "$REPLY" in
        y|Y|yes|YES) ;;
        *) say "Aborted. Nothing changed."; exit 0 ;;
    esac
fi

# ── stop running processes bound to the OpenDray default port ─
# Best-effort. `lsof` is on macOS + most Linux distros; if absent
# we skip the stop step and hope `rm -rf` wins the race.
PORT_CANDIDATES="8640 5433"
if command -v lsof >/dev/null 2>&1; then
    for p in $PORT_CANDIDATES; do
        PIDS=$(lsof -t -i ":$p" -sTCP:LISTEN 2>/dev/null || true)
        for pid in $PIDS; do
            if [ -n "$pid" ] && [ "$pid" != "$$" ]; then
                step "stopping PID $pid (port :$p)"
                kill -TERM "$pid" 2>/dev/null || true
                # Give it 2s, then SIGKILL.
                i=0
                while kill -0 "$pid" 2>/dev/null && [ $i -lt 10 ]; do
                    sleep 0.2
                    i=$((i + 1))
                done
                kill -0 "$pid" 2>/dev/null && kill -KILL "$pid" 2>/dev/null || true
                ok "stopped PID $pid"
            fi
        done
    done
fi

# ── remove ───────────────────────────────────────────────────
printf '%s' "$TARGETS" | while IFS= read -r t; do
    [ -z "$t" ] && continue
    step "removing $t"
    if [ -d "$t" ]; then
        rm -rf "$t" || { err "could not remove $t"; continue; }
    else
        rm -f "$t" || { err "could not remove $t"; continue; }
    fi
    ok "removed $t"
done

say ""
ok "OpenDray removed."
say ""
case ":$PATH:" in
    *":$INSTALL_DIR:"*)
        info "NOTE: $INSTALL_DIR is still in your PATH (harmless, but you"
        info "      can remove the 'export PATH=…' line from ~/.zshrc / ~/.bashrc"
        info "      if nothing else lives there)."
        ;;
esac
say ""
