-- M25 — Pluggable memory worker. Each LLM touchpoint
-- (gatekeeper, cleaner, gitactivity, transcript) can be served
-- either by the existing summarizer.Registry HTTP path (cheap,
-- local LM-Studio-friendly, low latency — good for high-frequency
-- tasks) or by spawning a headless Claude/Gemini agent session
-- (higher quality, higher latency, costs API tokens — good for
-- low-frequency narrative tasks).
--
-- memory_workers carries the per-task config. memory_worker_calls
-- captures every invocation's latency / outcome so operators can
-- see whether a swap is paying off in the UI.

CREATE TABLE IF NOT EXISTS memory_workers (
    task          TEXT PRIMARY KEY,
    kind          TEXT NOT NULL,
    summarizer_id TEXT,                          -- when kind='summarizer'; NULL = registry default
    provider_id   TEXT,                          -- when kind='agent': 'claude' | 'gemini'
    account_id    TEXT,                          -- when provider_id='claude': multi-account id
    enabled       BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT memory_workers_task_check
        CHECK (task IN ('gatekeeper','cleaner','gitactivity','transcript')),
    CONSTRAINT memory_workers_kind_check
        CHECK (kind IN ('summarizer','agent')),
    CONSTRAINT memory_workers_agent_needs_provider
        CHECK (kind = 'summarizer' OR (kind = 'agent' AND provider_id IS NOT NULL))
);

-- Seed with summarizer defaults so existing deployments behave
-- exactly as before until an operator chooses otherwise.
INSERT INTO memory_workers (task, kind) VALUES
    ('gatekeeper',  'summarizer'),
    ('cleaner',     'summarizer'),
    ('gitactivity', 'summarizer'),
    ('transcript',  'summarizer')
ON CONFLICT (task) DO NOTHING;

CREATE TABLE IF NOT EXISTS memory_worker_calls (
    id            BIGSERIAL PRIMARY KEY,
    task          TEXT NOT NULL,
    worker_kind   TEXT NOT NULL,
    provider_id   TEXT NOT NULL DEFAULT '',
    account_id    TEXT NOT NULL DEFAULT '',
    started_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    duration_ms   BIGINT NOT NULL,
    success       BOOLEAN NOT NULL,
    error_message TEXT NOT NULL DEFAULT '',
    input_bytes   INTEGER NOT NULL DEFAULT 0,
    output_bytes  INTEGER NOT NULL DEFAULT 0,
    tokens_in     INTEGER NOT NULL DEFAULT 0,
    tokens_out    INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS memory_worker_calls_task_started_idx
    ON memory_worker_calls (task, started_at DESC);

CREATE INDEX IF NOT EXISTS memory_worker_calls_started_idx
    ON memory_worker_calls (started_at DESC);
