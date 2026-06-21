-- Antigravity (agy) multi-account support. Mirrors claude_accounts
-- (0004), but an antigravity "account" is a dedicated HOME directory:
-- agy keys its entire credential + conversation state off $HOME
-- (<HOME>/.gemini/antigravity-cli/antigravity-oauth-token), so binding
-- a session to an account means spawning `agy` with HOME pointed there.
-- config_dir holds that per-account HOME. No token_path column: the
-- OAuth token lives under the HOME and is created out-of-band by the
-- guided `agy` login; opendray only points spawns at it.
CREATE TABLE antigravity_accounts (
    id           TEXT PRIMARY KEY DEFAULT 'agy_' || substr(md5(random()::text || clock_timestamp()::text), 1, 12),
    name         TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    config_dir   TEXT NOT NULL DEFAULT '',
    description  TEXT NOT NULL DEFAULT '',
    enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE sessions
    ADD COLUMN antigravity_account_id TEXT REFERENCES antigravity_accounts(id) ON DELETE SET NULL;
