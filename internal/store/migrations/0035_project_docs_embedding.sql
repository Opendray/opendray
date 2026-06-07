-- M-U Phase 2 — Add embedding columns to project_docs so goal / plan
-- join the semantic-search surface as first-class vectors instead of
-- the old lexical substring match (fixed 0.6 similarity).
--
-- Mirrors 0031_session_logs_embedding exactly: variable-dim vector
-- (BM25 / bge-m3 / OpenAI co-exist), embedder name for vector-space
-- filtering, embedding_at for debugging re-embeds. embedding NULL means
-- "not yet embedded" and is picked up by the doc backfill goroutine.
--
-- Only goal + plan are searched semantically; tech_stack /
-- recent_activity are spawn-banner-only and left unembedded. The
-- needs-embed partial index therefore restricts to those two kinds so
-- the backfill scan stays a single index lookup.

ALTER TABLE project_docs
    ADD COLUMN IF NOT EXISTS embedding    vector,
    ADD COLUMN IF NOT EXISTS embedder     TEXT,
    ADD COLUMN IF NOT EXISTS embedding_at TIMESTAMPTZ;

-- Rows still awaiting an embedding (goal/plan only). Drives the
-- backfill loop's batch scan.
CREATE INDEX IF NOT EXISTS project_docs_needs_embed_idx
    ON project_docs (updated_at)
 WHERE embedding IS NULL AND kind IN ('goal', 'plan');

-- Embedder filter — mirrors memories_embedder_idx / session_logs so the
-- cross-layer search can WHERE-restrict to a matching vector space
-- before the cosine pass.
CREATE INDEX IF NOT EXISTS project_docs_embedder_idx
    ON project_docs (embedder)
 WHERE embedder IS NOT NULL;
