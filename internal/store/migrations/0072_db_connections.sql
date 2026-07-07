-- 0072_db_connections — Database tool: per-project external database
-- connections (JetBrains-style database tool window).
--
-- A connection belongs to a project (keyed by cwd, matching project_docs /
-- project_lifecycle). Discrete host/port/db fields rather than a DSN string
-- so the UI can edit per-field and the password can be encrypted separately
-- (FieldCipher "v1:" envelope, same as channels.config secrets; plaintext
-- fallback when the backup key isn't armed).
--
-- driver is CHECK-constrained to 'postgres' for v1; widen in a later
-- migration when more engines land.
CREATE TABLE IF NOT EXISTS db_connections (
    id           TEXT PRIMARY KEY,
    cwd          TEXT NOT NULL,
    name         TEXT NOT NULL,
    driver       TEXT NOT NULL DEFAULT 'postgres' CHECK (driver IN ('postgres')),
    host         TEXT NOT NULL,
    port         INT  NOT NULL DEFAULT 5432,
    db_name      TEXT NOT NULL,
    username     TEXT NOT NULL,
    password_enc TEXT NOT NULL DEFAULT '',
    ssl_mode     TEXT NOT NULL DEFAULT 'prefer',
    read_only    BOOLEAN NOT NULL DEFAULT FALSE,
    options      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (cwd, name)
);

CREATE INDEX IF NOT EXISTS db_connections_cwd_idx ON db_connections (cwd);
