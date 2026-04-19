#!/usr/bin/env bash
# deploy_release.sh — one-shot update: rebuild backend → push to syz server,
#                     rebuild Android APK → upload to UNAS.
#
# Designed to run via the task-runner plugin (discovered as ./scripts/*.sh).
# Every input is an env var — no hard-coded hosts, no secrets in source.
# Set them once in scripts/deploy.env (gitignored) and the task-runner
# panel will pass them in.
#
# Phases (skip with SKIP_BACKEND=1 / SKIP_APK=1 / SKIP_UNAS=1):
#   1. Preflight  — tool + env var check; fail fast
#   2. Web bundle — flutter build web → app/build/web/
#   3. Go binary  — CGO=0 linux/amd64, embeds the fresh web bundle
#   4. SSH push   — scp binary to syz, systemctl restart, health check
#   5. APK build  — flutter build apk --release
#   6. UNAS upload— smbclient put APK into the project dir
#
# Exit codes (meaningful for task-runner UI):
#   0  success
#   1  missing / invalid env var
#   2  preflight tool missing
#   3  build failure (go or flutter)
#   4  remote deploy failure (ssh / scp / service)
#   5  APK upload failure (smbclient)
#
# Usage (manual):
#   scripts/deploy_release.sh                     # full pipeline
#   SKIP_APK=1 scripts/deploy_release.sh          # backend only
#   SKIP_BACKEND=1 scripts/deploy_release.sh      # APK + UNAS only
#
# Required env vars (see scripts/deploy.env.example):
#   OPENDRAY_DEPLOY_HOST       ssh destination, e.g. root@192.168.3.XX
#   OPENDRAY_DEPLOY_BIN_PATH   remote binary path, e.g. /opt/opendray/opendray
#   OPENDRAY_DEPLOY_SERVICE    systemd unit, e.g. opendray.service
#   OPENDRAY_HEALTH_URL        smoke-test URL, e.g. http://192.168.3.XX:8640/api/health
#   UNAS_HOST                  SMB host, e.g. //192.168.9.8/Claude_Workspace
#   UNAS_USER                  SMB user
#   UNAS_PASSWORD              SMB password (DO NOT commit — source from secret store)
#   UNAS_PATH                  subdir, e.g. OpenDray/android
#
# Optional:
#   OPENDRAY_SSH_KEY           default ~/.ssh/home_lab_key
#   FLUTTER_HOME               default ~/flutter
#   GO_BIN                     default /usr/local/go/bin/go
#   SKIP_BACKEND, SKIP_APK, SKIP_UNAS, NO_BUMP   flags ("1" to enable)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

# Load scripts/deploy.env if present (non-committed config).
if [[ -f "$SCRIPT_DIR/deploy.env" ]]; then
  # shellcheck disable=SC1091
  set -a
  source "$SCRIPT_DIR/deploy.env"
  set +a
fi

# ── Defaults ──────────────────────────────────────────────────────────
OPENDRAY_SSH_KEY="${OPENDRAY_SSH_KEY:-$HOME/.ssh/home_lab_key}"
FLUTTER_HOME="${FLUTTER_HOME:-$HOME/flutter}"
GO_BIN="${GO_BIN:-/usr/local/go/bin/go}"
SKIP_BACKEND="${SKIP_BACKEND:-0}"
SKIP_APK="${SKIP_APK:-0}"
SKIP_UNAS="${SKIP_UNAS:-0}"
NO_BUMP="${NO_BUMP:-0}"

export PATH="$FLUTTER_HOME/bin:$(dirname "$GO_BIN"):$PATH"

# ── Pretty logging ────────────────────────────────────────────────────
phase()  { printf '\n\033[1;34m▶ %s\033[0m\n' "$*"; }
ok()     { printf '  \033[1;32m✓\033[0m %s\n' "$*"; }
warn()   { printf '  \033[1;33m!\033[0m %s\n' "$*" >&2; }
fail()   { printf '  \033[1;31m✗ %s\033[0m\n' "$*" >&2; exit "${2:-1}"; }

