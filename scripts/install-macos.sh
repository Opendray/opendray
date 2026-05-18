#!/usr/bin/env bash
# scripts/install-macos.sh
# Interactive installation wizard for opendray on macOS (Intel + Apple Silicon).
#
# Defaults to a user-level LaunchAgent (runs when the user logs in) — best fit
# for a personal Mac or a Mac mini home server. Re-run with --launchd-daemon to
# install a system-wide LaunchDaemon instead.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "$SCRIPT_DIR/lib/common.sh"

# ── Defaults ─────────────────────────────────────────────────────────
: "${OPENDRAY_REPO:=Opendray/opendray_v2}"
: "${OPENDRAY_HOME:=$HOME/.opendray}"

LAUNCHD_SCOPE="agent"     # "agent" (user) or "daemon" (system)
FROM_SOURCE=0
SKIP_SERVICE=0

for arg in "$@"; do
    case "$arg" in
        --launchd-daemon) LAUNCHD_SCOPE="daemon" ;;
        --from-source)    FROM_SOURCE=1 ;;
        --skip-service)   SKIP_SERVICE=1 ;;
        -h|--help)
            cat <<'EOF'
opendray macOS installer wizard

Usage:
  bash scripts/install-macos.sh [options]

Options:
  --launchd-daemon    Install a /Library/LaunchDaemons unit (boot-time, all users).
                      Default: ~/Library/LaunchAgents (login-time, current user).
  --from-source       Build from this checkout instead of downloading a release.
  --skip-service      Install the binary + config but skip launchd registration.
  -h, --help          Show this help.
EOF
            exit 0 ;;
        *) log_warn "Unknown option: $arg (ignored)" ;;
    esac
done

# ───────────────────────────────────────────────────────────────────────
# Phase 0 — Sanity
# ───────────────────────────────────────────────────────────────────────

log_section "opendray installer — macOS"

[[ "$(uname -s)" == "Darwin" ]] || log_die "This installer is for macOS. On Linux use install-linux.sh."

ARCH_RAW="$(uname -m)"
case "$ARCH_RAW" in
    x86_64) ARCH="amd64"; BREW_PREFIX_DEFAULT="/usr/local" ;;
    arm64)  ARCH="arm64"; BREW_PREFIX_DEFAULT="/opt/homebrew" ;;
    *) log_die "Unsupported architecture: $ARCH_RAW" ;;
esac
log_ok "Architecture: $ARCH"

# Locate Homebrew (or refuse to proceed).
if have_cmd brew; then
    BREW_PREFIX="$(brew --prefix)"
else
    log_warn "Homebrew is not installed."
    cat <<'EOF'

This wizard uses brew to install Postgres, Node, pgvector, and pgvector.
Install Homebrew first, then rerun:

    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

EOF
    exit 1
fi
log_ok "Homebrew at $BREW_PREFIX"

# ───────────────────────────────────────────────────────────────────────
# Phase 1 — Plan + confirm
# ───────────────────────────────────────────────────────────────────────

log_section "What this wizard will do"

if [ "$LAUNCHD_SCOPE" = "daemon" ]; then
    SERVICE_TARGET="system-wide LaunchDaemon (/Library/LaunchDaemons, runs at boot)"
else
    SERVICE_TARGET="user LaunchAgent (~/Library/LaunchAgents, runs at login)"
fi

cat <<EOF
6 interactive steps. ^C anytime before service install — nothing is
irreversible until then.

  1) Verify base tools (curl, tar — should already be on macOS)
  2) Install Node.js + pnpm via brew              (for AI CLIs)
  3) Choose AI CLIs (Claude / Codex / Gemini)     (at least one)
  4) Postgres:
       (a) connect to an existing PG host                 — recommended for prod
       (b) install Postgres 16 + pgvector via brew        — easiest for single-Mac

  5) Bootstrap an opendray database, user, pgvector extension.
  6) Generate ~/.opendray/config.toml, run migrations, install
     a ${SERVICE_TARGET}, health-check the gateway.

EOF

