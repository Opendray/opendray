-- Claude multi-account support.
--
-- Each row points at the `claude-acc` tool's on-disk sandbox for one
-- subscription account (see ~/.claude-accounts/<name>/ and tokens/<name>.token).
-- The gateway reads the token file lazily at session-spawn time and injects
-- CLAUDE_CODE_OAUTH_TOKEN + CLAUDE_CONFIG_DIR as env overrides. Tokens never
-- enter Postgres — they stay chmod 600 on the host.
CREATE TABLE IF NOT EXISTS claude_accounts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    config_dir   TEXT NOT NULL,
    token_path   TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    enabled      BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_claude_accounts_enabled ON claude_accounts(enabled);

-- Optional per-session binding. NULL = use system keychain / env var (legacy).
ALTER TABLE sessions
    ADD COLUMN IF NOT EXISTS claude_account_id UUID REFERENCES claude_accounts(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_sessions_claude_account ON sessions(claude_account_id);
