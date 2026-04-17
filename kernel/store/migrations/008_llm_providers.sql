-- LLM Providers — address book of OpenAI-compatible model endpoints.
--
-- One row per upstream (Mac Ollama, LM Studio on another box, Groq
-- free tier, Gemini free tier, any custom OpenAI-compat service). NTC
-- itself runs no model; at session-spawn time the hub reads the
-- session's llm_provider_id + model, injects OPENAI_BASE_URL and
-- (optionally) Bearer token into the agent CLI (OpenCode, crush, …).
--
-- Rows are managed from the "LLM Providers" panel plugin
-- (plugins/panels/llm-providers/, category=endpoints). Model names
-- aren't stored here — we probe /v1/models on demand and let the user
-- pick (with free-text fallback when the upstream doesn't advertise).
CREATE TABLE IF NOT EXISTS llm_providers (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL UNIQUE,
    display_name  TEXT NOT NULL DEFAULT '',
    provider_type TEXT NOT NULL DEFAULT 'openai-compat',
    base_url      TEXT NOT NULL,
    api_key_env   TEXT NOT NULL DEFAULT '',
    description   TEXT NOT NULL DEFAULT '',
    enabled       BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_llm_providers_enabled ON llm_providers(enabled);

-- Per-session binding — mirrors the existing claude_account_id pattern
-- so spawn-time lookup is symmetric between Claude (oauth) and OpenCode
-- (provider-routed) sessions. NULL = no binding (agent reads whatever
-- env the host process has, i.e. the OpenCode default behaviour).
ALTER TABLE sessions
    ADD COLUMN IF NOT EXISTS llm_provider_id UUID
        REFERENCES llm_providers(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_sessions_llm_provider ON sessions(llm_provider_id);
