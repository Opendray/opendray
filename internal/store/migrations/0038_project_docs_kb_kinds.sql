-- 0038_project_docs_kb_kinds — widen project_docs.kind for the curated
-- knowledge-base pages (M-KB), fusing the KB into the existing note system
-- rather than a separate table:
--   - kb_infrastructure / kb_conventions / kb_lessons — global pages, stored
--     under the '__global__' sentinel cwd (cross-project reference).
--   - kb_handbook — one curated handbook per real project cwd.
--
-- AI-drafted (updated_by='agent', already allowed by 0027's CHECK). A human
-- edit (updated_by='operator') locks a page from further AI overwrite, so no
-- extra lock column is needed.

ALTER TABLE project_docs DROP CONSTRAINT IF EXISTS project_docs_kind_check;
ALTER TABLE project_docs ADD CONSTRAINT project_docs_kind_check
    CHECK (kind IN ('goal', 'plan', 'tech_stack', 'recent_activity',
                    'kb_infrastructure', 'kb_conventions', 'kb_lessons', 'kb_handbook'));
