-- 0052 — Cortex runtime settings (key/value).
--
-- Operator-tunable knobs for the unified module, editable from the
-- Cortex settings page without touching config.toml or restarting.
-- First key: spawn_mode.
--
--   spawn_mode = 'full' (default) — inject foundational rules + every
--                inject-flagged section/page in full (legacy shape).
--   spawn_mode = 'lean'           — inject foundational rules + a
--                compact INDEX of sections/pages; agents pull full
--                content on demand via the memory MCP (project_search
--                / doc_read). Saves spawn tokens and keeps long
--                sessions from drowning in upfront context.

CREATE TABLE cortex_settings (
    key        TEXT        PRIMARY KEY,
    value      TEXT        NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
