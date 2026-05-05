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

## Verify the bundle (without restoring)

`examples/verify-backup` ships a one-shot Go program that proves
the bundle round-trips without needing a target server:

```bash
go run ./examples/verify-backup \
  ~/.opendray/backups/2026/05/<id>.tar.gz.enc \
  "<your OPENDRAY_BACKUP_KEY>" \
  $(which pg_restore)
```

Output:

```
cipher fingerprint: e344173f214c7641
entry: manifest.json         431 bytes
entry: config.toml           1081 bytes
entry: dump.bin              91693 bytes

manifest fingerprint: e344173f214c7641
backup_id: bk_t53gcylshov5fylcd2szul
pg_version: 17.9
version: 1

--- pg_restore --list output (header only) ---
;     dbname: opendray_v2
;     TOC Entries: 108
…
```

The program decrypts via Cipher.Open, untars to /tmp, and runs
`pg_restore --list` (no DB needed) to confirm the dump.bin is a
structurally valid PostgreSQL custom-format dump.

## Restore from the UI

`/backups → Restore from file` accepts a bundle and runs it
through `pg_restore` against the DSN you pick. See the
"restore-and-import" tutorial section for the safety flow
(typing "I understand" when restoring over opendray's own DB).
