-- 0067 — per-conversation Claude account override for the Cortex "AI
-- discussion" channel.
--
-- 0062 let a conversation pin its cloud-agent provider + model; 0063 added
-- the local-model summarizer override. But Claude is a MULTI-ACCOUNT setup
-- (cliacct rows, each its own OAuth token + config dir), and the AI
-- discussion had no way to choose WHICH account a claude turn runs against
-- — so it fell back to the default config dir and could hit a "Not logged
-- in" prompt. This column lets the chat UI pin the account, threaded into
-- the worker as Config.AccountID (resolved to CLAUDE_CODE_OAUTH_TOKEN +
-- CLAUDE_CONFIG_DIR, the same plumbing sessions use).
--
--   claude_account_id: '' (use the default account/config) | a cliacct id
--
-- Only meaningful when provider_id = 'claude'; cleared for other providers
-- and for the local-model (summarizer) override.

ALTER TABLE cortex_conversations
    ADD COLUMN IF NOT EXISTS claude_account_id TEXT NOT NULL DEFAULT '';
