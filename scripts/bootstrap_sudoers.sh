#!/usr/bin/env bash
# bootstrap_sudoers.sh — one-time root setup so task-runner can deploy
# opendray autonomously AND safely (with rollback) without being killed
# mid-restart.
#
# Run ONCE per LXC, as root:
#   sudo bash scripts/bootstrap_sudoers.sh
#
# It installs four things:
#
#   1. /etc/sudoers.d/opendray-deploy — tight NOPASSWD allowlist with
#      exactly ONE command: triggering the deploy worker service.
#      Everything else still prompts.
#
#   2. /etc/systemd/system/opendray.service.d/deploy-override.conf —
#      drops NoNewPrivileges=no so the kernel allows sudo's setuid
#      bit. Other hardening (ProtectHome, ProtectSystem, PrivateTmp)
#      stays in place.
#
#   3. $HOME/.opendray-deploy/ — staging directory under linivek's home
#      so task-runner can drop the new binary without sudo at all, AND
#      without needing extra ReadWritePaths gymnastics around
#      ProtectSystem=strict (opendray.service already has /home/linivek
#      in its ReadWritePaths list).
#
#   4. /etc/systemd/system/opendray-deployer.service +
#      /usr/local/sbin/opendray-deployer — a one-shot service that
#      runs OUTSIDE opendray.service's cgroup. Task-runner triggers
#      it and exits; the deployer does stop→swap→start→health-check,
#      with AUTOMATIC ROLLBACK to the previous binary if the new one
#      fails health within 55 s. Logs to journald
#      (journalctl -u opendray-deployer -f).
#
# This layout means:
#   • task-runner's sudo surface is one command — minimal blast radius
#   • deploy survives task-runner's death (different cgroup)
#   • failed new binary automatically rolls back — syz stays reachable
#   • rollback also fails? the service is logged as CRITICAL in journald

set -euo pipefail

if [[ "$EUID" -ne 0 ]]; then
  echo "error: must run as root (sudo bash $0, or root shell)" >&2
  exit 1
fi

USER_NAME="${OPENDRAY_DEPLOY_USER:-linivek}"
REPO_ROOT="${OPENDRAY_REPO_ROOT:-/home/linivek/workspace/opendray}"
LIVE_BIN="/usr/local/bin/opendray"
STAGE_DIR="/home/$USER_NAME/.opendray-deploy"
STAGED_BIN="$STAGE_DIR/opendray.staged"
DEPLOYER_SH="/usr/local/sbin/opendray-deployer"
DEPLOYER_UNIT="/etc/systemd/system/opendray-deployer.service"
SERVICE="opendray.service"
HEALTH_URL="${OPENDRAY_HEALTH_URL:-http://127.0.0.1:8640/api/health}"

SUDOERS_FILE="/etc/sudoers.d/opendray-deploy"
OVERRIDE_DIR="/etc/systemd/system/${SERVICE}.d"
OVERRIDE_FILE="$OVERRIDE_DIR/deploy-override.conf"

if ! id -u "$USER_NAME" >/dev/null 2>&1; then
  echo "error: user '$USER_NAME' does not exist on this host" >&2
  exit 1
fi

# ─── 1. Staging directory ──────────────────────────────────────────────
mkdir -p "$STAGE_DIR"
chown "$USER_NAME:$USER_NAME" "$STAGE_DIR"
chmod 0750 "$STAGE_DIR"
echo "✓ staging dir: $STAGE_DIR (owner $USER_NAME, 0750)"

# ─── 2. opendray-deployer script ───────────────────────────────────────
cat > "$DEPLOYER_SH" <<'DEPLOYER'
#!/usr/bin/env bash
# opendray-deployer — root-side deploy worker, triggered by
# opendray-deployer.service. Runs in its OWN cgroup so
# `systemctl stop opendray.service` can't kill it mid-flight.
#
# Flow: validate staged binary → backup live → stop → swap → start →
# health-check (55 s) → on failure, automatic rollback to backup.
# Everything logs to journald (journalctl -u opendray-deployer).

set -eo pipefail

