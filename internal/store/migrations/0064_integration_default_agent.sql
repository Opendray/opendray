-- 0064 — per-integration default agent (provider + model + claude account)
-- and a first-class per-session model.
--
-- Third-party integrations (Cortex Phase 2 / #3) often spawn every
-- session against the same provider, model, and Claude account. Rather
-- than make the consumer repeat that on each POST /sessions, an operator
-- configures the integration's defaults once; session create fills any
-- field the request omits (the request still wins when it supplies one —
-- these are DEFAULTS, not enforcement).
--
--   default_provider_id:       '' (none) | 'claude' | 'codex' | 'gemini' | …
--   default_model:             '' (CLI/provider default) | a model id
--   default_claude_account_id: '' (none) | a cliacct id
--
-- `sessions.model` promotes model selection to a first-class request
-- field. Until now a model could only be pinned per-provider (provider
-- config `model`) or hand-typed into the spawn args (`--model X`). The
-- column lets a single session carry its own model — applied at spawn
-- time via the provider's model flag — so integration defaults (and a
-- future per-session model picker) have a clean home. Empty means
-- "fall back to the provider config default", preserving prior behavior.

ALTER TABLE integrations
    ADD COLUMN default_provider_id       TEXT NOT NULL DEFAULT '',
    ADD COLUMN default_model             TEXT NOT NULL DEFAULT '',
    ADD COLUMN default_claude_account_id TEXT NOT NULL DEFAULT '';

ALTER TABLE sessions
    ADD COLUMN model TEXT NOT NULL DEFAULT '';