require_var() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    fail "missing required env var: $name (see scripts/deploy.env.example)" 1
  fi
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required tool not found: $1" 2
}

# ── 1. Preflight ──────────────────────────────────────────────────────
phase "Preflight"

# Backend vars only needed if we're doing a backend deploy.
if [[ "$SKIP_BACKEND" != "1" ]]; then
  require_var OPENDRAY_DEPLOY_HOST
  require_var OPENDRAY_DEPLOY_BIN_PATH
  require_var OPENDRAY_DEPLOY_SERVICE
  require_var OPENDRAY_HEALTH_URL
  require_cmd ssh
  require_cmd scp
  require_cmd curl
  [[ -f "$GO_BIN" ]] || fail "go not found at $GO_BIN (set GO_BIN env var)" 2
fi

# APK phase needs flutter + Android SDK.
if [[ "$SKIP_APK" != "1" ]]; then
  [[ -x "$FLUTTER_HOME/bin/flutter" ]] \
    || fail "flutter not found at $FLUTTER_HOME/bin (set FLUTTER_HOME)" 2
  # Android SDK: flutter reports it in doctor; we check ANDROID_SDK_ROOT or
  # let flutter build apk fail with its own clear message below.
fi

# UNAS upload needs smbclient.
if [[ "$SKIP_UNAS" != "1" ]]; then
  require_var UNAS_HOST
  require_var UNAS_USER
  require_var UNAS_PASSWORD
  require_var UNAS_PATH
  require_cmd smbclient
fi

ok "env + tools ok"

# ── 2. Web bundle (always built when backend runs, since binary embeds it) ──
if [[ "$SKIP_BACKEND" != "1" ]]; then
  phase "Flutter web bundle"
  ( cd app && "$FLUTTER_HOME/bin/flutter" pub get >/dev/null )
  ( cd app && "$FLUTTER_HOME/bin/flutter" build web --release ) \
    || fail "flutter build web failed" 3
  ok "app/build/web ready"
fi

# ── 3. Go binary (linux/amd64) ────────────────────────────────────────
if [[ "$SKIP_BACKEND" != "1" ]]; then
  phase "Go binary (linux/amd64, static)"
  BIN_OUT="$REPO_ROOT/bin/opendray-linux-amd64"
  mkdir -p "$(dirname "$BIN_OUT")"
  VERSION="$(cd app && grep -E '^version:' pubspec.yaml | awk '{print $2}')"
  BUILD_SHA="$(git rev-parse --short HEAD 2>/dev/null || echo dev)"
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 "$GO_BIN" build \
    -trimpath \
    -ldflags "-s -w -X main.version=${VERSION} -X main.buildSha=${BUILD_SHA}" \
    -o "$BIN_OUT" \
    ./cmd/opendray \
    || fail "go build failed" 3
  BIN_SIZE="$(du -h "$BIN_OUT" | cut -f1)"
  ok "built $BIN_OUT ($BIN_SIZE, $VERSION@$BUILD_SHA)"
fi

# ── 4. SSH push to syz + restart ──────────────────────────────────────
if [[ "$SKIP_BACKEND" != "1" ]]; then
  phase "Deploy to $OPENDRAY_DEPLOY_HOST"
  BIN_OUT="$REPO_ROOT/bin/opendray-linux-amd64"
  SSH_OPTS="-i $OPENDRAY_SSH_KEY -o StrictHostKeyChecking=accept-new -o BatchMode=yes"

  # Stage to a tempfile, swap atomically.
  REMOTE_TMP="${OPENDRAY_DEPLOY_BIN_PATH}.new"
  scp $SSH_OPTS "$BIN_OUT" "$OPENDRAY_DEPLOY_HOST:$REMOTE_TMP" \
    || fail "scp failed" 4

  ssh $SSH_OPTS "$OPENDRAY_DEPLOY_HOST" bash -s <<EOF \
    || fail "remote deploy failed" 4
