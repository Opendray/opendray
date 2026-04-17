CREATE TABLE IF NOT EXISTS mcp_servers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    transport   TEXT NOT NULL DEFAULT 'stdio',
    command     TEXT NOT NULL DEFAULT '',
    args        JSONB NOT NULL DEFAULT '[]'::jsonb,
    env         JSONB NOT NULL DEFAULT '{}'::jsonb,
    url         TEXT NOT NULL DEFAULT '',
    headers     JSONB NOT NULL DEFAULT '{}'::jsonb,
    applies_to  JSONB NOT NULL DEFAULT '["*"]'::jsonb,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_mcp_servers_enabled ON mcp_servers(enabled);
