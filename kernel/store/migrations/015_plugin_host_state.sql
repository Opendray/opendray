-- plugin_host_state: per-plugin sidecar lifecycle stats maintained by
-- plugin/host.Supervisor. Lets the Settings UI show "restarted 3× in
-- last hour" and makes post-mortem investigation possible after a
-- supervisor restart.
--
-- Writes are best-effort — a supervisor must not fail to start a
-- sidecar because this table is temporarily unavailable.
--
-- M3 T5. Supervisor lands in T14.
CREATE TABLE IF NOT EXISTS plugin_host_state (
    plugin_name     TEXT        PRIMARY KEY REFERENCES plugins(name) ON DELETE CASCADE,
    last_started_at TIMESTAMPTZ,
    last_exit_code  INTEGER,
    restart_count   INTEGER     NOT NULL DEFAULT 0,
    last_error      TEXT
);
