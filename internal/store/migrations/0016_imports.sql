-- 0016_imports — admin-facing data import history.
--
-- Counterpart to 0015_exports: every successful or failed import
-- attempt logs one row here so operators can audit how the
-- current memory / integrations / custom_tasks tables came to
-- look the way they do.
--
-- A-restore (full pg_restore from a /backups bundle) is NOT
-- recorded here — those are one-shot operator operations whose
-- outcome is the entire database state and which only happen on
-- a fresh / parallel instance. Logs go to the audit_log + slog
-- channels instead.

CREATE TABLE imports (
    id              TEXT PRIMARY KEY,                -- "imp_<22 base32>"
    status          TEXT NOT NULL,                   -- 'pending'|'running'|'succeeded'|'failed'
    requested_by    TEXT NOT NULL,                   -- admin username
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at     TIMESTAMPTZ,
    source_filename TEXT,                            -- caller-provided original name (for audit)
    source_bytes    BIGINT NOT NULL DEFAULT 0,
    counts          JSONB NOT NULL DEFAULT '{}'::jsonb,  -- per-entity created/skipped/failed
    error           TEXT
);
CREATE INDEX imports_started_at_idx ON imports(started_at DESC);
CREATE INDEX imports_status_idx     ON imports(status);
