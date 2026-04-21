-- Docs-only: manual rollback for plugin_kv.
DROP INDEX IF EXISTS idx_plugin_kv_plugin;
DROP TABLE IF EXISTS plugin_kv;
