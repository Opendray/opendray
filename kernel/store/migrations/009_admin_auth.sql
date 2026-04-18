-- Admin credentials for the /api/auth/login endpoint.
--
-- Single-row table (enforced via the id=1 check) that holds the
-- currently-active username and a bcrypt hash of the password.
--
-- When the row is absent the login handler falls back to the
-- ADMIN_USERNAME / ADMIN_PASSWORD env vars — those act as the
-- bootstrap credentials on a fresh install. The first successful
-- change-credentials call writes the row; thereafter the env values
-- are ignored and the operator manages the password from the
-- Settings page.
CREATE TABLE IF NOT EXISTS admin_auth (
    id            INT PRIMARY KEY CHECK (id = 1),
    username      TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
