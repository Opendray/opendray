-- 0050 — the knowledge blueprint (Cortex).
--
-- Knowledge was 4 fixed pages (infrastructure / conventions / lessons
-- / reusable). New knowledge domains had nowhere to live, so
-- everything got crammed into those four ever-growing documents — bad
-- for humans AND for the AI, which had to swallow whole pages instead
-- of indexing fine-grained ones.
--
-- Fix: the SAME blueprint mechanism projects use now also governs the
-- global knowledge base. The '__global__' cwd gets a blueprint whose
-- sections ARE the knowledge pages; the operator (or a curation
-- conversation) can add new kb_* pages, classified by NATURE:
--
--   foundational — binding ground truth + rules; injected into every
--                  spawn as guardrails (when inject=true)
--   emergent     — distilled guidance; injected as reference
--                  (when inject=true)
--
-- Pages with inject=false stay out of the spawn banner entirely and
-- are reached on demand through cross-layer search (memquery /
-- memory_search), so agents index only what a task needs.

ALTER TABLE doc_blueprint_sections
    ADD COLUMN nature TEXT NOT NULL DEFAULT ''
        CHECK (nature IN ('', 'foundational', 'emergent'));

-- Seed the global blueprint with the four classic pages. They are
-- pinned (undeletable): the KB drafter keeps re-drafting them and the
-- spawn guardrails build on the foundational pair.
INSERT INTO doc_blueprint_sections
    (cwd, slug, title, description, position, maintainer_mode, prompt_hint, pinned, inject, nature)
VALUES
    ('__global__', 'kb_infrastructure', 'Infrastructure',
     'Standing ground truth about the home-lab/ecosystem: hosts, networks, databases, gateways — plus the binding rules for using them.',
     0, 'ai', '', TRUE, TRUE, 'foundational'),
    ('__global__', 'kb_conventions', 'Conventions',
     'The binding development conventions & policies: stack, source control, coding rules, release process.',
     1, 'ai', '', TRUE, TRUE, 'foundational'),
    ('__global__', 'kb_lessons', 'Lessons',
     'Distilled playbooks and hard-won lessons from past work — reference guidance, not law.',
     2, 'ai', '', TRUE, TRUE, 'emergent'),
    ('__global__', 'kb_reusable', 'Reusable features',
     'Catalog of features/components/patterns liftable into new projects.',
     3, 'ai', '', TRUE, TRUE, 'emergent')
ON CONFLICT (cwd, slug) DO NOTHING;
