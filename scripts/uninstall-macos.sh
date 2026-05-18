#!/usr/bin/env bash
# scripts/uninstall-macos.sh
# Interactive uninstaller for opendray on macOS.
#
# Mirrors install-macos.sh: removes the LaunchAgent (or LaunchDaemon)
# and the ~/.opendray tree. With --purge, also drops the database
# + role.

set -euo pipefail

if [ ! -t 0 ] && [ -r /dev/tty ]; then
    exec </dev/tty
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "$SCRIPT_DIR/lib/common.sh"

: "${OPENDRAY_HOME:=$HOME/.opendray}"
LABEL="com.opendray.opendray"

# We don't know which scope was used; check both.
USER_PLIST="$HOME/Library/LaunchAgents/${LABEL}.plist"
SYS_PLIST="/Library/LaunchDaemons/${LABEL}.plist"

PURGE="${OPENDRAY_PURGE:-0}"
ASSUME_YES="${OPENDRAY_YES:-0}"

for arg in "$@"; do
    case "$arg" in
        --purge)  PURGE=1 ;;
        --yes|-y) ASSUME_YES=1 ;;
        -h|--help)
            cat <<'EOF'
opendray macOS uninstaller

Usage:
  bash scripts/uninstall-macos.sh [options]

Options:
  --purge      Also drop the PostgreSQL database + role, delete
               ~/.opendray. Default mode only stops + unloads the
               launchd unit and removes the binary.
  -y, --yes    Skip confirmation prompts.
  -h, --help   Print this help.
EOF
            exit 0 ;;
        *) log_warn "Unknown option: $arg (ignored)" ;;
    esac
done

confirm() {
    local prompt="$1"
    [ "$ASSUME_YES" = "1" ] && return 0
    local answer
    ask_yes_no "$prompt" "n" answer
    [ "$answer" = "y" ]
}

# ───────────────────────────────────────────────────────────────────────
# Phase 0 — Sanity + plan
# ───────────────────────────────────────────────────────────────────────

log_section "opendray uninstaller — macOS"

[[ "$(uname -s)" == "Darwin" ]] || log_die "Run this on macOS only."

# Detect what's installed.
HAS_USER_PLIST=0
[ -f "$USER_PLIST" ] && HAS_USER_PLIST=1

HAS_SYS_PLIST=0
[ -f "$SYS_PLIST" ] && HAS_SYS_PLIST=1

HAS_HOME=0
[ -d "$OPENDRAY_HOME" ] && HAS_HOME=1

HAS_BIN=0
[ -x "$OPENDRAY_HOME/bin/opendray" ] && HAS_BIN=1

if [ "$HAS_USER_PLIST" = "0" ] && [ "$HAS_SYS_PLIST" = "0" ] && [ "$HAS_HOME" = "0" ]; then
    log_warn "No opendray install found at the standard locations."
    log_dim "  ~/.opendray/             (binary, config, data, logs)"
    log_dim "  ~/Library/LaunchAgents/${LABEL}.plist"
    log_dim "  /Library/LaunchDaemons/${LABEL}.plist"
    log_info "Nothing to do."
    exit 0
fi

log_section "What this uninstaller will do"

cat <<EOF
$([ "$HAS_USER_PLIST" = 1 ] && echo "  ✓ bootout + delete user LaunchAgent $USER_PLIST")
$([ "$HAS_SYS_PLIST"  = 1 ] && echo "  ✓ bootout + delete system LaunchDaemon $SYS_PLIST (needs sudo)")
$([ "$HAS_BIN"        = 1 ] && echo "  ✓ delete binary $OPENDRAY_HOME/bin/opendray")
EOF

if [ "$PURGE" = "1" ]; then
    cat <<EOF
  ${C_RED}--purge enabled — destructive steps below:${C_NC}
$([ "$HAS_HOME" = 1 ] && echo "  ✗ delete entire $OPENDRAY_HOME (config.toml, data/, logs/, bcrypt keyfile)")
  ✗ drop the PostgreSQL database + role (will prompt for superuser)
EOF
else
    cat <<EOF
  ${C_GRN}Keeping (safe default):${C_NC}
