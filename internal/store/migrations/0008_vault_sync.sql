-- 0008_vault_sync — single-row settings + last-run timestamps for the
-- vault auto-sync feature. The single row is enforced by a CHECK on
-- id=1 so updates are always UPDATE … WHERE id=1 and we never grow.

CREATE TABLE vault_sync_config (
    id                  INT PRIMARY KEY CHECK (id = 1),
    enabled             BOOLEAN NOT NULL DEFAULT FALSE,
    commit_interval_ms  BIGINT  NOT NULL DEFAULT 600000,  -- 10 minutes
    push_enabled        BOOLEAN NOT NULL DEFAULT TRUE,
    pull_enabled        BOOLEAN NOT NULL DEFAULT FALSE,
    pull_interval_ms    BIGINT  NOT NULL DEFAULT 3600000, -- 1 hour
    commit_message      TEXT    NOT NULL DEFAULT '',      -- empty → timestamped default
    last_commit_at      TIMESTAMPTZ,
    last_commit_hash    TEXT,
    last_push_at        TIMESTAMPTZ,
    last_pull_at        TIMESTAMPTZ,
    last_error          TEXT,
    last_error_at       TIMESTAMPTZ,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO vault_sync_config (id) VALUES (1);
