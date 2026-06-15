-- 0058_backup_verified — post-backup verification (backup redesign P2).
--
-- After a backup is written we read the blob back, decrypt it, and run
-- `pg_restore --list` to confirm the dump is a readable archive — so a
-- silently-corrupt backup is caught at creation time, not at disaster
-- time. verified_at records when that check last passed; verify_error
-- holds the failure detail when it didn't. Both NULL = not yet verified
-- (e.g. pg_restore unavailable, or a pre-0058 row). Pure additive.
ALTER TABLE backups ADD COLUMN IF NOT EXISTS verified_at  TIMESTAMPTZ;
ALTER TABLE backups ADD COLUMN IF NOT EXISTS verify_error TEXT;
