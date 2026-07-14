-- 0077_round_tables — Round Table (experimental): a cross-vendor AI GROUP
-- CHAT. Members are the seated providers (claude/codex/antigravity — the
-- three with a headless worker path) plus the operator. Anyone posts into
-- one shared thread; @mentioned members reply, reading the whole
-- conversation. Open-ended, like a Telegram group — no forced verdict; an
-- optional "summarize" asks a member to condense the discussion.
--
-- EXPERIMENTAL / rollback-able: idempotent (IF NOT EXISTS), touches NO
-- existing table/enum/CHECK. To remove the feature entirely, run
-- internal/roundtable/ROLLBACK.md (drops both tables) and delete the
-- feat/round-table branch.

CREATE TABLE IF NOT EXISTS round_tables (
    id                   TEXT PRIMARY KEY,
    topic                TEXT NOT NULL,
    cwd                  TEXT NOT NULL DEFAULT '',          -- optional project binding (memory recall + Phase 2 handoff)
    seats                JSONB NOT NULL DEFAULT '[]'::jsonb, -- [{provider, model, account_id}]
    status               TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'closed')),
    resulting_session_id TEXT NOT NULL DEFAULT '',          -- Phase 2 reserve: PTY session spawned from a summary
    origin               TEXT NOT NULL DEFAULT 'operator'
        CHECK (origin IN ('operator', 'integration')),
    integration_id       TEXT NOT NULL DEFAULT '',          -- isolation key when origin='integration'
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS round_tables_status_idx  ON round_tables (status);
CREATE INDEX IF NOT EXISTS round_tables_updated_idx ON round_tables (updated_at DESC);

-- One message in the group chat. seat_provider records WHICH member spoke
-- ('' for the operator / system). mentions lists the members a message
-- addressed (drives who replies). kind separates ordinary chat from an
-- on-demand summary.
CREATE TABLE IF NOT EXISTS round_table_messages (
    id             TEXT PRIMARY KEY,
    round_table_id TEXT NOT NULL REFERENCES round_tables(id) ON DELETE CASCADE,
    role           TEXT NOT NULL CHECK (role IN ('operator', 'seat', 'system')),
    seat_provider  TEXT NOT NULL DEFAULT '',
    seat_model     TEXT NOT NULL DEFAULT '',
    kind           TEXT NOT NULL DEFAULT 'message' CHECK (kind IN ('message', 'summary')),
    content        TEXT NOT NULL DEFAULT '',
    mentions       JSONB NOT NULL DEFAULT '[]'::jsonb,      -- providers this message @addressed
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS round_table_messages_table_idx
    ON round_table_messages (round_table_id, created_at);