$([ "$HAS_HOME" = 1 ] && echo "  · $OPENDRAY_HOME (config + data)")
  · the PostgreSQL database + role
EOF
fi

cat <<EOF

What this uninstaller will ${C_BLU}NOT${C_NC} touch:
  · Homebrew, Node.js, pnpm
  · The Claude / Codex / Gemini CLIs and their credentials
  · PostgreSQL itself (only the opendray database if --purge)

EOF

confirm "Proceed?" || { log_info "Aborted — nothing changed."; exit 0; }

# ───────────────────────────────────────────────────────────────────────
# Phase 1 — Unload + remove launchd units
# ───────────────────────────────────────────────────────────────────────

if [ "$HAS_USER_PLIST" = "1" ]; then
    log_step 1 "Unload user LaunchAgent"
    launchctl bootout "gui/$(id -u)/${LABEL}" 2>/dev/null || true
    rm -f "$USER_PLIST"
    log_ok "User LaunchAgent removed"
fi

if [ "$HAS_SYS_PLIST" = "1" ]; then
    log_step 1 "Unload system LaunchDaemon"
    run_priv launchctl bootout "system/${LABEL}" 2>/dev/null || true
    run_priv rm -f "$SYS_PLIST"
    log_ok "System LaunchDaemon removed"
fi

# ───────────────────────────────────────────────────────────────────────
# Phase 2 — Remove binary
# ───────────────────────────────────────────────────────────────────────

if [ "$HAS_BIN" = "1" ]; then
    log_step 2 "Remove binary"
    rm -f "$OPENDRAY_HOME/bin/opendray"
    log_ok "Removed $OPENDRAY_HOME/bin/opendray"
fi

# ───────────────────────────────────────────────────────────────────────
# Phase 3 — Purge
# ───────────────────────────────────────────────────────────────────────

if [ "$PURGE" != "1" ]; then
    log_section "Uninstall complete (gateway removed)"
    cat <<EOF
  The opendray gateway is stopped and removed from this Mac.

  Preserved for a future re-install:
$([ "$HAS_HOME" = 1 ] && echo "    $OPENDRAY_HOME/ (config.toml + data + logs)")
    PostgreSQL database (your existing data)

  Re-run the installer any time. To also drop data + DB, rerun with
  --purge.
EOF
    exit 0
fi

