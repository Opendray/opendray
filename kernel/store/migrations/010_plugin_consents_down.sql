-- Docs-only: forward-only migrations are the rule; this SQL is kept
-- alongside the up migration for manual rollback during incidents.
DROP TABLE IF EXISTS plugin_consents;
