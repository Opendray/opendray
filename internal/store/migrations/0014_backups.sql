-- 0014_backups — operator-facing disaster-recovery backups.
--
-- Three tables work together:
--   backup_targets   — pluggable storage destinations (local, smb, ...).
--   backup_schedules — recurring backup specs (interval-based; cron in v1.1).
--   backups          — every materialised dump, success or fail.
--
-- pg_dump output is streamed through cipher (AES-GCM, key derived from
-- env OPENDRAY_BACKUP_KEY) into the chosen target. Bundle layout +
-- restore protocol live in app/web/src/tutorial/sections/12-backup/.

CREATE TABLE backup_targets (
    id         TEXT PRIMARY KEY,                          -- "local" | "smb-unas" | ...
    kind       TEXT NOT NULL,                             -- 'local' | 'smb' | 's3' (s3 reserved, not impl in v1)
    config     JSONB NOT NULL DEFAULT '{}'::jsonb,        -- kind-specific; sensitive fields stored ciphertext
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE backup_schedules (
    id           TEXT PRIMARY KEY,
    target_id    TEXT NOT NULL REFERENCES backup_targets(id) ON DELETE RESTRICT,
    interval_sec INT  NOT NULL CHECK (interval_sec > 0),  -- v1: simple interval; v1.1 adds cron_expr
    retention    INT  NOT NULL DEFAULT 7 CHECK (retention >= 0),
    enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    last_run_at  TIMESTAMPTZ,
    next_run_at  TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX backup_schedules_due_idx
    ON backup_schedules(next_run_at) WHERE enabled = TRUE;

CREATE TABLE backups (
    id               TEXT PRIMARY KEY,                    -- "bk_<22 base32 chars>"
    schedule_id      TEXT REFERENCES backup_schedules(id) ON DELETE SET NULL,
    target_id        TEXT NOT NULL REFERENCES backup_targets(id) ON DELETE RESTRICT,
    status           TEXT NOT NULL,                       -- 'pending'|'running'|'succeeded'|'failed'|'deleted'
    triggered_by     TEXT NOT NULL,                       -- 'scheduler'|'manual'|'api'
    started_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at      TIMESTAMPTZ,
    bytes            BIGINT NOT NULL DEFAULT 0,
    sha256           TEXT,                                -- of the final ciphertext blob
    encrypted        BOOLEAN NOT NULL DEFAULT TRUE,
    key_fingerprint  TEXT,                                -- first 16 hex chars of SHA-256(derived-key); restore must match
    target_path      TEXT,                                -- relative to target root
    pg_version       TEXT,                                -- server_version reported by pg_dump
    opendray_version TEXT,
    git_sha          TEXT,
    error            TEXT,                                -- populated when status='failed'
    metadata         JSONB NOT NULL DEFAULT '{}'::jsonb   -- include_config, etc.
);
CREATE INDEX backups_started_at_idx ON backups(started_at DESC);
CREATE INDEX backups_status_idx     ON backups(status);
CREATE INDEX backups_schedule_idx   ON backups(schedule_id) WHERE schedule_id IS NOT NULL;
