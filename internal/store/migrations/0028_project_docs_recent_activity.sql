-- 0028_project_docs_recent_activity — widen project_docs.kind to
-- allow 'recent_activity'. M16c — LLM-summarised git activity so
-- the spawn-time banner includes "what was actually committed in
-- the last N days", a higher-level signal than per-tool hook
-- journal entries.
--
-- Same shape as tech_stack (0027): single row per cwd, replace-in-
-- place, scanner-managed (updated_by='scanner'), no proposal flow.

ALTER TABLE project_docs DROP CONSTRAINT IF EXISTS project_docs_kind_check;
ALTER TABLE project_docs ADD CONSTRAINT project_docs_kind_check
    CHECK (kind IN ('goal', 'plan', 'tech_stack', 'recent_activity'));
