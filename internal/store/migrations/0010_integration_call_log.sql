-- 0010_integration_call_log — per-call audit for integration-attributed
-- API traffic. Captures both directions:
--
--   inbound  — third-party app called our /api/v1/* with its API key
--   outbound — admin called /api/v1/proxy/{prefix}/* and we forwarded
--              the request to the integration's BaseURL
--
-- The lifecycle audit_log is for *configuration* changes (registered,
-- key_rotated, health_changed). This table is for *traffic*. They are
-- intentionally separate so retention, indexing, and write volume can
-- be tuned independently — call traffic is potentially 1000x audit.

CREATE TABLE integration_call_log (
    id              BIGSERIAL PRIMARY KEY,
    ts              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    integration_id  TEXT        NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
    direction       TEXT        NOT NULL CHECK (direction IN ('inbound', 'outbound')),
    method          TEXT        NOT NULL,
    path            TEXT        NOT NULL,
    status_code     INT         NOT NULL,
    duration_ms     INT         NOT NULL,
    bytes_written   BIGINT,
    request_id      TEXT,
    -- Optional resource the call touched, parsed from path/response.
    -- Filled best-effort by the middleware; nullable.
    resource_kind   TEXT,
    resource_id     TEXT
);

-- "Show me what integration X has been doing, newest first"
CREATE INDEX idx_intgr_call_by_intgr
    ON integration_call_log (integration_id, ts DESC);

-- "Show me everyone's recent calls"
CREATE INDEX idx_intgr_call_by_ts
    ON integration_call_log (ts DESC);

-- "Show me failures" — partial index keeps it tiny.
CREATE INDEX idx_intgr_call_errors
    ON integration_call_log (ts DESC)
    WHERE status_code >= 400;
