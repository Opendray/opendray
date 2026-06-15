-- 0063 — extend the per-conversation curation override to LOCAL models.
--
-- 0062 added provider_id/model for cloud-agent (claude/gemini/codex) overrides.
-- A discussion can also run on a configured summarizer/HTTP provider (LM Studio
-- / Ollama / OpenAI-compatible — i.e. a "local model"). This column pins which
-- summarizer_providers row to use. The three columns are mutually exclusive:
--
--   summarizer_id set → WorkerSummarizer (local/HTTP model)
--   provider_id   set → WorkerAgent (cloud-agent CLI + model)
--   both empty        → use the global `curation` worker config
--
-- Empty (the default) keeps existing conversations on the global default.

ALTER TABLE cortex_conversations
    ADD COLUMN summarizer_id TEXT NOT NULL DEFAULT '';
