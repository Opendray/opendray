-- 0062 — per-conversation model override for the Cortex curation channel.
--
-- The "discuss with AI" channel (CurationService) runs on the global
-- `curation` worker config (memory_workers). These columns let a single
-- conversation pin its own cloud-agent provider + model from the chat UI,
-- without changing the global default. Empty (the default) means "use the
-- configured curation worker" — so existing conversations are unaffected.
--
--   provider_id: '' (use global) | 'claude' | 'gemini' | 'codex'
--   model:       '' (CLI/global default) | a model id for that provider

ALTER TABLE cortex_conversations
    ADD COLUMN provider_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN model       TEXT NOT NULL DEFAULT '';