ask_yes_no "Ready to start?" "y" READY
[ "$READY" = "y" ] || { log_info "Aborted."; exit 0; }

# ───────────────────────────────────────────────────────────────────────
# Phase 2 — Base tools (mostly pre-installed on macOS)
# ───────────────────────────────────────────────────────────────────────

log_step 1 "Verify base tools"
require_cmds curl tar
log_ok "curl + tar present"

# psql we'll get from postgresql brew (or expect from local PG install).

# ───────────────────────────────────────────────────────────────────────
# Phase 3 — Node + pnpm
# ───────────────────────────────────────────────────────────────────────

log_step 2 "Install Node.js + pnpm"

if ! have_cmd node || [[ "$(node --version 2>/dev/null | sed 's/v//;s/\..*//')" -lt 20 ]]; then
    log_info "Installing node@22 via brew..."
    brew install node@22
    brew link --overwrite --force node@22 2>/dev/null || true
fi
log_ok "Node $(node --version)"

if ! have_cmd pnpm; then
    log_info "Installing pnpm via corepack..."
    corepack enable
    corepack prepare pnpm@latest --activate
fi
log_ok "pnpm $(pnpm --version 2>/dev/null || echo '?')"

# ───────────────────────────────────────────────────────────────────────
# Phase 4 — AI CLI selection
# ───────────────────────────────────────────────────────────────────────

log_step 3 "Install AI CLIs (at least one required)"

cat <<EOF
opendray spawns whichever CLI you select per-session. You can add more
later by rerunning the npm install commands by hand.

EOF

ask_yes_no "Install Claude Code (npm @anthropic-ai/claude-code)?" "y" WANT_CLAUDE
ask_yes_no "Install Gemini CLI (npm @google/gemini-cli)?"          "n" WANT_GEMINI
ask_yes_no "Install Codex CLI (npm @openai/codex)?"                "n" WANT_CODEX

INSTALLED_ANY=0
npm_install_global() {
    local pkg="$1" bin="$2"
    if have_cmd "$bin"; then
        log_ok "$bin already on PATH — skipping $pkg"
        INSTALLED_ANY=1
        return
    fi
    log_info "Installing $pkg ..."
    npm install -g --silent "$pkg" >/dev/null
    if have_cmd "$bin"; then
        log_ok "$bin installed: $($bin --version 2>/dev/null | head -1 || echo '?')"
        INSTALLED_ANY=1
    else
        log_warn "$pkg installed but '$bin' not on PATH — check 'npm bin -g' is in your shell init."
    fi
}

[ "$WANT_CLAUDE" = "y" ] && npm_install_global "@anthropic-ai/claude-code" claude
[ "$WANT_GEMINI" = "y" ] && npm_install_global "@google/gemini-cli"        gemini
[ "$WANT_CODEX"  = "y" ] && npm_install_global "@openai/codex"             codex

if [ "$INSTALLED_ANY" = "0" ] && ! have_cmd claude && ! have_cmd gemini && ! have_cmd codex; then
    log_warn "No AI CLI installed. opendray will run but session spawn will fail until you install one."
    ask_yes_no "Continue without an AI CLI?" "n" CONT_NO_CLI
    [ "$CONT_NO_CLI" = "y" ] || exit 0
fi

cat <<'EOF'

  Reminder: CLIs are installed but not yet logged in. Run their
  login commands interactively after this wizard:
    claude login         # browser OAuth
    gemini auth login
    codex login

EOF

# ───────────────────────────────────────────────────────────────────────
# Phase 5 — Postgres path
# ───────────────────────────────────────────────────────────────────────

log_step 4 "PostgreSQL"

ask_menu "Which Postgres should opendray talk to?" \
    "Use an existing PostgreSQL host (recommended)|Install PostgreSQL 16 + pgvector locally via brew" \
    PG_PATH_CHOICE

case "$PG_PATH_CHOICE" in
    Use*)     PG_MODE="existing" ;;
    Install*) PG_MODE="local"    ;;
    *) log_die "Unexpected choice: $PG_PATH_CHOICE" ;;
esac

