# Backup — quickstart

The feature is **off by default**. Two env vars turn it on:

```bash
export OPENDRAY_BACKUP_ENABLED=1
export OPENDRAY_BACKUP_KEY="$(openssl rand -base64 32)"
```

Restart opendray. The Backups page should now appear in the
sidebar under Platform.

## Record the key fingerprint

On `/backups` the top banner shows a 16-char hex
**Key fingerprint** — that's the first 8 bytes of
SHA-256(derived-key). Every backup row stamps this fingerprint;
restore later will refuse a blob whose stored fingerprint doesn't
match the running passphrase.

**Save this fingerprint alongside your passphrase in Vaultwarden
or your secrets manager.** If the fingerprint changes (passphrase
rotated), prior backups become unreadable on a fresh install.

## pg_dump prerequisite

opendray shells out to `pg_dump`. The binary's major version must
be ≥ your PostgreSQL server's major version. Inside an LXC / Docker
deployment you'll typically need:

```bash
apk add postgresql<MAJOR>-client     # alpine
apt-get install postgresql-client-<MAJOR>  # debian / ubuntu
```

The Backups → Status banner shows what version `pg_dump --version`
reports; if it's empty the trigger button is disabled.

## Take the first backup

1. Go to `/backups`.
2. Confirm the green "ok" banner shows a key fingerprint and
   pg_dump version.
3. Click **Backup now** (leave "include config.toml" on).
4. The row appears with status `running`, then `succeeded` —
   typical small instance: 1-3 seconds.
5. Click the **download arrow** to grab `<id>.tar.gz.enc`.

## Verify the bundle

```bash
# Decrypt with the same passphrase used by opendray:
go run ./cmd/opendray decrypt-backup --in <id>.tar.gz.enc --out plain.tar.gz
tar -tzf plain.tar.gz
# manifest.json
# config.toml
# dump.bin
```

Then `pg_restore --create plain.tar.gz` against an empty PG
instance to verify the dump is valid.

> **Note**: the `decrypt-backup` CLI subcommand is a v1.1 deliverable.
> Until then, decrypt by reading the file format documented in
> `internal/backup/cipher.go` (it's stable).
