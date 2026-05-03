-- 0006_custom_tasks — user-defined tasks surfaced by the Inspector's
-- Tasks tab alongside auto-discovered manifests (package.json,
-- Makefile, Cargo.toml, ...). One row per task; cwd='' means global
-- (visible from any session), otherwise the task is scoped to that
-- absolute working directory.

CREATE TABLE custom_tasks (
    id          TEXT PRIMARY KEY DEFAULT 'tsk_' || substr(md5(random()::text || clock_timestamp()::text), 1, 12),
    name        TEXT NOT NULL,
    command     TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    cwd         TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX custom_tasks_cwd_idx ON custom_tasks(cwd);
