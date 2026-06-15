-- 0046 — per-project doc blueprints (Cortex Phase 3).
--
-- The fixed Kind enum (goal/plan/tech_stack/recent_activity/overview)
-- forced every project — mobile app, service, CLI — into the same doc
-- structure ("被这些标签困住了格局"). A blueprint makes the section set
-- per-project data: project_docs.kind is reinterpreted as the section
-- slug (zero row rewrites — the PK and existing values are untouched),
-- and doc_blueprint_sections declares which sections exist for a cwd,
-- their order, and who maintains each one.
--
--   maintainer_mode:
--     ai      — drift detection auto-drafts it (proposal when locked)
--     human   — operator-authored; AI may only file proposals
--     scanner — mechanically rebuilt (tech_stack / recent_activity)
--
-- 'overview' is the reserved front page: pinned, undeletable, present
-- in every blueprint. kb_* slugs stay reserved global pages under the
-- '__global__' cwd, governed by the Knowledge layer — they never
-- appear in per-project blueprints.

CREATE TABLE doc_blueprint_sections (
    cwd             TEXT        NOT NULL,
    slug            TEXT        NOT NULL CHECK (slug ~ '^[a-z][a-z0-9_]{1,47}$'),
    title           TEXT        NOT NULL,
    description     TEXT        NOT NULL DEFAULT '',
    position        INT         NOT NULL DEFAULT 0,
    maintainer_mode TEXT        NOT NULL CHECK (maintainer_mode IN ('ai', 'human', 'scanner')),
    prompt_hint     TEXT        NOT NULL DEFAULT '',
    pinned          BOOLEAN     NOT NULL DEFAULT FALSE,
    -- inject: include this section's doc in the spawn banner. The
    -- overview is the human-facing front page and restates the other
    -- sections, so it seeds inject=FALSE to keep spawn cost flat.
    inject          BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (cwd, slug)
);

-- Seed the default blueprint for every project already known to
-- project_docs (excluding the global KB sentinel). The slugs map the
-- legacy fixed kinds 1:1, so existing doc rows keep working unchanged.
INSERT INTO doc_blueprint_sections (cwd, slug, title, description, position, maintainer_mode, pinned, inject)
SELECT DISTINCT cwd, s.slug, s.title, s.description, s.position, s.mode, s.pinned, s.inject
FROM project_docs,
     (VALUES
        ('overview',        'Overview',        'The project''s official document — the comprehensive page a developer reads to understand the whole project.', 0, 'ai',      TRUE,  FALSE),
        ('goal',            'Goal',            'Long-term intent: what this project is for and what done looks like.',                                          1, 'ai',      FALSE, TRUE),
        ('plan',            'Plan',            'The current roadmap / work-in-progress arc.',                                                                   2, 'ai',      FALSE, TRUE),
        ('tech_stack',      'Tech stack',      'Architecture, stack and repo structure — rebuilt mechanically by the project scanner.',                        3, 'scanner', FALSE, TRUE),
        ('recent_activity', 'Recent activity', 'Narrative summary of recent git history — rebuilt mechanically by the activity scanner.',                      4, 'scanner', FALSE, TRUE)
     ) AS s(slug, title, description, position, mode, pinned, inject)
WHERE cwd <> '__global__'
ON CONFLICT (cwd, slug) DO NOTHING;

-- kind becomes a slug: drop the enum CHECK (re-created by every
-- migration since 0025, most recently 0043) and constrain syntax only.
-- Validation against the blueprint happens in the service layer.
ALTER TABLE project_docs DROP CONSTRAINT IF EXISTS project_docs_kind_check;
ALTER TABLE project_docs ADD CONSTRAINT project_docs_kind_check
    CHECK (kind ~ '^[a-z][a-z0-9_]{1,63}$');

-- LATENT BUG FIX: project_doc_proposals.kind still carried 0025's
-- inline CHECK (kind IN ('goal','plan')) — 0027/0038/0043 only ever
-- relaxed project_docs. Every KB-page / overview update proposal (the
-- B3 Iterate edge) failed at INSERT. Same slug-syntax CHECK now.
ALTER TABLE project_doc_proposals DROP CONSTRAINT IF EXISTS project_doc_proposals_kind_check;
ALTER TABLE project_doc_proposals ADD CONSTRAINT project_doc_proposals_kind_check
    CHECK (kind ~ '^[a-z][a-z0-9_]{1,63}$');
