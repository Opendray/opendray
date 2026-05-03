-- 0007_session_parent — link spawned-by relationships between sessions
-- so the Inspector's Tasks tab can group its child shell sessions
-- under the originating session in the sidebar.
--
-- ON DELETE SET NULL is intentional: removing a parent leaves child
-- sessions running and visible at the top level. Cascade-delete would
-- silently kill long-running task processes (e.g. `pnpm dev`) when
-- the user just wanted to clean up the parent row.

ALTER TABLE sessions
    ADD COLUMN parent_session_id TEXT
        REFERENCES sessions(id) ON DELETE SET NULL;

CREATE INDEX sessions_parent_idx ON sessions(parent_session_id);
