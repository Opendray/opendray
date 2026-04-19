#!/usr/bin/env bash
# deploy_release.sh — syz LXC one-shot release:
#     preflight UNAS  →  build APK  →  upload (verified)  →  deploy backend
#
# UNAS access goes through scripts/unas_upload.py (pure-Python smbprotocol
# client) because the LXC sandbox blocks installing samba-client. Bootstrap:
#     pip3 install --user smbprotocol
#
# DESIGN GOAL: never leave the system in a half-updated state. The APK
# MUST be reachable on UNAS before we touch the running backend. If UNAS
# is unreachable, the script aborts before building — never deploys.
#
# This script is tailored for the syz LXC container specifically:
#   • Always LOCAL mode (no SSH). Task-runner runs inside the same
#     opendray service we're about to restart, so the restart is
#     forked as a detached worker that outlives task-runner's death.
#   • UNAS is the single source of truth for new APKs. Upload is
#     preflight-tested + size-verified post-upload — a failed upload
#     aborts the whole pipeline before backend deploy.
#   • Filename + directory layout follow the upload-unas skill spec:
#       //192.168.9.8/Claude_Workspace/OpenDray/android/
#         OpenDray-v<version>+<build>-<yyyyMMdd>.apk
#
# PHASE ORDER (no phase skippable unless you know what you're doing):
#   0. Preflight UNAS    — connectivity + auth + target dir check
#   1. Preflight local   — tool + backend paths
#   2. Flutter web       — build web (Go embed input)
#   3. Go binary         — CGO=0 linux/amd64, version stamped
#   4. Flutter APK       — bump pubspec.yaml build number, build release
#   5. UNAS upload       — smbprotocol put + verify by size
#   6. Deploy backend    — fork detached restart worker, exit
#
# EXIT CODES:
#   0   success (or detached backend restart initiated)
#   1   missing env / credentials
#   2   tool not found
#   3   build failure
#   4   backend deploy failure
#   5   UNAS upload / verify failure (backend NOT deployed)
#   6   UNAS unreachable at preflight (nothing built, nothing touched)
#
# CONFIG: scripts/deploy.env (gitignored). Minimum viable contents:
#     UNAS_PASSWORD="..."          # required
#
# Everything else has sane defaults pinned to syz LXC + home-lab conventions.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

