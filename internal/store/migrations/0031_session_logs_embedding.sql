-- M-PB — Add embedding columns to session_logs so the journal
-- joins the layer-5 semantic search surface. Combined with the new
-- project_search MCP tool, agents can ask "what did we decide
-- about X" and hit journal entries the same way they hit memories.
--
-- Strategy:
--   - embedding NULL means "not yet embedded" — caught by the
--     startup backfill goroutine which calls memory.Embedder on
--     batches of 50 rows.
--   - embedder TEXT records which embedder produced the vector so
--     mismatched-vector-space queries get filtered out (matches
--     the existing memories.embedder column convention).
--   - embedding_at lets operators debug "this journal entry is
--     stale because we re-embedded everything yesterday but this
--     one row's embedder is still the old name."
--
-- Why no fixed dimension: same reason memories doesn't pin one —
-- BM25 / bge-m3 / OpenAI all co-exist, dim varies per row.

ALTER TABLE session_logs
    ADD COLUMN IF NOT EXISTS embedding    vector,
    ADD COLUMN IF NOT EXISTS embedder     TEXT,
    ADD COLUMN IF NOT EXISTS embedding_at TIMESTAMPTZ;

-- Partial index over rows still awaiting backfill makes the
-- startup scan a single index lookup rather than a sequential scan
-- across the whole table.
CREATE INDEX IF NOT EXISTS session_logs_needs_embed_idx
    ON session_logs (created_at)
 WHERE embedding IS NULL;

-- Embedder filter — mirrors memories_embedder_idx semantics so the
-- cross-layer search can WHERE-restrict before the cosine pass.
CREATE INDEX IF NOT EXISTS session_logs_embedder_idx
    ON session_logs (embedder)
 WHERE embedder IS NOT NULL;
