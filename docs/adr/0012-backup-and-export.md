# ADR 0012 — Backup & Export

**Status**: Accepted, 2026-05-05
**Authors**: Linivek, Claude (planner + impl)
**Replaces**: —

## Context

opendray accumulates state worth recovering: integrations + their
hashed API keys, memories, custom tasks, audit logs, channel
messages, sessions, vault git refs. Up to v0.x there was no
operator-facing way to take that state off the box short of
`pg_dump` from the host. Two distinct user needs surfaced:

1. **Disaster recovery** — the LXC dies / migrates; the operator
   wants a recent encrypted dump on UNAS or local disk to restore
   from.
2. **Data export** — "give me my memories / integrations as a
   downloadable bundle so I can audit, archive, or move them."

These have different shapes (whole-DB dump vs. selected
logical-entity rows), different cadences (recurring vs. one-shot),
different audiences (admin operator vs. admin operator wearing a
different hat — opendray has no end-user concept), and different
storage targets (off-box vs. on-box-then-download). They share a
cipher, a target abstraction, and a bundling primitive.

## Decision

Build **one** `internal/backup` package with **two** distinct
service surfaces:

- **A (runtime backups)**: `pg_dump --format=custom` →
  `tar(manifest, [config], dump)` → AES-GCM-chunked → pluggable
  `BackupTarget`. Driven by a Scheduler goroutine that polls the
  `backup_schedules` table (FOR UPDATE SKIP LOCKED).
- **C (data exports)**: streaming JSONL of selected tables
  (memories, integrations metadata, custom_tasks) into a zip with
  a manifest. Synchronous; user holds the HTTP request.

### Encryption

- Master key: derived from `OPENDRAY_BACKUP_KEY` env via
  PBKDF2-HMAC-SHA256 (200k iterations, fixed salt
  `"opendray-v1-backup"`). Fixed salt is intentional — passphrase
  is the only secret, and we want every backup taken under the
  same passphrase to share a derived key so they all decrypt.
- A: **whole-bundle** AES-256-GCM. Stream is chunked (64 KiB
  plaintext frames). Each frame's nonce is random 96-bit; AAD =
  frame index (prevents reordering); a terminator frame with
  ptLen=0 distinguishes "ended" from "truncated."
- C: **field-level** AES-256-GCM, base64url envelope `v1:…`.
  Sensitive values inside JSONL rows (e.g. SMB password in
  `backup_targets.config`, future plaintext API keys) are wrapped
  individually. Zip itself is not encrypted; manifest + counts
  remain readable without the passphrase.

### Storage targets

`BackupTarget` interface = `{Put, Get, Delete, HealthCheck}` with
streaming I/O. v1 ships:

- **local** — disk under `cfg.backup.local_dir` (default
  `~/.opendray/backups`). Atomic write via temp + rename.
- **smb** — pure-Go SMB2 client (no host `cifs-utils` dep). Per-op
  dial / auth / mount / write / unmount cycle. Password stored
  AEAD-encrypted in `backup_targets.config`.
- **s3** — interface honoured but instantiation refused
  (`ErrTargetUnsupported`). v1.1.

### Schemas

Three DDL files land:

- `0014_backups.sql` — `backup_targets`, `backup_schedules`,
  `backups`. Soft-delete (`status='deleted'`) keeps audit history.
  `key_fingerprint` column on `backups` lets restore reject blobs
  taken under a prior passphrase.
- `0015_exports.sql` — `exports` with `download_token` (32-byte
  base64url, second factor on download), `expires_at` (default
  24h, reaped by the scheduler).

### HTTP surface (admin-only)

- `/backups` (CRUD + download), `/backup-schedules` (CRUD),
  `/backup-targets` (CRUD + `/test`), `/backup-status`.
- `/exports` (CRUD + `/download?token=`).

opendray has no "regular user" concept — all of the above ride
the admin bearer. The C-surface download endpoint additionally
requires the per-export token so a leaked URL alone isn't enough.

### Scheduler

A single goroutine, started from `app.New` when the feature is
on. Wakes every 30s and:

1. Reaps expired exports (cheap; runs every tick).
2. Tries to claim one due `backup_schedules` row.
3. Runs the backup synchronously, applies retention.

No cron expressions in v1 — `interval_sec` only. Multiple
opendray instances cooperate via `FOR UPDATE SKIP LOCKED`.

## Decisions deliberately deferred (out of v1)

- **S3 / GCS / R2 targets.** Interface ready; impl deferred.
- **Reverse import.** Both A and C are export-only.
- **Cron expressions on schedules.** v1 = `interval_sec`.
- **PITR / WAL archiving.** Snapshot only.
- **Automatic key rotation.** Lose passphrase = lose ability to
  decrypt prior bundles. UI shows the key fingerprint so
  operators record it externally.
- **Plaintext API key recovery.** All keys are bcrypt — no
  plaintext exists to export. The "include plaintext" UI option
  is plumbed for v1.1 (which may introduce a plaintext cache for
  selected system integrations) but in v1 produces only a
  manifest note: "no recoverable plaintext keys."
- **opendray's own `decrypt-backup` CLI.** v1 expects operators
  to use the documented file format; cipher.go is stable.
- **Multi-admin / RBAC.** Single admin model unchanged.

## Consequences

- Operators must distribute `OPENDRAY_BACKUP_KEY` to every
  opendray instance that should be able to read shared backups.
- The opendray container needs `pg_dump` matching the server
  major version. Updated in deployment docs; the Backups page
  banner explicitly surfaces "pg_dump unavailable" so the
  failure mode is obvious.
- LXC tax: SMB target reaches the share over TCP/445; firewall
  rules between the opendray LXC and the file server must allow
  it (we don't go through the host's mount namespace).
- Two new tutorial group + ADR; no other module is touched
  except `cfg.Backup` (new section), `app.go` (mount + scheduler),
  and Sidebar/Router (new `/backups` + `/export` routes).
