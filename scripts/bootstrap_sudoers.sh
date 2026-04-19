#!/usr/bin/env bash
# bootstrap_sudoers.sh — one-time root setup so task-runner can deploy
# opendray without prompting for a sudo password.
#
# Run ONCE per LXC, as root:
#
#   sudo bash scripts/bootstrap_sudoers.sh
#     (or as root shell: bash scripts/bootstrap_sudoers.sh)
#
# What it does:
#   • Drops /etc/sudoers.d/opendray-deploy with a TIGHT command allowlist
#     covering only the four ops deploy_release.sh needs:
#       install → /usr/local/bin/opendray.new
#       systemctl stop  opendray.service
#       systemctl start opendray.service
#       mv     /usr/local/bin/opendray.new → /usr/local/bin/opendray
#   • chmod 440 (the only mode visudo accepts for sudoers drop-ins)
#   • Validates syntax with `visudo -c` before leaving it in place —
#     syntax errors get rolled back so you can't accidentally lock
#     yourself out of sudo.
#
# Nothing about this grants linivek general root. The four commands
# above and nothing else. Any other sudo invocation still prompts
# for a password as usual.

set -euo pipefail

if [[ "$EUID" -ne 0 ]]; then
  echo "error: must run as root (sudo bash $0, or root shell)" >&2
  exit 1
fi

USER_NAME="${OPENDRAY_DEPLOY_USER:-linivek}"
REPO_ROOT="${OPENDRAY_REPO_ROOT:-/home/linivek/workspace/opendray}"
BIN_SRC="$REPO_ROOT/bin/opendray-linux-amd64"
BIN_STAGING="/usr/local/bin/opendray.new"
BIN_FINAL="/usr/local/bin/opendray"
SERVICE="opendray.service"

SUDOERS_FILE="/etc/sudoers.d/opendray-deploy"

if ! id -u "$USER_NAME" >/dev/null 2>&1; then
  echo "error: user '$USER_NAME' does not exist on this host" >&2
  exit 1
fi

# Generate into a tempfile first; validate with visudo; only then install.
# This is the standard defence against bricking sudo via a syntax error.
TMP="$(mktemp /tmp/opendray-sudoers.XXXXXX)"
trap 'rm -f "$TMP"' EXIT

cat > "$TMP" <<EOF
# Managed by scripts/bootstrap_sudoers.sh — re-run that script to update.
# Purpose: let opendray's task-runner plugin (running as $USER_NAME) deploy
# a new binary + restart the service, without a password prompt. The list
# below is the complete allowlist; any other sudo invocation still prompts.

$USER_NAME ALL=(root) NOPASSWD: /usr/bin/install -m 0755 $BIN_SRC $BIN_STAGING
$USER_NAME ALL=(root) NOPASSWD: /usr/bin/systemctl stop $SERVICE
$USER_NAME ALL=(root) NOPASSWD: /usr/bin/systemctl start $SERVICE
$USER_NAME ALL=(root) NOPASSWD: /usr/bin/systemctl is-active --quiet $SERVICE
$USER_NAME ALL=(root) NOPASSWD: /usr/bin/mv $BIN_STAGING $BIN_FINAL
EOF

chmod 440 "$TMP"

if ! visudo -c -f "$TMP" >/dev/null; then
  echo "error: sudoers syntax check FAILED — nothing installed" >&2
  visudo -c -f "$TMP" >&2 || true
  exit 2
fi

# Move into place atomically with the correct mode + ownership.
install -m 0440 -o root -g root "$TMP" "$SUDOERS_FILE"

# Final re-check on the installed file itself.
if ! visudo -c -f "$SUDOERS_FILE" >/dev/null; then
  echo "error: installed sudoers file FAILED its own check — removing" >&2
  rm -f "$SUDOERS_FILE"
  exit 2
fi

echo "✓ installed $SUDOERS_FILE"
echo "  user: $USER_NAME"
echo "  repo: $REPO_ROOT"
echo "  service: $SERVICE"
echo
echo "Smoke-test (run AS $USER_NAME, expect no password prompt):"
echo "  sudo -n /usr/bin/systemctl is-active --quiet $SERVICE && echo ok"