# Read old config to find DSN bits.
OLD_CONFIG="$OPENDRAY_HOME/config.toml"
OD_DB_URL=""
if [ -f "$OLD_CONFIG" ]; then
    OD_DB_URL="$(awk '
        /^\[database\]/ { in_db = 1; next }
        /^\[/           { in_db = 0 }
        in_db && /^[[:space:]]*url[[:space:]]*=/ {
            sub(/^[^"]*"/, "")
            sub(/".*$/, "")
            print
            exit
        }
    ' "$OLD_CONFIG")"
fi

PG_SUPER_USER=""
PG_SUPER_HOST=""
PG_SUPER_PORT="5432"
PG_SUPER_PW=""
PG_SUPER_DB="postgres"
OD_DB_NAME=""
OD_DB_USER=""

if [ -n "$OD_DB_URL" ]; then
    TMP="${OD_DB_URL#postgres://}"
    OD_DB_USER="${TMP%%:*}"
    TMP="${TMP#*@}"
    HOSTPORT="${TMP%%/*}"
    OD_DB_NAME="${TMP#*/}"
    OD_DB_NAME="${OD_DB_NAME%%\?*}"
    PG_SUPER_HOST="${HOSTPORT%:*}"
    [ "$HOSTPORT" != "$PG_SUPER_HOST" ] && PG_SUPER_PORT="${HOSTPORT#*:}"
    log_info "Detected from config: $OD_DB_USER@$PG_SUPER_HOST:$PG_SUPER_PORT/$OD_DB_NAME"
fi

log_step 3 "Drop the opendray database"

if [ -z "$OD_DB_NAME" ]; then
    log_warn "Couldn't parse a DB URL from the old config."
    ask_with_default "Database name to drop"  "opendray"      OD_DB_NAME
    ask_with_default "App role to drop"       "opendray_user" OD_DB_USER
    ask_with_default "PG host"                "127.0.0.1"     PG_SUPER_HOST
    ask_with_default "PG port"                "5432"          PG_SUPER_PORT
fi

cat <<EOF

To drop $OD_DB_NAME we need superuser access on the PG host.

EOF

ask_menu "How should we connect as superuser?" \
    "Local trust auth as your macOS user (works with brew-installed PG)|Network auth with a superuser name + password" \
    PG_SUPER_PATH

if [[ "$PG_SUPER_PATH" == Local* ]]; then
    PG_SUPER_USER="$USER"
    PG_SUPER_PW=""
else
    ask_with_default "Superuser name"  "postgres" PG_SUPER_USER
    ask_with_default "Maintenance DB"  "postgres" PG_SUPER_DB
    ask_password    "Superuser password"           PG_SUPER_PW
fi

run_psql_super() {
    local sql="$1" target_db="${2:-$PG_SUPER_DB}"
    if [ -z "$PG_SUPER_PW" ]; then
        psql -h "$PG_SUPER_HOST" -p "$PG_SUPER_PORT" -U "$PG_SUPER_USER" -d "$target_db" \
            -v ON_ERROR_STOP=1 -c "$sql"
    else
        PGPASSWORD="$PG_SUPER_PW" psql \
            -h "$PG_SUPER_HOST" -p "$PG_SUPER_PORT" -U "$PG_SUPER_USER" -d "$target_db" \
            -v ON_ERROR_STOP=1 -c "$sql"
    fi
}

log_info "Terminating any active connections to '$OD_DB_NAME'..."
run_psql_super "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$OD_DB_NAME' AND pid <> pg_backend_pid()" \
    >/dev/null 2>&1 || true

log_info "Dropping database '$OD_DB_NAME'..."
run_psql_super "DROP DATABASE IF EXISTS \"$OD_DB_NAME\"" || log_warn "DROP DATABASE failed — manual cleanup needed."

if [ -n "$OD_DB_USER" ]; then
    log_info "Dropping role '$OD_DB_USER'..."
    run_psql_super "DROP ROLE IF EXISTS \"$OD_DB_USER\"" || log_warn "DROP ROLE failed — manual cleanup needed."
fi
log_ok "Database + role dropped"

# ───────────────────────────────────────────────────────────────────────
# Phase 4 — Delete ~/.opendray
# ───────────────────────────────────────────────────────────────────────

# Deletions are unconditional in --purge mode (see uninstall-linux.sh
# for the rationale — Phase 0 detection is just a UX hint; the actual
# cleanup is best-effort blanket rm -rf).

log_step 4 "Delete $OPENDRAY_HOME"
rm -rf "$OPENDRAY_HOME"
log_ok "Removed $OPENDRAY_HOME (config.toml + opendray.env + launcher + data + logs)"

# Post-delete verification.
log_info "Verifying nothing survived..."
SURVIVORS=()
for p in "$OPENDRAY_HOME" "$USER_PLIST" "$SYS_PLIST"; do
    [ -e "$p" ] && SURVIVORS+=("$p")
done
if [ "${#SURVIVORS[@]}" -gt 0 ]; then
    log_err "Some paths still exist after --purge — manual cleanup needed:"
    for p in "${SURVIVORS[@]}"; do
        printf "  %s\n" "$p" >&2
        ls -la "$p" 2>&1 | head -5 | sed 's/^/    /' >&2
    done
    exit 1
fi
log_ok "Verified — no opendray paths remain on disk"

# ───────────────────────────────────────────────────────────────────────
# Done
# ───────────────────────────────────────────────────────────────────────

log_section "Purge complete"

cat <<EOF
  opendray and all of its persistent state on this Mac have been removed:

    launchd unit + binary    ✓ gone
    ~/.opendray              ✓ gone
    database '$OD_DB_NAME' + role '$OD_DB_USER'   ✓ dropped

  Still installed:
    · Homebrew, Node.js
    · The AI CLIs and their credentials
    · PostgreSQL server itself + any other databases

  Re-install any time with:
    curl -fsSL https://raw.githubusercontent.com/Opendray/opendray_v2/main/scripts/install.sh | bash
EOF
