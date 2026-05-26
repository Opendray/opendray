#!/usr/bin/env bash
# scripts/install-macos.sh
# Interactive installation wizard for opendray on macOS (Intel + Apple Silicon).
#
# Defaults to a user-level LaunchAgent (runs when the user logs in) — best fit
# for a personal Mac or a Mac mini home server. Re-run with --launchd-daemon to
# install a system-wide LaunchDaemon instead.

set -euo pipefail

# Reattach stdin to the controlling terminal so prompts work even when
# we arrived here via `curl … | bash` (the inherited stdin is the curl
# pipe at EOF, which makes every `read` fail immediately).
if [ ! -t 0 ] && [ -r /dev/tty ]; then
    exec </dev/tty
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "$SCRIPT_DIR/lib/common.sh"

# ── Defaults ─────────────────────────────────────────────────────────
: "${OPENDRAY_REPO:=Opendray/opendray}"
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
  2) Install Node.js via brew                     (for AI CLIs)
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
# Phase 3 — Node.js (for the AI CLIs)
# ───────────────────────────────────────────────────────────────────────
#
# pnpm is intentionally NOT installed here. Default path needs only Node
# (for `npm install -g` of the AI CLIs) — pnpm is `--from-source`-only.

log_step 2 "Install Node.js"

if ! have_cmd node || [[ "$(node --version 2>/dev/null | sed 's/v//;s/\..*//')" -lt 20 ]]; then
    log_info "Installing node@22 via brew..."
    brew install node@22
    brew link --overwrite --force node@22 2>/dev/null || true
