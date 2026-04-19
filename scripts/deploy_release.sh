#!/usr/bin/env bash
# deploy_release.sh — one-shot update: rebuild backend → swap binary →
#                     restart service, rebuild Android APK → UNAS upload.
#
# Designed to run via the task-runner plugin (./scripts/*.sh discovery).
#
# Two deploy modes:
#   • LOCAL (default)  — same box as the running opendray service. No SSH
#                        required. The restart is forked as a detached
#                        background worker so task-runner can exit
#                        cleanly before systemd terminates it (task-runner
#                        IS a subprocess of the service being restarted).
#                        Progress logged to /tmp/opendray-deploy-<ts>.log.
#   • REMOTE           — set OPENDRAY_DEPLOY_HOST to an ssh target and
#                        the script will scp + ssh systemctl restart.
#                        Health check runs inline (we're not being killed).
#
# Phase order (every phase skippable):
#   1. Preflight   — tool + mode detect; auto-skip UNAS if not configured
#   2. Web bundle  — flutter build web → app/build/web/ (Go embed input)
#   3. Go binary   — CGO=0 linux/amd64 with version + sha stamped
#   4. APK build   — flutter build apk --release, stamps name
#   5. UNAS upload — smbclient put APK into project dir
#   6. Deploy      — backend LAST (local mode forks detached; remote inline)
#
# The backend deploy is last on purpose: once we kick the restart in
# local mode, this script is dead. APK+UNAS run BEFORE that so nothing
# important is lost if restart fails.
#
# Exit codes (meaningful in task-runner UI):
#   0  success
#   1  missing / invalid env var
#   2  preflight tool missing
#   3  build failure (go or flutter)
#   4  remote deploy failure (ssh / scp / service)
#   5  APK upload failure (smbclient)
#
# Usage:
#   scripts/deploy_release.sh                    # full pipeline, local mode
#   SKIP_APK=1 scripts/deploy_release.sh         # backend only
#   SKIP_BACKEND=1 scripts/deploy_release.sh     # APK + UNAS only
#   SKIP_UNAS=1 scripts/deploy_release.sh        # don't upload to UNAS
#   OPENDRAY_DEPLOY_HOST=root@192.168.3.X ./deploy_release.sh  # remote mode
#
# Env vars (all optional; see scripts/deploy.env.example for ALL knobs):
#   OPENDRAY_DEPLOY_HOST       empty → local; set → remote ssh target
#   OPENDRAY_DEPLOY_BIN_PATH   default /opt/opendray/opendray
#   OPENDRAY_DEPLOY_SERVICE    default opendray.service
#   OPENDRAY_HEALTH_URL        default http://127.0.0.1:8640/api/health
#   OPENDRAY_SSH_KEY           default ~/.ssh/home_lab_key (remote only)
#   UNAS_HOST, UNAS_USER, UNAS_PASSWORD, UNAS_PATH
#                              set all 4 to enable; any missing → auto-skip
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

# Deploy mode detection.
#   LOCAL  — task-runner is running on the same box as the opendray service
#            (the normal case; no SSH, no extra config needed). All backend
#            defaults apply: /opt/opendray/opendray, opendray.service,
#            http://127.0.0.1:8640/api/health.
#   REMOTE — user explicitly set OPENDRAY_DEPLOY_HOST to something other
#            than local/localhost. scp + ssh path kicks in.
if [[ -z "${OPENDRAY_DEPLOY_HOST:-}" \
      || "$OPENDRAY_DEPLOY_HOST" == "local" \
      || "$OPENDRAY_DEPLOY_HOST" == "localhost" ]]; then
  DEPLOY_MODE="local"
else
  DEPLOY_MODE="remote"
fi

# Sensible defaults for local mode. Users override via env only if their
# install deviates from the standard layout.
OPENDRAY_DEPLOY_BIN_PATH="${OPENDRAY_DEPLOY_BIN_PATH:-/opt/opendray/opendray}"
OPENDRAY_DEPLOY_SERVICE="${OPENDRAY_DEPLOY_SERVICE:-opendray.service}"
OPENDRAY_HEALTH_URL="${OPENDRAY_HEALTH_URL:-http://127.0.0.1:8640/api/health}"

# Sudo is only used when not already root — inside LXC you typically run
# as root and this is a no-op.
if [[ $EUID -eq 0 ]]; then
  SUDO=""
else
  SUDO="sudo"
fi

# Backend tools.
if [[ "$SKIP_BACKEND" != "1" ]]; then
  require_cmd curl
  [[ -f "$GO_BIN" ]] || fail "go not found at $GO_BIN (set GO_BIN env var)" 2
  if [[ "$DEPLOY_MODE" == "remote" ]]; then
    require_cmd ssh
    require_cmd scp
  else
    require_cmd systemctl
  fi
fi

# APK phase needs flutter + Android SDK.
if [[ "$SKIP_APK" != "1" ]]; then
  [[ -x "$FLUTTER_HOME/bin/flutter" ]] \
    || fail "flutter not found at $FLUTTER_HOME/bin (set FLUTTER_HOME)" 2
  # Android SDK readiness is checked by `flutter build apk` itself — it
  # emits a clear error if the SDK isn't configured.
fi

