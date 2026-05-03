-- 0009_audit_log_index — make audit_log queryable.
--
-- The Sink writes lifecycle events to audit_log on every publish, but
-- there was no read path until now. Two indexes cover the two common
-- access patterns surfaced by the Activity UI:
--
--   1. per-object timeline   — "show all events for session X"
--      → idx_audit_subject (subject_kind, subject_id, ts DESC)
--
--   2. per-topic stream      — "show recent integration.* events"
--      → idx_audit_action (action, ts DESC)
--
-- Both end with `ts DESC` so the leading rows of an index scan are
-- the most recent — no extra sort step needed for the common
-- "give me the last N" query.

CREATE INDEX IF NOT EXISTS idx_audit_subject
    ON audit_log (subject_kind, subject_id, ts DESC);

CREATE INDEX IF NOT EXISTS idx_audit_action
    ON audit_log (action, ts DESC);

CREATE INDEX IF NOT EXISTS idx_audit_ts
    ON audit_log (ts DESC);
