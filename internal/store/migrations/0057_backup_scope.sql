-- 0057_backup_scope — full-instance backups (backup redesign P1).
--
-- A backup can now capture more than the Postgres dump. `kind`
-- distinguishes a plain DB dump ('db_only' — the existing behaviour and
-- the default for every prior row) from a 'full_instance' bundle that
-- also carries the vault (notes/skills/mcp), secrets.env and
-- config.toml: everything needed to rebuild a working instance, not
-- just its database.
--
-- triggered_by gains a 'pre_migrate' value used by snapshots taken
-- automatically before schema migrations run. The column is free-text
-- TEXT (allowed values live in comments, not a CHECK constraint), so
-- no constraint needs relaxing. Pure additive change.
ALTER TABLE backups          ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'db_only';
ALTER TABLE backup_schedules ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'db_only';