if [ "$PG_MODE" = "local" ]; then
    log_info "Installing postgresql@16 + pgvector via brew..."
    brew install postgresql@16 pgvector
    brew services start postgresql@16

    # Brew links postgresql@16's binaries via keg-only by default — make psql available.
    if ! have_cmd psql; then
        export PATH="$BREW_PREFIX/opt/postgresql@16/bin:$PATH"
    fi

    # Give PG a moment to come up.
    sleep 2

    PG_SUPER_HOST="127.0.0.1"
    PG_SUPER_PORT="5432"
    PG_SUPER_USER="$USER"       # brew PG creates the superuser as $USER
    PG_SUPER_DB="postgres"
    PG_SUPER_PW=""              # brew PG default: no password, trust auth via socket
    log_ok "Local Postgres up; superuser = $USER (trust auth on socket)"
else
    cat <<'EOF'

We'll connect to your existing PG host as a superuser only to create the
opendray database, user, and pgvector extension. After bootstrap,
opendray reconnects with its own least-privilege user.

EOF
    require_cmds psql || log_die "psql is not installed — 'brew install libpq && brew link --force libpq' to get it."
    ask_with_default "PG host"           "localhost" PG_SUPER_HOST
    ask_with_default "PG port"           "5432"      PG_SUPER_PORT
    ask_with_default "Superuser name"    "postgres"  PG_SUPER_USER
    ask_with_default "Maintenance DB"    "postgres"  PG_SUPER_DB
    ask_password    "Superuser password"             PG_SUPER_PW

    log_info "Testing superuser connection..."
    if ! PGPASSWORD="$PG_SUPER_PW" psql \
            -h "$PG_SUPER_HOST" -p "$PG_SUPER_PORT" -U "$PG_SUPER_USER" -d "$PG_SUPER_DB" \
            -tAc "SELECT 1" >/dev/null 2>&1; then
        log_die "Cannot connect as $PG_SUPER_USER@$PG_SUPER_HOST:$PG_SUPER_PORT"
    fi
    log_ok "Superuser connection OK"

    PGPASSWORD="$PG_SUPER_PW" psql -h "$PG_SUPER_HOST" -p "$PG_SUPER_PORT" \
        -U "$PG_SUPER_USER" -d "$PG_SUPER_DB" \
        -tAc "SELECT 1 FROM pg_available_extensions WHERE name='vector'" 2>/dev/null \
        | grep -q 1 \
        || log_die "pgvector extension is not available on this PG server. Install it (brew install pgvector, or build from source) and rerun."
    log_ok "pgvector available on server"
fi

# ───────────────────────────────────────────────────────────────────────
# Phase 6 — Bootstrap opendray DB
# ───────────────────────────────────────────────────────────────────────

log_step 5 "Bootstrap opendray database"

ask_with_default "Database name"                   "opendray"      OD_DB_NAME
ask_with_default "App DB user (CRUD only)"         "opendray_user" OD_DB_USER

DEFAULT_DB_PW="$(gen_password 24)"
ask_with_default "App DB password (Enter = random)" "$DEFAULT_DB_PW" OD_DB_PW

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

log_info "Creating database '$OD_DB_NAME' (skip if exists)..."
if ! run_psql_super "SELECT 1 FROM pg_database WHERE datname = '$OD_DB_NAME'" | grep -q 1; then
    run_psql_super "CREATE DATABASE \"$OD_DB_NAME\""
fi
log_ok "Database ready"

log_info "Creating role '$OD_DB_USER'..."
if run_psql_super "SELECT 1 FROM pg_roles WHERE rolname = '$OD_DB_USER'" | grep -q 1; then
    run_psql_super "ALTER USER \"$OD_DB_USER\" WITH ENCRYPTED PASSWORD '${OD_DB_PW//\'/\'\'}'"
else
    run_psql_super "CREATE USER \"$OD_DB_USER\" WITH ENCRYPTED PASSWORD '${OD_DB_PW//\'/\'\'}'"
fi
log_ok "Role ready"

