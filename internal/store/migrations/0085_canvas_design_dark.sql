-- 0085_canvas_design_dark — a second set of tokens for dark mode.
--
-- 0084 stored one palette, so a canvas built from the tokens was always light:
-- the preview follows the operator's theme, and a fixed light palette looks
-- wrong the moment they switch. Colour tokens now come in pairs; the scale
-- tokens (font, radius, spacing, base size) stay single because they don't
-- change with the theme.
--
-- Keys absent from tokens_dark fall back to their light value, so a project
-- that only cares about one theme sets nothing here.
--
-- Additive and idempotent. Rollback:
--   ALTER TABLE canvas_design_systems DROP COLUMN tokens_dark;

ALTER TABLE canvas_design_systems
    ADD COLUMN IF NOT EXISTS tokens_dark JSONB NOT NULL DEFAULT '{}'::jsonb;
