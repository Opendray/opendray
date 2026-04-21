-- 017 DOWN: re-add plugins.config + drop session FKs.
-- The re-added column is empty since legacy values were migrated
-- into plugin_kv; rolling back does not restore them.

ALTER TABLE sessions DROP CONSTRAINT IF EXISTS sessions_llm_provider_id_fkey;
ALTER TABLE sessions DROP CONSTRAINT IF EXISTS sessions_claude_account_id_fkey;

ALTER TABLE plugins ADD COLUMN IF NOT EXISTS config jsonb NOT NULL DEFAULT '{}'::jsonb;
