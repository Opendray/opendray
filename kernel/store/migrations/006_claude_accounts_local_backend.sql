-- Extend claude_accounts to support a second backend type: "local".
--
-- backend_type = 'oauth'  → existing behaviour; config_dir + token_path drive
--                            CLAUDE_CODE_OAUTH_TOKEN / CLAUDE_CONFIG_DIR injection.
-- backend_type = 'local'  → NTC's in-process Anthropic↔OpenAI proxy at
--                            /v1/messages on the gateway; spawn injects
--                            ANTHROPIC_BASE_URL=http://127.0.0.1:<port> and
--                            reads provider/base_url/model/api_key_env from
--                            this row to route the request at call time.
--
-- All new columns are nullable / defaulted so existing rows keep working
-- with no migration backfill required.
ALTER TABLE claude_accounts
    ADD COLUMN IF NOT EXISTS backend_type TEXT NOT NULL DEFAULT 'oauth',
    ADD COLUMN IF NOT EXISTS provider     TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS base_url     TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS model        TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS api_key_env  TEXT NOT NULL DEFAULT '';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'claude_accounts_backend_type_chk'
    ) THEN
        ALTER TABLE claude_accounts
            ADD CONSTRAINT claude_accounts_backend_type_chk
            CHECK (backend_type IN ('oauth', 'local'));
    END IF;
END$$;

CREATE INDEX IF NOT EXISTS idx_claude_accounts_backend ON claude_accounts(backend_type);

-- The old ollama / lmstudio agent plugins spawned a raw REPL in a PTY,
-- which isn't useful as an agentic CLI inside NTC. They're replaced by
-- "local" claude_accounts rows routed through the Anthropic↔OpenAI
-- proxy. Drop the seeded plugin rows so they disappear from the
-- Providers page after the upgrade. Running sessions that still point
-- at these session types will continue to drain naturally.
DELETE FROM plugins WHERE name IN ('ollama', 'lmstudio');

