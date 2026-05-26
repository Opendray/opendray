#!/usr/bin/env bash
# scripts/install-linux.sh
# Interactive installation wizard for opendray on Ubuntu / Debian Linux.
#
# Phases:
#   0. Sanity: distro + arch + sudo
#   1. Plan summary + confirmation
#   2. Base tools (curl, ca-certificates, build essentials, postgresql-client)
#   3. Node.js (for AI CLIs); pnpm only when --from-source
#   4. AI CLI selection + install (Claude / Codex / Gemini)
#   5. PostgreSQL path: use existing OR install locally
#   6. Bootstrap opendray DB / user / pgvector
#   7. opendray credentials + listen address
#   8. Download release tarball (or build from source if --from-source)
#   9. Generate config.toml + run migrations
#  10. Install systemd service + health check
#
# Designed to be re-runnable: most steps detect "already done" and skip.

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

# ── Defaults (can be overridden by env or prompts) ───────────────────
: "${OPENDRAY_REPO:=Opendray/opendray}"
: "${OPENDRAY_PREFIX:=/usr/local}"             # binary install prefix
: "${OPENDRAY_CONFIG_DIR:=/etc/opendray}"
: "${OPENDRAY_DATA_DIR:=/var/lib/opendray}"
: "${OPENDRAY_LOG_DIR:=/var/log/opendray}"
: "${OPENDRAY_SERVICE_USER:=opendray}"
: "${OPENDRAY_SERVICE_NAME:=opendray}"

FROM_SOURCE=0
SKIP_SERVICE=0

for arg in "$@"; do
    case "$arg" in
        --from-source)  FROM_SOURCE=1 ;;
        --skip-service) SKIP_SERVICE=1 ;;
        -h|--help)
            cat <<'EOF'
opendray Linux installer wizard

Usage:
  bash scripts/install-linux.sh [options]

Options:
  --from-source     Build opendray from this checkout instead of downloading a release.
  --skip-service    Install the binary + config but skip systemd unit registration.
  -h, --help        Show this help.
EOF
            exit 0 ;;
        *) log_warn "Unknown option: $arg (ignored)" ;;
    esac
done

# ───────────────────────────────────────────────────────────────────────
# Phase 0 — Sanity
# ───────────────────────────────────────────────────────────────────────

log_section "opendray installer — Linux"

if [ ! -f /etc/os-release ]; then
    log_die "/etc/os-release missing — can't identify this distribution."
fi
# shellcheck disable=SC1091
. /etc/os-release

case "${ID:-}${ID_LIKE:+ $ID_LIKE}" in
    *ubuntu*|*debian*)
        log_ok "Distribution: ${PRETTY_NAME:-$ID}"
        ;;
    *)
        log_warn "This wizard is tuned for Ubuntu / Debian. Detected: ${PRETTY_NAME:-$ID}"
        ask_yes_no "Continue anyway? Package commands may need manual tweaks." "n" CONTINUE_NONDEB
        [ "$CONTINUE_NONDEB" = "y" ] || exit 0
        ;;
esac

ARCH_RAW="$(uname -m)"
case "$ARCH_RAW" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) log_die "Unsupported architecture: $ARCH_RAW (need amd64 or arm64)" ;;
esac
log_ok "Architecture: $ARCH"

require_root_or_sudo

# ───────────────────────────────────────────────────────────────────────
# Phase 1 — Show plan, confirm
# ───────────────────────────────────────────────────────────────────────

log_section "What this wizard will do"

cat <<EOF
This installer walks through 6 interactive steps. You can ^C anytime
before the systemd unit is started — nothing irreversible happens before that.

  1) Install base tools             (apt: curl, build-essential, postgresql-client)
  2) Install Node.js                (needed for the AI CLIs)
  3) Choose & install AI CLIs       (Claude / Codex / Gemini — at least one required)
  4) Postgres path:
       (a) use an existing PostgreSQL host                     — recommended for prod
       (b) install PostgreSQL 16 + pgvector locally via apt    — easiest for single-box

  5) Bootstrap an opendray database, user, and the pgvector extension.
  6) Generate /etc/opendray/config.toml, run schema migration,
     install a systemd unit, and health-check the gateway.

EOF

ask_yes_no "Ready to start?" "y" READY
[ "$READY" = "y" ] || { log_info "Aborted."; exit 0; }

# ───────────────────────────────────────────────────────────────────────
# Phase 2 — Base tools
# ───────────────────────────────────────────────────────────────────────

log_step 1 "Install base tools"

APT_BASE_PKGS=(curl ca-certificates gnupg lsb-release tar postgresql-client xz-utils)
APT_BUILD_PKGS=(build-essential pkg-config git)

