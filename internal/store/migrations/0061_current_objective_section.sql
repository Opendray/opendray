-- 0061 — short-term "current objective" doc section + per-section write policy.
--
-- Goal/plan modelled only the LONG-TERM tier ("北极星" + roadmap arc): they
-- change rarely and the agent-side MCP write is proposal-gated. But most work
-- is driven by SHORT-TERM objectives established live in a session, completed,
-- then rolled to the next — and those never had a home that the in-session
-- agent (the writer with the richest context) could update directly. Plans
-- came out stale because the only auto-writer was the post-session drift
-- detector working off a lossy transcript summary.
--
-- This migration adds:
--   1. write_policy on each blueprint section — 'direct' lets the in-session
--      agent write the live doc (when unlocked); 'proposal' keeps the
--      operator-approval gate. goal/plan stay 'proposal' (unchanged);
--      current_objective is 'direct'.
--   2. a 'current_objective' section in every project blueprint: ai-maintained,
--      injected, direct-write — the current short-term objective + its steps.

ALTER TABLE doc_blueprint_sections
    ADD COLUMN write_policy TEXT NOT NULL DEFAULT 'proposal'
        CHECK (write_policy IN ('direct', 'proposal'));

-- Make room at slot 3 (right after 'plan') so the short-term objective sorts
-- between the long-term roadmap and the scanner sections. Bumps every
-- per-project section at/after that slot down by one; the global KB blueprint
-- is untouched.
UPDATE doc_blueprint_sections
   SET position = position + 1, updated_at = NOW()
 WHERE cwd <> '__global__' AND position >= 3;

-- Seed the section for every known project. Source the cwd set from BOTH
-- the blueprint table AND project_docs: a pre-0046 project may have doc rows
-- but no blueprint rows yet (it gets lazily seeded on first ListSections),
-- so the union guarantees it still receives current_objective now. New
-- projects get it from defaultSections() at lazy-seed time.
INSERT INTO doc_blueprint_sections
    (cwd, slug, title, description, position, maintainer_mode, write_policy, prompt_hint, pinned, inject)
SELECT cwd,
       'current_objective',
       'Current Objective',
       'The short-term objective we are working on right now and its immediate steps — set in-session, completed, then rolled to the next. Not permanent.',
       3,
       'ai',
       'direct',
       'This is the CURRENT short-term objective plus its immediate steps. It is expected to change FREQUENTLY: when a session establishes a new immediate objective, replace this with it; when a session completes it, roll it forward to the next objective and note what was just finished. Do NOT treat it as long-term intent — that is the Goal.',
       FALSE,
       TRUE
  FROM (
        SELECT DISTINCT cwd FROM doc_blueprint_sections WHERE cwd <> '__global__'
        UNION
        SELECT DISTINCT cwd FROM project_docs WHERE cwd <> '__global__'
       ) AS projects
ON CONFLICT (cwd, slug) DO NOTHING;
