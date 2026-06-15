# Disaster Recovery Handbook

How to rebuild an opendray-v2 instance from a backup — including the
worst case: a fresh machine where nothing but your off-site backup
bundle and your Recovery Kit survived.

Read this **before** you need it. Recovery has a hard prerequisite
(the backup passphrase) that is unrecoverable if you didn't plan for
it, and the steps below assume you did.

> Scope: this covers the operator-facing **disaster-recovery backups**
> (encrypted PostgreSQL dumps, optionally bundling the whole instance).
> The admin-facing data *exports* (memories / integrations / custom
> tasks) are a separate, narrower surface — see the Operator Guide.

---

## 1. What's in a backup

A backup is a single AES-256-GCM-encrypted bundle (`<id>.tar.gz.enc`).
There are two kinds:

| Kind | Contents | Rebuilds |
|------|----------|----------|
| `db_only` (default) | `manifest.json` + `dump.bin` (the `pg_dump`) | the database only |
| `full_instance` | the above **plus** `vault.tar` (notes / skills / mcp), `secrets.env`, and `config.toml` | a working instance on a fresh machine |

A `db_only` backup is **not** sufficient for bare-metal recovery — it
restores your data but not the secrets, vault, or config needed to run.
For real disaster recovery, schedule or take `full_instance` backups.

> The vault tar **excludes `.git`** by default (it's recoverable from
> your remote and keeps the bundle small). `claude-accounts` (OAuth
> tokens) are **not** bundled by default — they cross a trust boundary;
> the manifest records their path so you know what to re-authenticate.

Every backup's encryption fingerprint is stored on the row. The UI's
**Verified** badge means opendray decrypted the blob and ran
`pg_restore --list` against it after writing — proof it's restorable,
not just that it was written.

---

## 2. The one thing you must keep off-site: the key

Backups are encrypted with a key derived from `OPENDRAY_BACKUP_KEY`
(the backup passphrase). **opendray deliberately never puts that key
inside a backup** — otherwise the backup would encrypt and ship its own
lock combination. Consequence:

> A backup bundle without the passphrase is unrecoverable. There is no
> backdoor.

You have two ways to survive the loss of the host that held the key:

1. **Store the passphrase itself** off-site (a password manager,
   Vaultwarden, a sealed envelope). Simplest.
2. **Recovery Kit** — opendray wraps the backup passphrase under a
   *separate* recovery passphrase you choose, producing
   `opendray-recovery-kit.json`. You store the kit (safe to keep next
   to the backups) and memorise / vault the recovery passphrase. Losing
   either one alone is fine; an attacker needs both.

Generate a Recovery Kit from the Backups page (web or mobile →
**Recovery Kit**), or it's offered the first time you arm backups.
Re-issue it whenever you rotate `OPENDRAY_BACKUP_KEY`.

The kit records which backup-key fingerprint it unlocks, so you can
tell at a glance whether a given kit matches a given pile of backups.

---

## 3. Bare-metal recovery, from zero

You have: a fresh host, the backup bundle(s), and **either** the backup
passphrase **or** (Recovery Kit + recovery passphrase). Steps:

### 3a. Install opendray + PostgreSQL

Install the `opendray` binary ([install-binary.md](install-binary.md))
and a PostgreSQL 15+ server. Create an empty database and a DSN for it.
Make sure `pg_restore`'s major version matches the server.

### 3b. Recover the backup passphrase

If you stored the passphrase directly, skip to 3c with it in hand.

If you kept a Recovery Kit, reconstruct the passphrase with the
`recover-key` subcommand:

```bash
# Print the recovered passphrase to stdout:
OPENDRAY_RECOVERY_PASSPHRASE='your-recovery-passphrase' \
  opendray recover-key --kit opendray-recovery-kit.json

# …or write it straight to the default keyfile so the next start
# can decrypt existing backups:
OPENDRAY_RECOVERY_PASSPHRASE='your-recovery-passphrase' \
  opendray recover-key --kit opendray-recovery-kit.json --install
```

`recover-key` prints the backup-key fingerprint the kit unlocks before
doing anything, so you can confirm it matches your backups. Omit
`OPENDRAY_RECOVERY_PASSPHRASE` to be prompted on stdin instead. Add
`--overwrite` to replace an existing keyfile.

### 3c. Bring opendray up with the recovered key

Point opendray at the empty database and the recovered key, then start
it so migrations create the schema:

```bash
export OPENDRAY_BACKUP_ENABLED=1
export OPENDRAY_BACKUP_KEY='…the recovered passphrase…'   # or use --install above
export OPENDRAY_BACKUP_PG_RESTORE_PATH=/path/to/matching/pg_restore

opendray migrate -config config.toml   # create the schema
opendray serve   -config config.toml   # start the gateway
```

Log in to the admin UI. The Backups page should now show your existing
backups as decryptable (the key fingerprint banner matches the rows).

### 3d. Restore the bundle

