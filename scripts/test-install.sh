#!/usr/bin/env bash
# test-install.sh — simulates a fresh one-click install on this machine.
#
# Clears any lingering setup state, launches the freshly-built binary in
# setup mode under a throwaway config path, and lets you walk through
# the browser wizard. On completion, tears itself down cleanly.
#
# Usage:
#   ./scripts/test-install.sh            # uses bin/opendray-<os>-<arch>
#   OPENDRAY_NO_BROWSER=1 ./scripts/...   # print URL only, don't launch
#
set -euo pipefail

cd "$(dirname "$0")/.."

OS=$(go env GOOS)
ARCH=$(go env GOARCH)
BIN="bin/opendray-${OS}-${ARCH}"

if [[ ! -x "$BIN" ]]; then
  echo "→ binary not found, building: $BIN"
  make release-all >/dev/null
fi

# Kill any existing OpenDray on :8640 — both make dev-backend and a
# stale test run land there.
pkill -f "opendray" 2>/dev/null || true
sleep 1

# Throwaway location for this test cycle. Real users would just get
# ~/.opendray/config.toml; we use /tmp so repeated runs are isolated.
CONFIG=/tmp/opendray-test/config.toml
DATA=/tmp/opendray-test
rm -rf "$DATA" ~/.opendray/setup-token
mkdir -p "$DATA"

cat <<'EOS'
╔═══════════════════════════════════════════════════════════════╗
║   OpenDray install test                                       ║
║                                                               ║
║   Launching the binary in setup mode.                         ║
║   Your browser should open to the wizard automatically.       ║
║                                                               ║
║   Follow the wizard to completion, then Ctrl-C here to stop.  ║
║   Re-run this script to test again with a clean slate.        ║
╚═══════════════════════════════════════════════════════════════╝
EOS
echo ""

# env -i wipes the environment so stray DB_HOST / JWT_SECRET from your
# dev shell don't short-circuit the wizard. We pass PATH and HOME so
# the binary can still find npm / write config, and OPENDRAY_CONFIG so
# the test writes to /tmp instead of touching ~/.opendray/config.toml.
exec env -i \
  PATH="$PATH" \
  HOME="$HOME" \
  OPENDRAY_CONFIG="$CONFIG" \
  ${OPENDRAY_NO_BROWSER:+OPENDRAY_NO_BROWSER="$OPENDRAY_NO_BROWSER"} \
  "$BIN"