run_psql_super "GRANT ALL PRIVILEGES ON DATABASE \"$OD_DB_NAME\" TO \"$OD_DB_USER\""
run_psql_super "CREATE EXTENSION IF NOT EXISTS vector" "$OD_DB_NAME"
run_psql_super "GRANT ALL ON SCHEMA public TO \"$OD_DB_USER\""    "$OD_DB_NAME"
log_ok "pgvector enabled + privileges granted"

APP_PG_HOST="$PG_SUPER_HOST"
APP_PG_PORT="$PG_SUPER_PORT"

log_info "Verifying app-user connection..."
if ! test_pg_dsn "$OD_DB_USER" "$OD_DB_PW" "$APP_PG_HOST" "$APP_PG_PORT" "$OD_DB_NAME"; then
    log_die "App user cannot connect. Check pg_hba.conf for a host entry covering $APP_PG_HOST."
fi
log_ok "App user can connect"

# ───────────────────────────────────────────────────────────────────────
# Phase 7 — Admin password + listen
# ───────────────────────────────────────────────────────────────────────

log_step 6 "opendray admin credentials + network"

cat <<'EOF'
Initial admin password (you'll be forced to rotate it after first UI
login — opendray writes a bcrypt keyfile, then the plaintext below
becomes inert).

EOF
DEFAULT_ADMIN_PW="$(gen_password 20)"
ask_with_default "Initial admin password (Enter = random)" "$DEFAULT_ADMIN_PW" OD_ADMIN_PW

cat <<'EOF'

Listen address:
  127.0.0.1:8770   loopback (safe default; reach via SSH tunnel or reverse proxy)
  0.0.0.0:8770     all interfaces (LAN reachable — use only on trusted networks)

EOF
ask_with_default "Listen address" "127.0.0.1:8770" OD_LISTEN

# ───────────────────────────────────────────────────────────────────────
# Phase 8 — Binary install
# ───────────────────────────────────────────────────────────────────────

log_step 7 "Install opendray binary"

OPENDRAY_BIN="$OPENDRAY_HOME/bin/opendray"
mkdir -p "$OPENDRAY_HOME/bin" "$OPENDRAY_HOME/logs" "$OPENDRAY_HOME/data"

if [ "$FROM_SOURCE" = "1" ]; then
    [ -d "$SCRIPT_DIR/../cmd/opendray" ] || log_die "--from-source given but no cmd/opendray dir at $SCRIPT_DIR/.."
    have_cmd go || log_die "go toolchain required for --from-source. Install: brew install go"
    log_info "Building from source..."
    ( cd "$SCRIPT_DIR/.." && go build -trimpath -ldflags="-s -w" -o "$OPENDRAY_BIN" ./cmd/opendray )
else
    log_info "Fetching latest release tag..."
    LATEST_TAG="$(curl -fsSL "https://api.github.com/repos/$OPENDRAY_REPO/releases/latest" \
        | grep -oE '"tag_name":\s*"[^"]+"' | head -1 \
        | sed 's/.*"\([^"]*\)"$/\1/')"
    [ -n "$LATEST_TAG" ] || log_die "Could not resolve latest release tag"
    log_ok "Latest release: $LATEST_TAG"

    VERSION="${LATEST_TAG#v}"
    TARBALL_NAME="opendray_${VERSION}_darwin_${ARCH}.tar.gz"
    TARBALL_URL="https://github.com/$OPENDRAY_REPO/releases/download/$LATEST_TAG/$TARBALL_NAME"

    TMP_TARBALL="$(mktemp -t opendray.XXXXXX.tar.gz)"
    register_cleanup_file "$TMP_TARBALL"
    TMP_EXTRACT="$(mktemp -d -t opendray-extract.XXXXXX)"

    log_info "Downloading $TARBALL_NAME..."
    curl -fsSL --retry 3 -o "$TMP_TARBALL" "$TARBALL_URL" \
        || log_die "Download failed: $TARBALL_URL"

    SUMS_URL="https://github.com/$OPENDRAY_REPO/releases/download/$LATEST_TAG/SHA256SUMS"
    TMP_SUMS="$(mktemp -t opendray-sums.XXXXXX)"
    register_cleanup_file "$TMP_SUMS"
    if curl -fsSL --retry 2 -o "$TMP_SUMS" "$SUMS_URL" 2>/dev/null; then
        if grep -q " $TARBALL_NAME$" "$TMP_SUMS"; then
            EXPECTED="$(grep " $TARBALL_NAME$" "$TMP_SUMS" | awk '{print $1}')"
            ACTUAL="$(shasum -a 256 "$TMP_TARBALL" | awk '{print $1}')"
            [ "$EXPECTED" = "$ACTUAL" ] \
                || log_die "Checksum mismatch! Expected $EXPECTED got $ACTUAL"
            log_ok "SHA-256 verified"
        else
            log_warn "$TARBALL_NAME not listed in SHA256SUMS — skipping checksum check"
        fi
    fi

    tar -xzf "$TMP_TARBALL" -C "$TMP_EXTRACT"
    BINARY_FOUND="$(find "$TMP_EXTRACT" -type f -name opendray -perm -u+x | head -1)"
    [ -n "$BINARY_FOUND" ] || log_die "No opendray binary in $TARBALL_NAME"
    install -m 0755 "$BINARY_FOUND" "$OPENDRAY_BIN"
    rm -rf "$TMP_EXTRACT"