fi
log_ok "Node $(node --version)"

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
    log_info "Installing $pkg (~30–90 s — npm registry download)..."
    # No --silent / no /dev/null: 50–100 MB packages with a silent install look
    # indistinguishable from a hang. Let npm's progress bar through.
    npm install -g "$pkg"
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
    # pgvector's Homebrew bottle only ships the extension for specific
    # PG majors (currently 17 + 18 — NOT 16). Install pgvector first,
    # then provision the newest PG major it actually supports. Never
    # hardcode a version pgvector may not cover, or CREATE EXTENSION
    # vector fails after the DB is already bootstrapped.
    log_info "Installing pgvector via brew..."
    brew install pgvector

    # Collect the PG majors pgvector actually ships an extension for.
    _supported=""
    for _ctl in "$BREW_PREFIX"/opt/pgvector/share/postgresql@*/extension/vector.control; do
        [ -e "$_ctl" ] || continue
        _supported="$_supported $(printf '%s' "$_ctl" | sed -E 's#.*/postgresql@([0-9]+)/.*#\1#')"
    done
    [ -n "$_supported" ] || log_die "pgvector installed but no supported PostgreSQL major detected (check 'brew list pgvector')."

    # Prefer a supported version that's ALREADY installed — avoids adding
    # yet another Postgres on machines that already run one. Fall back to
    # the newest supported version on a clean machine.
    PG_VER=""
    for _v in $(printf '%s\n' $_supported | sort -rn | uniq); do
        if [ -d "$BREW_PREFIX/opt/postgresql@$_v" ]; then PG_VER="$_v"; break; fi
    done
    if [ -n "$PG_VER" ]; then
        log_info "Reusing already-installed, pgvector-supported PostgreSQL $PG_VER"
    else
        PG_VER="$(printf '%s\n' $_supported | sort -rn | uniq | head -1 || true)"
        log_info "pgvector supports PostgreSQL $PG_VER — installing postgresql@$PG_VER"
    fi
    PG_FORMULA="postgresql@$PG_VER"
    brew install "$PG_FORMULA"   # no-op if already present

    # Always reference THIS PG's bins explicitly + put them first on PATH.
    # Another linked postgresql@NN (common on multi-version Macs) otherwise
    # shadows psql / pg_isready / pg_config and you talk to the wrong server.
    PG_BIN="$BREW_PREFIX/opt/$PG_FORMULA/bin"
    export PATH="$PG_BIN:$PATH"

    # Port selection. Use the port THIS instance is actually configured
    # for — postgresql.conf carries the 5432 default, or a non-default
    # port a previous installer run wrote. A `brew services restart`
    # picks up whatever the conf says regardless of what's free, so we
    # must probe the same port we'll actually bind (otherwise the
    # readiness check below waits on the wrong port and times out).
    _pgconf="$BREW_PREFIX/var/$PG_FORMULA/postgresql.conf"
    # A fresh postgresql.conf ships the port line COMMENTED ("#port = 5432"),
    # so this grep finds no match and exits 1. Under `set -euo pipefail`
    # that aborts the whole installer right after the Postgres install —
    # before the `${PG_SUPER_PORT:-5432}` fallback below can apply. The
    # `|| true` keeps the no-match case from tripping errexit (same guard
    # the lsof probe just below already uses).
    PG_SUPER_PORT="$(grep -E '^[[:space:]]*port[[:space:]]*=' "$_pgconf" 2>/dev/null | tail -1 | sed -E 's/^[^=]*=[[:space:]]*([0-9]+).*/\1/' || true)"
    PG_SUPER_PORT="${PG_SUPER_PORT:-5432}"

    # A conflict only if that port is held by a process that ISN'T this
    # instance's own server (matched by its data dir on the command line).
    # `|| true`: lsof exits non-zero on a free port (the common fresh
    # case) and set -o pipefail + set -e would otherwise abort here.
    _busy_pid="$(lsof -nP -iTCP:"$PG_SUPER_PORT" -sTCP:LISTEN -t 2>/dev/null | head -1 || true)"
    if [ -n "$_busy_pid" ] && ! ps -p "$_busy_pid" -o command= 2>/dev/null | grep -qF "$BREW_PREFIX/var/$PG_FORMULA"; then
        log_warn "Port $PG_SUPER_PORT is in use by PID $_busy_pid ($(ps -p "$_busy_pid" -o comm= 2>/dev/null | tail -1))."
        _alt=$((PG_SUPER_PORT + 1))
        while lsof -nP -iTCP:"$_alt" -sTCP:LISTEN -t >/dev/null 2>&1; do _alt=$((_alt + 1)); done
        ask_with_default "Port for opendray's PostgreSQL ($PG_FORMULA)" "$_alt" PG_SUPER_PORT
        if ! grep -qE "^[[:space:]]*port[[:space:]]*=[[:space:]]*$PG_SUPER_PORT([^0-9]|$)" "$_pgconf" 2>/dev/null; then
            printf '\n# set by opendray installer (port conflict)\nport = %s\n' "$PG_SUPER_PORT" >> "$_pgconf"
        fi
    fi

    # Idempotent service start — reuse if healthy, recover from an error
    # state via bootout, else start. Avoids `launchctl bootstrap exited 5`
    # when the service is already loaded.
    case "$(brew services list | awk -v f="$PG_FORMULA" '$1==f {print $2}')" in
        started) log_info "$PG_FORMULA already loaded — restarting to apply config"; brew services restart "$PG_FORMULA" ;;
        error)   launchctl bootout "gui/$(id -u)/homebrew.mxcl.$PG_FORMULA" 2>/dev/null || true; brew services restart "$PG_FORMULA" ;;
        *)       brew services start "$PG_FORMULA" ;;
    esac

    # Readiness probe instead of a blind `sleep 2`.
    log_info "Waiting for PostgreSQL on 127.0.0.1:$PG_SUPER_PORT ..."
    _tries=30
    while [ "$_tries" -gt 0 ]; do
        "$PG_BIN/pg_isready" -h 127.0.0.1 -p "$PG_SUPER_PORT" >/dev/null 2>&1 && break
        sleep 1; _tries=$((_tries - 1))
    done
    "$PG_BIN/pg_isready" -h 127.0.0.1 -p "$PG_SUPER_PORT" >/dev/null 2>&1 \
        || log_die "PostgreSQL did not become ready on :$PG_SUPER_PORT — check: tail $BREW_PREFIX/var/log/$PG_FORMULA.log"

    PG_SUPER_HOST="127.0.0.1"
    PG_SUPER_USER="$USER"       # brew PG creates the superuser as $USER
    PG_SUPER_DB="postgres"
    PG_SUPER_PW=""              # brew PG default: no password, trust auth on loopback

    # Validate pgvector is actually visible to THIS server before relying
    # on CREATE EXTENSION downstream — fail loud and early, not mid-bootstrap.
    if ! "$PG_BIN/psql" -h 127.0.0.1 -p "$PG_SUPER_PORT" -U "$PG_SUPER_USER" -d postgres \
            -tAc "SELECT 1 FROM pg_available_extensions WHERE name='vector'" 2>/dev/null | grep -q 1; then
        log_die "pgvector not visible to $PG_FORMULA despite install — aborting before bootstrap."
    fi
    log_ok "Local PostgreSQL $PG_VER ready on :$PG_SUPER_PORT (superuser=$USER, trust auth) + pgvector"
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

