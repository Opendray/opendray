#!/usr/bin/env bash
# verify-deploy.sh — sanity-check a freshly-deployed OpenDray backend.
#
# The "I redeployed but nothing changed" problem almost always comes from
# running the old binary (cached image, missed rebuild, wrong service
# reload). This script calls /api/health and /api/providers, then prints
# the exact fields an operator can compare against their expectation.
#
# Usage:
#   scripts/verify-deploy.sh                                # prompts for URL
#   scripts/verify-deploy.sh http://192.168.3.21:8600       # explicit host
#   OPENDRAY_URL=... scripts/verify-deploy.sh               # env override
#
# Requires: curl, jq.
set -euo pipefail

URL="${1:-${OPENDRAY_URL:-}}"
if [[ -z "${URL}" ]]; then
  read -r -p "OpenDray base URL (e.g. http://192.168.3.21:8600): " URL
fi
URL="${URL%/}"

if ! command -v jq >/dev/null 2>&1; then
  echo "error: jq is required — apt install jq / brew install jq" >&2
  exit 2
fi

section() { printf '\n\033[1m── %s ──\033[0m\n' "$1"; }
kv()      { printf '  %-28s %s\n' "$1" "$2"; }

section "GET ${URL}/api/health"
health=$(curl -fsS "${URL}/api/health" 2>&1 || true)
if [[ -z "${health}" ]] || ! echo "${health}" | jq . >/dev/null 2>&1; then
  echo "  HEALTH CHECK FAILED:"
  echo "  ${health}"
  exit 1
fi
kv "status"    "$(jq -r '.status // "—"'    <<<"${health}")"
kv "version"   "$(jq -r '.version // "—"'   <<<"${health}")"
kv "buildSha"  "$(jq -r '.buildSha // "—"'  <<<"${health}")"
kv "buildTime" "$(jq -r '.buildTime // "—"' <<<"${health}")"
kv "sessions"  "$(jq -r '.sessions // "—"'  <<<"${health}")"
kv "plugins"   "$(jq -r '.plugins // "—"'   <<<"${health}")"

# If buildTime is "unknown", the binary was built without the ldflags
# stamp — someone ran `go build` instead of `make release-linux`. Flag it
# loudly so the operator doesn't waste time hunting.
bt=$(jq -r '.buildTime // ""' <<<"${health}")
if [[ "${bt}" == "unknown" || -z "${bt}" ]]; then
  echo
  echo "  ⚠  buildTime is 'unknown' — this binary was NOT built via"
  echo "     'make release-linux'. Re-run the release Makefile target to"
  echo "     stamp version/buildSha/buildTime, or your deploy may still be"
  echo "     running the previous image."
fi

section "M5 Phase 5 migration check — tier-1 plugins"
providers=$(curl -fsS "${URL}/api/providers" 2>&1 || true)
if ! echo "${providers}" | jq . >/dev/null 2>&1; then
  echo "  /api/providers returned non-JSON — skipping tier-1 check"
  echo "  (are you logged in? this endpoint may require auth)"
else
  for name in terminal file-browser claude; do
    pub=$(echo "${providers}" | jq -r \
      --arg n "${name}" \
      '.[] | select(.provider.name == $n) | .provider.publisher // ""')
    if [[ "${pub}" == "opendray-builtin" ]]; then
      printf '  %-14s  ✓ v1 migrated (publisher=%s)\n' "${name}" "${pub}"
    elif [[ -z "${pub}" ]]; then
      printf '  %-14s  ✗ NOT migrated (publisher empty — running old manifest)\n' "${name}"
    else
      printf '  %-14s  ?  publisher=%s (unexpected)\n' "${name}" "${pub}"
    fi
  done
fi

section "Next steps"
echo "  If buildTime matches what your CI/deploy pipeline stamped, the"
echo "  right binary is running. Compare against the previous deploy's"
echo "  buildTime to prove a successful recompile."
echo
echo "  If a tier-1 plugin still shows publisher empty, the 'plugins' embed"
echo "  shipped with the binary predates M5 A1/A2/A3 — rebuild from HEAD."
