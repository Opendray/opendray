-- Docs-only: manual rollback for the plugin_secret nonce column.
ALTER TABLE plugin_secret DROP COLUMN IF EXISTS nonce;
