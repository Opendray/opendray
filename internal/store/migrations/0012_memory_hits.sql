-- 0012_memory_hits — track which memories actually get used.
--
-- hit_count: total number of times this memory was returned by a
--   search (post-threshold filter). Bumped in a single bulk
--   statement after Service.Search hands rows back to the caller.
-- last_hit_at: timestamp of the most recent hit. NULL until first
--   use; lets the inspector sort by "freshest" or surface stale
--   memories that haven't been touched in months.
--
-- No backfill: existing rows start at 0 / NULL, which is correct.
ALTER TABLE memories
    ADD COLUMN IF NOT EXISTS hit_count   BIGINT      NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_hit_at TIMESTAMPTZ;
