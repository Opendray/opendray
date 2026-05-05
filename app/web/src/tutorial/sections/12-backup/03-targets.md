# Backup — targets

A *target* is "where the encrypted bundle ends up." opendray
ships with two implementations:

## local

Writes blobs into a directory on the same host running opendray.

- Default root: `~/.opendray/backups/` (or
  `cfg.backup.local_dir` from config.toml).
- Atomic: each blob is written to `.<id>.tar.gz.enc.part` and
  renamed onto the final path on success. A crash during write
  leaves only the temp file; the next scheduler tick will GC it.
- The `local` target is auto-created on first boot of the feature
  with id = `"local"`. Don't delete it unless you've added another
  target you want to be the default.

## smb

Writes to any SMB / CIFS share via a pure-Go SMB2 client (no host
`cifs-utils` dependency, so it works inside an unprivileged LXC).

UI: `/backups → Targets → New target`, kind = `smb`. Required
fields:

| Field | Example |
|---|---|
| Host | `192.168.9.8` |
| Port | `445` (default) |
| Share | `Claude_Workspace` |
| User | `linivek` |
| Password | (stored AES-GCM-encrypted, never echoed back) |
| Path prefix | `opendray/backups` |

Click **Test** after creating to verify auth + write access; the
test writes and removes a small probe file under the path prefix.

## What gets persisted

`backup_targets` in PG holds three columns:

- `id` — operator-chosen or auto-generated (`tgt_…`).
- `kind` — `local` / `smb` / (`s3` reserved).
- `config` — JSONB. Sensitive fields (e.g. SMB password) are
  AES-GCM wrapped with the master backup passphrase before write,
  so a leaked DB row alone doesn't reveal the SMB credential.

GET /backup-targets always returns redacted config (`password:
"********"`); the encrypted form never leaves the server.

## What's missing in v1

- **S3-compatible target** — interface is honoured but the service
  refuses to instantiate one (returns
  `ErrTargetUnsupported`). Land in v1.1 alongside any cloud-bound
  deployment.
- **Free-space reporting** — the UI doesn't yet ask the target how
  much room is left. For now operators monitor the volume / share
  themselves.
