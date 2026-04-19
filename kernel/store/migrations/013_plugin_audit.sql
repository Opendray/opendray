-- plugin_audit: append-only log of every capability-gated call a plugin
-- made. Written synchronously by the bridge middleware before the call
-- proceeds; rows are never rewritten. Indexed by (plugin_name, ts DESC)
-- so the audit page can pull the recent N rows for one plugin fast.
CREATE TABLE IF NOT EXISTS plugin_audit (
    id           BIGSERIAL PRIMARY KEY,
    ts           TIMESTAMPTZ NOT NULL DEFAULT now(),
    plugin_name  TEXT NOT NULL,
    ns           TEXT NOT NULL,
    method       TEXT NOT NULL,
    caps         TEXT[] NOT NULL DEFAULT '{}',
    result       TEXT NOT NULL,
    duration_ms  INT  NOT NULL DEFAULT 0,
    args_hash    TEXT NOT NULL DEFAULT '',
    message      TEXT
);

CREATE INDEX IF NOT EXISTS idx_plugin_audit_plugin_ts ON plugin_audit(plugin_name, ts DESC);
