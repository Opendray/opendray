# Restart

Most Server settings require the gateway process to restart
before they take effect — `listen` rebinds a socket, `log.level`
reconfigures the slog handler, idle thresholds are read once at
startup, etc. The bottom of every Server section has a **Restart
server** button that handles this in-place.

## What happens when you click Restart

1. Browser POSTs `/api/v1/admin/restart`.
2. Gateway returns `202 Accepted` then waits 500 ms (so the
   response can flush).
3. Gateway calls `syscall.Exec(os.Executable(), os.Args, …)` —
   the running process is **replaced** with a fresh execution
   of the same binary. PID stays the same; the listening socket
   drops momentarily and rebinds.
4. The browser shows a full-screen overlay with a tick counter.
5. Every 1 s the browser polls `/api/v1/health`. First success →
   "Server restarted" toast, page auto-reloads.
6. Timeout after 30 s → toast error, overlay clears so you can
   investigate manually.

Total downtime: **~1-3 seconds** in normal cases.

## Confirm dialogs

Two guards trigger before the restart actually fires:

- **Unsaved changes**: "Restart will use the LAST SAVED config.
  Continue?" — if you typed something but didn't click Save,
  it's discarded. Click cancel, hit Save, then Restart.
- **Generic confirm**: "Restart the opendray gateway? All open
  terminal sessions will reconnect automatically." — final OK
  before the exec.

Both confirms come up regardless of which Server section
triggered the restart.

## What survives a restart

- **Database state** — Postgres is external; sessions, audit
  log, channels, integrations, cliacct rows all persist.
- **Active terminal sessions' DB rows** — they get reconciled to
  `ended` on startup since the PTY is gone, but the session ID
  + tab survives. You hit **Start** to respawn.
- **Channel webhooks & integrations** — their DB-backed state
  survives, the channels reconnect on startup.
- **Browser state** — the auto-reload preserves your route,
  bearer token, and theme.

## What gets lost

- **Live log tail buffer** — the in-memory ring is cleared.
  The first records after restart appear empty for a moment.
- **Subscriber WebSockets** — events viewer, log stream,
  session terminal streams all drop and auto-reconnect.
- **`config.toml` comments** — every Save through the UI uses
  the BurntSushi/toml encoder, which doesn't preserve comments.
  If you keep careful comments, edit the file by hand and skip
  the UI Save for that section.

## Restart vs `pkill` + relaunch

Functionally identical — the self-exec path is just a
convenience. If your launch wrapper has its own re-exec logic
(systemd, a docker-style PID 1), you can stop the gateway via
the UI and let the supervisor restart it. The UI doesn't care
which path actually brings the process back; it just polls
`/health` until it's green.

If you have **no supervisor** — for example you launched via
`nohup go run ./cmd/opendray serve &` from a terminal that's
since been closed — `syscall.Exec` is the only way back up
without manual intervention. Confirm before clicking the
button: a failed self-exec leaves you with no daemon.

## Manual recovery

If the UI restart hangs and the 30 s timeout fires, the
gateway is likely in a degraded state (DB unreachable, listen
port collision, broken `config.toml`). Recover from the shell:

```bash
# Find and kill any stuck process.
pkill -f "opendray serve"

# Tail the log to see why startup failed.
tail -f /tmp/opendray.log    # or journalctl -fu opendray, etc.

# Fix the underlying issue, then relaunch.
go run ./cmd/opendray serve -config config.toml
```

If `config.toml` is the problem, the file lives at the path
shown in the Settings header (mono small print under each
section title).
