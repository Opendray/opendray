-- 0077_round_tables — Round Table (experimental): cross-vendor
-- multi-agent discussion. A deterministic chair drives N heterogeneous
-- provider seats (claude/codex/antigravity) through a fixed
-- propose → critique → synthesize schedule and produces a structured
-- Verdict for the operator to approve.
--
-- EXPERIMENTAL / rollback-able: this migration is idempotent
-- (IF NOT EXISTS) and touches NO existing tables or enums. To roll the
-- feature back entirely, run internal/roundtable/ROLLBACK.md (drops both
-- tables) and delete the feat/round-table branch — nothing else in the
-- schema references these tables.

CREATE TABLE IF NOT EXISTS round_tables (
    id                    TEXT PRIMARY KEY,
    topic                 TEXT NOT NULL,
    cwd                   TEXT NOT NULL DEFAULT '',          -- optional project binding (memory recall + Phase 2 handoff)
    seats                 JSONB NOT NULL DEFAULT '[]'::jsonb, -- [{provider, model, account_id}]
    status                TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'running', 'awaiting_verdict', 'failed', 'closed')),
    verdict               JSONB,                             -- structured synthesis (null until awaiting_verdict)
    resulting_session_id  TEXT NOT NULL DEFAULT '',          -- Phase 2 reserve: PTY session spawned on approval
    error                 TEXT NOT NULL DEFAULT '',
    origin                TEXT NOT NULL DEFAULT 'operator'
        CHECK (origin IN ('operator', 'integration')),
    integration_id        TEXT NOT NULL DEFAULT '',          -- isolation key when origin='integration'
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS round_tables_status_idx  ON round_tables (status);
CREATE INDEX IF NOT EXISTS round_tables_updated_idx ON round_tables (updated_at DESC);

CREATE TABLE IF NOT EXISTS round_table_turns (
    id             TEXT PRIMARY KEY,
    round_table_id TEXT NOT NULL REFERENCES round_tables(id) ON DELETE CASCADE,
    beat           TEXT NOT NULL CHECK (beat IN ('propose', 'critique', 'synthesize')),
    seat_provider  TEXT NOT NULL DEFAULT '',                 -- which seat spoke; '' for the chair
    seat_model     TEXT NOT NULL DEFAULT '',
    role           TEXT NOT NULL CHECK (role IN ('seat', 'chair', 'system')),
    content        TEXT NOT NULL DEFAULT '',                 -- raw seat output
    structured     JSONB,                                    -- parsed structured fields
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS round_table_turns_table_idx
    ON round_table_turns (round_table_id, created_at);
