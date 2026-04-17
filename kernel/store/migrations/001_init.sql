-- Sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL DEFAULT '',
    session_type    TEXT NOT NULL DEFAULT 'claude',
    claude_session_id TEXT,
    cwd             TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'stopped',
    model           TEXT NOT NULL DEFAULT '',
    pid             INTEGER,
    extra_args      JSONB DEFAULT '[]'::jsonb,
    env_overrides   JSONB DEFAULT '{}'::jsonb,
    total_cost_usd  NUMERIC(10,4) DEFAULT 0,
    input_tokens    BIGINT DEFAULT 0,
    output_tokens   BIGINT DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_active_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_sessions_last_active ON sessions(last_active_at DESC);

-- Plugins table
CREATE TABLE IF NOT EXISTS plugins (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL UNIQUE,
    version         TEXT NOT NULL DEFAULT '0.0.0',
    enabled         BOOLEAN NOT NULL DEFAULT true,
    manifest        JSONB NOT NULL DEFAULT '{}'::jsonb,
    health_status   TEXT NOT NULL DEFAULT 'unknown',
    health_checked_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_plugins_enabled ON plugins(enabled);