You can restore either through the UI (Backups → **Restore**) by
uploading the `.tar.gz.enc` bundle, or by re-uploading a bundle you
downloaded. The restore is a deliberate **two-step, dry-run-first**
flow on both web and mobile:

1. **Preview (dry run)** — opendray decrypts the bundle, verifies the
   fingerprint, and reports a *plan*: the dump size, and where
   `config.toml`, `secrets.env`, and the vault files would land. It
   **changes nothing**.
2. **Apply restore** — only after you review the plan and (when
   restoring into opendray's own database) type the confirmation
   phrase. Apply first takes an automatic **pre-restore safety
   snapshot**, then writes the components and runs `pg_restore`.

On apply of a `full_instance` bundle:

- `config.toml` is written, moving any existing file aside to `.bak`
  (a second restore timestamps the `.bak` rather than clobbering it).
- `vault.tar` is unpacked (with path-traversal protection).
- `secrets.env` is written `0600`.
- the dump is replayed with `pg_restore` (`--clean --if-exists` when
  you ticked *clean*).

After a restore, set `OPENDRAY_BACKUP_KEY` / config to match the
recovered instance and restart so the running process picks up the
restored `config.toml` and secrets.

---

## 4. Restoring into an external database (advanced)

The Restore form accepts a **target DSN**. Leave it empty to restore
into opendray's own database (the dangerous default — gated by the
confirmation phrase). Provide a DSN to replay a bundle into *any* other
Postgres instead — useful for inspecting a backup on a throwaway DB
without touching production. Apply mode always requires the
confirmation phrase regardless of target, because an external DSN can
just as easily point at production.

---

## 5. Upgrade safety net (pre-migrate snapshots)

Before applying schema migrations, opendray automatically takes a
**pre-migrate snapshot** so an upgrade is always preceded by a
restorable point. This is **fail-closed**: if the snapshot can't be
taken, the migration is blocked rather than risking an unprotected
upgrade.

If you must proceed without one (e.g. the backup subsystem is
intentionally off), set the escape hatch:

```bash
OPENDRAY_SKIP_PREMIGRATE_BACKUP=1 opendray migrate -config config.toml
```

When no backup cipher is configured, the snapshot falls back to a
local `0600` **plaintext** dump under the pre-migrate directory. Treat
that file as sensitive and remove it once the upgrade is confirmed
good.

---

## 6. Health & monitoring

The Backups page (web strip + mobile overview) and `GET
/api/v1/backup-health` surface an at-a-glance roll-up:

- **last good backup** — staleness signal
- **recent failures** — failed runs in the last 24h
- **verify failures** — succeeded backups whose restore-verify failed
  (the blob may not be restorable — investigate before trusting it)
- **overdue schedules** — enabled schedules more than 5 minutes past
  their next run

Backup failures and verification failures are also dispatched to your
configured notification channels (Telegram, etc.) so a silently broken
backup pipeline can't masquerade as a healthy one.

---

## 7. Drill it

A backup you've never restored is a hypothesis, not a safety net. At
least once:

1. Take a `full_instance` backup.
2. On a throwaway host (or with a throwaway target DSN), run the full
   §3 recovery from the bundle + Recovery Kit.
3. Confirm the restored instance starts and the data is intact.

Then you'll know the handbook works for *your* deployment — and that
your off-site key really does unlock your backups.

---

## 8. 3-2-1, briefly

Aim for **3** copies of your data, on **2** different media, with **1**
off-site. A backup or schedule can **fan out to several targets at
once** (local + SMB/S3/WebDAV/SFTP/rclone): pick more than one
destination and opendray writes the *same* sealed bundle to each in a
single run, grouped under one fan-out id. A target that's temporarily
down fails only its own copy — the others still land. Retention is
per-target. Keep the Recovery Kit (or passphrase) off-site too —
backups and their key on the same dead host is one copy, not two.

## 9. Content-dedup ("incremental")

pg_dump has no native incremental mode, so opendray dedups by content.
Each backup records the sha256 of its *plaintext* bundle; when a later
run on the same target produces an identical hash, opendray skips the
re-upload and points the new row at the existing blob (flagged
**deduped** in the UI). The row is still a complete, restorable backup —
it just shares storage with its identical predecessor. Retention is
reference-aware: a shared blob is removed only once no retained row
still points at it, so a deduped backup is never left dangling. Note
that logical dumps often differ byte-for-byte between runs (ordering,
timestamps), so dedup helps most on genuinely-unchanged databases.

## 10. Credential encryption at rest

Once backups are armed (a backup key is configured), opendray encrypts
**git-host API tokens** at rest with the same AES-GCM key, transparently
— no action needed. Tokens saved before backups were armed stay
plaintext until next saved. Because the key is the backup passphrase,
**rotating that passphrase makes existing encrypted tokens
unrecoverable** — re-enter affected git-host tokens after a rotation
(the UI shows the token as empty/needs-re-entry).