NEED_INSTALL=()
for pkg in "${APT_BASE_PKGS[@]}" "${APT_BUILD_PKGS[@]}"; do
    if ! dpkg-query -W -f='${Status}' "$pkg" 2>/dev/null | grep -q "install ok installed"; then
        NEED_INSTALL+=("$pkg")
    fi
done

if [ "${#NEED_INSTALL[@]}" -gt 0 ]; then
    log_info "Installing missing apt packages: ${NEED_INSTALL[*]}"
    run_priv apt-get update -qq
    DEBIAN_FRONTEND=noninteractive run_priv apt-get install -y -qq "${NEED_INSTALL[@]}"
fi
log_ok "Base tools ready"

# ───────────────────────────────────────────────────────────────────────
# Phase 3 — Node.js (for the AI CLIs)
# ───────────────────────────────────────────────────────────────────────
#
# We deliberately do NOT install pnpm here. The default path (release-
# tarball binary install) doesn't need pnpm at all — the AI CLIs are
# installed via `npm install -g` and the opendray binary is downloaded
# pre-built. pnpm is only required for `--from-source` builds (web bundle),
# and that branch installs it lazily, with visible progress output.
#
# corepack's silent download path was hanging on slow networks here.

log_step 2 "Install Node.js"

NODE_NEEDED=1
if have_cmd node; then
    NODE_VER_RAW="$(node --version 2>/dev/null || true)"   # "v22.10.0"
    NODE_MAJ="${NODE_VER_RAW#v}"
    NODE_MAJ="${NODE_MAJ%%.*}"
    if [[ "$NODE_MAJ" =~ ^[0-9]+$ ]] && [ "$NODE_MAJ" -ge 20 ]; then
        log_ok "Node.js $NODE_VER_RAW already installed"
        NODE_NEEDED=0
    else
        log_warn "Node.js $NODE_VER_RAW is too old (need ≥ 20). Will install Node 22 LTS."
    fi
fi

if [ "$NODE_NEEDED" = "1" ]; then
    log_info "Installing Node.js 22 LTS via NodeSource..."
    curl -fsSL https://deb.nodesource.com/setup_22.x | run_priv_env bash -
    DEBIAN_FRONTEND=noninteractive run_priv apt-get install -y -qq nodejs
    log_ok "Node.js $(node --version) installed"
fi

# ───────────────────────────────────────────────────────────────────────
# Phase 4 — AI CLI selection
# ───────────────────────────────────────────────────────────────────────

log_step 3 "Install AI CLIs (you need at least one)"

cat <<EOF
opendray spawns whichever CLI you select on a per-session basis. You
need at least one; you can add others later by re-running this wizard
or running the npm command by hand.

EOF

ask_yes_no "Install Claude Code (npm @anthropic-ai/claude-code)?" "y" WANT_CLAUDE
ask_yes_no "Install Gemini CLI (npm @google/gemini-cli)?" "n" WANT_GEMINI
ask_yes_no "Install Codex CLI (npm @openai/codex)?" "n" WANT_CODEX

INSTALLED_ANY=0

npm_install_global() {
    local pkg="$1" bin="$2"
    if have_cmd "$bin"; then
        log_ok "$bin already on PATH — skipping $pkg install"
        INSTALLED_ANY=1
        return 0
    fi
    log_info "Installing $pkg (~30–90 s — npm registry download)..."
    # `npm install -g` writes under /usr/lib/node_modules — needs root unless prefix is rewritten.
    # No --silent / no /dev/null redirect: AI CLI packages are 50–100 MB, and a silent install on a
    # slow link looks indistinguishable from a hang. Let npm's progress bar through.
    run_priv npm install -g "$pkg"
    if have_cmd "$bin"; then
        log_ok "$bin installed: $($bin --version 2>/dev/null | head -1 || echo 'version unknown')"
        INSTALLED_ANY=1
    else
        log_warn "$pkg installed but '$bin' is not on PATH — check 'npm bin -g' is in your \$PATH"
    fi
}

[ "$WANT_CLAUDE" = "y" ] && npm_install_global "@anthropic-ai/claude-code" claude
[ "$WANT_GEMINI" = "y" ] && npm_install_global "@google/gemini-cli" gemini
[ "$WANT_CODEX"  = "y" ] && npm_install_global "@openai/codex" codex

# Don't fail hard — user might want to install CLIs manually later.
# But warn loudly if literally nothing landed.
if [ "$INSTALLED_ANY" = "0" ] && ! have_cmd claude && ! have_cmd gemini && ! have_cmd codex; then
    log_warn "No AI CLI is on PATH. opendray will install fine, but session spawn will fail until you install one."
    ask_yes_no "Continue without an AI CLI?" "n" CONT_NO_CLI
    [ "$CONT_NO_CLI" = "y" ] || exit 0
