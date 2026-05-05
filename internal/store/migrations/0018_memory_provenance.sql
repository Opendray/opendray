-- 0018_memory_provenance — record where every memory came from.
--
-- source_kind classifies provenance so the inspector can show
-- "this fact was extracted by the summarizer" vs "user typed this
-- in the UI" vs "MCP tool call from an agent". The summarizer
-- dedup logic also reads source_kind to avoid re-extracting its
-- own outputs.
--
-- Existing rows are backfilled to source_kind='manual' which is
-- the correct interpretation for anything stored before this
-- migration: pre-Phase-A all writes came from the manual UI /
-- mcp_call / mirror. We coarse-classify those to 'manual' since
-- the discriminator is gone.

ALTER TABLE memories
    ADD COLUMN IF NOT EXISTS source_kind        TEXT NOT NULL DEFAULT 'manual'
        CHECK (source_kind IN ('manual','mcp_call','summarizer','mirror_claude_md','imported')),
    ADD COLUMN IF NOT EXISTS source_ref         TEXT,
    ADD COLUMN IF NOT EXISTS summarizer_session TEXT,
    ADD COLUMN IF NOT EXISTS confidence         REAL;

CREATE INDEX IF NOT EXISTS memories_source_kind_idx
    ON memories(source_kind);
CREATE INDEX IF NOT EXISTS memories_summarizer_session_idx
    ON memories(summarizer_session)
    WHERE summarizer_session IS NOT NULL;