ask_pg_identifier "Database name"                   "opendray"      OD_DB_NAME
ask_pg_identifier "App DB user (CRUD only)"         "opendray_user" OD_DB_USER

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
Pick the initial admin login. You'll be forced to rotate the password
after first UI login (opendray writes a bcrypt keyfile, after which
the plaintext below becomes inert — `user` stays authoritative).

EOF
ask_with_default "Admin username" "admin" OD_ADMIN_USER

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

    if ! have_cmd pnpm; then
        log_info "Installing pnpm globally for the web build (~30 s — npm registry)..."
        npm install -g pnpm@latest
    fi
    have_cmd pnpm || log_die "pnpm is required for --from-source builds (the React SPA goes into the Go binary via go:embed). Install pnpm and rerun."

    log_info "Building web bundle (pnpm install + build)..."
    ( cd "$SCRIPT_DIR/../app/web" && pnpm install --frozen-lockfile && pnpm build )

    log_info "Building Go binary..."
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

# ── Put `opendray` on the interactive PATH ───────────────────────────
# The binary lives under $OPENDRAY_HOME/bin (default ~/.opendray/bin),
# which is NOT on the default macOS PATH — the launchd service runs via
# the absolute path, but an interactive `opendray …` wouldn't resolve.
# Link it into Homebrew's bin: it's already on PATH for any working brew
# setup and user-writable (no sudo), and resolves on both Intel
# (/usr/local/bin) and Apple Silicon (/opt/homebrew/bin).
OD_PATH_NOTE=""
if [ "$(command -v opendray 2>/dev/null || true)" = "$OPENDRAY_BIN" ]; then
    log_ok "opendray already on PATH"
else
    OD_LINK_DIR="$BREW_PREFIX/bin"
    if mkdir -p "$OD_LINK_DIR" 2>/dev/null && [ -w "$OD_LINK_DIR" ]; then
        ln -sf "$OPENDRAY_BIN" "$OD_LINK_DIR/opendray"
        log_ok "Linked opendray → $OD_LINK_DIR/opendray (on PATH)"
    else
        OD_PATH_NOTE="$OPENDRAY_HOME/bin"
        log_warn "Could not link into $OD_LINK_DIR — add opendray to PATH yourself:"
        log_warn "    echo 'export PATH=\"$OD_PATH_NOTE:\$PATH\"' >> ~/.zshrc && source ~/.zshrc"
    fi
fi

# ───────────────────────────────────────────────────────────────────────
# Phase 9 — Config + migrate
# ───────────────────────────────────────────────────────────────────────

log_step 8 "Write config + apply migrations"

OD_CONFIG_PATH="$OPENDRAY_HOME/config.toml"
OD_ENV_PATH="$OPENDRAY_HOME/opendray.env"
OD_LAUNCHER="$OPENDRAY_HOME/bin/opendray-launcher.sh"
OD_DSN="$(build_dsn "$OD_DB_USER" "$OD_DB_PW" "$APP_PG_HOST" "$APP_PG_PORT" "$OD_DB_NAME")"

# ── 1. opendray.env — SECRETS LIVE HERE ──────────────────────────────
# 0600 user-only. The launcher script (next step) sources it before
# exec'ing opendray; opendray reads via env-var overrides.
cat > "$OD_ENV_PATH" <<EOF
# opendray secrets — generated by install-macos.sh on $(date -u +%Y-%m-%dT%H:%M:%SZ)
#
# Keep this file 0600. opendray reads these via env-var overrides
# (see internal/config/config.go) and they take precedence over
# anything in config.toml.
OPENDRAY_DATABASE_URL=$OD_DSN
OPENDRAY_ADMIN_PASSWORD=$OD_ADMIN_PW
EOF
chmod 0600 "$OD_ENV_PATH"

