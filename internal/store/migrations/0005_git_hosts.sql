-- 0005_git_hosts — per-host API tokens for the Inspector's Git
-- panel. PR / issue / repo metadata fetches need a bearer token; one
-- entry per (host, kind) tuple. Tokens are stored plaintext — admin-
-- only API surface, same trust model as the claude_accounts on-disk
-- token files.

CREATE TABLE git_hosts (
    id          TEXT PRIMARY KEY DEFAULT 'gh_' || substr(md5(random()::text || clock_timestamp()::text), 1, 12),
    kind        TEXT NOT NULL CHECK (kind IN ('github', 'gitea', 'gitlab')),
    host        TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL DEFAULT '',
    token       TEXT NOT NULL DEFAULT '',
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
