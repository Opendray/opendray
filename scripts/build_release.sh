#!/usr/bin/env bash
# build_release.sh — APK-only release: bump version → build → upload to UNAS.
#
# This is the mobile-app release path. It does NOT touch the backend.
# Use scripts/deploy_release.sh when you also need to ship a new
# gateway binary to syz LXC.
#
# UNAS upload goes through scripts/unas_upload.py (pure-Python
# smbprotocol client) because the syz LXC sandbox blocks `sudo apt
# install samba-client`. Bootstrap once per host:
#     pip3 install --user smbprotocol
#
# Phases:
#   0. Preflight UNAS    — connectivity + credentials + target dir
#   1. Preflight local   — flutter, java, android sdk
#   2. Bump pubspec      — increments the +BUILD suffix (unless NO_BUMP=1)
#   3. Flutter APK       — clean build, release mode
#   4. UNAS upload       — put + size-verify (3 retries)
#
# Exit codes:
#   0   success
#   1   missing credential / env
#   2   required tool not found
#   3   build failure
#   5   UNAS upload / verify failure
#   6   UNAS unreachable at preflight (nothing built)
#
# Env knobs (also read from scripts/deploy.env if present):
#   UNAS_PASSWORD      — required; falls back to ~/.config/opendray/unas.pw
#   UNAS_SERVER        — default 192.168.9.8
#   UNAS_SHARE         — default Claude_Workspace
#   UNAS_USER          — default linivek
#   UNAS_PROJECT       — default OpenDray  (UNAS dir: <PROJECT>/android)
#   FLUTTER_HOME       — default $HOME/flutter
#   JAVA_HOME          — default $HOME/opt/java
#   ANDROID_SDK_ROOT   — default $HOME/Android/Sdk
#   NO_BUMP=1          — skip pubspec.yaml build-number increment
#   NO_UPLOAD=1        — build only; skip the UNAS upload (useful for
#                        local smoke tests — the APK lands at
#                        app/build/app/outputs/flutter-apk/)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

