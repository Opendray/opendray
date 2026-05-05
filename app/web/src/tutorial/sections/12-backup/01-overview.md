# Backup — overview

opendray ships with two complementary safety nets for the data
your gateway accumulates:

- **A — Disaster-recovery backups** (`/backups`). Encrypted full
  PostgreSQL dumps written to a pluggable storage target. Manual
  trigger or recurring schedule. Aimed at "the box died, restore
  on a new one."
- **C — Data exports** (`/export`). One-shot zip bundles of
  selected logical entities (memories, integrations metadata,
  custom tasks). Aimed at "I want to take my data with me."

Both ride a shared cipher (AES-256-GCM, key derived from the
`OPENDRAY_BACKUP_KEY` env var via PBKDF2). Without that env var
set the entire feature stays off — see Quickstart below.

## Why two surfaces

| Question | A | C |
|---|---|---|
| What's inside? | Whole PG dump + config.toml + manifest | A few JSONL tables + manifest, no dump |
| Encrypted? | Whole bundle (tar.gz inside AES-GCM stream) | Zip is plaintext; sensitive fields wrapped per-row |
| Triggered by? | Manual / scheduler | Manual |
| Best for? | Restore an entire opendray instance | Migration, audit, "give me my data" |
| Restore tool? | `pg_restore` after decrypting | Future import flow (v1.1) |

## What's NOT here

- **No S3 / GCS target yet.** The interface is open, but only
  `local` and `smb` ship in v1.
- **No reverse import.** Both bundles are export-only in v1.
- **No PITR / WAL archiving.** A pg_dump is a snapshot, not a
  continuous log.
- **No automatic key rotation.** Lose the passphrase, lose the
  ability to decrypt prior backups. Record the key fingerprint
  shown on the Backups page in your secrets manager.
