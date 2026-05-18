# Operator Guide

This guide is for operators running an opendray-v2 deployment in
production-like settings. Developer-flavoured setup is in
[`docs/quickstart.md`](quickstart.md).

## Topology

opendray ships as a **single static Go binary** with the React admin
SPA embedded via `go:embed`. Architecture:

```
        ┌─────────────────────────┐
client ─┤  HTTPS reverse proxy    │
        │  (Cloudflare Tunnel,    │
        │   nginx, Caddy, etc.)   │
        └────────────┬────────────┘
                     │ HTTP, plaintext
                     ▼
        ┌─────────────────────────┐
        │  opendray binary :8770  │
        │  ├── /api/v1/*          │
        │  ├── /admin/* (SPA)     │
        │  └── /api/v1/health     │
        └────────────┬────────────┘
                     │ TCP, pgx/v5
                     ▼
        ┌─────────────────────────┐
        │  PostgreSQL 15+         │
        └─────────────────────────┘
```

opendray itself does not terminate TLS. Run it behind Cloudflare
Tunnel, nginx, Caddy, or any other reverse proxy.

## CLI subcommands

```
opendray serve     [-config FILE]    # start the gateway
opendray migrate   [-config FILE]    # apply pending DB migrations and exit
opendray notes     <subcommand>      # filesystem notes vault ops
opendray skill     <subcommand>      # inspect/load agent skills
opendray mcp       <subcommand>      # inspect MCP server registry
opendray mcp-memory                  # stdio MCP server (used by Claude/Codex)
opendray version                     # print build info and exit
```

Only `serve` runs an HTTP listener. Every other subcommand exits as
soon as its task is done.

## Configuration reference

The full default config is in [`config.example.toml`](../config.example.toml).
Every field can be overridden by an `OPENDRAY_*` env variable. Env wins
over file.

### Top-level

| TOML key | Env override | Default | Notes |
|---|---|---|---|
| `listen` | `OPENDRAY_LISTEN` | `127.0.0.1:8770` | HTTP listen address |

### `[database]`

| Key | Env | Default | Notes |
|---|---|---|---|
| `url` | `OPENDRAY_DATABASE_URL` | _(required)_ | PostgreSQL 15+ DSN. Use a project-scoped role with CRUD-only privileges, never a superuser. |
| `max_conns` | `OPENDRAY_DATABASE_MAX_CONNS` | `16` | Pool cap for the pgx connection pool. |

### `[admin]`

| Key | Env | Default | Notes |
|---|---|---|---|
| `user` | `OPENDRAY_ADMIN_USER` | `admin` | Single-admin model — there's no multi-user system in v1. Used as the bootstrap username before the operator changes credentials via the UI; superseded by the keyfile thereafter. |
| `password` | `OPENDRAY_ADMIN_PASSWORD` | _(required for first boot)_ | Bootstrap-only plaintext password. The first time the operator changes credentials via the UI (`/api/v1/auth/change-credentials`), opendray writes a bcrypt-hashed keyfile at `$HOME/.opendray/secrets/admin.key` (mode `0600`, parent dir `0700`). From that point on the keyfile is authoritative and this value is **ignored** even if you keep updating it. |
| `token_ttl` | `OPENDRAY_ADMIN_TOKEN_TTL` | `24h` | Bearer token absolute lifetime for browser logins (`/api/v1/auth/login`). |
| `mobile_token_ttl` | `OPENDRAY_ADMIN_MOBILE_TOKEN_TTL` | `30d` (`720h`) | Bearer token absolute lifetime for the Flutter mobile app (`/api/v1/auth/mobile-login`). Longer because the device gates access via biometrics + secure storage. |

> **Where the password lives at rest** — credential precedence in
> `LoadCreds` (`internal/auth/keyfile.go`):
>
> 1. If `OPENDRAY_ADMIN_KEY_FILE` is set, the file at that path is
>    authoritative. Missing or unreadable here is a hard error
>    (operator intended to use it).
> 2. Otherwise, if `$HOME/.opendray/secrets/admin.key` exists, that
>    keyfile is used (bcrypt-hashed credentials).
> 3. Otherwise, fall back to `[admin].user` + `[admin].password` from
>    config/env (plaintext, bootstrap only).
>
> For docker / systemd `LoadCredential` / k8s secret deployments,
> point `OPENDRAY_ADMIN_KEY_FILE` at the injected file — this skips
> step 2 entirely so a stale home-dir keyfile never wins over the
> deployment's intent.

### `[log]`