fi

cat <<'EOF'

  ┌─ Heads up ──────────────────────────────────────────────────────┐
  │ The CLIs are installed but NOT logged in. After this wizard,    │
  │ run each login command as the opendray service user so the      │
  │ daemon can read the resulting credentials:                      │
  │                                                                 │
  │   sudo -u opendray -H claude auth login                         │
  │   sudo -u opendray -H codex login --device-auth                 │
  │   sudo -u opendray -H GEMINI_CLI_NO_BROWSER=true gemini         │
  │     (gemini has no `login` subcommand; run it once,             │
  │      paste the device code from codeassist.google.com/          │
  │      authcode at the prompt, then ^C to exit)                   │
  │                                                                 │
  │ Credentials are written under                                   │
  │   /var/lib/opendray/.{codex,gemini,claude}/                     │
  │ which is where the daemon reads them at session spawn time.     │
  └─────────────────────────────────────────────────────────────────┘
EOF

# ───────────────────────────────────────────────────────────────────────
# Phase 5 — PostgreSQL path selection
# ───────────────────────────────────────────────────────────────────────

log_step 4 "PostgreSQL"

ask_menu "Which Postgres should opendray talk to?" \
    "Use an existing PostgreSQL host (recommended if you already have one)|Install PostgreSQL 16 + pgvector locally via apt" \
    PG_PATH_CHOICE

case "$PG_PATH_CHOICE" in
    Use*)
        PG_MODE="existing"
        ;;
    Install*)
        PG_MODE="local"
        ;;
    *) log_die "Unexpected choice: $PG_PATH_CHOICE" ;;
esac

if [ "$PG_MODE" = "local" ]; then
    log_info "Installing PostgreSQL 16 + pgvector via apt..."

    # PG ≥16 on stable Ubuntu 24.04+ ships from the default repo. On older
    # releases the PGDG repo is needed; we add it conditionally.
    if ! apt-cache show postgresql-16 >/dev/null 2>&1; then
        log_info "Adding PostgreSQL APT repository (PGDG)..."
        CODENAME="$(lsb_release -cs)"
        run_priv install -d /usr/share/postgresql-common/pgdg
        curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc \
            | run_priv tee /usr/share/postgresql-common/pgdg/apt.postgresql.org.asc >/dev/null
        echo "deb [signed-by=/usr/share/postgresql-common/pgdg/apt.postgresql.org.asc] https://apt.postgresql.org/pub/repos/apt $CODENAME-pgdg main" \
            | run_priv tee /etc/apt/sources.list.d/pgdg.list >/dev/null
        run_priv apt-get update -qq
    fi

    DEBIAN_FRONTEND=noninteractive run_priv apt-get install -y -qq \
        postgresql-16 postgresql-16-pgvector

    run_priv systemctl enable --now postgresql
    log_ok "PostgreSQL 16 installed and running"

    PG_SUPER_HOST="127.0.0.1"
    PG_SUPER_PORT="5432"
    PG_SUPER_USER="postgres"
    PG_SUPER_DB="postgres"
    PG_SUPER_PW_AUTH="peer"   # local install: peer auth via the postgres OS user

else
    cat <<'EOF'

We'll connect to your existing PG host as a superuser only to create the
opendray database, user, and pgvector extension. After bootstrap,
opendray reconnects with its own least-privilege user.

EOF
    ask_with_default "PG host"               "localhost" PG_SUPER_HOST
    ask_with_default "PG port"               "5432"      PG_SUPER_PORT
    ask_with_default "Superuser name"        "postgres"  PG_SUPER_USER
    ask_with_default "Maintenance DB"        "postgres"  PG_SUPER_DB
    ask_password    "Superuser password"                 PG_SUPER_PW
    PG_SUPER_PW_AUTH="password"

    log_info "Testing superuser connection..."
    if ! PGPASSWORD="$PG_SUPER_PW" psql \
            -h "$PG_SUPER_HOST" -p "$PG_SUPER_PORT" -U "$PG_SUPER_USER" -d "$PG_SUPER_DB" \
            -tAc "SELECT 1" >/dev/null 2>&1; then
        log_die "Cannot connect as $PG_SUPER_USER@$PG_SUPER_HOST:$PG_SUPER_PORT — check host/port/credentials and try again."
    fi
    log_ok "Superuser connection OK"

    # Probe pgvector availability (extension must be installable on this server).
    PGPASSWORD="$PG_SUPER_PW" psql -h "$PG_SUPER_HOST" -p "$PG_SUPER_PORT" \
        -U "$PG_SUPER_USER" -d "$PG_SUPER_DB" \
        -tAc "SELECT 1 FROM pg_available_extensions WHERE name='vector'" 2>/dev/null \
        | grep -q 1 \
        || log_die "pgvector extension is not available on this PG server. Install postgresql-<ver>-pgvector (or build from https://github.com/pgvector/pgvector) and rerun."
    log_ok "pgvector extension is available on this server"
