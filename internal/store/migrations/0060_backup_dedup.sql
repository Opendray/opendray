-- 0060_backup_dedup — content-dedup "incremental" backups (P3).
--
-- pg_dump has no native incremental mode, so opendray dedups by
-- content: each backup records `content_hash`, the sha256 of its
-- *plaintext* bundle (pre-seal — Seal's per-run nonce makes the
-- ciphertext unstable). When the next run on the same target produces
-- an identical hash, opendray skips re-uploading and points the new row
-- at the existing blob, flagging it `deduped`. Retention is then
-- reference-aware: a blob is removed only once no non-deleted row still
-- references its target_path. Pure additive change.
ALTER TABLE backups ADD COLUMN IF NOT EXISTS content_hash TEXT;
ALTER TABLE backups ADD COLUMN IF NOT EXISTS deduped BOOLEAN NOT NULL DEFAULT FALSE;

-- Dedup lookups are "latest succeeded row on this target with this hash".
CREATE INDEX IF NOT EXISTS backups_dedup_idx
    ON backups(target_id, content_hash)
    WHERE status = 'succeeded' AND content_hash IS NOT NULL;
