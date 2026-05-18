#!/usr/bin/env bash
# scripts/uninstall-linux.sh
# Interactive uninstaller for opendray on Linux (Ubuntu / Debian).
#
# Default mode:    remove the running gateway (binary + systemd unit),
#                  KEEP config + data + database.
# --purge:         also drop the DB, role, config, data dir, logs,
#                  and service user.

set -euo pipefail

# Reattach stdin to the controlling terminal when invoked via `curl | bash`.
if [ ! -t 0 ] && [ -r /dev/tty ]; then
    exec </dev/tty
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "$SCRIPT_DIR/lib/common.sh"

# ── Defaults (mirror install-linux.sh) ───────────────────────────────
: "${OPENDRAY_PREFIX:=/usr/local}"
: "${OPENDRAY_CONFIG_DIR:=/etc/opendray}"
: "${OPENDRAY_DATA_DIR:=/var/lib/opendray}"
: "${OPENDRAY_LOG_DIR:=/var/log/opendray}"
: "${OPENDRAY_SERVICE_USER:=opendray}"
: "${OPENDRAY_SERVICE_NAME:=opendray}"

# Also accept env-var forms so `curl … | OPENDRAY_PURGE=1 bash` works,
# bypassing the `bash -s -- --flag` syntax that's easy to mangle when
# pasting a multi-line command into a terminal.
PURGE="${OPENDRAY_PURGE:-0}"
ASSUME_YES="${OPENDRAY_YES:-0}"

for arg in "$@"; do
    case "$arg" in
        --purge)     PURGE=1 ;;
        --yes|-y)    ASSUME_YES=1 ;;
        -h|--help)
            cat <<'EOF'
opendray Linux uninstaller

Usage:
  bash scripts/uninstall-linux.sh [options]

Options:
  --purge      Also drop the PostgreSQL database + role, delete config,
               data directory, logs, and the service user. Default mode
               only removes the running gateway and keeps user data so
               you can re-install later and resume.
  -y, --yes    Skip all confirmation prompts. CAUTION when combined with
               --purge: that combination deletes data with no second
               chance. Useful for automation.
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

log_section "opendray uninstaller — Linux"

[ -f /etc/os-release ] || log_die "/etc/os-release missing — can't identify the distribution."
# shellcheck disable=SC1091
. /etc/os-release
case "${ID:-}${ID_LIKE:+ $ID_LIKE}" in
    *ubuntu*|*debian*) ;;
    *) log_warn "Distro is ${PRETTY_NAME:-$ID}. Uninstall steps assume Ubuntu / Debian; some may not match." ;;
esac

require_root_or_sudo

# Detect what's actually installed so we report accurately.
HAS_UNIT=0
[ -f "/etc/systemd/system/${OPENDRAY_SERVICE_NAME}.service" ] && HAS_UNIT=1

HAS_BINARY=0
[ -x "$OPENDRAY_PREFIX/bin/opendray" ] && HAS_BINARY=1

HAS_CONFIG=0
[ -d "$OPENDRAY_CONFIG_DIR" ] && HAS_CONFIG=1

HAS_DATA=0
[ -d "$OPENDRAY_DATA_DIR" ] && HAS_DATA=1

HAS_LOGS=0
[ -d "$OPENDRAY_LOG_DIR" ] && HAS_LOGS=1

HAS_USER=0
id "$OPENDRAY_SERVICE_USER" >/dev/null 2>&1 && HAS_USER=1

# Bail early if literally nothing to uninstall.
if [ "$HAS_UNIT" = "0" ] && [ "$HAS_BINARY" = "0" ] && [ "$HAS_CONFIG" = "0" ] && [ "$HAS_DATA" = "0" ]; then
    log_warn "No opendray install found at the standard locations."
    log_dim "  binary:   $OPENDRAY_PREFIX/bin/opendray"
    log_dim "  config:   $OPENDRAY_CONFIG_DIR"
    log_dim "  data:     $OPENDRAY_DATA_DIR"
    log_dim "  unit:     /etc/systemd/system/${OPENDRAY_SERVICE_NAME}.service"
    log_info "Nothing to do."
    exit 0
fi

log_section "What this uninstaller will do"

cat <<EOF
$([ "$HAS_UNIT" = 1 ] && echo "  ✓ stop systemd service '${OPENDRAY_SERVICE_NAME}' and disable on boot")
$([ "$HAS_UNIT" = 1 ] && echo "  ✓ delete unit file /etc/systemd/system/${OPENDRAY_SERVICE_NAME}.service")
$([ "$HAS_BINARY" = 1 ] && echo "  ✓ delete binary $OPENDRAY_PREFIX/bin/opendray")
EOF

if [ "$PURGE" = "1" ]; then
    cat <<EOF
  ${C_RED}--purge enabled — destructive steps below:${C_NC}
