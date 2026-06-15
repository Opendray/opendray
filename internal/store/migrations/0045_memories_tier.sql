-- 0045 — memory tiers (Cortex Phase 2).
--
-- tier is a storage-level fact filtered at the store layer, so every
-- consumer (ambient injector, knowledge anchorer, KB drafter, conflict
-- detector, list/search APIs) sees durable-only by default — one filter
-- instead of N call-site patches.
--
--   durable    — normal long-term memory (all existing rows)
--   quarantine — captured from a session whose integration policy is
--                'quarantine'; excluded from consolidation and injection
--                until explicitly promoted; auto-expired after a TTL.

ALTER TABLE memories
    ADD COLUMN tier TEXT NOT NULL DEFAULT 'durable'
        CHECK (tier IN ('durable', 'quarantine')),
    ADD COLUMN quarantine_expires_at TIMESTAMPTZ;

-- The quarantine review queue lists/counts only this slice; keep the
-- index partial so the durable majority pays nothing.
CREATE INDEX IF NOT EXISTS idx_memories_quarantine
    ON memories (tier, quarantine_expires_at)
    WHERE tier = 'quarantine';
