-- 0053 — skill usage tracking (Cortex distillation workbench).
--
-- Skills are injected into every spawn; without usage data the skill
-- list only ever grows and quietly becomes its own injection burden.
-- use_count / last_used_at are bumped when a session's transcript
-- actually references a skill, so the workbench can surface
-- never-used skills as retirement candidates.

ALTER TABLE knowledge_nodes
    ADD COLUMN use_count    INT NOT NULL DEFAULT 0,
    ADD COLUMN last_used_at TIMESTAMPTZ;
