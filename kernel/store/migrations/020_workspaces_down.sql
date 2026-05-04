-- 020_workspaces_down — rollback. Drops scoping columns + the table.
-- Existing data on the workspace_id columns is lost; the resources
-- themselves stay intact (just lose their workspace assignment).

DROP INDEX IF EXISTS idx_plugin_consents_workspace;
ALTER TABLE plugin_consents DROP COLUMN IF EXISTS workspace_id;

DROP INDEX IF EXISTS idx_plugin_secret_ws_plugin_key;
DROP INDEX IF EXISTS idx_plugin_secret_workspace;
ALTER TABLE plugin_secret DROP COLUMN IF EXISTS workspace_id;
-- Restoring the original UNIQUE(plugin, key) is left to the operator
-- if they need it back; depends on whether any duplicates snuck in.

DROP INDEX IF EXISTS idx_plugin_kv_ws_plugin_key;
DROP INDEX IF EXISTS idx_plugin_kv_workspace;
ALTER TABLE plugin_kv DROP COLUMN IF EXISTS workspace_id;

DROP INDEX IF EXISTS idx_llm_providers_workspace;
ALTER TABLE llm_providers DROP COLUMN IF EXISTS workspace_id;

DROP INDEX IF EXISTS idx_claude_accounts_ws_name;
DROP INDEX IF EXISTS idx_claude_accounts_workspace;
ALTER TABLE claude_accounts DROP COLUMN IF EXISTS workspace_id;

DROP INDEX IF EXISTS idx_sessions_workspace;
ALTER TABLE sessions DROP COLUMN IF EXISTS workspace_id;

DROP INDEX IF EXISTS idx_workspaces_one_default;
DROP TABLE IF EXISTS workspaces;