fi

log_ok "Installed: $OPENDRAY_BIN"
"$OPENDRAY_BIN" version

# ───────────────────────────────────────────────────────────────────────
# Phase 9 — Config + migrate
# ───────────────────────────────────────────────────────────────────────

log_step 8 "Write config + apply migrations"

OD_CONFIG_PATH="$OPENDRAY_HOME/config.toml"
OD_DSN="$(build_dsn "$OD_DB_USER" "$OD_DB_PW" "$APP_PG_HOST" "$APP_PG_PORT" "$OD_DB_NAME")"

cat > "$OD_CONFIG_PATH" <<EOF
# opendray gateway configuration — generated by install-macos.sh on $(date -u +%Y-%m-%dT%H:%M:%SZ)

listen = "$OD_LISTEN"

[database]
url = "$OD_DSN"

[admin]
# Initial bootstrap password. Rotate via the UI after first login.
password = "$OD_ADMIN_PW"

[log]
level = "info"
format = "json"

[runtime]
data_dir = "$OPENDRAY_HOME/data"
EOF

chmod 0600 "$OD_CONFIG_PATH"
log_ok "Config: $OD_CONFIG_PATH (mode 0600)"

log_info "Applying migrations..."
"$OPENDRAY_BIN" migrate -config "$OD_CONFIG_PATH"
log_ok "Migrations applied"

# ───────────────────────────────────────────────────────────────────────
# Phase 10 — launchd registration
# ───────────────────────────────────────────────────────────────────────

if [ "$SKIP_SERVICE" = "1" ]; then
    log_section "Manual start command"
    cat <<EOF
  $OPENDRAY_BIN serve -config $OD_CONFIG_PATH

EOF
    exit 0
fi

log_step 9 "Install launchd unit"

LABEL="com.opendray.opendray"

if [ "$LAUNCHD_SCOPE" = "daemon" ]; then
    PLIST_PATH="/Library/LaunchDaemons/${LABEL}.plist"
    DOMAIN="system"
    USER_KV="<key>UserName</key><string>$USER</string>"
else
    mkdir -p "$HOME/Library/LaunchAgents"
    PLIST_PATH="$HOME/Library/LaunchAgents/${LABEL}.plist"
    DOMAIN="gui/$(id -u)"
    USER_KV=""
fi

# Render plist via a tmp file (Daemon path writes via run_priv).
TMP_PLIST="$(mktemp -t opendray-plist.XXXXXX)"
register_cleanup_file "$TMP_PLIST"