| Key | Env | Default | Notes |
|---|---|---|---|
| `level` | `OPENDRAY_LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `format` | `OPENDRAY_LOG_FORMAT` | `text` | `text` (slog default) or `json` |
| `file` | _(none)_ | _(stdout only)_ | Optional rotating log file (10 MB max, keeps 5 archives). |

Every level/format also writes to an in-memory ring buffer
(~2,000 records) served at `/admin/logs` for live tailing.

### `[session]`

| Key | Env | Default | Notes |
|---|---|---|---|
| `idle_threshold` | `OPENDRAY_SESSION_IDLE_THRESHOLD` | `30s` | A session that emits no stdout for this long fires `session.idle`. |
| `idle_interval` | `OPENDRAY_SESSION_IDLE_INTERVAL` | `5s` | Idle-detector poll cadence. |

### `[vault]` and `[mcp]`

Filesystem paths for the notes vault and MCP server registry. See
`config.example.toml`; rarely changed.

### `[backup]`

Optional encrypted-backup subsystem. Disabled by default.

| Key | Env | Default | Notes |
|---|---|---|---|
| `enabled` | `OPENDRAY_BACKUP_ENABLED` | `false` | Master toggle. |
| `local_dir` | `OPENDRAY_BACKUP_LOCAL_DIR` | `~/.opendray/backups` | Default local target root. |
| `export_dir` | `OPENDRAY_BACKUP_EXPORT_DIR` | OS-specific | Staging dir for `/export` zip bundles. |
| `pg_dump_path` | `OPENDRAY_BACKUP_PG_DUMP_PATH` | _(PATH lookup)_ | Override `pg_dump` binary location. Must match the server's major version. |
| `pg_restore_path` | `OPENDRAY_BACKUP_PG_RESTORE_PATH` | _(PATH lookup)_ | Override `pg_restore` location. |

Plus one **env-only** secret:

| Env | Default | Notes |
|---|---|---|
| `OPENDRAY_BACKUP_KEY` | _(required when backups are enabled)_ | Master passphrase, AES-256-GCM key derivation. **Never write this in `config.toml`** — by design the loader rejects it there. |

## Database lifecycle

Migrations live in `internal/store/migrations/*.sql`, embedded into
the binary at build time. There are 23 migrations as of this writing,
named `0001_initial.sql` through `0023_*.sql`.

The runner:
- Maintains a `schema_migrations(version, applied_at)` table.
- Applies any unapplied versions in lexical filename order.
- Each migration runs in its own transaction.

```bash
opendray migrate -config config.toml
```

It's idempotent — already-applied versions are skipped, so it's safe
to re-run on every deploy. There are **no down-migrations**; rollback
is manual via raw SQL. To re-run a specific migration after a fix:

```sql
DELETE FROM schema_migrations WHERE version = '0014_backups';
```

…then re-run `opendray migrate`.

## Health endpoint

`GET /api/v1/health` (no auth required):

```json
{
  "status": "ok",
  "version": "1.0.0",
  "commit": "abc1234",
  "uptime_s": 12345,
  "db_ok": true
}
```

- HTTP 200 + `db_ok: true` → ready for traffic
- HTTP 503 + `db_ok: false` → degraded; the gateway is up but Postgres
  ping failed within 2 seconds

This single endpoint serves both liveness and readiness for k8s-style
probes.

## Logging

Standard library `log/slog` with two handlers:

- **Text** (default) — human-readable
- **JSON** — for ingestion by log shippers (Vector, Fluentd, etc.)

Outputs:
- Always to stdout/stderr
- Optionally to a rotating file (`[log].file`, 10 MB × 5 archives)
- Always to an in-memory ring buffer surfaced at `/admin/logs`

Set `[log].level = "debug"` to enable verbose component-level traces.
Production: keep at `info` with `format = "json"` if shipping logs.

## Backup subsystem

Two surfaces:

1. **Disaster-recovery backups** — full PostgreSQL dumps via `pg_dump`,
   encrypted client-side with AES-256-GCM, written to a configured
   target (local FS, SMB, WebDAV, SFTP, rclone, or S3).
2. **Data exports** — admin-triggered zip bundles of memories /
   integrations / custom tasks, downloaded via signed URL.

Enabling:

```bash
export OPENDRAY_BACKUP_ENABLED=1
export OPENDRAY_BACKUP_KEY="$(openssl rand -base64 32)"

# pg_dump must match the server's major version. On macOS:
export OPENDRAY_BACKUP_PG_DUMP_PATH=/opt/homebrew/opt/postgresql@17/bin/pg_dump
export OPENDRAY_BACKUP_PG_RESTORE_PATH=/opt/homebrew/opt/postgresql@17/bin/pg_restore

opendray serve -config config.toml
```

The admin sidebar grows a Backups page (`/backups`) and an Export page
(`/export`). The full lifecycle (target setup, schedules, restore
flow) is documented in §Backup below
section.

State persists in three DB tables (created by migration `0014_backups.sql`):
- `backup_targets` — destination configs
- `backup_schedules` — recurring specs
- `backups` — audit log of every dump

Rotate `OPENDRAY_BACKUP_KEY` carefully: backups encrypted with the old
key remain decryptable only with the old key. Keep the old passphrase
out of band until those backups are rotated out of retention.

## Process lifecycle

opendray traps `SIGINT` and `SIGTERM` and runs a graceful shutdown
(15-second window):

1. HTTP server stops accepting new requests
2. In-flight HTTP requests get up to 60 seconds to finish (per-request
   read/write timeout)
3. `Manager.Shutdown` sends `SIGTERM` to every running PTY session
4. `Hub.Shutdown` drains pending channel notifications (Telegram, etc.)
5. Background subsystems drain with per-subsystem timeouts:
   - Health checker: 2s
   - Audit sink: 5s
   - Vault sync: 2s
   - Backup scheduler: 2s (if enabled)
   - Capture engine: 2s

The grace window is **15 seconds** total for the supervisor to forward
to subsystems plus the per-subsystem timeouts above. Plan your
deploy/restart cycle so requests finish or fail fast — long-running
PTY sessions get a SIGTERM and may not complete in 15s; the agent
process is responsible for handling that.

## Sessions / PTY

opendray runs interactive PTYs via `creack/pty`, which works on macOS
and Linux (and other Unix-likes). **Windows is not supported** — there
is no PTY abstraction.

Each session has:
- A 1 MiB ring buffer of stdout for replay (`/api/v1/sessions/{id}/buffer`)
- A virtual-terminal emulator (`vt10x.Terminal`) for screen snapshots
- An idle detector firing `session.idle` events on the event bus

Session states: `pending` → `running` → (`idle`?) → (`stopped` | `ended`).

## Admin auth model

There's one human admin. The flow:

1. Client `POST /api/v1/auth/login` with `{user, password}`
2. opendray verifies the credentials against the active source
   (see [`[admin]`](#admin) for precedence):
   - **Keyfile source** — bcrypt-hashes the submitted password and
     compares against the stored hash. A dummy bcrypt compare runs
     even on username mismatch so the response time doesn't leak
     "user exists" vs "user doesn't".
   - **Config source** (bootstrap only) — constant-time compares the
     plaintext password from `[admin].password` / env.
3. On match, opendray issues a 32-byte random bearer token
4. Token lives in process memory only (`map[string]TokenInfo`); it's
   **lost on restart** — every operator restart forces re-login on
   active web sessions
5. Tokens expire absolutely after `[admin].token_ttl` (default 24h)
6. WebSocket endpoints accept the token via `?token=` query parameter
   (browsers can't set custom headers on WS handshakes)

Failed and successful logins both publish events (`admin.login_failed`,
`admin.login_success`) that the audit sink persists.

### Rotating credentials from the UI

`POST /api/v1/auth/change-credentials` (Settings → Admin → Change
Password in the web UI, mirror form in the mobile app) atomically:

- bcrypt-hashes the new password (min length 8) and writes a fresh
  `admin.key` to `$HOME/.opendray/secrets/admin.key` via temp-file +
  `rename` (so a crashed write leaves the previous keyfile intact)
- hot-swaps the in-memory `AdminCreds` so the next login uses the
  new pair without a restart
- revokes every existing bearer token (every other open browser /
  mobile session needs to log in again)

The plaintext password never reaches disk and never appears in logs.
After the first rotation, the `OPENDRAY_ADMIN_PASSWORD` env var is
inert — change it all you want, opendray won't read it again until
the keyfile is deleted.

## Unified memory subsystem

opendray ships with a cross-CLI project memory layer. Each Claude /
Codex / Gemini spawn boots with a shared markdown banner derived
from five sources:

- **Tech stack & structure** (scanner-managed, auto-refreshed)
- **Project goal** (operator-set, agent-proposable)
- **Project plan** (operator-set, agent-proposable)
- **Recent activity** (LLM-summarised git log, refreshed every 24h)
- **Recent journal** (auto-appended session-end summaries)

Long-term facts an agent stores via the memory MCP are filtered by
a server-side LLM gatekeeper, deduplicated by vector similarity,
and periodically reviewed by an LLM librarian (operator approves
deletions). Three layers of isolation defend against transcript
leakage between projects.

UI surfaces:

- **Mobile / web → Memory → Project** — 7-tab page for the
  per-project goal/plan/tech/activity/journal/inbox/cleanup
- **Memory → Cleanup inbox** — cross-project pending cleanup
  decisions
- **Session detail → 🏁 Project memory** — jump from any session
  into its project's memory page

Operator-facing detail and SQL recipes for validation are in
[`docs/memory-system.md`](memory-system.md).

## Where to look next

- [`docs/memory-system.md`](memory-system.md) — operator guide for the unified memory layer
- [`config.example.toml`](../config.example.toml) — full annotated config
- [`docs/quickstart.md`](quickstart.md) — developer setup
- [`docs/integration-guide.md`](integration-guide.md) — building external integrations
- [`deploy/`](../deploy/) — systemd unit + Proxmox LXC notes for production install
