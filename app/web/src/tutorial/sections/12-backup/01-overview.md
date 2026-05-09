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

## Where backups can go

Six target kinds covering ≈99% of user storage habits:

- **`local`** — directory on the opendray host (default fallback)
- **`smb`** — Windows shares, home NAS (Synology / QNAP / UNAS)
- **`s3`** — AWS S3, Cloudflare R2, B2, MinIO, Alibaba Cloud OSS (阿里云 OSS), Tencent Cloud COS (腾讯云 COS), …
- **`webdav`** — Nextcloud, ownCloud, Synology DSM (群晖 DSM), Box, Jianguoyun (坚果云), …
- **`sftp`** — any SSH-accessible server (VPS, Hetzner Storage Box)
- **`rclone`** — passthrough to 70+ extra backends (Google Drive,
  OneDrive, Dropbox, Baidu Pan (百度网盘), Aliyun Drive (阿里云盘), …)

See **Targets** for the per-kind field list. All sensitive fields
(passwords, secret keys, private keys) are AES-256-GCM encrypted
at rest using the master backup passphrase.

## What's NOT here (v1 scope cuts)

- **No reverse import for restore** — restore replays a `pg_dump`
  via `pg_restore`. The `/export` zip has its own import flow
  (`/export → Import section`).
- **No PITR / WAL archiving** — a pg_dump is a point-in-time
  snapshot, not a continuous log. For sub-hour RPO use
  PostgreSQL's own WAL archiving on top of opendray.
- **No automatic key rotation** — lose the passphrase, lose the
  ability to decrypt prior backups. Record the key fingerprint
  shown on the Backups page in your secrets manager (Vaultwarden,
  1Password, etc.).
- **No edit-target UI** — to change a target's config (e.g. rotate
  S3 credentials), delete + recreate. v1.1.
