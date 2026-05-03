-- 0004_claude_accounts — replaces the v2-bootstrap API-key account
-- model (which was wrong for Claude Code's OAuth-based auth) with the
-- v1 multi-account model: rows hold metadata only, while the OAuth
-- token sits chmod 600 at token_path on the gateway host. The token
-- file is managed by the host tool `claude-acc` (login/logout) or via
-- the import-local endpoint.

CREATE TABLE claude_accounts (
    id           TEXT PRIMARY KEY DEFAULT 'cla_' || substr(md5(random()::text || clock_timestamp()::text), 1, 12),
    name         TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    config_dir   TEXT NOT NULL DEFAULT '',
    token_path   TEXT NOT NULL DEFAULT '',
    description  TEXT NOT NULL DEFAULT '',
    enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Per-session bindings.
--   claude_account_id: which account to inject (CLAUDE_CODE_OAUTH_TOKEN
--     + CLAUDE_CONFIG_DIR) at spawn time. NULL = let the CLI fall back
--     to its system-keychain login.
--   claude_session_id: opaque id captured from claude's stdout so a
--     restart can `--resume` and keep the conversation state.
ALTER TABLE sessions
    ADD COLUMN claude_account_id TEXT REFERENCES claude_accounts(id) ON DELETE SET NULL,
    ADD COLUMN claude_session_id TEXT;
