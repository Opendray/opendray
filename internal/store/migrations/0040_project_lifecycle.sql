-- 0040_project_lifecycle — P-D: per-project lifecycle status.
-- A project moves active → paused → archived as it goes idle or is shelved.
-- archived/paused projects are frozen: excluded from spawn injection and from
-- cross-project Knowledge distillation, surfaced read-only.
--
-- Projects without a row are treated as 'active' by the service layer, so this
-- table stores only explicit decisions (operator button or idle auto-suggest
-- that was accepted). Keyed by cwd to match project_docs / session_logs.
CREATE TABLE IF NOT EXISTS project_lifecycle (
    cwd        TEXT PRIMARY KEY,
    status     TEXT NOT NULL DEFAULT 'active'
               CHECK (status IN ('active', 'paused', 'archived')),
    updated_by TEXT NOT NULL DEFAULT 'operator',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
