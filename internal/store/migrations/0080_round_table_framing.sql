-- 0080_round_table_framing — a table-level "framing" directive for the Round
-- Table group chat (experimental). Shared across all members and injected into
-- every reply's system prompt: it sets the current topic and the roles /
-- relationships between members ("claude leads architecture, codex only hunts
-- security holes"). Editable live so a long-running table can be re-framed as
-- the discussion moves to a new topic.
--
-- Forward-only, idempotent, non-destructive: a plain ADD COLUMN IF NOT EXISTS
-- with a default. Rolls back with
-- `ALTER TABLE round_tables DROP COLUMN framing;` (see ROLLBACK.md).
ALTER TABLE round_tables
    ADD COLUMN IF NOT EXISTS framing TEXT NOT NULL DEFAULT '';
