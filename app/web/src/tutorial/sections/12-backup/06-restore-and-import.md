# Backup — restore (A) and import (C)

The reverse direction of both surfaces lives in v1. `/backups` has
a "Restore from file" button (replays an encrypted bundle into
PostgreSQL); `/export` grew an Import section (replays an export
zip into the live tables).

## A — Restore from a `.tar.gz.enc` bundle

Use this when the box died and you're rebuilding on a fresh
opendray instance, OR when you want to roll a single instance
back to an earlier snapshot.

### Pre-requisite

The **same** `OPENDRAY_BACKUP_KEY` that produced the bundle must
be set on the running opendray. The manifest stamps the key
fingerprint; restore refuses bundles whose fingerprint doesn't
match the running cipher.

`pg_restore` must be on PATH (or `cfg.backup.pg_restore_path`),
major version ≥ the bundle's `pg_version`.

### Two modes

| Target DSN | What happens | Confirmation |
|---|---|---|
| **blank** (default) | Restores into opendray's own database (DANGEROUS — drops + replays every table). | Requires typing `I understand`. |
| **explicit DSN** | `pg_restore --dbname=<your DSN>` against an external/parallel DB. opendray's runtime DB is untouched. | None beyond admin auth. |

`--clean --if-exists` is on by default (drops tables before
replaying). Toggle off only if you're restoring into a freshly
created empty DB.

### What's restored

The bundle's `dump.bin` is a `pg_dump --format=custom` of every
opendray table — sessions, memories, integrations (with
api_key_hash bcrypt blobs intact), audit, etc. The
`config.toml` entry inside the tar is **not** auto-applied; if
you want to merge it in you do that by hand.

### Audit

Restore is a one-shot operator action; outcome lives in slog
(see `/settings/server` log viewer) plus the `pg_restore_output`
field in the API response. No DB row is created — restoring the
DB would have erased it.

## C — Import a `.zip` export bundle

Use this to migrate memories / integrations / custom_tasks from
one opendray to another, or to roll back a single logical
section without touching the rest of the database.

### Conflict policy v1

| Entity | Conflict key | Behaviour on conflict |
|---|---|---|
| Memories | `id` | Skipped (existing row wins). |
| Integrations | `id` and `route_prefix` (UNIQUE) | Skipped. |
| Custom tasks | `id` | Skipped. |

The Import row's `counts` field reports the breakdown
(created / skipped / failed) per entity.

### Memories: the embedder caveat

Imported memories are written with
`embedder = 'imported_v1'` and a NULL embedding column.
Search() ignores them until you trigger a re-embed pass.

After import:

1. Go to `/memory`.
2. Open the Maintenance tab (top right).
3. Click **Re-embed all under current embedder**.

Re-embedding uses whatever embedder the receiving opendray runs
(BM25 / HTTP / LocalONNX) — there's no requirement that the
sending instance used the same one.

### Integrations: the auth caveat

Bundles never carry plaintext API keys (opendray stores only
bcrypt hashes). On import, integration rows arrive with:

- `enabled = false` (so they can't be reached even by accident)
- `api_key_hash = "imported:no-plaintext-key-rotate-before-use"`
  (a non-bcrypt sentinel — no plaintext will ever validate
  against it)

Before the integration can authenticate again, **rotate the key**
from the integrations page. opendray returns the new plaintext
once; record it in your secrets manager and configure the caller.

### Audit trail

Every import attempt — success or failure — writes a row to the
`imports` table. The History panel on `/export` shows the last
20 attempts with per-entity counts. Failed sections don't abort
the whole import; the row's `error` field captures the first
section that failed and you can scroll the slog for details.
