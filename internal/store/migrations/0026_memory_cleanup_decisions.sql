-- 0026_memory_cleanup_decisions — pending LLM-proposed cleanups for
-- existing memories. M13 / cleaner subsystem.
--
-- Why a separate table from memories: a cleanup proposal is a
-- mutable decision-in-progress (pending → approved/rejected →
-- executed). Embedding that lifecycle into the memories row itself
-- would mix data with workflow state and complicate every read
-- path. Following the same pattern as project_doc_proposals
-- (migration 0025) — agent / LLM proposes, operator approves.
--
-- verdict values:
--   'keep'      — LLM says this memory is still useful (recorded
--                 so we can audit "we considered cleaning it and
--                 decided not to"; usually no follow-up action)
--   'stale'     — refers to obsolete code / decisions / state.
--                 If approved, the memory is deleted.
--   'duplicate' — near-paraphrase of another memory. merge_into
--                 must be set. If approved, this memory is deleted
--                 and merge_into.metadata.merged_from accumulates
--                 the dropped id.
--
-- status values:
--   'pending'   — awaiting operator decision
--   'approved'  — operator approved; ready for executor
--   'rejected'  — operator rejected; row stays for audit
--   'executed'  — approved + the delete/merge was performed
--   'expired'   — proposal got stale (memory was deleted /
--                 modified out from under us). Set by executor when
--                 it can't apply the action.
CREATE TABLE IF NOT EXISTS memory_cleanup_decisions (
    id            TEXT PRIMARY KEY,
    memory_id     TEXT NOT NULL,
    -- We do NOT FK to memories(id) because:
    --   1. memories can be deleted out-of-band (operator UI, scope
    --      bulk delete) and we'd lose audit history with ON DELETE
    --      CASCADE.
    --   2. NULL would be ambiguous (deleted vs never existed?)
    -- Instead we record memory_text_snapshot below so the UI can
    -- show "what we were going to clean" even after the underlying
    -- row is gone.
    memory_scope     TEXT NOT NULL,
    memory_scope_key TEXT NOT NULL DEFAULT '',
    memory_text_snapshot TEXT NOT NULL,
    verdict       TEXT NOT NULL CHECK (verdict IN ('keep', 'stale', 'duplicate')),
    reason        TEXT NOT NULL DEFAULT '',
    -- merge_into is the memory id to merge into when verdict =
    -- 'duplicate'. NULL for keep/stale. We don't FK either.
    merge_into    TEXT,
    -- run_id groups decisions from the same cleaner.Run invocation
    -- so the UI can show "this batch ran at 2026-05-12 03:00, 14
    -- decisions" instead of an unstructured stream.
    run_id        TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'approved', 'rejected', 'executed', 'expired')),
    -- summarizer_provider_id tells the operator which LLM proposed
    -- this — useful when comparing the quality of different
    -- gatekeeper backends.
    summarizer_provider_id TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    decided_at    TIMESTAMPTZ,
    executed_at   TIMESTAMPTZ
);

-- Inbox query: "give me everything I owe a decision on, newest first."
CREATE INDEX IF NOT EXISTS memory_cleanup_decisions_pending_idx
    ON memory_cleanup_decisions (created_at DESC)
    WHERE status = 'pending';

-- Per-scope filter for the project-scoped inbox view.
CREATE INDEX IF NOT EXISTS memory_cleanup_decisions_scope_idx
    ON memory_cleanup_decisions (memory_scope, memory_scope_key, created_at DESC);

-- Run-grouping query: "show me decisions from this cleaner run".
CREATE INDEX IF NOT EXISTS memory_cleanup_decisions_run_idx
    ON memory_cleanup_decisions (run_id, created_at);
