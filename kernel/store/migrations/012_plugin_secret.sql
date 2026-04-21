-- plugin_secret: per-plugin encrypted secret store backing opendray.secret.*.
-- ciphertext is sealed by the host's data-encryption key, not this row's
-- contents — rotation happens at the key layer. Scaffolded in M1; no
-- live writers until M2.
CREATE TABLE IF NOT EXISTS plugin_secret (
    plugin_name TEXT NOT NULL REFERENCES plugins(name) ON DELETE CASCADE,
    key         TEXT NOT NULL,
    ciphertext  BYTEA NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (plugin_name, key)
);
