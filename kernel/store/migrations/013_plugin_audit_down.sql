-- Docs-only: manual rollback for plugin_audit.
DROP INDEX IF EXISTS idx_plugin_audit_plugin_ts;
DROP TABLE IF EXISTS plugin_audit;