# ── Load optional config ──────────────────────────────────────────────
# Shared with deploy_release.sh — UNAS creds live there.
if [[ -f "$SCRIPT_DIR/deploy.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$SCRIPT_DIR/deploy.env"
  set +a
fi

# ── Defaults ──────────────────────────────────────────────────────────
UNAS_SERVER="${UNAS_SERVER:-192.168.9.8}"
UNAS_SHARE="${UNAS_SHARE:-Claude_Workspace}"
UNAS_USER="${UNAS_USER:-linivek}"
UNAS_PROJECT="${UNAS_PROJECT:-OpenDray}"
if [[ -z "${UNAS_PASSWORD:-}" ]]; then
  PW_FILE="$HOME/.config/opendray/unas.pw"
  if [[ -f "$PW_FILE" && -r "$PW_FILE" ]]; then
    UNAS_PASSWORD="$(tr -d '\n\r' <"$PW_FILE")"
  fi
fi

FLUTTER_HOME="${FLUTTER_HOME:-$HOME/flutter}"
JAVA_HOME="${JAVA_HOME:-$HOME/opt/java}"
ANDROID_SDK_ROOT="${ANDROID_SDK_ROOT:-$HOME/Android/Sdk}"
ANDROID_HOME="${ANDROID_HOME:-$ANDROID_SDK_ROOT}"
export JAVA_HOME ANDROID_SDK_ROOT ANDROID_HOME

NO_BUMP="${NO_BUMP:-0}"
NO_UPLOAD="${NO_UPLOAD:-0}"

export PATH="$JAVA_HOME/bin:$FLUTTER_HOME/bin:$PATH"

TRACE_LOG="/tmp/opendray-build-trace-$(date +%Y%m%dT%H%M%S).log"
exec > >(tee -a "$TRACE_LOG")
exec 2> >(tee -a "$TRACE_LOG" >&2)
printf '[trace] %s — build_release.sh pid %s — log: %s\n' \
  "$(date -Is)" "$$" "$TRACE_LOG"

# ── Pretty logging (matches deploy_release.sh for muscle-memory) ─────
phase()  { printf '\n\033[1;34m▶ %s\033[0m\n' "$*"; }
ok()     { printf '  \033[1;32m✓\033[0m %s\n' "$*"; }
warn()   { printf '  \033[1;33m!\033[0m %s\n' "$*" >&2; }
fail()   {
  printf '\n  \033[1;31m✗ %s\033[0m\n' "$*"
  printf '\n  \033[1;31m✗ %s\033[0m\n' "$*" >&2
  printf '     (exit %s; full trace: %s)\n' "${2:-1}" "$TRACE_LOG"
  exit "${2:-1}"
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required tool not found: $1" 2
}

# ── UNAS wrappers (shared pattern with deploy_release.sh) ────────────
# Password flows over stdin, never argv, so it never hits /proc/*/cmdline.
UNAS_REL_DIR="${UNAS_PROJECT}/android"
UNAS_PY="$SCRIPT_DIR/unas_upload.py"

unas_ping()  { printf '%s' "$UNAS_PASSWORD" | python3 "$UNAS_PY" ping  "$UNAS_SERVER" "$UNAS_SHARE" "$UNAS_USER"; }
unas_mkdir() { printf '%s' "$UNAS_PASSWORD" | python3 "$UNAS_PY" mkdir "$UNAS_SERVER" "$UNAS_SHARE" "$UNAS_USER" "$1"; }
unas_put()   { printf '%s' "$UNAS_PASSWORD" | python3 "$UNAS_PY" put   "$UNAS_SERVER" "$UNAS_SHARE" "$UNAS_USER" "$1" "$2"; }
unas_size()  { printf '%s' "$UNAS_PASSWORD" | python3 "$UNAS_PY" size  "$UNAS_SERVER" "$UNAS_SHARE" "$UNAS_USER" "$1" "$2"; }

# ── 0. Preflight UNAS (skipped when NO_UPLOAD=1) ─────────────────────
if [[ "$NO_UPLOAD" != "1" ]]; then
  phase "Preflight UNAS connectivity"
  require_cmd python3
  [[ -f "$UNAS_PY" ]] || fail "unas_upload.py missing at $UNAS_PY" 1
  python3 -c 'import smbprotocol' 2>/dev/null \
    || fail "smbprotocol not installed. run: pip3 install --user smbprotocol" 2

  if [[ -z "${UNAS_PASSWORD:-}" ]]; then
    fail "UNAS_PASSWORD not set. Options:
    (a) echo 'PASSWORD' > ~/.config/opendray/unas.pw && chmod 600 ~/.config/opendray/unas.pw
    (b) export UNAS_PASSWORD=...
    (c) add UNAS_PASSWORD=... to scripts/deploy.env
    (or rerun with NO_UPLOAD=1 to build locally without uploading)" 1
  fi

  if ! unas_ping >/dev/null 2>&1; then
    fail "cannot reach UNAS at //$UNAS_SERVER/$UNAS_SHARE as $UNAS_USER — check network, credentials, or share permissions." 6
  fi
  ok "UNAS reachable at //$UNAS_SERVER/$UNAS_SHARE"

  if ! unas_mkdir "$UNAS_REL_DIR" >/dev/null 2>&1; then
    fail "cannot create/access UNAS target dir $UNAS_REL_DIR" 6
  fi
  ok "UNAS target dir $UNAS_REL_DIR ready"
else
  warn "NO_UPLOAD=1 — skipping UNAS preflight and upload"
fi

# ── 1. Preflight local ────────────────────────────────────────────────
phase "Preflight local toolchain"
[[ -d app ]] || fail "app/ not found — run from opendray repo root" 1
[[ -x "$FLUTTER_HOME/bin/flutter" ]] \
  || fail "flutter not found at $FLUTTER_HOME/bin" 2
[[ -x "$JAVA_HOME/bin/java" ]] \
  || fail "java not found at $JAVA_HOME/bin/java (need JDK 17 for APK build)" 2
[[ -d "$ANDROID_SDK_ROOT/platforms" ]] \
  || fail "Android SDK platforms dir missing at $ANDROID_SDK_ROOT/platforms" 2
ok "flutter + jdk + android sdk present"

# ── 2. Bump pubspec ───────────────────────────────────────────────────
phase "Version bump"
cd app

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
  ok "version: ${CURRENT} (no bump — NO_BUMP=1)"
fi

# ── 3. Flutter APK (release, Linux-safe clean rebuild) ───────────────
phase "Flutter APK (release)"

# Mutagen-synced repos from a Mac leave local.properties, .dart_tool/,
# and .flutter-plugins-dependencies with Homebrew / /Users paths baked
# in — these poison every subsequent Linux build because `flutter pub
# get` preserves (rather than regenerates) the plugins file. Full
# `flutter clean` + immediate local.properties rewrite is the only
# reliable workaround. See deploy_release.sh for the full post-mortem.
rm -f android/local.properties
"$FLUTTER_HOME/bin/flutter" clean >/dev/null
"$FLUTTER_HOME/bin/flutter" pub get >/dev/null

BUILD_DATE="$(date +%Y%m%d)"

# Write local.properties AFTER pub get and right before the build, so
# any mutagen re-sync racing in the background doesn't stomp on us
# before Gradle reads it.
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

# Stamp the APK per upload-unas skill convention:
#   <Project>-v<X.Y.Z>+<build>-<yyyyMMdd>.apk
STAMPED="${UNAS_PROJECT}-v${NEXT}-${BUILD_DATE}.apk"
APK_PATH="$REPO_ROOT/app/build/app/outputs/flutter-apk/$STAMPED"
cp "$SRC_APK" "$APK_PATH"
LOCAL_SIZE="$(stat -c %s "$APK_PATH")"
APK_SIZE_H="$(du -h "$APK_PATH" | cut -f1)"
ok "APK: $STAMPED ($APK_SIZE_H, $LOCAL_SIZE bytes)"
ok "local path: $APK_PATH"
cd "$REPO_ROOT"

# ── 4. UNAS upload (with retry + size verification) ──────────────────
if [[ "$NO_UPLOAD" == "1" ]]; then
  phase "Done (local-only build)"
  ok "APK ready at $APK_PATH — upload skipped (NO_UPLOAD=1)"
  exit 0
fi

phase "Upload to UNAS"
APK_NAME="$(basename "$APK_PATH")"

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
  fail "UNAS put failed after 3 attempts. APK is at $APK_PATH for manual upload." 5
fi

# Independent size check — defends against a race where a put reports
# success but the remote file gets replaced between put and our check.
REMOTE_SIZE_CHECK="$(unas_size "$UNAS_REL_DIR" "$APK_NAME")" \
  || fail "UNAS listing lost $APK_NAME after put." 5

if [[ "$REMOTE_SIZE_CHECK" != "$LOCAL_SIZE" ]]; then
  fail "size mismatch on re-check: local=$LOCAL_SIZE UNAS=$REMOTE_SIZE_CHECK" 5
fi
ok "verified on UNAS: $APK_NAME ($REMOTE_SIZE_CHECK bytes match local)"
ok "path: //$UNAS_SERVER/$UNAS_SHARE/$UNAS_REL_DIR/$APK_NAME"

phase "Done"
ok "APK shipped: $UNAS_REL_DIR/$APK_NAME"
ok "install on device: pull from //$UNAS_SERVER/$UNAS_SHARE/$UNAS_REL_DIR/"
