-- 0020_memory_capture_rules — capture rules + injection profiles + cost log.
--
-- Three tables, one migration, because they're conceptually one
-- feature (ambient memory) and the application wiring needs all
-- three before anything works.

-- ── capture rules ──────────────────────────────────────────────
-- A rule says "for these sessions, every N new messages, run the
-- summarizer to extract durable facts and store them as memory
-- (with dedup against existing memories)".
--
-- session_id NULL = global default applying to every session
-- without a session-scoped override. trigger_kind is constrained
-- to 'after_messages' in Phase A; trigger_config is JSONB so
-- future on_idle / k_chars trigger types don't need new schema.
-- summarizer_provider_id NULL = use the registry's is_default row.

CREATE TABLE IF NOT EXISTS memory_capture_rules (
    id                        TEXT PRIMARY KEY,
    session_id                TEXT REFERENCES sessions(id) ON DELETE CASCADE,
    name                      TEXT NOT NULL,
    enabled                   BOOLEAN NOT NULL DEFAULT TRUE,
    trigger_kind              TEXT NOT NULL
        CHECK (trigger_kind IN ('after_messages')),
    trigger_config            JSONB NOT NULL DEFAULT '{}'::jsonb,
    summarizer_provider_id    TEXT REFERENCES memory_summarizer_providers(id) ON DELETE SET NULL,
    dedup_threshold           REAL NOT NULL DEFAULT 0.85
        CHECK (dedup_threshold BETWEEN 0 AND 1),
    target_scope              TEXT NOT NULL DEFAULT 'project'
        CHECK (target_scope IN ('session','project','global')),
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS memory_capture_rules_session_idx
    ON memory_capture_rules(session_id) WHERE session_id IS NOT NULL;

-- "at most one global default rule" — partial unique on the
-- session_id IS NULL slot. Multiple per-session rules are fine.
CREATE UNIQUE INDEX IF NOT EXISTS memory_capture_rules_global_idx
    ON memory_capture_rules((session_id IS NULL))
    WHERE session_id IS NULL AND enabled = TRUE;

-- ── injection profiles ─────────────────────────────────────────
-- Decides what (if anything) gets prepended to the agent's system
-- prompt at session spawn. Phase A strategies: 'none' (default —
-- model still uses memory_search on demand) | 'top_k_recent'
-- (inject N most recent project-scoped memories).

CREATE TABLE IF NOT EXISTS memory_injection_profiles (
    id              TEXT PRIMARY KEY,
    session_id      TEXT REFERENCES sessions(id) ON DELETE CASCADE,
    strategy_kind   TEXT NOT NULL
        CHECK (strategy_kind IN ('none','top_k_recent')),
    config          JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS memory_injection_profiles_global_idx
    ON memory_injection_profiles((session_id IS NULL))
    WHERE session_id IS NULL;

CREATE INDEX IF NOT EXISTS memory_injection_profiles_session_idx
    ON memory_injection_profiles(session_id) WHERE session_id IS NOT NULL;

-- ── summarizer call log ────────────────────────────────────────
-- One row per summarizer invocation. Drives the cost panel + lets
-- operators audit "why did this fact get extracted".
--
-- input_tokens/output_tokens are reported by the provider;
-- estimated_usd is computed locally from the pricing table at
-- call time so future price changes don't retroactively rewrite
-- history. facts_extracted/facts_stored differ when dedup or
-- store-failure drops some — both numbers exposed for transparency.

CREATE TABLE IF NOT EXISTS memory_summarizer_calls (
    id                        TEXT PRIMARY KEY,
    rule_id                   TEXT REFERENCES memory_capture_rules(id) ON DELETE SET NULL,
    provider_id               TEXT REFERENCES memory_summarizer_providers(id) ON DELETE SET NULL,
    session_id                TEXT REFERENCES sessions(id) ON DELETE SET NULL,
    started_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at               TIMESTAMPTZ,
    duration_ms               INTEGER,
    input_tokens              INTEGER NOT NULL DEFAULT 0,
    output_tokens             INTEGER NOT NULL DEFAULT 0,
    estimated_usd             NUMERIC(12,6) NOT NULL DEFAULT 0,
    facts_extracted           INTEGER NOT NULL DEFAULT 0,
    facts_stored              INTEGER NOT NULL DEFAULT 0,
    facts_skipped_dedup       INTEGER NOT NULL DEFAULT 0,
    status                    TEXT NOT NULL
        CHECK (status IN ('succeeded','failed','timeout','provider_unavailable')),
    error                     TEXT,
    raw_response_truncated    TEXT
);

CREATE INDEX IF NOT EXISTS memory_summarizer_calls_started_idx
    ON memory_summarizer_calls(started_at DESC);
CREATE INDEX IF NOT EXISTS memory_summarizer_calls_session_idx
    ON memory_summarizer_calls(session_id) WHERE session_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS memory_summarizer_calls_provider_idx
    ON memory_summarizer_calls(provider_id) WHERE provider_id IS NOT NULL;
