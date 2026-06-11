-- 0054 — per-skill enable switch (Cortex distillation rework).
--
-- With hundreds of distilled skills, a project typically needs 1-2.
-- enabled=false removes the skill's SKILL.md from the vault skills
-- dir (so no session loads it) while keeping the node + its history;
-- re-enabling re-renders the file. The spawn knowledge banner also
-- lists enabled skills only.

ALTER TABLE knowledge_nodes
    ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT TRUE;
