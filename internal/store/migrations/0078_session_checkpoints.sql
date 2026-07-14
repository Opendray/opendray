-- 0078_session_checkpoints — context-level checkpoints for sessions
-- (priority ② of the state-machine-hardening plan: context backup/restore).
--
-- A checkpoint captures the recoverable *context* of a session at a point
-- in time: the working tree's uncommitted git diff, untracked (non-ignored)
-- files, and the operator input history. The heavy payload lives on disk
-- under storage_path (filesystem + DB-metadata split, so large diffs/files
-- don't bloat the DB); this row is the queryable manifest.
--
-- git-aware, .gitignore-respecting capture: only tracked changes (diff vs
-- HEAD) and untracked-but-not-ignored files are snapshot, so node_modules /
-- build artifacts / secrets under .gitignore are never copied. A non-git
-- cwd records metadata only (is_git=false).
--
-- ON DELETE CASCADE: removing a session removes its checkpoint rows. The
-- on-disk payload is reaped separately by the checkpoint service.
CREATE TABLE IF NOT EXISTS session_checkpoints (
    id              TEXT PRIMARY KEY,
    session_id      TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- what triggered the capture: 'interrupted' (gateway shutdown) or 'manual'.
    trigger         TEXT NOT NULL CHECK (trigger IN ('interrupted', 'manual')),
    cwd             TEXT NOT NULL,
    is_git          BOOLEAN NOT NULL DEFAULT FALSE,
    git_head        TEXT,                        -- HEAD sha at capture ('' / NULL when none)
    git_dirty       BOOLEAN NOT NULL DEFAULT FALSE,
    diff_bytes      BIGINT NOT NULL DEFAULT 0,
    untracked_files INTEGER NOT NULL DEFAULT 0,
    untracked_bytes BIGINT NOT NULL DEFAULT 0,
    input_bytes     BIGINT NOT NULL DEFAULT 0,
    -- true when any cap (diff size / per-file / total / file count) clipped
    -- the capture, so the UI can flag the checkpoint as partial.
    truncated       BOOLEAN NOT NULL DEFAULT FALSE,
    storage_path    TEXT NOT NULL,               -- on-disk checkpoint dir
    note            TEXT
);

CREATE INDEX IF NOT EXISTS session_checkpoints_session_created_idx
    ON session_checkpoints (session_id, created_at DESC);
