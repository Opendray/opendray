-- Denormalize session OUTCOME onto session_logs so the experience compiler's
-- episode feedstock survives session pruning.
--
-- The compiler distils cross-project skills only from procedures that
-- SUCCEEDED in >=2 sessions, and it reads that success signal (exit_code /
-- state / timing) from the `sessions` table via
--   session_logs l JOIN sessions s ON s.id = l.session_id.
-- But `sessions` is ephemeral (rows are DELETEd on prune), so the JOIN
-- silently drops every session_summary whose session row is gone — measured
-- at 112 summaries -> 2 surviving the JOIN. The compiler therefore never sees
-- enough recurrence and distils nothing.
--
-- Fix: the journaler already KNOWS the outcome when it writes the
-- session_summary (SessionSucceeded + ExitCode + Started/Ended), so persist it
-- here on the durable row. The episode source then reads session_logs alone
-- (no JOIN) and the whole journal corpus becomes feedstock. Columns are
-- nullable — only session_summary rows carry them; a boot-time backfill
-- recovers the values for historical rows by parsing the summary body.
ALTER TABLE session_logs
    ADD COLUMN IF NOT EXISTS outcome_state TEXT,        -- 'ended' | 'stopped' (NULL = non-summary / not backfilled)
    ADD COLUMN IF NOT EXISTS exit_code     INT,         -- process exit code; NULL when unknown / still running
    ADD COLUMN IF NOT EXISTS started_at    TIMESTAMPTZ, -- session start (duration / time-cost proxy)
    ADD COLUMN IF NOT EXISTS ended_at      TIMESTAMPTZ; -- session end

-- The compiler scans recent session_summary rows; a partial index keeps that
-- scan cheap as the journal grows.
CREATE INDEX IF NOT EXISTS session_logs_episode_idx
    ON session_logs (created_at)
    WHERE kind = 'session_summary';
