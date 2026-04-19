-- plugin_consents: install-time capability grants, one row per installed plugin.
-- manifest_hash pins the exact manifest the user consented to — if a plugin
-- update changes its declared permissions the install flow re-prompts and
-- rewrites this row.
CREATE TABLE IF NOT EXISTS plugin_consents (
    plugin_name   TEXT PRIMARY KEY REFERENCES plugins(name) ON DELETE CASCADE,
    manifest_hash TEXT NOT NULL,
    perms_json    JSONB NOT NULL,
    granted_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
