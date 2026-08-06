-- 0083_canvas_kind — the Canvas is not only UI mocks. It is a general visual
-- thinking surface an agent renders into: an early-stage draft of a screen, a
-- targeted refinement of an existing design, and equally a flowchart, a mind
-- map, or a relationship/entity diagram. Recording the KIND lets the panel
-- group + icon them, and lets the agent know what it is drawing (a diagram is
-- authored as self-contained inline SVG/HTML, not as a screen mock).
--
-- Additive and idempotent: one nullable-with-default column on canvas_artifacts.
-- Existing rows become 'ui', which is what they all are today.
-- Rollback: ALTER TABLE canvas_artifacts DROP COLUMN kind;

ALTER TABLE canvas_artifacts
    ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'ui';
