-- 0001_initial — opendray-v2 baseline schema (design §10).
-- All tables are created in one migration to keep M0 simple. Subsequent
-- DDL changes land as additional NNNN_*.sql files.

CREATE TABLE providers (
    id            TEXT PRIMARY KEY,
    manifest_hash TEXT NOT NULL,
    config        JSONB NOT NULL DEFAULT '{}'::jsonb,
    enabled       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE sessions (
    id          TEXT PRIMARY KEY,
    name        TEXT,
    provider_id TEXT NOT NULL REFERENCES providers(id),
    cwd         TEXT NOT NULL,
    args        JSONB NOT NULL DEFAULT '[]'::jsonb,
    state       TEXT NOT NULL,
    pid         INT,
    started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at    TIMESTAMPTZ,
    exit_code   INT
);
CREATE INDEX sessions_state_idx ON sessions(state);

CREATE TABLE integrations (
    id               TEXT PRIMARY KEY,
    name             TEXT NOT NULL UNIQUE,
    base_url         TEXT NOT NULL,
    route_prefix     TEXT NOT NULL UNIQUE,
    api_key_hash     TEXT NOT NULL,
    scopes           JSONB NOT NULL DEFAULT '[]'::jsonb,
    version          TEXT,
    enabled          BOOLEAN NOT NULL DEFAULT TRUE,
    health_status    TEXT NOT NULL DEFAULT 'unknown',
    health_payload   JSONB,
    health_last_seen TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    rotated_at       TIMESTAMPTZ
);

CREATE TABLE channels (
    id         TEXT PRIMARY KEY,
    kind       TEXT NOT NULL,
    config     JSONB NOT NULL,
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE channel_messages (
    id              BIGSERIAL PRIMARY KEY,
    channel_id      TEXT NOT NULL REFERENCES channels(id),
    direction       TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    session_id      TEXT,
    author          TEXT,
    text            TEXT NOT NULL,
    metadata        JSONB,
    ts              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX channel_messages_conv_ts_idx
    ON channel_messages(conversation_id, ts DESC);

CREATE TABLE audit_log (
    id           BIGSERIAL PRIMARY KEY,
    ts           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actor_kind   TEXT NOT NULL,
    actor_id     TEXT,
    action       TEXT NOT NULL,
    subject_kind TEXT,
    subject_id   TEXT,
    metadata     JSONB
);