set -euo pipefail
chmod 0755 "$REMOTE_TMP"
systemctl stop "$OPENDRAY_DEPLOY_SERVICE"
mv "$REMOTE_TMP" "$OPENDRAY_DEPLOY_BIN_PATH"
systemctl start "$OPENDRAY_DEPLOY_SERVICE"
systemctl is-active --quiet "$OPENDRAY_DEPLOY_SERVICE"
EOF
  ok "service restarted"

  # Health check with exponential backoff (max ~15s).
  phase "Health check"
  for attempt in 1 2 3 4 5; do
    sleep_for=$((attempt * attempt))
    if curl -fsS --max-time 5 "$OPENDRAY_HEALTH_URL" >/dev/null 2>&1; then
      ok "health ok on attempt $attempt"
      break
    fi
    if [[ $attempt -eq 5 ]]; then
      fail "health check failed after 5 attempts — service may be crashing" 4
    fi
    warn "not ready yet, retry in ${sleep_for}s"
    sleep "$sleep_for"
  done
fi

# ── 5. APK build ──────────────────────────────────────────────────────
APK_PATH=""
if [[ "$SKIP_APK" != "1" ]]; then
  phase "Flutter APK (release)"

  # Version bump mirrors app/build_release.sh (Linux-portable sed).
  cd app
  CURRENT=$(grep -E '^version:' pubspec.yaml | awk '{print $2}')
  NAME="${CURRENT%+*}"
  BUILD="${CURRENT#*+}"

  if [[ "$NO_BUMP" != "1" ]]; then
    BUILD=$((BUILD + 1))
    NEXT="${NAME}+${BUILD}"
    sed -i.bak "s/^version: .*/version: ${NEXT}/" pubspec.yaml
    rm -f pubspec.yaml.bak
    ok "version: ${CURRENT} → ${NEXT}"
  else
    NEXT="$CURRENT"
    ok "version: ${CURRENT} (no bump)"
  fi

  BUILD_DATE="$(date +%Y-%m-%d)"
  "$FLUTTER_HOME/bin/flutter" pub get >/dev/null
  "$FLUTTER_HOME/bin/flutter" build apk --release \
    --dart-define=BUILD_DATE="$BUILD_DATE" \
    || { cd "$REPO_ROOT"; fail "flutter build apk failed (Android SDK configured?)" 3; }

  SRC_APK="build/app/outputs/flutter-apk/app-release.apk"
  [[ -f "$SRC_APK" ]] || { cd "$REPO_ROOT"; fail "APK not found at $SRC_APK" 3; }

  # Rename for clarity.
  STAMPED="opendray-${NEXT/+/-}-${BUILD_DATE}.apk"
  APK_PATH="$REPO_ROOT/app/build/app/outputs/flutter-apk/$STAMPED"
  cp "$SRC_APK" "$APK_PATH"
  APK_SIZE="$(du -h "$APK_PATH" | cut -f1)"
  ok "APK: $STAMPED ($APK_SIZE)"
  cd "$REPO_ROOT"
fi

# ── 6. UNAS upload ────────────────────────────────────────────────────
if [[ "$SKIP_UNAS" != "1" ]]; then
  phase "Upload to UNAS"
  if [[ -z "$APK_PATH" ]]; then
    # SKIP_APK was set but user still wants upload — find most recent APK.
    APK_PATH="$(ls -t app/build/app/outputs/flutter-apk/*.apk 2>/dev/null | head -1 || true)"
    [[ -n "$APK_PATH" ]] || fail "no APK to upload (SKIP_APK=1 and none cached)" 1
    warn "using cached APK: $APK_PATH"
  fi

  APK_NAME="$(basename "$APK_PATH")"
  APK_DIR="$(dirname "$APK_PATH")"

  # smbclient uses -U user%pass; quote password in case of special chars.
  smbclient "$UNAS_HOST" -U "$UNAS_USER%$UNAS_PASSWORD" \
    -D "$UNAS_PATH" \
    -c "lcd $APK_DIR; put $APK_NAME" \
    || fail "smbclient upload failed" 5

  ok "uploaded $APK_NAME → $UNAS_HOST/$UNAS_PATH/"
fi

phase "Done"
ok "OpenDray deployment complete"
