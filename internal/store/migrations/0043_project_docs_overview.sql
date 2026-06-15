-- 0043_project_docs_overview — widen project_docs.kind for the project's
-- official OVERVIEW document. The Overview is the rich, AI-maintained,
-- human-readable page a developer reads to understand the whole project
-- (features + architecture + foundations). Per-project (Notes), AI-drafted
-- (updated_by='agent'); a human edit locks it.
ALTER TABLE project_docs DROP CONSTRAINT IF EXISTS project_docs_kind_check;
ALTER TABLE project_docs ADD CONSTRAINT project_docs_kind_check
    CHECK (kind IN ('goal', 'plan', 'tech_stack', 'recent_activity', 'overview',
                    'kb_infrastructure', 'kb_conventions', 'kb_lessons',
                    'kb_reusable', 'kb_handbook'));
