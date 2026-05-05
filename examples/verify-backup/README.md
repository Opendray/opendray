# verify-backup

One-shot tool that proves a `.tar.gz.enc` bundle from `/backups`
round-trips correctly — decrypt → untar → `pg_restore --list` —
without needing a target PostgreSQL server.

Use this when you want to know "is this bundle still readable
under my current OPENDRAY_BACKUP_KEY" without committing to a
full restore.

## Usage

```bash
go run ./examples/verify-backup \
  <path-to-.tar.gz.enc> \
  <OPENDRAY_BACKUP_KEY> \
  <path-to-pg_restore>
```

Example:

```bash
go run ./examples/verify-backup \
  ~/.opendray/backups/2026/05/bk_xxx.tar.gz.enc \
  "$OPENDRAY_BACKUP_KEY" \
  /opt/homebrew/opt/postgresql@17/bin/pg_restore
```

The dump.bin is extracted to `/tmp/extracted-dump.bin` so you can
inspect it further (e.g. `pg_restore --list /tmp/extracted-dump.bin`)
or `pg_restore --dbname=...` it into a target DB by hand.

## Why it isn't an `opendray` subcommand

The full restore flow lives in the admin UI (`/backups →
Restore from file`) and HTTP API (`POST /backups/restore`),
which is the supported path. This example is a developer / SRE
tool — concise enough to read in one screen, runs against the
package's exported `backup.Cipher` interface, and stays out of
the production binary surface.