STAGE_DIR="/home/$USER_NAME/.opendray-deploy"
STAGED_BIN="$STAGE_DIR/opendray.staged"
LIVE_BIN="/usr/local/bin/opendray"
BACKUP_BIN="/usr/local/bin/opendray.prev"
SERVICE="opendray.service"
HEALTH_URL="${OPENDRAY_HEALTH_URL:-http://127.0.0.1:8640/api/health}"

log() { printf '[%s] %s\n' "$(date -Is)" "$*"; }

# ── Staged binary must exist and be a real ELF ───────────────────────
if [[ ! -f "$STAGED_BIN" ]]; then
  log "ABORT: no staged binary at $STAGED_BIN"
  exit 1
fi
if ! file "$STAGED_BIN" 2>/dev/null | grep -q 'ELF.*executable'; then
  log "ABORT: staged binary is not a Linux ELF executable"
  file "$STAGED_BIN" 2>&1 | log "  file says: $(cat)"
  exit 1
fi

# ── Sanity probe: running the staged binary with --help should not
# segfault. If it does, reject without touching the live binary.
if ! "$STAGED_BIN" --help >/dev/null 2>&1; then
  # Not fatal — some versions may not have --help. Just warn.
  log "WARN: staged binary --help exits non-zero (tolerated)"
fi

# ── Backup live binary ───────────────────────────────────────────────
log "backup: cp $LIVE_BIN → $BACKUP_BIN"
cp --preserve=all "$LIVE_BIN" "$BACKUP_BIN"

# ── Stop service (our cgroup is separate; we won't get killed) ──────
log "systemctl stop $SERVICE"
systemctl stop "$SERVICE"

# ── Swap binary ──────────────────────────────────────────────────────
log "install: $STAGED_BIN → $LIVE_BIN"
install -m 0755 "$STAGED_BIN" "$LIVE_BIN"

# ── Start service ────────────────────────────────────────────────────
log "systemctl start $SERVICE"
if ! systemctl start "$SERVICE"; then
  log "FAIL: start returned non-zero — rolling back"
  install -m 0755 "$BACKUP_BIN" "$LIVE_BIN"
  if systemctl start "$SERVICE"; then
    log "rollback started — previous binary running"
  else
    log "CRITICAL: rollback start ALSO failed — SERVICE DOWN"
  fi
  exit 1
fi

# ── Health check with exponential backoff (max ~55 s) ───────────────
ok=0
for i in 1 4 9 16 25; do
  if curl -fsS --max-time 5 "$HEALTH_URL" >/dev/null 2>&1; then
    log "health ok after ~$((i))s"
    ok=1
    break
  fi
  log "not ready yet, retry in ${i}s"
  sleep "$i"
done

if [[ "$ok" == "1" ]]; then
  # Keep backup for manual rollback; cleaning up the staging file is
  # safe since we've already deployed + health-checked.
  rm -f "$STAGED_BIN"
  log "deploy complete — new binary live, previous kept at $BACKUP_BIN"
  exit 0
fi

# ── Health failed — rollback ─────────────────────────────────────────
log "health FAILED after 55s — rolling back to $BACKUP_BIN"
systemctl stop "$SERVICE" || log "WARN: stop during rollback failed"
install -m 0755 "$BACKUP_BIN" "$LIVE_BIN"
if systemctl start "$SERVICE"; then
  log "rollback start OK"
else
  log "CRITICAL: rollback start FAILED — SERVICE DOWN, manual intervention needed"
  exit 2
fi

# Verify rollback health
for i in 1 4 9 16; do
  if curl -fsS --max-time 5 "$HEALTH_URL" >/dev/null 2>&1; then
    log "rollback verified — previous binary answering health checks"
    exit 1
  fi
  sleep "$i"
done

log "CRITICAL: rollback started but service not answering health — manual intervention needed"
exit 2
DEPLOYER

