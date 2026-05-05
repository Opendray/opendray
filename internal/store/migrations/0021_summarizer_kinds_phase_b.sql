-- 0021_summarizer_kinds_phase_b — extend the CHECK on
-- memory_summarizer_providers.kind to accept the Phase B
-- backends:
--
--   openai      — real OpenAI API (gpt-4o-mini etc.)
--   lmstudio    — LM Studio's OpenAI-compatible local endpoint
--   integration — any opendray-registered integration whose
--                 scopes include 'memory:summarize'; opendray
--                 forwards summarize calls via reverse proxy
--
-- We DROP the old constraint by name and ADD the new one in a
-- single transaction so a partially-applied migration leaves
-- behind no half-state.

ALTER TABLE memory_summarizer_providers
    DROP CONSTRAINT IF EXISTS memory_summarizer_providers_kind_check;

ALTER TABLE memory_summarizer_providers
    ADD CONSTRAINT memory_summarizer_providers_kind_check
    CHECK (kind IN ('anthropic','ollama','openai','lmstudio','integration'));