# UNAS upload — auto-skip if not fully configured, so the deploy still
# completes the backend phase even when UNAS creds aren't in place yet.
if [[ "$SKIP_UNAS" != "1" ]]; then
  if [[ -z "${UNAS_HOST:-}" || -z "${UNAS_USER:-}" \
        || -z "${UNAS_PASSWORD:-}" || -z "${UNAS_PATH:-}" ]]; then
    warn "UNAS_* not fully configured; skipping UNAS upload"
    warn "to enable: copy scripts/deploy.env.example → scripts/deploy.env and fill UNAS_*"
    SKIP_UNAS=1
  else
    require_cmd smbclient
  fi
fi

ok "mode: $DEPLOY_MODE"
ok "binary: $OPENDRAY_DEPLOY_BIN_PATH"
ok "service: $OPENDRAY_DEPLOY_SERVICE"
ok "health: $OPENDRAY_HEALTH_URL"

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

# Backend binary staged but NOT yet swapped — we do APK + UNAS first so a
# deploy-triggered restart (which likely kills this script in local mode)
# doesn't lose work downstream.

# ── 4. APK build ──────────────────────────────────────────────────────
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

# ── 5. UNAS upload ────────────────────────────────────────────────────
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

# ── 6. Deploy backend (LAST — local mode detaches to survive self-kill) ──
if [[ "$SKIP_BACKEND" != "1" ]]; then
  BIN_OUT="$REPO_ROOT/bin/opendray-linux-amd64"
  TMP_TARGET="${OPENDRAY_DEPLOY_BIN_PATH}.new"

  if [[ "$DEPLOY_MODE" == "local" ]]; then
    phase "Deploy (local, detached restart)"
    # Stage the new binary NOW while we're still alive — atomic rename is
    # cheap and can't fail if FS has space.
    $SUDO install -m 0755 "$BIN_OUT" "$TMP_TARGET" \
      || fail "install staged binary failed" 4
    ok "staged new binary at $TMP_TARGET"

    # Task-runner is almost certainly a subprocess of the service we're
    # about to stop. systemctl stop kills our whole process group — so
    # we fork a detached worker that survives task-runner's death. It
    # does the mv + restart + health check and appends to a log the
    # user can tail after the workbench reconnects.
    LOG="/tmp/opendray-deploy-$(date +%Y%m%dT%H%M%S).log"
    nohup bash -c "
      exec >>'$LOG' 2>&1
      set -eo pipefail
      trap 'echo \"[deploy] \$(date -Is) — script exiting with code \$?\"' EXIT
      echo '[deploy] '\$(date -Is)' — starting local restart'
      # Give the parent task-runner a moment to return its exit code to
      # the user before we start terminating the service.
      sleep 2
      $SUDO systemctl stop '$OPENDRAY_DEPLOY_SERVICE'
      $SUDO mv '$TMP_TARGET' '$OPENDRAY_DEPLOY_BIN_PATH'
      $SUDO systemctl start '$OPENDRAY_DEPLOY_SERVICE'
      for i in 1 4 9 16 25; do
        if curl -fsS --max-time 5 '$OPENDRAY_HEALTH_URL' >/dev/null 2>&1; then
          echo '[deploy] '\$(date -Is)' — health ok'
          exit 0
        fi
        echo '[deploy] '\$(date -Is)' — not ready, retry in '\$i's'
        sleep \$i
      done
      echo '[deploy] '\$(date -Is)' — HEALTH FAILED — run: journalctl -u $OPENDRAY_DEPLOY_SERVICE -n 100'
      exit 1
    " </dev/null >/dev/null 2>&1 &
    DEPLOY_PID=$!
    disown "$DEPLOY_PID"
    ok "detached restart worker forked (pid $DEPLOY_PID)"
    ok "log: $LOG"
    warn "task-runner will lose connection during restart — reconnect the app and tail the log above"
    # In local mode we do NOT run a health check from this script —
    # the detached worker owns it. Exit 0 here; the worker's outcome
    # appears in the log.
  else
    phase "Deploy to $OPENDRAY_DEPLOY_HOST (remote)"
    SSH_OPTS="-i $OPENDRAY_SSH_KEY -o StrictHostKeyChecking=accept-new -o BatchMode=yes"

    scp $SSH_OPTS "$BIN_OUT" "$OPENDRAY_DEPLOY_HOST:$TMP_TARGET" \
      || fail "scp failed" 4

    ssh $SSH_OPTS "$OPENDRAY_DEPLOY_HOST" bash -s <<EOF \
      || fail "remote deploy failed" 4
set -euo pipefail
chmod 0755 "$TMP_TARGET"
systemctl stop "$OPENDRAY_DEPLOY_SERVICE"
mv "$TMP_TARGET" "$OPENDRAY_DEPLOY_BIN_PATH"
systemctl start "$OPENDRAY_DEPLOY_SERVICE"
systemctl is-active --quiet "$OPENDRAY_DEPLOY_SERVICE"
EOF
    ok "service restarted on $OPENDRAY_DEPLOY_HOST"

    # Remote deploy: health check is safe inline because we're not the
    # ones being restarted.
    phase "Health check"
    for attempt in 1 2 3 4 5; do
      sleep_for=$((attempt * attempt))
      if curl -fsS --max-time 5 "$OPENDRAY_HEALTH_URL" >/dev/null 2>&1; then
        ok "health ok on attempt $attempt"
        break
      fi
      if [[ $attempt -eq 5 ]]; then
        fail "health check failed after 5 attempts — check journalctl on $OPENDRAY_DEPLOY_HOST" 4
      fi
      warn "not ready yet, retry in ${sleep_for}s"
      sleep "$sleep_for"
    done
  fi
fi

phase "Done"
ok "OpenDray deployment complete"
