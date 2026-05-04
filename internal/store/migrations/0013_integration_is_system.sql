-- 0013_integration_is_system — flag rows that opendray manages
-- itself (e.g. opendray-memory's auto-registered MCP integration)
-- so the UI can group them and refuse delete/rotate from operators.
--
-- Operator-created integrations stay is_system = FALSE; opendray
-- bootstraps anything internal with TRUE so the integration list
-- doesn't deceive the operator into thinking they once registered
-- "opendray-memory" themselves.
ALTER TABLE integrations
    ADD COLUMN IF NOT EXISTS is_system BOOLEAN NOT NULL DEFAULT FALSE;

-- The internally-managed memory integration may already exist on
-- upgraded installs (created by opendray pre-flag). Backfill once.
UPDATE integrations SET is_system = TRUE WHERE name = 'opendray-memory';
