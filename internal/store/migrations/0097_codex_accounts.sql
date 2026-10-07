-- OpenAI Codex CLI (codex) multi-account support. Mirrors grok_accounts
-- (0094): a codex "account" is a dedicated CODEX_HOME directory holding
-- its own <CODEX_HOME>/auth.json (ChatGPT OAuth or API key), created
-- out-of-band by `CODEX_HOME=<dir> codex login` (or the API-key login
-- endpoint). config_dir holds that per-account CODEX_HOME. No token
-- column: opendray never stores the credential, it only points spawns at
-- the account's home.
CREATE TABLE codex_accounts (
    id           TEXT PRIMARY KEY DEFAULT 'cdx_' || substr(md5(random()::text || clock_timestamp()::text), 1, 12),
    name         TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    config_dir   TEXT NOT NULL DEFAULT '',
    description  TEXT NOT NULL DEFAULT '',
    enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE sessions
    ADD COLUMN codex_account_id TEXT REFERENCES codex_accounts(id) ON DELETE SET NULL;
