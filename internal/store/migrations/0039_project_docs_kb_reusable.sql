-- 0039_project_docs_kb_reusable — widen project_docs.kind for the M-KB
-- "reusable features" page (kb_reusable): a global catalog of features /
-- components / patterns already built across projects that can be lifted into
-- a new project. AI-drafted (updated_by='agent'); human edit locks it.

ALTER TABLE project_docs DROP CONSTRAINT IF EXISTS project_docs_kind_check;
ALTER TABLE project_docs ADD CONSTRAINT project_docs_kind_check
    CHECK (kind IN ('goal', 'plan', 'tech_stack', 'recent_activity',
                    'kb_infrastructure', 'kb_conventions', 'kb_lessons',
                    'kb_reusable', 'kb_handbook'));