fi

# ───────────────────────────────────────────────────────────────────────
# Phase 6 — Bootstrap opendray DB / user / extension
# ───────────────────────────────────────────────────────────────────────

log_step 5 "Bootstrap opendray database"

ask_pg_identifier "Database name"                    "opendray"        OD_DB_NAME
ask_pg_identifier "Application DB user (CRUD only)"  "opendray_user"   OD_DB_USER

DEFAULT_DB_PW="$(gen_password 24)"
ask_with_default "App DB password (Enter = random)" "$DEFAULT_DB_PW" OD_DB_PW

# Each SQL statement runs as its own `psql -c` to dodge heredoc paste / quoting issues.
# We use a temp .pgpass so the superuser password is never on the command line.

run_psql_super() {
    local sql="$1"
    if [ "$PG_SUPER_PW_AUTH" = "peer" ]; then
        run_priv_as postgres psql -v ON_ERROR_STOP=1 -d "$PG_SUPER_DB" -c "$sql"
    else
        PGPASSWORD="$PG_SUPER_PW" psql -v ON_ERROR_STOP=1 \
            -h "$PG_SUPER_HOST" -p "$PG_SUPER_PORT" -U "$PG_SUPER_USER" -d "$PG_SUPER_DB" \
            -c "$sql"
    fi
}

# Idempotent ordering: create DB first (skip if exists), then role, then grants.
log_info "Creating database '$OD_DB_NAME' (skip if exists)..."
if ! run_psql_super "SELECT 1 FROM pg_database WHERE datname = '$OD_DB_NAME'" | grep -q 1; then
    run_psql_super "CREATE DATABASE \"$OD_DB_NAME\""
fi
log_ok "Database ready"

log_info "Creating role '$OD_DB_USER' (or updating password if it exists)..."
if run_psql_super "SELECT 1 FROM pg_roles WHERE rolname = '$OD_DB_USER'" | grep -q 1; then
    run_psql_super "ALTER USER \"$OD_DB_USER\" WITH ENCRYPTED PASSWORD '${OD_DB_PW//\'/\'\'}'"
else
    run_psql_super "CREATE USER \"$OD_DB_USER\" WITH ENCRYPTED PASSWORD '${OD_DB_PW//\'/\'\'}'"
fi
log_ok "Role ready"

run_psql_super "GRANT ALL PRIVILEGES ON DATABASE \"$OD_DB_NAME\" TO \"$OD_DB_USER\""

# Now connect *into* the new DB to enable the extension and grant on public schema.
run_psql_super_indb() {
    local sql="$1"
    if [ "$PG_SUPER_PW_AUTH" = "peer" ]; then
        run_priv_as postgres psql -v ON_ERROR_STOP=1 -d "$OD_DB_NAME" -c "$sql"
    else
        PGPASSWORD="$PG_SUPER_PW" psql -v ON_ERROR_STOP=1 \
            -h "$PG_SUPER_HOST" -p "$PG_SUPER_PORT" -U "$PG_SUPER_USER" -d "$OD_DB_NAME" \
            -c "$sql"
    fi
}

run_psql_super_indb "CREATE EXTENSION IF NOT EXISTS vector"
run_psql_super_indb "GRANT ALL ON SCHEMA public TO \"$OD_DB_USER\""

log_ok "pgvector enabled in '$OD_DB_NAME' + privileges granted"

# Verify the app user can connect.
log_info "Verifying app-user connection..."
if [ "$PG_MODE" = "local" ]; then
    # Local install: connect via 127.0.0.1 + password (pg_hba default is md5 for host).
    APP_PG_HOST="127.0.0.1"
    APP_PG_PORT="5432"
else
    APP_PG_HOST="$PG_SUPER_HOST"
    APP_PG_PORT="$PG_SUPER_PORT"
fi

if ! test_pg_dsn "$OD_DB_USER" "$OD_DB_PW" "$APP_PG_HOST" "$APP_PG_PORT" "$OD_DB_NAME"; then
    log_die "App user can connect... no. Check pg_hba.conf authentication settings (host entry for $APP_PG_HOST)."
fi
log_ok "App user can connect to $OD_DB_NAME"

# ───────────────────────────────────────────────────────────────────────
# Phase 7 — opendray credentials + listen address
# ───────────────────────────────────────────────────────────────────────

log_step 6 "opendray admin credentials + network"

cat <<'EOF'
Pick the initial admin login. You'll be forced to rotate the password
after your first UI login (opendray writes a bcrypt keyfile, after
which the plaintext in config.toml is inert).

