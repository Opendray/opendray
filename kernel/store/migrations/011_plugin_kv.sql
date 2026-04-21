-- plugin_kv: per-plugin key-value store backing opendray.storage.*.
-- Scaffolded in M1; no live writers until M2's bridge runtime.
CREATE TABLE IF NOT EXISTS plugin_kv (
    plugin_name TEXT NOT NULL REFERENCES plugins(name) ON DELETE CASCADE,
    key         TEXT NOT NULL,
    value       JSONB NOT NULL,
    size_bytes  INT  NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (plugin_name, key)
);

CREATE INDEX IF NOT EXISTS idx_plugin_kv_plugin ON plugin_kv(plugin_name);
