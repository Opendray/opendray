-- 0044 — session provenance + per-integration memory policy (Cortex Phase 2).
--
-- opendray is a third-party-consumable platform: sessions are created by
-- the operator (web/mobile UI), by the local CLI, or by external apps via
-- scoped API keys. Until now those were indistinguishable, so every
-- session fed the memory capture pipeline equally and third-party temp
-- sessions polluted the operator's long-term memory (the complaint that
-- triggered the Cortex re-architecture).
--
-- origin is derived from the authenticated principal at create time —
-- never client-supplied. Existing rows predate scoped-key session
-- creation, so backfilling them as 'operator' is correct by construction.

ALTER TABLE sessions
    ADD COLUMN origin TEXT NOT NULL DEFAULT 'operator'
        CHECK (origin IN ('operator', 'integration', 'cli')),
    ADD COLUMN integration_id TEXT;

-- memory_policy declares what the capture pipeline does with sessions an
-- integration creates:
--   none       — the session never produces memory
--   quarantine — facts land in the quarantine tier (excluded from
--                consolidation + spawn injection; reviewable + promotable)
--   full       — trusted: facts are durable, same as operator sessions
-- Default quarantine: safe-by-default for every existing registration.
ALTER TABLE integrations
    ADD COLUMN memory_policy TEXT NOT NULL DEFAULT 'quarantine'
        CHECK (memory_policy IN ('none', 'quarantine', 'full'));

-- System integrations are opendray-managed and serve the OPERATOR's
-- own sessions (e.g. the auto-registered opendray-memory MCP that
-- agents use to store/recall memory). Quarantining those writes would
-- break the operator's own memory pipeline — they are trusted.
UPDATE integrations SET memory_policy = 'full' WHERE COALESCE(is_system, FALSE);