$([ "$HAS_CONFIG" = 1 ] && echo "  ✗ delete config directory $OPENDRAY_CONFIG_DIR (config.toml + any rotated bcrypt keys)")
$([ "$HAS_DATA" = 1 ]   && echo "  ✗ delete data directory $OPENDRAY_DATA_DIR (sessions, notes, vault, backups)")
$([ "$HAS_LOGS" = 1 ]   && echo "  ✗ delete log directory $OPENDRAY_LOG_DIR")
$([ "$HAS_USER" = 1 ]   && echo "  ✗ remove service account '$OPENDRAY_SERVICE_USER'")
  ✗ drop the PostgreSQL database + role (will prompt for superuser)
EOF
else
    cat <<EOF
  ${C_GRN}Keeping (safe default):${C_NC}
$([ "$HAS_CONFIG" = 1 ] && echo "  · config directory $OPENDRAY_CONFIG_DIR")
$([ "$HAS_DATA" = 1 ]   && echo "  · data directory $OPENDRAY_DATA_DIR")
$([ "$HAS_LOGS" = 1 ]   && echo "  · log directory $OPENDRAY_LOG_DIR")
$([ "$HAS_USER" = 1 ]   && echo "  · service account '$OPENDRAY_SERVICE_USER'")
  · the PostgreSQL database + role
EOF
fi

cat <<EOF

What this uninstaller will ${C_BLU}NOT${C_NC} touch (these are general-purpose tools
the wizard merely installed for opendray's benefit):
  · Node.js, npm, pnpm
  · The Claude / Codex / Gemini CLIs (they wrap your accounts)
  · PostgreSQL itself (only the opendray database if --purge)
  · apt packages (build-essential, postgresql-client, etc.)

EOF

confirm "Proceed?" || { log_info "Aborted — nothing changed."; exit 0; }

# ───────────────────────────────────────────────────────────────────────
# Phase 1 — Stop + remove the systemd unit
# ───────────────────────────────────────────────────────────────────────

if [ "$HAS_UNIT" = "1" ]; then
    log_step 1 "Stop and remove systemd unit"
    if run_priv systemctl is-active --quiet "$OPENDRAY_SERVICE_NAME"; then
        log_info "Stopping ${OPENDRAY_SERVICE_NAME}..."
        run_priv systemctl stop "$OPENDRAY_SERVICE_NAME"
    fi
    if run_priv systemctl is-enabled --quiet "$OPENDRAY_SERVICE_NAME" 2>/dev/null; then
        run_priv systemctl disable "$OPENDRAY_SERVICE_NAME"
    fi
    run_priv rm -f "/etc/systemd/system/${OPENDRAY_SERVICE_NAME}.service"
    run_priv systemctl daemon-reload
    run_priv systemctl reset-failed "$OPENDRAY_SERVICE_NAME" 2>/dev/null || true
    log_ok "systemd unit removed"
fi

# ───────────────────────────────────────────────────────────────────────
# Phase 2 — Remove the binary
# ───────────────────────────────────────────────────────────────────────

if [ "$HAS_BINARY" = "1" ]; then
    log_step 2 "Remove binary"
    run_priv rm -f "$OPENDRAY_PREFIX/bin/opendray"
    log_ok "Removed $OPENDRAY_PREFIX/bin/opendray"
fi

# ───────────────────────────────────────────────────────────────────────
# Phase 3 — Purge (only if --purge)
# ───────────────────────────────────────────────────────────────────────

if [ "$PURGE" != "1" ]; then
    log_section "Uninstall complete (gateway removed)"
    cat <<EOF
  The opendray gateway is stopped and removed from this host.

  Preserved for a future re-install:
$([ "$HAS_CONFIG" = 1 ] && echo "    $OPENDRAY_CONFIG_DIR/config.toml")
$([ "$HAS_DATA" = 1 ]   && echo "    $OPENDRAY_DATA_DIR/  (bcrypt keyfile, sessions, notes, vault)")
$([ "$HAS_LOGS" = 1 ]   && echo "    $OPENDRAY_LOG_DIR/    (audit history)")
$([ "$HAS_USER" = 1 ]   && echo "    user '$OPENDRAY_SERVICE_USER'")
    PostgreSQL database (your existing data)

  Re-run the installer any time to bring opendray back. To also drop
  data + DB, run this uninstaller again with --purge.
EOF
    exit 0
fi

# Read the old config (if it still exists) to know how to drop the DB.
# We need: DSN, app DB user, app DB name.
OLD_CONFIG="$OPENDRAY_CONFIG_DIR/config.toml"
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
PG_SUPER_PW_AUTH="password"
PG_SUPER_DB="postgres"
OD_DB_NAME=""
OD_DB_USER=""

if [ -n "$OD_DB_URL" ]; then
    # postgres://<user>:<pw>@<host>:<port>/<db>?...
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
    ask_with_default "Database name to drop"     "opendray"        OD_DB_NAME
    ask_with_default "App role to drop"          "opendray_user"   OD_DB_USER
    ask_with_default "PG host"                   "127.0.0.1"       PG_SUPER_HOST
    ask_with_default "PG port"                   "5432"            PG_SUPER_PORT
fi

cat <<EOF

To drop $OD_DB_NAME and role $OD_DB_USER we need superuser access on
the PG host. Two options:
EOF

ask_menu "How should we connect as superuser?" \
    "Local peer auth as the 'postgres' OS user (works if PG was installed locally by the wizard)|Network auth with a superuser name + password" \
    PG_SUPER_PATH

case "$PG_SUPER_PATH" in
    Local*)  PG_SUPER_PW_AUTH="peer"; PG_SUPER_USER="postgres" ;;
    Network*)
        PG_SUPER_PW_AUTH="password"
        ask_with_default "Superuser name"  "postgres" PG_SUPER_USER
        ask_with_default "Maintenance DB"  "postgres" PG_SUPER_DB
        ask_password    "Superuser password"           PG_SUPER_PW
        ;;