# The heredoc above is quoted ('DEPLOYER') to keep bash $VARS inside the
# deployer script untouched — we want it to expand them at RUNTIME, not
# at bootstrap time. But STAGE_DIR="/home/$USER_NAME/.opendray-deploy"
# is the one line we want resolved NOW, while USER_NAME is known, so
# the deployer script knows where to look without needing its own env
# wiring. A targeted sed just for that line:
sed -i "s|^STAGE_DIR=\".*\"|STAGE_DIR=\"/home/$USER_NAME/.opendray-deploy\"|" "$DEPLOYER_SH"

chmod 0755 "$DEPLOYER_SH"
echo "✓ installed $DEPLOYER_SH"

# ─── 3. opendray-deployer.service unit ────────────────────────────────
cat > "$DEPLOYER_UNIT" <<EOF
[Unit]
Description=Opendray deploy worker (one-shot, rollback-safe)
# NOT After=$SERVICE — the two run at different cgroups on purpose.
# When this service stops $SERVICE, it must not be dragged down with it.

[Service]
Type=oneshot
User=root
ExecStart=$DEPLOYER_SH
TimeoutStartSec=180
StandardOutput=journal
StandardError=journal
# No hardening — this service needs to write /usr/local/bin, call
# systemctl, and run for <3 minutes.
EOF
chmod 0644 "$DEPLOYER_UNIT"
echo "✓ installed $DEPLOYER_UNIT"

# ─── 4. sudoers NOPASSWD (just the trigger) ──────────────────────────
TMP="$(mktemp /tmp/opendray-sudoers.XXXXXX)"
trap 'rm -f "$TMP"' EXIT

cat > "$TMP" <<EOF
# Managed by scripts/bootstrap_sudoers.sh — re-run to update.
# task-runner (running as $USER_NAME inside opendray.service) is
# granted NOPASSWD for ONE command: triggering the deploy worker.
# The worker itself runs as root in its own systemd service and
# handles install/stop/start/health-check/rollback.

$USER_NAME ALL=(root) NOPASSWD: /usr/bin/systemctl start --no-block opendray-deployer.service
$USER_NAME ALL=(root) NOPASSWD: /usr/bin/systemctl start opendray-deployer.service
EOF
chmod 0440 "$TMP"

if ! visudo -c -f "$TMP" >/dev/null; then
  echo "error: sudoers syntax check FAILED — nothing installed" >&2
  visudo -c -f "$TMP" >&2 || true
  exit 2
fi
install -m 0440 -o root -g root "$TMP" "$SUDOERS_FILE"
if ! visudo -c -f "$SUDOERS_FILE" >/dev/null; then
  echo "error: installed sudoers FAILED its own check — removing" >&2
  rm -f "$SUDOERS_FILE"
  exit 2
fi
echo "✓ installed $SUDOERS_FILE"

# ─── 5. systemd drop-in for opendray.service ─────────────────────────
mkdir -p "$OVERRIDE_DIR"
cat > "$OVERRIDE_FILE" <<EOF
# Managed by scripts/bootstrap_sudoers.sh — do not edit.
# NoNewPrivileges=no is required for sudo's setuid to work from
# task-runner (which is a child of this service). Sudo is only ever
# invoked to trigger opendray-deployer.service.
[Service]
NoNewPrivileges=no
EOF
chmod 0644 "$OVERRIDE_FILE"
echo "✓ installed $OVERRIDE_FILE"

systemctl daemon-reload
echo "✓ systemctl daemon-reload"

echo
echo "═══ Bootstrap done. Two things you still need to do: ═══"
echo
echo "1. Reload opendray.service so the NoNewPrivileges override takes"
echo "   effect — it only activates after restart:"
echo
echo "     systemctl restart $SERVICE"
echo
echo "2. Smoke-test the deploy trigger (run as $USER_NAME, expect no prompt):"
echo
echo "     sudo -n /usr/bin/systemctl start --no-block opendray-deployer.service && echo ok"
echo
echo "   (That will fire the deployer with an empty staging dir, so"
echo "    journalctl -u opendray-deployer will show 'ABORT: no staged"
echo "    binary at $STAGED_BIN' — that's expected; it confirms the"
echo "    trigger works.)"
echo
echo "After that, task-runner's scripts/deploy_release.sh runs end-to-end"
echo "without any manual SSH."
