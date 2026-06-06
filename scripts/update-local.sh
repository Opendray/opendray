#!/usr/bin/env bash
# scripts/update-local.sh — a local "opendray update" for UNRELEASED code.
#
# `opendray update` pulls a published GitHub release. When you've changed
# opendray locally but haven't cut a release, there is nothing to pull. This
# rebuilds the React SPA + the Go binary from THIS checkout and installs it
# over the running install (~/.opendray/bin/opendray) — byte-for-byte the
# artifact that `install-macos.sh --from-source` produces (web embedded via
# go:embed).
#
# Mirrors the mobile build_release.sh idea: one command to ship local code.
#
# Usage:
#   scripts/update-local.sh             # build + install (does NOT restart)
#   scripts/update-local.sh --restart   # build + install + restart the gateway
#
# NOTE: a restart drops every live session — including any cloud-agent
# session driving the very terminal you launched this from. That is why the
# restart is opt-in and lives in its own "Restart backend" task.
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="${OPENDRAY_BIN:-$HOME/.opendray/bin/opendray}"
RESTART=0
[ "${1:-}" = "--restart" ] && RESTART=1

command -v pnpm >/dev/null 2>&1 || { echo "✗ pnpm not found (needed for the web build)"; exit 1; }
command -v go   >/dev/null 2>&1 || { echo "✗ go not found"; exit 1; }

echo "==> [1/2] building web SPA (gets embedded into the binary via go:embed)"
( cd "$REPO" && pnpm --filter web build )

COMMIT="$(git -C "$REPO" rev-parse --short HEAD 2>/dev/null || echo local)"
echo "==> [2/2] building opendray (local-$COMMIT) -> $BIN"
mkdir -p "$(dirname "$BIN")"
( cd "$REPO" && go build -trimpath \
    -ldflags="-s -w -X github.com/opendray/opendray-v2/internal/version.Version=local-$COMMIT -X github.com/opendray/opendray-v2/internal/version.Commit=$COMMIT" \
    -o "$BIN" ./cmd/opendray )

echo "✓ installed $BIN"
if [ "$RESTART" = 1 ]; then
  echo "==> restarting gateway (drops live sessions)…"
  "$BIN" restart
else
  echo "↻ not restarted — run the 'Restart backend' task (or 'opendray restart') to go live."
fi
