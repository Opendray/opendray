#!/usr/bin/env bash
# verify-deploy.sh — sanity-check a freshly-deployed OpenDray backend.
#
# "I redeployed but nothing changed" almost always means a cached
# binary. Hits /api/health + /api/providers and prints buildTime +
# tier-1 migration status so the operator can tell at a glance whether
# the right image is live.
#
# Usage:
#   scripts/verify-deploy.sh <url>
#   OPENDRAY_URL=<url> scripts/verify-deploy.sh
#   scripts/verify-deploy.sh --help
#
# Degrades gracefully without jq — falls back to python3 or grep.

usage() {
  cat <<'USAGE'
usage: verify-deploy.sh [url]
       OPENDRAY_URL=<url> verify-deploy.sh

Arguments:
  url   OpenDray base URL, e.g. http://192.168.3.21:8600

Examples:
  scripts/verify-deploy.sh http://192.168.3.21:8600
  OPENDRAY_URL=http://192.168.3.21:8600 scripts/verify-deploy.sh
USAGE
}

case "${1:-}" in
  -h|--help|help) usage; exit 0 ;;
esac

URL="${1:-${OPENDRAY_URL:-}}"
if [[ -z "${URL}" ]]; then
  echo "error: backend URL required" >&2
  echo >&2
  usage >&2
  exit 2
fi
URL="${URL%/}"

# Pick a JSON extractor. jq is best; python3 is almost always present;
# grep is the last resort and handles the three specific fields we need.
json_get() {
  local field="$1" body="$2"
  if command -v jq >/dev/null 2>&1; then
    printf '%s' "${body}" | jq -r --arg k "${field}" '.[$k] // "—"'
    return
  fi
  if command -v python3 >/dev/null 2>&1; then
    printf '%s' "${body}" | python3 -c "
import json, sys
try:
    d = json.load(sys.stdin)
    print(d.get('${field}', '—'))
except Exception:
    print('—')
"
    return
  fi
  # grep fallback — handles flat top-level keys only.
  printf '%s' "${body}" | \
    grep -oE "\"${field}\"[[:space:]]*:[[:space:]]*\"[^\"]*\"" | \
    sed -E "s/.*:[[:space:]]*\"([^\"]*)\"/\1/" | head -n1 || echo "—"
}

# providers is a JSON array — we need a per-name filter.
provider_publisher() {
  local name="$1" body="$2"
  if command -v jq >/dev/null 2>&1; then
    printf '%s' "${body}" | jq -r \
      --arg n "${name}" \
      '.[] | select(.provider.name == $n) | .provider.publisher // ""' | head -n1
    return
  fi
  if command -v python3 >/dev/null 2>&1; then
    printf '%s' "${body}" | python3 -c "
import json, sys
try:
    arr = json.load(sys.stdin)
    for e in arr:
        p = e.get('provider', {})
        if p.get('name') == '${name}':
            print(p.get('publisher', ''))
            sys.exit(0)
    print('')
except Exception:
    print('')
"
    return
  fi
  echo ""
}

section() { printf '\n\033[1m── %s ──\033[0m\n' "$1"; }
kv()      { printf '  %-28s %s\n' "$1" "$2"; }

section "GET ${URL}/api/health"
health="$(curl -fsS --connect-timeout 5 "${URL}/api/health" 2>&1)"
curl_rc=$?
if [[ ${curl_rc} -ne 0 ]]; then
  echo "  HEALTH CHECK FAILED (curl exit ${curl_rc}):"
  echo "  ${health}"
  exit 1
fi

# Sanity: is it even JSON? If not, print the raw body and bail.
if [[ "${health:0:1}" != "{" ]]; then
  echo "  Non-JSON response:"
  echo "  ${health}"
  exit 1
fi

kv "status"    "$(json_get status "${health}")"
kv "version"   "$(json_get version "${health}")"
kv "buildSha"  "$(json_get buildSha "${health}")"
kv "buildTime" "$(json_get buildTime "${health}")"
kv "sessions"  "$(json_get sessions "${health}")"
kv "plugins"   "$(json_get plugins "${health}")"

bt="$(json_get buildTime "${health}")"
if [[ "${bt}" == "unknown" || "${bt}" == "—" ]]; then
  echo
  echo "  ⚠  buildTime is '${bt}' — this binary was NOT built via"
  echo "     'make release-linux' (or an older pre-buildTime build)."
  echo "     Recompile + redeploy before trusting the rest of this check."
fi

section "M5 Phase 5 — tier-1 plugins publisher check"
providers="$(curl -fsS --connect-timeout 5 "${URL}/api/providers" 2>&1)"
curl_rc=$?
if [[ ${curl_rc} -ne 0 ]] || [[ "${providers:0:1}" != "[" ]]; then
  echo "  /api/providers unavailable (curl=${curl_rc}) — may require auth."
  echo "  Skipping tier-1 check."
else
  for name in terminal file-browser claude; do
    pub="$(provider_publisher "${name}" "${providers}")"
    if [[ "${pub}" == "opendray-builtin" ]]; then
      printf '  %-14s  ✓ v1 migrated (publisher=%s)\n' "${name}" "${pub}"
    elif [[ -z "${pub}" ]]; then
      printf '  %-14s  ✗ NOT migrated (publisher empty — old plugins embed)\n' "${name}"
    else
      printf '  %-14s  ?  publisher=%s (unexpected)\n' "${name}" "${pub}"
    fi
  done
fi

section "Interpretation"
echo "  • buildTime should differ from the last deploy — if it's the same,"
echo "    the binary wasn't actually recompiled."
echo "  • All three tier-1 plugins should show publisher=opendray-builtin."
echo "  • If either fails: check your deploy path (old binary copied, or"
echo "    'go build' without ldflags). The M5 changes live in git; the"
echo "    question is whether the running binary has them compiled in."