# ── 2. config.toml — NON-SECRET configuration ────────────────────────
cat > "$OD_CONFIG_PATH" <<EOF
# opendray gateway configuration — generated by install-macos.sh on $(date -u +%Y-%m-%dT%H:%M:%SZ)
#
# This file is intentionally non-secret. Database URL and admin
# password are loaded from $OD_ENV_PATH via the launcher script
# (~/.opendray/bin/opendray-launcher.sh) at service start.

listen = "$OD_LISTEN"

[database]
# url loaded from OPENDRAY_DATABASE_URL (see opendray.env)

[admin]
# user is fine to keep here — it's only an identifier.
# password loaded from OPENDRAY_ADMIN_PASSWORD (see opendray.env).
user = "$OD_ADMIN_USER"

[log]
level = "info"
format = "json"

[runtime]
data_dir = "$OPENDRAY_HOME/data"
EOF
chmod 0644 "$OD_CONFIG_PATH"

# ── 3. Launcher script — sources env then execs opendray ─────────────
# launchd's plist EnvironmentVariables block would put secrets directly
# in the plist XML (effectively another plaintext copy). A wrapper
# script that sources the env file lets us keep secrets in one place
# with strict permissions.
cat > "$OD_LAUNCHER" <<EOF
#!/bin/sh
# opendray launchd wrapper — sources $OD_ENV_PATH then execs opendray.
# Don't edit this file by hand; rerun the installer to regenerate.
set -e
set -a
. "$OD_ENV_PATH"
set +a
exec "$OPENDRAY_BIN" serve -config "$OD_CONFIG_PATH"
EOF
chmod 0700 "$OD_LAUNCHER"

log_ok "config.toml: $OD_CONFIG_PATH (0644, no secrets)"
log_ok "opendray.env: $OD_ENV_PATH (0600 — secrets here)"
log_ok "launcher:     $OD_LAUNCHER (0700)"

log_info "Applying migrations..."
# Wizard's shell env (set via `OPENDRAY_… = …` prefix) is picked up by
# opendray's env-var override layer.
OPENDRAY_DATABASE_URL="$OD_DSN" \
OPENDRAY_ADMIN_PASSWORD="$OD_ADMIN_PW" \
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

# Build the service PATH. launchd does NOT read shell rc files, so the
# daemon only sees this PATH — if an AI CLI lives somewhere else (e.g.
# Claude Code's native installer puts `claude` in ~/.local/bin, not the
# brew bin), opendray can't spawn it and sessions fail with the CLI "not
# found". Seed the standard dirs, then prepend wherever each installed
# CLI actually resolves right now, plus the native-installer location.
SVC_PATH="${BREW_PREFIX}/bin:/usr/local/bin:/usr/bin:/bin"
for _cli in claude gemini codex; do
    _clipath="$(command -v "$_cli" 2>/dev/null || true)"
    [ -n "$_clipath" ] || continue
    _clidir="$(cd "$(dirname "$_clipath")" 2>/dev/null && pwd || true)"
    [ -n "$_clidir" ] || continue
    case ":$SVC_PATH:" in *":$_clidir:"*) ;; *) SVC_PATH="$_clidir:$SVC_PATH" ;; esac
done
case ":$SVC_PATH:" in *":$HOME/.local/bin:"*) ;; *) SVC_PATH="$HOME/.local/bin:$SVC_PATH" ;; esac

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
        <string>${OD_LAUNCHER}</string>
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
        <string>${SVC_PATH}</string>
        <key>HOME</key>
        <string>${HOME}</string>
    </dict>
</dict>
</plist>
EOF

