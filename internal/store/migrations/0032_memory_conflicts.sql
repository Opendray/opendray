-- M-PC — Cross-layer conflict ledger.
--
-- A daily LLM librarian (ConflictDetector worker) scans each
-- project's plan + top facts + recent journal and asks: "do any
-- of these claims contradict each other?" When yes, a row lands
-- here for operator review. Same approval-flow shape as
-- memory_cleanup_decisions and project_doc_proposals — agents
-- can't auto-resolve conflicts; the operator decides.
--
-- Why a separate table (not memory_summarizer_calls + JSON):
-- conflicts are an addressable backlog — we GET them, change
-- their status, and reference them from UI badges. Building that
-- on top of an audit log would require denormalising on every
-- read.
--
-- layer_a / layer_b enumerate which memory subsystem the two
-- conflicting items come from. ref_a / ref_b are the row IDs
-- inside that layer (memories.id for 'fact', project_docs.id for
-- 'goal'/'plan', session_logs.id for 'journal'). One side can be
-- 'self' when a layer contradicts itself (two facts disagree).
--
-- evidence is the LLM-authored markdown explaining the
-- contradiction — shown verbatim in the inbox so the operator
-- has the rationale on hand.

CREATE TABLE IF NOT EXISTS memory_conflicts (
    id           TEXT PRIMARY KEY,
    cwd          TEXT NOT NULL,
    layer_a      TEXT NOT NULL CHECK (layer_a IN ('fact','plan','goal','journal')),
    ref_a        TEXT NOT NULL,
    layer_b      TEXT NOT NULL CHECK (layer_b IN ('fact','plan','goal','journal')),
    ref_b        TEXT NOT NULL,
    evidence     TEXT NOT NULL DEFAULT '',
    severity     TEXT NOT NULL DEFAULT 'medium'
        CHECK (severity IN ('low','medium','high')),
    status       TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending','accepted','dismissed','expired')),
    detected_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    decided_at   TIMESTAMPTZ,
    decided_by   TEXT NOT NULL DEFAULT ''
);

-- Pending conflicts under one cwd is the hot path (operator
-- opens the Conflicts tab); index covers it directly.
CREATE INDEX IF NOT EXISTS memory_conflicts_cwd_pending_idx
    ON memory_conflicts (cwd, detected_at DESC)
 WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS memory_conflicts_status_idx
    ON memory_conflicts (status, detected_at DESC);

-- Extend the memory_workers task catalogue with conflict_detector.
-- Same idiom as 0030_memory_workers_plan_drift: drop + re-add the
-- CHECK constraint, then seed a summarizer default.

ALTER TABLE memory_workers
    DROP CONSTRAINT IF EXISTS memory_workers_task_check;

ALTER TABLE memory_workers
    ADD CONSTRAINT memory_workers_task_check
        CHECK (task IN (
            'gatekeeper','cleaner','gitactivity','transcript',
            'plan_drift','conflict_detector'
        ));

INSERT INTO memory_workers (task, kind) VALUES
    ('conflict_detector', 'summarizer')
ON CONFLICT (task) DO NOTHING;
