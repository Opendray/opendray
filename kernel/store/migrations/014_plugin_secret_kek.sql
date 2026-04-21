-- plugin_secret_kek: per-plugin wrapped Data-Encryption-Key (DEK) used by
-- opendray.secret.*. The raw DEK is 32 random bytes; it's stored here
-- encrypted under the host Key-Encryption-Key (KEK). The KEK itself is
-- never persisted — it's derived at runtime via HKDF-SHA256 from the
-- admin bcrypt hash (see kernel/auth/secret_kek.go, lands in M3 T7).
--
-- Rotating the admin password rotates the KEK; a one-shot rewrap walk
-- at login time unwraps every wrapped_dek with the old KEK and rewraps
-- with the new one, bumping kek_kid accordingly.
--
-- M3 T4. The secret namespace (api_secret.go) lands in T13.
CREATE TABLE IF NOT EXISTS plugin_secret_kek (
    plugin_name  TEXT        PRIMARY KEY REFERENCES plugins(name) ON DELETE CASCADE,
    wrapped_dek  BYTEA       NOT NULL,
    kek_kid      TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
