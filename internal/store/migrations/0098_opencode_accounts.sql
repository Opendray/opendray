-- OpenCode multi-account support. Unlike claude/agy/grok, an opencode
-- "account" is not a directory: opencode reads its whole credential set
-- from the OPENCODE_AUTH_CONTENT env var when present (same JSON shape as
-- ~/.local/share/opencode/auth.json, and it fully REPLACES the on-disk
-- set). So an account is a named credential bundle, injected per spawn,
-- while every account shares the one session DB
-- (~/.local/share/opencode/opencode.db) — which is what lets a live
-- session switch accounts and `--session <id>` resume the same chat.
--
-- credentials_enc holds the bundle as a FieldCipher "v1:" AES-GCM
-- envelope (the live backup cipher, same as summarizer api keys). Never
-- plaintext: creation is refused until the cipher is armed. providers is
-- the non-secret list of provider ids in the bundle, for display.
CREATE TABLE opencode_accounts (
    id              TEXT PRIMARY KEY DEFAULT 'oc_' || substr(md5(random()::text || clock_timestamp()::text), 1, 12),
    name            TEXT NOT NULL UNIQUE,
    display_name    TEXT NOT NULL DEFAULT '',
    description     TEXT NOT NULL DEFAULT '',
    credentials_enc TEXT NOT NULL DEFAULT '',
    providers       JSONB NOT NULL DEFAULT '[]'::jsonb,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE sessions
    ADD COLUMN opencode_account_id TEXT REFERENCES opencode_accounts(id) ON DELETE SET NULL;
