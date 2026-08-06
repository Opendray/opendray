-- 0084_canvas_design_system — one design system per project, so the Canvas
-- stops re-inventing a look on every render.
--
-- Without it the agent re-derives colours, type and spacing from scratch each
-- time and the results drift. With it the gateway can (a) put the tokens in
-- every canvas request prompt and (b) inject them as CSS variables into the
-- rendered document, so consistency is structural rather than a hope.
--
-- tokens: a small JSON object of design tokens (colour / font / radius /
-- spacing …). Kept as JSONB rather than columns because the vocabulary will
-- grow and every field is optional.
-- notes: the free-text half — voice, density, things to avoid — the part
-- tokens cannot express.
--
-- Additive and idempotent: a new table, touching nothing existing.
-- Rollback: DROP TABLE canvas_design_systems;

CREATE TABLE IF NOT EXISTS canvas_design_systems (
    cwd        TEXT PRIMARY KEY,
    tokens     JSONB NOT NULL DEFAULT '{}'::jsonb,
    notes      TEXT  NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
