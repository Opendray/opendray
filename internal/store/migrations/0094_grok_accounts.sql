-- Grok Build (grok) multi-account support. Mirrors antigravity_accounts
-- (0069): a grok "account" is a dedicated GROK_HOME directory. grok keys
-- its credential + config state off GROK_HOME (<GROK_HOME>/auth.json,
-- config.toml, trusted_folders.toml), so binding a session to an account
-- means spawning `grok` with GROK_HOME pointed there. config_dir holds
-- that per-account GROK_HOME. No token column: the login token lives at
-- <GROK_HOME>/auth.json and is created out-of-band by `GROK_HOME=<dir>
-- grok login`; opendray only points spawns at it.
CREATE TABLE grok_accounts (
    id           TEXT PRIMARY KEY DEFAULT 'grok_' || substr(md5(random()::text || clock_timestamp()::text), 1, 12),
    name         TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    config_dir   TEXT NOT NULL DEFAULT '',
    description  TEXT NOT NULL DEFAULT '',
    enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE sessions
    ADD COLUMN grok_account_id TEXT REFERENCES grok_accounts(id) ON DELETE SET NULL;