EOF

ask_with_default "Admin username" "admin" OD_ADMIN_USER

DEFAULT_ADMIN_PW="$(gen_password 20)"
ask_with_default "Initial admin password (Enter = random)" "$DEFAULT_ADMIN_PW" OD_ADMIN_PW

cat <<'EOF'

Listen address — where the gateway binds:
  127.0.0.1:8770   loopback only, safest; reach over SSH tunnel or a reverse proxy
  0.0.0.0:8770     all interfaces, LAN reachable (only do this on trusted networks)

EOF

ask_with_default "Listen address" "127.0.0.1:8770" OD_LISTEN

# ───────────────────────────────────────────────────────────────────────
# Phase 8 — Binary install
# ───────────────────────────────────────────────────────────────────────

log_step 7 "Install opendray binary"

if [ "$FROM_SOURCE" = "1" ]; then
    [ -d "$SCRIPT_DIR/../cmd/opendray" ] || log_die "--from-source given but no cmd/opendray dir found at $SCRIPT_DIR/.."
    have_cmd go || log_die "go toolchain required for --from-source. Install with: 'apt install golang-go' (or use a Go 1.25+ build)."

    if ! have_cmd pnpm; then
        log_info "Installing pnpm globally for the web build (~30 s — npm registry)..."
        run_priv npm install -g pnpm@latest
    fi
    have_cmd pnpm || log_die "pnpm is required for --from-source builds (the React SPA goes into the Go binary via go:embed). Install pnpm and rerun."

    log_info "Building web bundle (pnpm install + build)..."
    ( cd "$SCRIPT_DIR/../app/web" && pnpm install --frozen-lockfile && pnpm build )

    log_info "Building Go binary (this takes ~30 s)..."
    ( cd "$SCRIPT_DIR/.." && go build -trimpath -ldflags="-s -w" -o /tmp/opendray-binary ./cmd/opendray )
    run_priv install -m 0755 /tmp/opendray-binary "$OPENDRAY_PREFIX/bin/opendray"
    rm -f /tmp/opendray-binary
else
    log_info "Fetching latest release tag from GitHub..."
    LATEST_TAG="$(curl -fsSL "https://api.github.com/repos/$OPENDRAY_REPO/releases/latest" \
        | grep -oE '"tag_name":\s*"[^"]+"' | head -1 \
        | sed 's/.*"\([^"]*\)"$/\1/')"

    [ -n "$LATEST_TAG" ] || log_die "Could not resolve latest release tag from $OPENDRAY_REPO"
    log_ok "Latest release: $LATEST_TAG"

    VERSION="${LATEST_TAG#v}"
    TARBALL_NAME="opendray_${VERSION}_linux_${ARCH}.tar.gz"
    TARBALL_URL="https://github.com/$OPENDRAY_REPO/releases/download/$LATEST_TAG/$TARBALL_NAME"

    TMP_TARBALL="$(mktemp --suffix=.tar.gz)"
    register_cleanup_file "$TMP_TARBALL"
    TMP_EXTRACT="$(mktemp -d)"
    register_cleanup_file "$TMP_EXTRACT/_dummy"   # also wipe the dir below

    log_info "Downloading $TARBALL_NAME..."
    if ! curl -fsSL --retry 3 -o "$TMP_TARBALL" "$TARBALL_URL"; then
        log_die "Download failed: $TARBALL_URL"
    fi

    # Best-effort checksum verification using SHA256SUMS shipped on the same release.
    SUMS_URL="https://github.com/$OPENDRAY_REPO/releases/download/$LATEST_TAG/SHA256SUMS"
    TMP_SUMS="$(mktemp)"
    register_cleanup_file "$TMP_SUMS"
    if curl -fsSL --retry 2 -o "$TMP_SUMS" "$SUMS_URL" 2>/dev/null; then
        if grep -q " $TARBALL_NAME$" "$TMP_SUMS"; then
            EXPECTED="$(grep " $TARBALL_NAME$" "$TMP_SUMS" | awk '{print $1}')"
            ACTUAL="$(sha256sum "$TMP_TARBALL" | awk '{print $1}')"
            if [ "$EXPECTED" = "$ACTUAL" ]; then
                log_ok "SHA-256 verified ($EXPECTED)"
            else
                log_die "Checksum mismatch! Expected $EXPECTED, got $ACTUAL. Refusing to install."
            fi
        else
            log_warn "$TARBALL_NAME not listed in SHA256SUMS — skipping checksum check."
        fi
    else
        log_warn "Could not fetch SHA256SUMS — skipping checksum check."
    fi

    log_info "Extracting..."
    tar -xzf "$TMP_TARBALL" -C "$TMP_EXTRACT"
    BINARY_FOUND="$(find "$TMP_EXTRACT" -type f -name opendray -perm -u+x | head -1)"
    [ -n "$BINARY_FOUND" ] || log_die "No 'opendray' binary inside $TARBALL_NAME"

    run_priv install -m 0755 "$BINARY_FOUND" "$OPENDRAY_PREFIX/bin/opendray"
    rm -rf "$TMP_EXTRACT"