cat > "$TMP_PLIST" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${LABEL}</string>
    ${USER_KV}
    <key>ProgramArguments</key>
    <array>
        <string>${OPENDRAY_BIN}</string>
        <string>serve</string>
        <string>-config</string>
        <string>${OD_CONFIG_PATH}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>
    <key>StandardOutPath</key>
    <string>${OPENDRAY_HOME}/logs/opendray.log</string>
    <key>StandardErrorPath</key>
    <string>${OPENDRAY_HOME}/logs/opendray.err</string>
    <key>WorkingDirectory</key>
    <string>${OPENDRAY_HOME}</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>${BREW_PREFIX}/bin:/usr/local/bin:/usr/bin:/bin</string>
        <key>HOME</key>
        <string>${HOME}</string>
    </dict>
</dict>
</plist>
EOF

if [ "$LAUNCHD_SCOPE" = "daemon" ]; then
    run_priv install -m 0644 -o root -g wheel "$TMP_PLIST" "$PLIST_PATH"
    run_priv launchctl bootout "$DOMAIN/$LABEL" 2>/dev/null || true
    run_priv launchctl bootstrap "$DOMAIN" "$PLIST_PATH"
    run_priv launchctl enable    "$DOMAIN/$LABEL"
    run_priv launchctl kickstart -k "$DOMAIN/$LABEL"
else
    install -m 0644 "$TMP_PLIST" "$PLIST_PATH"
    launchctl bootout "$DOMAIN/$LABEL" 2>/dev/null || true
    launchctl bootstrap "$DOMAIN" "$PLIST_PATH"
    launchctl enable    "$DOMAIN/$LABEL"
    launchctl kickstart -k "$DOMAIN/$LABEL"
fi
log_ok "launchd unit loaded: $PLIST_PATH"

# Health check
log_info "Waiting for the gateway..."
HEALTH_URL="http://${OD_LISTEN}/api/v1/health"
[[ "$OD_LISTEN" == 0.0.0.0:* ]] && HEALTH_URL="http://127.0.0.1:${OD_LISTEN##*:}/api/v1/health"

for i in $(seq 1 20); do
    if curl -fsS "$HEALTH_URL" 2>/dev/null | grep -q '"status":"ok"'; then
        log_ok "Health endpoint OK"
        break
    fi
    sleep 1
    if [ "$i" = 20 ]; then
        log_err "Health check did not succeed within 20s."
        log_dim "Tail $OPENDRAY_HOME/logs/opendray.err for details:"
        tail -30 "$OPENDRAY_HOME/logs/opendray.err" 2>/dev/null || true
        exit 1
    fi
done

# ───────────────────────────────────────────────────────────────────────
# Summary
# ───────────────────────────────────────────────────────────────────────

log_section "Install complete"

WEB_URL="http://${OD_LISTEN}/admin/"
[[ "$OD_LISTEN" == 0.0.0.0:* ]] && WEB_URL="http://<this-Mac>:${OD_LISTEN##*:}/admin/"

cat <<EOF
  ${C_BLU}Admin UI${C_NC}        ${WEB_URL}
  ${C_BLU}Login as${C_NC}        admin
  ${C_BLU}Password${C_NC}        ${OD_ADMIN_PW}   ${C_YEL}← rotate via Settings → Admin on first login${C_NC}

  ${C_BLU}Config${C_NC}          ${OD_CONFIG_PATH}
  ${C_BLU}Logs${C_NC}            ${OPENDRAY_HOME}/logs/opendray.{log,err}
  ${C_BLU}Service${C_NC}         launchctl kickstart -k ${DOMAIN}/${LABEL}     # restart
                  launchctl bootout      ${DOMAIN}/${LABEL}     # stop+unload

  ${C_BLU}Database${C_NC}        ${OD_DB_USER}@${APP_PG_HOST}:${APP_PG_PORT}/${OD_DB_NAME}
  ${C_BLU}DB password${C_NC}     ${OD_DB_PW}   ${C_YEL}← save this somewhere safe${C_NC}

  Next:
    1. Open the admin UI, log in, rotate the admin password.
    2. Run 'claude login' / 'gemini auth login' / 'codex login' to finish CLI auth.
    3. Providers → register the CLI binary path (e.g. \$(which claude)).
    4. Sessions → New session → spawn your first session.

  See docs/getting-started.md for the post-install walkthrough.
EOF
