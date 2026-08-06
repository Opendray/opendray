-- 0082_canvas_artifacts — Canvas (beta): a live HTML preview surface any
-- cloud agent can push to, so the operator sees the real rendered UI while
-- it is being built instead of only screenshots + prose. The agent calls the
-- `canvas_render` MCP tool (title + html); the gateway upserts one artifact
-- per (cwd, slug) and broadcasts `canvas.updated`, and the web Canvas panel
-- renders the html in a sandboxed <iframe srcdoc>. The operator's on-canvas
-- annotations (pins / region selections) are formatted and seeded back into
-- the session as a prompt — a two-way visual channel, no persistence needed
-- for the feedback itself.
--
-- Scoped by cwd (project working dir), exactly like the session Inspector
-- tabs, so a Canvas tab shows the artifacts for its session's project.
--
-- BETA / rollback-able: idempotent (IF NOT EXISTS), touches NO existing
-- table/enum/CHECK. To remove the feature entirely: drop this table, delete
-- internal/canvas/, the canvas_render MCP tool, and the web Canvas panel.

CREATE TABLE IF NOT EXISTS canvas_artifacts (
    id         TEXT PRIMARY KEY,
    cwd        TEXT NOT NULL,
    slug       TEXT NOT NULL DEFAULT 'default',   -- named canvas within a project (agent picks; default 'default')
    title      TEXT NOT NULL DEFAULT '',
    html       TEXT NOT NULL DEFAULT '',
    version    INTEGER NOT NULL DEFAULT 1,         -- bumped on every re-render of the same (cwd, slug)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (cwd, slug)
);

CREATE INDEX IF NOT EXISTS canvas_artifacts_cwd_idx
    ON canvas_artifacts (cwd, updated_at DESC);