# (Re)load a launchd unit, tolerant of an already-loaded label.
# `launchctl bootout` is asynchronous: a naive `bootout || true; bootstrap`
# races on a re-install — the old instance is still draining when bootstrap
# runs, which fails with "Bootstrap failed: 5: Input/output error". Wait for
# the old instance to actually disappear, then bootstrap; if it still won't
# load (already-loaded / transient EIO), bootout + retry once. A genuinely
# broken load is caught by the health check that follows.
reload_launchd_unit() {
    local domain="$1" label="$2" plist="$3" priv="${4:-}"
    local i
    $priv launchctl bootout "$domain/$label" 2>/dev/null || true
    for i in 1 2 3 4 5 6 7 8 9 10; do
        $priv launchctl print "$domain/$label" >/dev/null 2>&1 || break
        sleep 1
    done
    if ! $priv launchctl bootstrap "$domain" "$plist" 2>/dev/null; then
        $priv launchctl bootout "$domain/$label" 2>/dev/null || true
        sleep 2
        $priv launchctl bootstrap "$domain" "$plist" 2>/dev/null || true
    fi
    $priv launchctl enable    "$domain/$label" 2>/dev/null || true
    $priv launchctl kickstart -k "$domain/$label" 2>/dev/null || true
}

if [ "$LAUNCHD_SCOPE" = "daemon" ]; then
    run_priv install -m 0644 -o root -g wheel "$TMP_PLIST" "$PLIST_PATH"
    reload_launchd_unit "$DOMAIN" "$LABEL" "$PLIST_PATH" run_priv
else
    install -m 0644 "$TMP_PLIST" "$PLIST_PATH"
    reload_launchd_unit "$DOMAIN" "$LABEL" "$PLIST_PATH"
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

# Build a clickable URL. For 0.0.0.0 listens, resolve the Mac's primary
# IP via `ipconfig getifaddr` (en0 = Wi-Fi / wired on most setups; en1 = fallback).
PORT="${OD_LISTEN##*:}"
HOST_PART="${OD_LISTEN%:*}"
case "$HOST_PART" in
    0.0.0.0|"")
        LAN_IP=""
        for iface in en0 en1 en2 en3; do
            LAN_IP="$(ipconfig getifaddr "$iface" 2>/dev/null)"
            [ -n "$LAN_IP" ] && break
        done
        [ -n "$LAN_IP" ] || LAN_IP="<this-Mac-ip>"
        WEB_URL="http://${LAN_IP}:${PORT}/admin/"
        WEB_URL_NOTE="  (LAN — reachable from any device on the same network)"
        ;;
    *)
        WEB_URL="http://${OD_LISTEN}/admin/"
        WEB_URL_NOTE=""
        ;;
esac

if [ -n "$OD_PATH_NOTE" ]; then
    OD_CLI_LINE="opendray   ${C_YEL}← add ${OD_PATH_NOTE} to PATH first (see above)${C_NC}"
else
    OD_CLI_LINE="opendray version"
fi

cat <<EOF
  ${C_BLU}Admin UI${C_NC}        ${WEB_URL}${WEB_URL_NOTE}
  ${C_BLU}Username${C_NC}        ${OD_ADMIN_USER}
  ${C_BLU}Password${C_NC}        ${OD_ADMIN_PW}   ${C_YEL}← rotate via Settings → Admin on first login${C_NC}

  ${C_BLU}CLI${C_NC}             ${OD_CLI_LINE}
  ${C_BLU}Config${C_NC}          ${OD_CONFIG_PATH}
  ${C_BLU}Logs${C_NC}            ${OPENDRAY_HOME}/logs/opendray.{log,err}
  ${C_BLU}Service${C_NC}         launchctl kickstart -k ${DOMAIN}/${LABEL}     # restart
                  launchctl bootout      ${DOMAIN}/${LABEL}     # stop+unload

  ${C_BLU}Database${C_NC}        ${OD_DB_USER}@${APP_PG_HOST}:${APP_PG_PORT}/${OD_DB_NAME}
  ${C_BLU}DB password${C_NC}     ${OD_DB_PW}   ${C_YEL}← save this somewhere safe${C_NC}

  Next:
    1. Open the admin UI (link above is clickable in most terminals),
       log in as ${OD_ADMIN_USER}, rotate the admin password.
    2. Run 'claude login' / 'gemini auth login' / 'codex login' to finish CLI auth.
    3. Providers → register the CLI binary path (e.g. \$(which claude)).
    4. Sessions → New session → spawn your first session.

  See docs/getting-started.md for the post-install walkthrough.
EOF