fi

log_ok "Installed: $OPENDRAY_PREFIX/bin/opendray"
"$OPENDRAY_PREFIX/bin/opendray" version

# ───────────────────────────────────────────────────────────────────────
# Phase 9 — Generate config.toml + migrate
# ───────────────────────────────────────────────────────────────────────

log_step 8 "Write config + apply schema migrations"

OD_CONFIG_PATH="$OPENDRAY_CONFIG_DIR/config.toml"
OD_ENV_PATH="$OPENDRAY_CONFIG_DIR/opendray.env"
OD_DSN="$(build_dsn "$OD_DB_USER" "$OD_DB_PW" "$APP_PG_HOST" "$APP_PG_PORT" "$OD_DB_NAME")"

run_priv install -d -m 0750 "$OPENDRAY_CONFIG_DIR"
run_priv install -d -m 0750 "$OPENDRAY_DATA_DIR"
run_priv install -d -m 0755 "$OPENDRAY_LOG_DIR"

# ── 1. opendray.env — SECRETS LIVE HERE ──────────────────────────────
# Strict mode 0640, root:opendray. systemd reads as root via
# EnvironmentFile=; opendray's `migrate` subcommand reads it directly
# while running as the service user. No other shell user on the box
# can see these values.
TMP_ENV="$(mktemp)"
register_cleanup_file "$TMP_ENV"
cat > "$TMP_ENV" <<EOF
# opendray secrets — generated by install-linux.sh on $(date -Iseconds)
#
# Keep this file 0640 root:opendray. opendray reads these via env-var
# overrides (see internal/config/config.go) and they take precedence
# over anything left in config.toml.
OPENDRAY_DATABASE_URL=$OD_DSN
OPENDRAY_ADMIN_PASSWORD=$OD_ADMIN_PW
EOF
run_priv install -m 0640 "$TMP_ENV" "$OD_ENV_PATH"

# ── 2. config.toml — NON-SECRET configuration ────────────────────────
# This file is intentionally safe to read (mode 0644). Anything that
# could leak credentials is omitted; opendray fills those from the
# env file above.
TMP_CFG="$(mktemp)"
register_cleanup_file "$TMP_CFG"
cat > "$TMP_CFG" <<EOF
# opendray gateway configuration — generated by install-linux.sh on $(date -Iseconds)
#
# This file is intentionally non-secret (mode 0644). Database URL and
# admin password are loaded from /etc/opendray/opendray.env (0640) at
# service start, via systemd's EnvironmentFile= directive.
#
# Override anything in this file with OPENDRAY_<KEY>=… environment
# variables — see config.example.toml for the full list.

listen = "$OD_LISTEN"

[database]
# url loaded from OPENDRAY_DATABASE_URL (see opendray.env)

[admin]
# user is fine to keep here — it's only an identifier, not a secret.
# password loaded from OPENDRAY_ADMIN_PASSWORD (see opendray.env).
# After your first UI password rotation, opendray writes a bcrypt
# keyfile under the service user's home and the env-loaded password
# becomes inert.
user = "$OD_ADMIN_USER"

[log]
level = "info"
format = "json"

[runtime]
data_dir = "$OPENDRAY_DATA_DIR"
EOF
run_priv install -m 0644 "$TMP_CFG" "$OD_CONFIG_PATH"

# Create the service user before running migrate so the runtime artifacts
# (data_dir contents, future bcrypt keyfile) land with the right ownership.
if ! id "$OPENDRAY_SERVICE_USER" >/dev/null 2>&1; then
    log_info "Creating service account '$OPENDRAY_SERVICE_USER'..."
    run_priv useradd --system --home-dir "$OPENDRAY_DATA_DIR" --shell /usr/sbin/nologin "$OPENDRAY_SERVICE_USER"
fi
run_priv chown -R "$OPENDRAY_SERVICE_USER:$OPENDRAY_SERVICE_USER" "$OPENDRAY_DATA_DIR" "$OPENDRAY_LOG_DIR"
run_priv chown root:"$OPENDRAY_SERVICE_USER" "$OPENDRAY_CONFIG_DIR" "$OD_CONFIG_PATH" "$OD_ENV_PATH"
run_priv chmod 0750 "$OPENDRAY_CONFIG_DIR"
run_priv chmod 0644 "$OD_CONFIG_PATH"
run_priv chmod 0640 "$OD_ENV_PATH"

