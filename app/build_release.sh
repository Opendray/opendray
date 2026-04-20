#!/usr/bin/env bash
# Build OpenDray Flutter app (Android APK + iOS IPA).
#
# Behavior:
#   • Reads version from pubspec.yaml (format: X.Y.Z+BUILD)
#   • Auto-increments BUILD number and writes it back
#   • Stamps BUILD_DATE (YYYY-MM-DD) into the binary via --dart-define
#   • Builds APK (release) and IPA (release, no codesign by default)
#
# Usage:
#   ./build_release.sh                # bump build, build apk + ipa
#   ./build_release.sh --apk-only     # skip ipa
#   ./build_release.sh --no-bump      # don't increment build number
set -euo pipefail

APK_ONLY=0
BUMP=1
for arg in "$@"; do
  case "$arg" in
    --apk-only) APK_ONLY=1 ;;
    --no-bump)  BUMP=0 ;;
    *) echo "unknown arg: $arg" >&2; exit 2 ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

PUBSPEC="pubspec.yaml"
CURRENT=$(grep -E '^version:' "$PUBSPEC" | awk '{print $2}')
NAME="${CURRENT%+*}"
BUILD="${CURRENT#*+}"

if [[ "$BUMP" == "1" ]]; then
  BUILD=$((BUILD + 1))
  NEXT="${NAME}+${BUILD}"
  # macOS sed requires -i ''
  sed -i '' "s/^version: .*/version: ${NEXT}/" "$PUBSPEC"
  echo "version: ${CURRENT} → ${NEXT}"
else
  NEXT="$CURRENT"
  echo "version: ${CURRENT} (no bump)"
fi

BUILD_DATE=$(date +%Y-%m-%d)
# UTC ISO8601-basic timestamp — stamped alongside BUILD_DATE so two APKs
# built on the same day (same pubspec version, same build number) still
# render distinct labels in Settings → About. Compare timestamps to tell
# "the APK I pushed 5 minutes ago" from "the APK that was there before".
BUILD_TIMESTAMP=$(date -u '+%Y%m%dT%H%M%SZ')
echo "build date: ${BUILD_DATE}"
echo "build timestamp: ${BUILD_TIMESTAMP}"

flutter pub get

echo "→ building APK (release)"
flutter build apk --release \
  --dart-define=BUILD_DATE="${BUILD_DATE}" \
  --dart-define=BUILD_TIMESTAMP="${BUILD_TIMESTAMP}"

if [[ "$APK_ONLY" == "0" ]]; then
  echo "→ building IPA (release, no codesign)"
  flutter build ipa --release --no-codesign \
    --dart-define=BUILD_DATE="${BUILD_DATE}" \
    --dart-define=BUILD_TIMESTAMP="${BUILD_TIMESTAMP}"
fi

echo "done: ${NEXT} @ ${BUILD_TIMESTAMP}"