esac

run_psql_super() {
    local sql="$1" target_db="${2:-$PG_SUPER_DB}"
    if [ "$PG_SUPER_PW_AUTH" = "peer" ]; then
        run_priv_as postgres psql -v ON_ERROR_STOP=1 -d "$target_db" -c "$sql"
    else
        PGPASSWORD="$PG_SUPER_PW" psql -v ON_ERROR_STOP=1 \
            -h "$PG_SUPER_HOST" -p "$PG_SUPER_PORT" -U "$PG_SUPER_USER" -d "$target_db" \
            -c "$sql"
    fi
}

# Terminate any active connections to the target DB before dropping it.
# Without this, DROP DATABASE fails with "database is being accessed by other users".
log_info "Terminating any active connections to '$OD_DB_NAME'..."
run_psql_super "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$OD_DB_NAME' AND pid <> pg_backend_pid()" \
    >/dev/null 2>&1 || true

log_info "Dropping database '$OD_DB_NAME' (skip if absent)..."
run_psql_super "DROP DATABASE IF EXISTS \"$OD_DB_NAME\"" || log_warn "DROP DATABASE failed — manual cleanup needed."

if [ -n "$OD_DB_USER" ]; then
    log_info "Dropping role '$OD_DB_USER' (skip if absent)..."
    run_psql_super "DROP ROLE IF EXISTS \"$OD_DB_USER\"" || log_warn "DROP ROLE failed — manual cleanup needed."
fi
log_ok "Database + role dropped"

# ───────────────────────────────────────────────────────────────────────
# Phase 4 — Delete config / data / logs / user
# ───────────────────────────────────────────────────────────────────────
#
# Deletions are unconditional in --purge mode. We don't gate on the
# HAS_* flags from Phase 0 — those represent "was present at the start
# of this run", but a previous partial run might have left a stray
# config.toml that Phase 0 didn't see (e.g., when the script crashed
# mid-execution). `rm -rf ...` is a no-op on already-absent paths
# anyway, so always running it costs nothing.

log_step 4 "Delete files + service account"

run_priv rm -rf "$OPENDRAY_CONFIG_DIR"
log_ok "Removed $OPENDRAY_CONFIG_DIR (config.toml + opendray.env if present)"

run_priv rm -rf "$OPENDRAY_DATA_DIR"
log_ok "Removed $OPENDRAY_DATA_DIR (data + bcrypt keyfile)"

run_priv rm -rf "$OPENDRAY_LOG_DIR"
log_ok "Removed $OPENDRAY_LOG_DIR (logs)"

if id "$OPENDRAY_SERVICE_USER" >/dev/null 2>&1; then
    # `userdel -r` would also wipe the home dir; we've already nuked it
    # via OPENDRAY_DATA_DIR, so plain `userdel` is enough.
    run_priv userdel "$OPENDRAY_SERVICE_USER" 2>/dev/null || true
    log_ok "Removed service user '$OPENDRAY_SERVICE_USER'"
fi

# Post-delete verification — bail loudly if any of the standard paths
# survived (e.g., bind mount, immutable flag, ENOENT race). The whole
# point of --purge is "no trace left"; if something's still on disk,
# the operator needs to know now, not when they re-install.
log_info "Verifying nothing survived..."
SURVIVORS=()
for p in "$OPENDRAY_CONFIG_DIR" "$OPENDRAY_DATA_DIR" "$OPENDRAY_LOG_DIR" \
         "/etc/systemd/system/${OPENDRAY_SERVICE_NAME}.service" \
         "$OPENDRAY_PREFIX/bin/opendray"; do
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
  opendray and all of its persistent state on this host have been
  removed:

    binary, systemd unit          ✓ gone
    config / data / logs          ✓ gone
    service user '$OPENDRAY_SERVICE_USER'              ✓ gone
    database '$OD_DB_NAME' + role '$OD_DB_USER'   ✓ dropped

  Still on the host (and NOT touched by uninstall):
    · Node.js / npm
    · The AI CLIs (claude / codex / gemini) and their credentials
    · PostgreSQL server itself + any other databases on it
    · apt packages

  Re-install any time with:
    curl -fsSL https://raw.githubusercontent.com/Opendray/opendray_v2/main/scripts/install.sh | bash
EOF
