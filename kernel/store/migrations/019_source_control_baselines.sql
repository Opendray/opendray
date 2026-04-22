-- source_control_baselines: per-session "what did Claude touch this run?"
-- snapshots. When a Claude session starts, the Source Control panel
-- can record the current HEAD SHA for each repository so the Changes
-- tab highlights only the work done during this session rather than
-- any accumulated local edits.
--
-- Replaces the previous in-memory map in gateway/git/git.go.Manager,
-- which evaporated on every gateway restart and leaked entries for
-- abandoned sessions. DB-backed gives us survival across restarts plus
-- a clean cascade on session deletion.
--
-- repo_path is stored as the absolute path the plugin's allowedRoots
-- resolved to at snapshot time — the Source Control panel scopes every
-- lookup by (session_id, repo_path) because a single session may touch
-- multiple repos.
CREATE TABLE IF NOT EXISTS source_control_baselines (
    session_id   UUID        NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    repo_path    TEXT        NOT NULL,
    head_sha     TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (session_id, repo_path)
);

CREATE INDEX IF NOT EXISTS idx_source_control_baselines_session
    ON source_control_baselines (session_id);
