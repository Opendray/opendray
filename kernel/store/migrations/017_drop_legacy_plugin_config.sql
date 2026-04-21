-- 017: drop plugins.config JSONB column + add session FK constraints.
--
-- plugins.config was the pre-v1 home for plugin configuration. After
-- M5 Phase 5/6 every plugin uses the v1 Configure pipeline, which
-- writes non-secret values to plugin_kv.__config.* and secrets to
-- plugin_secret. Data migration from plugins.config → plugin_kv ran
-- in the application's Tier 1 cleanup (two legacy rows for claude +
-- file-browser on 2026-04-21); any remaining values would have been
-- '{}' already and can be safely discarded.
--
-- The session FKs close a long-standing integrity gap: sessions
-- carry claude_account_id / llm_provider_id UUIDs but nothing
-- prevented the referenced row from being deleted. ON DELETE SET
-- NULL matches the app's semantics — a session whose account/
-- provider vanished shows "—" until the user picks a new one.

ALTER TABLE plugins DROP COLUMN IF EXISTS config;

-- Normalise any accidental empty-string values before adding the FK.
-- sessions.claude_account_id and llm_provider_id are UUID already so
-- this is a defensive cleanup in case earlier migrations allowed text.
UPDATE sessions SET claude_account_id = NULL WHERE claude_account_id::text = '';
UPDATE sessions SET llm_provider_id   = NULL WHERE llm_provider_id::text   = '';

ALTER TABLE sessions DROP CONSTRAINT IF EXISTS sessions_claude_account_id_fkey;
ALTER TABLE sessions
    ADD CONSTRAINT sessions_claude_account_id_fkey
    FOREIGN KEY (claude_account_id) REFERENCES claude_accounts(id)
    ON DELETE SET NULL;

ALTER TABLE sessions DROP CONSTRAINT IF EXISTS sessions_llm_provider_id_fkey;
ALTER TABLE sessions
    ADD CONSTRAINT sessions_llm_provider_id_fkey
    FOREIGN KEY (llm_provider_id) REFERENCES llm_providers(id)
    ON DELETE SET NULL;
