# Backup — exports (Plan C)

`/export` is the operator-facing **data export** view. Distinct
from `/backups` (which is operator-facing disaster recovery), an
export is a one-shot zip bundle of *selected* logical entities.

## When to use what

| Need | Use |
|---|---|
| "the box died, restore everything" | `/backups` (full PG dump) |
| "I want to take my memories to another tool" | `/export` (memories.jsonl) |
| "give me an audit trail of integrations" | `/export` (integrations.jsonl, metadata) |
| "WAL replay / point-in-time recovery" | Out of scope for v1 |

## Scope flags

- **Memories** — every row in the `memories` table (text +
  scope + metadata + hit counts). Embedding vectors are
  omitted; an importer would need to re-embed.
- **Integrations** — three modes:
  - `none` (skip the integrations table entirely)
  - `metadata` (recommended) — id, name, route prefix, scopes;
    no API key material.
  - `plaintext` — opt-in. v1 has no recoverable plaintext
    keys (everything is bcrypt-hashed), so the bundle ends up
    metadata-only with a `notes[]` entry in the manifest
    documenting that fact. The option exists so the
    confirmation flow + manifest format are stable when v1.1
    adds a plaintext key cache.
- **Custom tasks** — operator-defined tasks shown in the
  Inspector's Tasks tab.

## Bundle format

```
<exp_…>.zip
  manifest.json       ← what's inside, key fingerprint, counts, notes[]
  memories.jsonl      ← one JSON object per line
  integrations.jsonl  ← one JSON object per line, sans api_key_hash
  custom_tasks.jsonl  ← one JSON object per line
```

Unlike disaster-recovery backups, the **zip itself is not
encrypted**. Sensitive *fields* inside (e.g. plaintext API keys
when supported in v1.1) are AEAD-wrapped per row. Rationale:
operators want to be able to read the manifest + counts without
needing the master passphrase, and the actual sensitive
material is per-field in the future.

## Lifecycle

- Bundles live under `cfg.backup.export_dir` (default
  `~/.opendray/exports/`).
- Each bundle has a 24-hour expiry. The scheduler reaps
  expired files every 30s and flips the row to
  `status='expired'`.
- Download requires both the admin bearer (already enforced by
  the page) AND the per-export `download_token`. The token is
  shown via GET `/exports/{id}` (the list view redacts it).
- Delete is manual: click the trash icon to remove the file
  immediately and drop the row.
