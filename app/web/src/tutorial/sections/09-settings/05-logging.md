# Logging

Settings → **Server → Logging** owns three knobs and a live tail.

## Knobs

| Field | toml key | Default | Notes |
|---|---|---|---|
| Log level | `log.level` | `info` | `debug` / `info` / `warn` / `error`. Lines below this level are dropped before any handler runs. |
| Format | `log.format` | `text` | `text` is the default human-readable line format; `json` is one log per line as a JSON object. |
| Log file | `log.file` | *(empty)* | Optional path. When set, every line is also written here in addition to stderr. Auto-rotates at 10 MB, keeps 5 backups, max 30 days. |

`log.file` accepts `~/` and is created on first write. If the
directory doesn't exist opendray creates it (mode `0o755`). The
file is opened append-only; concurrent runs of opendray would
interleave entries — don't share the same path between two
running instances.

Changing any of the three requires a **Restart** — the slog
handler is built once at startup.

## Live tail

Below the form, the **Live tail** panel shows the most recent
log lines streamed straight from the running process:

- The gateway keeps the last **2,000 records** in an in-memory
  ring buffer (~1-2 MB RSS). On WebSocket connect, the panel
  replays the entire ring then forwards every new record as it
  arrives.
- **Color by level**: debug (grey) / info (default) / warn
  (amber) / error (red).
- **Counts**: tiny `D · I · W · E` stats at the top show how many
  records of each level are currently in the panel.
- **`live` / `offline` indicator** confirms the WebSocket is
  connected.

### Toolbar

| Button | Action |
|---|---|
| 🔍 search box | Substring filter, case-insensitive, against the rendered text line |
| **Pause** | Stop auto-scroll. Records still arrive but the view stops jumping. |
| **Clear** | Wipe the local view. **Server-side ring is untouched** — the next page reload replays the full buffer. |
| **Download** | Save the entire ring as `opendray-YYYYMMDD-HHMMSS.log`. Plain text, one record per line. |

The browser holds at most 5,000 records locally to keep the
DOM responsive — older lines drop off the head when you exceed
that. The server's ring is the source of truth; if you scroll
to the top and want even older context, download.

## When to use each level

- **debug** during development or when debugging a specific
  bug. Verbose; not safe for production.
- **info** default for production. Every HTTP request is logged
  at info; idle/start/stop events too.
- **warn** for "things still work but something looked wrong" —
  e.g. Telegram backoff, channel auth flakes.
- **error** for "something failed and a user noticed or will
  notice". Actionable; rare.

If you bump to `error` to quiet a noisy production log, use
**Restart server** at the bottom of the page; reload the
browser; the live tail starts fresh.

## Where else are logs?

- **stderr** of the gateway process — wherever you redirected
  it (terminal, `nohup ... > /tmp/opendray.log`, systemd
  journal, Docker stdout). Not affected by `log.file` setting;
  the multi-writer fans to both.
- **Audit log** (DB-backed, NOT the same thing) — tracks
  every admin action (config update, restart, integration CRUD).
  Lives under the Activity page filter, not here.
