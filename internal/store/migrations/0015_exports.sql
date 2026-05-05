-- 0015_exports — admin-facing data export bundles.
--
-- Distinct from 0014 backups: an export is a one-shot, time-bounded
-- zip of selected logical entities (memories / integrations metadata
-- / custom tasks) generated for a specific operator and downloaded
-- once via a download_token. Files are kept in a local export dir
-- (cfg.Backup.ExportDir) and reaped on expires_at.
--
-- Export bundles are NOT encrypted as a whole — only sensitive
-- fields (e.g. plaintext API keys, when explicitly opted in) are
-- AEAD-wrapped per-row. Rationale lives in the v1 design doc.

CREATE TABLE exports (
    id             TEXT PRIMARY KEY,                       -- "exp_<22 base32 chars>"
    status         TEXT NOT NULL,                          -- 'pending'|'running'|'ready'|'failed'|'expired'
    requested_by   TEXT NOT NULL,                          -- admin username (for audit)
    scope          JSONB NOT NULL,                         -- {memories:bool, integrations:'none'|'metadata'|'plaintext', custom_tasks:bool}
    started_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at    TIMESTAMPTZ,
    expires_at     TIMESTAMPTZ NOT NULL,                   -- default NOW() + 24h; cleaner reaps after this
    bytes          BIGINT NOT NULL DEFAULT 0,
    sha256         TEXT,                                   -- of the zip on disk
    download_token TEXT NOT NULL UNIQUE,                   -- 32 base64url bytes; orthogonal to admin bearer
    file_path      TEXT,                                   -- absolute path under cfg.Backup.ExportDir
    error          TEXT
);
CREATE INDEX exports_expires_at_idx ON exports(expires_at);
CREATE INDEX exports_status_idx     ON exports(status);
