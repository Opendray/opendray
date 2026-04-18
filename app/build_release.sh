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
echo "build date: ${BUILD_DATE}"

flutter pub get

echo "→ building APK (release)"
flutter build apk --release \
  --dart-define=BUILD_DATE="${BUILD_DATE}"

if [[ "$APK_ONLY" == "0" ]]; then
  echo "→ building IPA (release, no codesign)"
  flutter build ipa --release --no-codesign \
    --dart-define=BUILD_DATE="${BUILD_DATE}"
fi

echo "done: ${NEXT}"