log_ok "config.toml: $OD_CONFIG_PATH (0644, no secrets — safe to read)"
log_ok "opendray.env: $OD_ENV_PATH (0640 root:$OPENDRAY_SERVICE_USER — secrets here)"

log_info "Applying migrations (idempotent)..."
# Pass secrets via the wizard's shell env so opendray-as-service-user picks
# them up. The actual service uses systemd's EnvironmentFile= once started.
OPENDRAY_DATABASE_URL="$OD_DSN" \
OPENDRAY_ADMIN_PASSWORD="$OD_ADMIN_PW" \
    run_priv_as_env "$OPENDRAY_SERVICE_USER" "$OPENDRAY_PREFIX/bin/opendray" migrate -config "$OD_CONFIG_PATH"
log_ok "Migrations applied"

# ───────────────────────────────────────────────────────────────────────
# Phase 10 — systemd unit + health check
# ───────────────────────────────────────────────────────────────────────

if [ "$SKIP_SERVICE" = "1" ]; then
    log_info "--skip-service set — leaving service installation to you."
    log_section "Manual start command"
    cat <<EOF
  sudo -u $OPENDRAY_SERVICE_USER $OPENDRAY_PREFIX/bin/opendray serve -config $OD_CONFIG_PATH

EOF
    exit 0
fi

log_step 9 "Install systemd unit + start service"

UNIT_PATH="/etc/systemd/system/${OPENDRAY_SERVICE_NAME}.service"

TMP_UNIT="$(mktemp)"
register_cleanup_file "$TMP_UNIT"
cat > "$TMP_UNIT" <<EOF
[Unit]
Description=opendray gateway
Documentation=https://github.com/$OPENDRAY_REPO
After=network-online.target postgresql.service
Wants=network-online.target

[Service]
Type=simple
User=$OPENDRAY_SERVICE_USER
Group=$OPENDRAY_SERVICE_USER
# Secrets — systemd loads these into the service env before exec, so
# the database URL and admin bootstrap password never appear in
# config.toml on disk or in 'ps aux' / journalctl output.
EnvironmentFile=$OD_ENV_PATH
# Where the dashboard's "Update now" drops its request file for the
# privileged self-update oneshot to act on (must match the .path unit).
Environment=OPENDRAY_STATE_DIR=$OPENDRAY_DATA_DIR
ExecStart=$OPENDRAY_PREFIX/bin/opendray serve -config $OD_CONFIG_PATH
Restart=on-failure
RestartSec=5s
StandardOutput=append:$OPENDRAY_LOG_DIR/opendray.log
StandardError=append:$OPENDRAY_LOG_DIR/opendray.err

# ── Hardening ────────────────────────────────────────────────────────
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$OPENDRAY_DATA_DIR $OPENDRAY_LOG_DIR
PrivateTmp=true
PrivateDevices=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictNamespaces=true
RestrictRealtime=true
LockPersonality=true
# MemoryDenyWriteExecute is intentionally NOT enabled: the V8/Node-based
# CLIs (codex, gemini) JIT-compile and must flip pages RW->RX via mprotect,
# which a W^X filter blocks (SIGSYS → the child dies on spawn). Claude
# survives via an interpreter-only fallback; codex/gemini do not. The
# blast radius without it is the unprivileged $OPENDRAY_SERVICE_USER user.

[Install]
WantedBy=multi-user.target
EOF

run_priv install -m 0644 "$TMP_UNIT" "$UNIT_PATH"

# ── Self-update units (in-dashboard "Update now") ────────────────────
# The daemon is unprivileged and can't replace its own binary or restart
# the unit, so "Update now" only drops a request file; this root oneshot,
# activated by the path unit when the file appears, does the privileged
# work. It installs the official latest (checksum-verified) and clears the
# request.
TMP_SU_SVC="$(mktemp)"; register_cleanup_file "$TMP_SU_SVC"
cat > "$TMP_SU_SVC" <<EOF
[Unit]
Description=Apply a queued opendray upgrade (privileged)
After=network-online.target ${OPENDRAY_SERVICE_NAME}.service
Wants=network-online.target

[Service]
Type=oneshot
Environment=OPENDRAY_STATE_DIR=$OPENDRAY_DATA_DIR
ExecStart=$OPENDRAY_PREFIX/bin/opendray self-update --apply
ProtectHome=yes
EOF

TMP_SU_PATH="$(mktemp)"; register_cleanup_file "$TMP_SU_PATH"
cat > "$TMP_SU_PATH" <<EOF
[Unit]
Description=Watch for an opendray in-dashboard upgrade request

[Path]
PathExists=$OPENDRAY_DATA_DIR/selfupdate.request
Unit=opendray-selfupdate.service

