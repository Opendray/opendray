-- Additional operator logins (multi-user Tier 1). The primary admin keeps
-- living in the keyfile / config / env (internal/auth/keyfile.go) so it
-- stays the recovery path; this table only holds the extra users an
-- admin creates. role gates the admin-only route group; it is NOT an
-- isolation boundary — every session still runs as the opendray OS user.
-- Usernames are unique case-insensitively; password_hash is bcrypt.
CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY DEFAULT 'usr_' || substr(md5(random()::text || clock_timestamp()::text), 1, 12),
    username      TEXT NOT NULL,
    display_name  TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
    disabled      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS users_username_lower_idx ON users (lower(username));

-- owner is the username of the operator who created the session, stamped
-- from the authenticated principal (never the request body). NULL on rows
-- created before this migration and on integration / CLI sessions.
ALTER TABLE sessions
    ADD COLUMN IF NOT EXISTS owner TEXT;