# ── Load optional config ──────────────────────────────────────────────
if [[ -f "$SCRIPT_DIR/deploy.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$SCRIPT_DIR/deploy.env"
  set +a
fi

# ── Defaults (home-lab conventions; override via deploy.env if needed) ──
# UNAS — pinned to the single home-lab NAS documented in the upload-unas skill.
UNAS_SERVER="${UNAS_SERVER:-192.168.9.8}"
UNAS_SHARE="${UNAS_SHARE:-Claude_Workspace}"
UNAS_USER="${UNAS_USER:-linivek}"
UNAS_PROJECT="${UNAS_PROJECT:-OpenDray}"
# Password: env var → ~/.config/opendray/unas.pw → error
if [[ -z "${UNAS_PASSWORD:-}" ]]; then
  PW_FILE="$HOME/.config/opendray/unas.pw"
  if [[ -f "$PW_FILE" && -r "$PW_FILE" ]]; then
    UNAS_PASSWORD="$(tr -d '\n\r' <"$PW_FILE")"
  fi
fi

# Backend (LOCAL always — task-runner lives inside this service).
# Default path matches the running syz LXC install (verified via
# `systemctl cat opendray.service` → /usr/local/bin/opendray).
OPENDRAY_DEPLOY_BIN_PATH="${OPENDRAY_DEPLOY_BIN_PATH:-/usr/local/bin/opendray}"
OPENDRAY_DEPLOY_SERVICE="${OPENDRAY_DEPLOY_SERVICE:-opendray.service}"
OPENDRAY_HEALTH_URL="${OPENDRAY_HEALTH_URL:-http://127.0.0.1:8640/api/health}"

# Tool locations.
FLUTTER_HOME="${FLUTTER_HOME:-$HOME/flutter}"
GO_BIN="${GO_BIN:-/usr/local/go/bin/go}"

# Android build tooling — Flutter reads SDK/JDK from ~/.config/flutter/settings,
# but the Gradle task spawned during `flutter build apk` needs JAVA_HOME set
# in its environment. task-runner spawns with a minimal env, so we must
# re-export here rather than rely on login shell rc files.
JAVA_HOME="${JAVA_HOME:-$HOME/opt/java}"
ANDROID_SDK_ROOT="${ANDROID_SDK_ROOT:-$HOME/Android/Sdk}"
ANDROID_HOME="${ANDROID_HOME:-$ANDROID_SDK_ROOT}"
export JAVA_HOME ANDROID_SDK_ROOT ANDROID_HOME

# Behaviour flags.
NO_BUMP="${NO_BUMP:-0}"

export PATH="$JAVA_HOME/bin:$FLUTTER_HOME/bin:$(dirname "$GO_BIN"):$PATH"

# ── Pretty logging ────────────────────────────────────────────────────
phase()  { printf '\n\033[1;34m▶ %s\033[0m\n' "$*"; }
ok()     { printf '  \033[1;32m✓\033[0m %s\n' "$*"; }
warn()   { printf '  \033[1;33m!\033[0m %s\n' "$*" >&2; }
fail()   { printf '  \033[1;31m✗ %s\033[0m\n' "$*" >&2; exit "${2:-1}"; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required tool not found: $1" 2
}

# Sudo only when we're not root (typical LXC case: you ARE root).
if [[ $EUID -eq 0 ]]; then SUDO=""; else SUDO="sudo"; fi

# ── 0. Preflight UNAS (MUST succeed or we abort before building) ─────
# NOTE: we use a pure-Python SMB client (scripts/unas_upload.py, backed
# by the `smbprotocol` pip package) instead of the system `smbclient`
# binary — the syz LXC sandbox blocks `sudo apt install samba-client`,
# so we picked a library that installs cleanly via `pip install --user`.
phase "Preflight UNAS connectivity"
require_cmd python3
require_cmd curl
require_cmd "$GO_BIN"
[[ -x "$FLUTTER_HOME/bin/flutter" ]] \
  || fail "flutter not found at $FLUTTER_HOME/bin" 2
[[ -x "$JAVA_HOME/bin/java" ]] \
  || fail "java not found at $JAVA_HOME/bin/java (need JDK 17 for APK build)" 2
[[ -d "$ANDROID_SDK_ROOT/platforms" ]] \
  || fail "Android SDK platforms dir missing at $ANDROID_SDK_ROOT/platforms" 2

python3 -c 'import smbprotocol' 2>/dev/null \
  || fail "smbprotocol not installed. run: pip3 install --user smbprotocol" 2

if [[ -z "${UNAS_PASSWORD:-}" ]]; then
  fail "UNAS_PASSWORD not set. Options:
    (a) echo 'PASSWORD' > ~/.config/opendray/unas.pw && chmod 600 ~/.config/opendray/unas.pw
    (b) export UNAS_PASSWORD=...
    (c) add UNAS_PASSWORD=... to scripts/deploy.env" 1
fi

UNAS_REL_DIR="${UNAS_PROJECT}/android"
UNAS_PY="$SCRIPT_DIR/unas_upload.py"
[[ -f "$UNAS_PY" ]] || fail "unas_upload.py missing at $UNAS_PY" 1

# Wrapper: feed password over stdin, never argv (argv leaks to /proc/*/cmdline).
unas() {
  printf '%s' "$UNAS_PASSWORD" \
    | python3 "$UNAS_PY" "$@" "$UNAS_SERVER" "$UNAS_SHARE" "$UNAS_USER"
}
# NB: unas_upload.py's argv order is <cmd> <server> <share> <user> [args...]
# so we reshape here.
unas_ping()  { printf '%s' "$UNAS_PASSWORD" | python3 "$UNAS_PY" ping "$UNAS_SERVER" "$UNAS_SHARE" "$UNAS_USER"; }
unas_mkdir() { printf '%s' "$UNAS_PASSWORD" | python3 "$UNAS_PY" mkdir "$UNAS_SERVER" "$UNAS_SHARE" "$UNAS_USER" "$1"; }
unas_put()   { printf '%s' "$UNAS_PASSWORD" | python3 "$UNAS_PY" put "$UNAS_SERVER" "$UNAS_SHARE" "$UNAS_USER" "$1" "$2"; }
unas_size()  { printf '%s' "$UNAS_PASSWORD" | python3 "$UNAS_PY" size "$UNAS_SERVER" "$UNAS_SHARE" "$UNAS_USER" "$1" "$2"; }

if ! unas_ping >/dev/null 2>&1; then
  fail "cannot reach UNAS at //$UNAS_SERVER/$UNAS_SHARE as $UNAS_USER — check network, credentials, or share permissions.
  Backend was NOT touched; nothing built." 6
fi
ok "UNAS reachable at //$UNAS_SERVER/$UNAS_SHARE"

if ! unas_mkdir "$UNAS_REL_DIR" >/dev/null 2>&1; then
  fail "cannot create/access UNAS target dir $UNAS_REL_DIR" 6
fi
ok "UNAS target dir $UNAS_REL_DIR ready"

# ── 1. Preflight local ───────────────────────────────────────────────
phase "Preflight local"
require_cmd systemctl
[[ -d app ]] || fail "app/ not found — run from opendray repo root" 1
[[ -d cmd/opendray ]] || fail "cmd/opendray not found" 1
ok "tools + layout ok"

# ── 2. Flutter web bundle (Go embed input) ───────────────────────────
phase "Flutter web bundle"
( cd app && "$FLUTTER_HOME/bin/flutter" pub get >/dev/null )
( cd app && "$FLUTTER_HOME/bin/flutter" build web --release ) \
  || fail "flutter build web failed" 3
ok "app/build/web ready"

# ── 3. Go binary ─────────────────────────────────────────────────────
phase "Go binary (linux/amd64, static)"
BIN_OUT="$REPO_ROOT/bin/opendray-linux-amd64"
mkdir -p "$(dirname "$BIN_OUT")"
VERSION_YAML="$(cd app && grep -E '^version:' pubspec.yaml | awk '{print $2}')"
BUILD_SHA="$(git rev-parse --short HEAD 2>/dev/null || echo dev)"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 "$GO_BIN" build \
  -trimpath \
  -ldflags "-s -w -X main.version=${VERSION_YAML} -X main.buildSha=${BUILD_SHA}" \
  -o "$BIN_OUT" \
  ./cmd/opendray \
  || fail "go build failed" 3
BIN_SIZE="$(du -h "$BIN_OUT" | cut -f1)"
ok "built $BIN_OUT ($BIN_SIZE, $VERSION_YAML@$BUILD_SHA)"

# ── 4. Flutter APK (bump build number, release) ──────────────────────
phase "Flutter APK (release)"
cd app

# If the repo was ever rsync'd from a Mac dev box, three artefacts
# survive with Homebrew / /Users paths baked in and cross-contaminate
# every subsequent Linux build:
#
#   android/local.properties       (sdk.dir + flutter.sdk)
#   .flutter-plugins-dependencies  (plugin source paths → Mac pub-cache)
#   .dart_tool/                    (compiled Dart hooks with Mac paths)
#
# `flutter pub get` PRESERVES existing entries in the plugins file
# rather than regenerating them, which leaks Mac paths into every
# build it touches (including back into local.properties). The only
# reliable escape is a full `flutter clean` — then pub get rebuilds
# everything from scratch using ~/.config/flutter/settings.
#
# Trade-off: every run discards build/ too, so incremental builds are
# gone. Worth it for correctness; first build after clean is the slow
# one, subsequent runs in the same session reuse Gradle's daemon cache.
rm -f android/local.properties
"$FLUTTER_HOME/bin/flutter" clean >/dev/null

CURRENT="$(grep -E '^version:' pubspec.yaml | awk '{print $2}')"
VERSION="${CURRENT%+*}"
BUILD="${CURRENT#*+}"

if [[ "$NO_BUMP" != "1" ]]; then
  BUILD=$((BUILD + 1))
  NEXT="${VERSION}+${BUILD}"
  sed -i "s/^version: .*/version: ${NEXT}/" pubspec.yaml
  ok "version: ${CURRENT} → ${NEXT}"
else
  NEXT="$CURRENT"
  ok "version: ${CURRENT} (no bump)"
fi

BUILD_DATE="$(date +%Y%m%d)"
"$FLUTTER_HOME/bin/flutter" pub get >/dev/null

# MUTAGEN RACE FIX — the opendray repo is live-synced from a Mac via
# mutagen-agent, which re-writes android/local.properties with Mac
# paths (Homebrew Flutter, /Users/... Android SDK) any time the Mac
# side's file differs from ours. `flutter pub get` + `flutter clean`
# write the correct Linux paths, then mutagen overwrites them a few
# hundred ms later. Gradle then reads Mac paths → build dies with
# "Included build '/opt/homebrew/share/flutter/...' does not exist".
#
# We win the race by force-writing local.properties IMMEDIATELY
# before `flutter build apk` fires — Gradle's settings.gradle.kts
# reads it in the first ~300ms, long before mutagen's next sync cycle.
# Subsequent mutagen overwrites don't matter: Gradle doesn't re-read
# the file during the build.
#
# Permanent fix lives on the Mac: add these globs to mutagen sync
# ignores so machine-local artefacts stop travelling across:
#   app/android/local.properties
#   app/.flutter-plugins-dependencies
#   app/.dart_tool/
#   app/build/
cat > android/local.properties <<EOF
sdk.dir=$ANDROID_SDK_ROOT
flutter.sdk=$FLUTTER_HOME
flutter.buildMode=release
flutter.versionName=$VERSION
flutter.versionCode=$BUILD
EOF

"$FLUTTER_HOME/bin/flutter" build apk --release \
  --dart-define=BUILD_DATE="$BUILD_DATE" \
  || { cd "$REPO_ROOT"; fail "flutter build apk failed (Android SDK configured?)" 3; }

SRC_APK="build/app/outputs/flutter-apk/app-release.apk"
[[ -f "$SRC_APK" ]] || { cd "$REPO_ROOT"; fail "APK not produced at $SRC_APK" 3; }

# Filename per upload-unas skill convention:
#   <Project>-v<X.Y.Z>+<build>-<yyyyMMdd>.apk
STAMPED="${UNAS_PROJECT}-v${NEXT}-${BUILD_DATE}.apk"
APK_PATH="$REPO_ROOT/app/build/app/outputs/flutter-apk/$STAMPED"
cp "$SRC_APK" "$APK_PATH"
LOCAL_SIZE="$(stat -c %s "$APK_PATH")"
APK_SIZE_H="$(du -h "$APK_PATH" | cut -f1)"
ok "APK: $STAMPED ($APK_SIZE_H, $LOCAL_SIZE bytes)"
cd "$REPO_ROOT"

# ── 5. UNAS upload (with retry + size verification) ──────────────────
phase "Upload to UNAS"
APK_NAME="$(basename "$APK_PATH")"
APK_DIR="$(dirname "$APK_PATH")"

# unas_put self-verifies (re-stats the remote file after upload and
# aborts with exit 1 on size mismatch) and prints the remote byte
# count to stdout on success.
UPLOAD_OK=0
REMOTE_SIZE=""
for attempt in 1 2 3; do
  if REMOTE_SIZE="$(unas_put "$UNAS_REL_DIR" "$APK_PATH")"; then
    UPLOAD_OK=1
    break
  fi
  warn "upload attempt $attempt failed; retrying in $((attempt * 3))s"
  sleep $((attempt * 3))
done

if [[ "$UPLOAD_OK" != "1" ]]; then
  fail "UNAS put failed after 3 attempts — backend NOT deployed. APK is still at $APK_PATH for manual upload." 5
fi

# Double-check with an independent size call (defensive — catches races
# where the put reported success but the remote file was replaced out
# from under us between our upload and this check).
REMOTE_SIZE_CHECK="$(unas_size "$UNAS_REL_DIR" "$APK_NAME")" \
  || fail "UNAS listing lost $APK_NAME after put — Backend NOT deployed." 5

if [[ "$REMOTE_SIZE_CHECK" != "$LOCAL_SIZE" ]]; then
  fail "size mismatch on re-check: local=$LOCAL_SIZE UNAS=$REMOTE_SIZE_CHECK — Backend NOT deployed." 5
fi
ok "verified on UNAS: $APK_NAME ($REMOTE_SIZE_CHECK bytes match local)"
ok "path: //$UNAS_SERVER/$UNAS_SHARE/$UNAS_REL_DIR/$APK_NAME"

# ── 6. Deploy backend (detached — task-runner WILL be killed) ────────
phase "Deploy backend (detached restart)"
TMP_TARGET="${OPENDRAY_DEPLOY_BIN_PATH}.new"

# Stage the new binary now while we're still alive.
$SUDO install -m 0755 "$BIN_OUT" "$TMP_TARGET" \
  || fail "failed to stage new binary at $TMP_TARGET" 4
ok "staged at $TMP_TARGET"

# The detached worker sleeps briefly (so this script can return to
# task-runner with exit 0), stops the service (which kills task-runner),
# atomically swaps the binary, restarts, and runs health-check — all
# appended to a timestamped log.
LOG="/tmp/opendray-deploy-$(date +%Y%m%dT%H%M%S).log"

nohup bash -c "
  exec >>'$LOG' 2>&1
  set -eo pipefail
  echo '[deploy] '\$(date -Is)' — detached worker starting'
  sleep 2   # let task-runner return exit 0 first
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
  echo '[deploy] '\$(date -Is)' — HEALTH FAILED — journalctl -u $OPENDRAY_DEPLOY_SERVICE -n 200'
  exit 1
" </dev/null >/dev/null 2>&1 &
DEPLOY_PID=$!
disown "$DEPLOY_PID"
ok "detached restart worker forked (pid $DEPLOY_PID)"
ok "log: $LOG"
warn "task-runner will lose connection shortly — reconnect the app, then:"
warn "  tail -f $LOG"

phase "Done"
ok "APK shipped: $UNAS_REL_DIR/$APK_NAME"
ok "backend restart: running in background, see log above"