[Install]
WantedBy=multi-user.target
EOF

run_priv install -m 0644 "$TMP_SU_SVC" "/etc/systemd/system/opendray-selfupdate.service"
run_priv install -m 0644 "$TMP_SU_PATH" "/etc/systemd/system/opendray-selfupdate.path"

run_priv systemctl daemon-reload
run_priv systemctl enable --now "$OPENDRAY_SERVICE_NAME"
run_priv systemctl enable --now opendray-selfupdate.path
log_ok "Service '$OPENDRAY_SERVICE_NAME' enabled and started (+ self-update watcher)"

# Health check loop.
log_info "Waiting for the gateway to respond..."
HEALTH_URL="http://${OD_LISTEN}/api/v1/health"
# If user picked 0.0.0.0, hit it via 127.0.0.1 for the loopback check.
[[ "$OD_LISTEN" == 0.0.0.0:* ]] && HEALTH_URL="http://127.0.0.1:${OD_LISTEN##*:}/api/v1/health"

for i in $(seq 1 20); do
    if curl -fsS "$HEALTH_URL" 2>/dev/null | grep -q '"status":"ok"'; then
        log_ok "Health endpoint OK"
        break
    fi
    sleep 1
    if [ "$i" = 20 ]; then
        log_err "Health check did not succeed within 20s."
        log_dim "Tail $OPENDRAY_LOG_DIR/opendray.err for details:"
        run_priv tail -30 "$OPENDRAY_LOG_DIR/opendray.err" 2>/dev/null || true
        exit 1
    fi
done

# ───────────────────────────────────────────────────────────────────────
# Summary
# ───────────────────────────────────────────────────────────────────────

log_section "Install complete"

# Build a clickable URL. For 0.0.0.0 listens, resolve the host's primary
# LAN IP so the operator can actually click and open it from another
# machine — `<this-host>` was a documentation placeholder that came
# through to the terminal as literal text.
PORT="${OD_LISTEN##*:}"
HOST_PART="${OD_LISTEN%:*}"
case "$HOST_PART" in
    0.0.0.0|"")
        LAN_IP=""
        if have_cmd hostname; then
            LAN_IP="$(hostname -I 2>/dev/null | awk '{print $1}')"
        fi
        if [ -z "$LAN_IP" ] && have_cmd ip; then
            LAN_IP="$(ip route get 1.1.1.1 2>/dev/null | awk '/src/ {for(i=1;i<=NF;i++) if($i=="src") {print $(i+1); exit}}')"
        fi
        [ -n "$LAN_IP" ] || LAN_IP="<this-host-ip>"
        WEB_URL="http://${LAN_IP}:${PORT}/admin/"
        WEB_URL_NOTE="  (LAN — reachable from any device on the same network)"
        ;;
    *)
        WEB_URL="http://${OD_LISTEN}/admin/"
        WEB_URL_NOTE=""
        ;;
esac

cat <<EOF
  ${C_BLU}Admin UI${C_NC}        ${WEB_URL}${WEB_URL_NOTE}
  ${C_BLU}Username${C_NC}        ${OD_ADMIN_USER}
  ${C_BLU}Password${C_NC}        ${OD_ADMIN_PW}   ${C_YEL}← rotate via Settings → Admin on first login${C_NC}

  ${C_BLU}Config${C_NC}          ${OD_CONFIG_PATH}
  ${C_BLU}Logs${C_NC}            ${OPENDRAY_LOG_DIR}/opendray.{log,err}
  ${C_BLU}Service${C_NC}         systemctl {status,restart,stop} ${OPENDRAY_SERVICE_NAME}

  ${C_BLU}Database${C_NC}        ${OD_DB_USER}@${APP_PG_HOST}:${APP_PG_PORT}/${OD_DB_NAME}
  ${C_BLU}DB password${C_NC}     ${OD_DB_PW}   ${C_YEL}← save this somewhere safe${C_NC}

  Next:
    1. Open the admin UI (link above is clickable in most terminals),
       log in as ${OD_ADMIN_USER}, rotate the admin password.
    2. Finish logging your AI CLI(s) in as the opendray service user so
       the daemon can read the resulting credentials under
       /var/lib/opendray/.{codex,gemini,claude}/ :
         sudo -u opendray -H claude auth login
         sudo -u opendray -H codex login --device-auth
         sudo -u opendray -H GEMINI_CLI_NO_BROWSER=true gemini
       (gemini has no 'login' subcommand — run it once interactively,
        paste the device code from codeassist.google.com/authcode at
        the prompt, then ^C to exit.)
    3. Providers → register the CLI binary path (e.g. \$(which claude)).
    4. Sessions → New session → spawn your first session.

  See docs/getting-started.md for the post-install walkthrough.
EOF
