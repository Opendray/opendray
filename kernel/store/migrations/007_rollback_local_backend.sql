-- Rollback the "local" backend_type experiment on claude_accounts.
--
-- The Anthropic↔OpenAI proxy (gateway/llm_proxy/) works in isolation
-- but wrapping Claude CLI around local models didn't produce a usable
-- agent in practice (tool_use distribution mismatch, CLI-side caching
-- assumptions, small-model reliability). We're pivoting to adding a
-- first-class OpenAI-native agent plugin (OpenCode) and a new
-- llm_providers table that it reads directly.
--
-- Dropping the columns here so the schema matches the code that uses
-- it. Translation code in gateway/llm_proxy/translate.go is preserved
-- for future reuse (e.g. Anthropic-API-compatible gateways).
ALTER TABLE claude_accounts
    DROP CONSTRAINT IF EXISTS claude_accounts_backend_type_chk;

DROP INDEX IF EXISTS idx_claude_accounts_backend;

ALTER TABLE claude_accounts
    DROP COLUMN IF EXISTS backend_type,
    DROP COLUMN IF EXISTS provider,
    DROP COLUMN IF EXISTS base_url,
    DROP COLUMN IF EXISTS model,
    DROP COLUMN IF EXISTS api_key_env;
