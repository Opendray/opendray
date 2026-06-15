-- 0059_backup_fanout — multi-target (3-2-1) fan-out (backup redesign P3).
--
-- A single backup invocation can now write the SAME sealed bundle to
-- several targets at once (e.g. local disk + an off-site S3), so the
-- 3-2-1 rule is one click instead of N schedules. The rows produced by
-- one fan-out share a `group_id`; each still carries its own target_id
-- and target_path so download / restore / retention stay per-target.
--
-- backup_schedules gains `target_ids` — the full set of destinations a
-- schedule fans out to. The legacy single `target_id` column stays as
-- the primary/first target (and keeps its FK), and existing schedules
-- are backfilled to a one-element array so old rows behave unchanged.
-- Pure additive change.
ALTER TABLE backups ADD COLUMN IF NOT EXISTS group_id TEXT;
CREATE INDEX IF NOT EXISTS backups_group_idx ON backups(group_id) WHERE group_id IS NOT NULL;

ALTER TABLE backup_schedules
    ADD COLUMN IF NOT EXISTS target_ids TEXT[] NOT NULL DEFAULT '{}';
UPDATE backup_schedules
   SET target_ids = ARRAY[target_id]
 WHERE cardinality(target_ids) = 0;
