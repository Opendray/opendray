-- 0069 — Loop Engine (autoloop): persistent autonomous agent loops.
--
-- "Loop Engineering" (2026) is the shift from hand-writing prompts to writing
-- a system that drives the agent: a recursive goal, a way to find work, act,
-- verify, remember, and a stopping condition. opendray already has every
-- building block (PTY sessions to act, worker.Registry to verify, eventbus +
-- channels to escalate, memory to remember) — this migration persists the
-- orchestration layer (internal/autoloop) so loops survive a gateway restart.
--
-- A Loop is 1:1 with a session it drives. Two kinds share one lifecycle:
--   - interval: re-feed `prompt` every `interval_seconds`; stop on caps.
--   - goal:     feed `prompt` (seed), wait for turn_completed, run the
--               `judge_task` worker → continue|done|escalate|fail.
--
-- Provider-agnostic: the engine drives sessions at the PTY layer
-- (ExpectTurn/Input/turn_completed), so a loop works the same over
-- claude/codex/gemini/antigravity/opencode — `Loop` holds no provider field;
-- the bound session determines the CLI.

CREATE TABLE session_loops (
    id               TEXT PRIMARY KEY,
    session_id       TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    -- origin mirrors session.Origin: who created the loop.
    origin           TEXT NOT NULL DEFAULT 'operator'
                     CHECK (origin IN ('operator', 'integration')),
    integration_id   TEXT REFERENCES integrations(id) ON DELETE SET NULL,
    kind             TEXT NOT NULL CHECK (kind IN ('interval', 'goal')),
    status           TEXT NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending', 'running', 'paused',
                                       'done', 'stopped', 'failed', 'escalated')),
    -- goal mode: the recursive objective the judge evaluates against.
    goal             TEXT NOT NULL DEFAULT '',
    -- interval mode: re-fed each tick. goal mode: the seed prompt.
    prompt           TEXT NOT NULL,
    -- interval mode only; floored at 30s by the engine. NULL for goal loops.
    interval_seconds INT,
    -- hard caps (at least one of max_iterations / deadline_at is enforced at
    -- creation; deadline_at is mandatory per the design's guardrail-first rule).
    max_iterations   INT NOT NULL DEFAULT 20,
    deadline_at      TIMESTAMPTZ,
    -- consecutive judge=fail / turn-timeout count that flips the loop to
    -- 'escalated' (hand it to a human) rather than retrying forever.
    failure_cap      INT NOT NULL DEFAULT 3,
    -- worker TaskKind used to verify a goal turn (e.g. 'loop_judge'); NULL =
    -- no verification (interval loops, or a goal loop that only caps on count).
    judge_task       TEXT,
    iteration        INT NOT NULL DEFAULT 0,
    last_verdict     TEXT,
    last_reason      TEXT,
    -- forward-compat spillover (e.g. future token/cost budget).
    config           JSONB NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at       TIMESTAMPTZ,
    ended_at         TIMESTAMPTZ
);

-- ReconcileStartup scans only live loops; partial index keeps it cheap.
CREATE INDEX idx_session_loops_live
    ON session_loops (status)
    WHERE status IN ('running', 'paused');
CREATE INDEX idx_session_loops_session ON session_loops (session_id);

-- Per-iteration audit. This is a SCOPED audit trail (one row per turn of a
-- known loop), not a generic raw event stream — opendray deliberately avoids
-- generic event-log viewers.
CREATE TABLE session_loop_runs (
    id         BIGSERIAL PRIMARY KEY,
    loop_id    TEXT NOT NULL REFERENCES session_loops(id) ON DELETE CASCADE,
    iteration  INT NOT NULL,
    prompt     TEXT NOT NULL,
    verdict    TEXT,
    reason     TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at   TIMESTAMPTZ
);
CREATE INDEX idx_session_loop_runs_loop ON session_loop_runs (loop_id, iteration);

-- Register 'loop_judge' as a memory-worker touchpoint: the goal-mode verifier
-- that reads a turn's output and returns continue|done|escalate|fail. Seeded
-- kind='summarizer' so it is provider-agnostic by default (an HTTP judge that
-- works the same regardless of which CLI the driven session runs) and cheap;
-- an operator can flip it to an agent worker later (subject to the per-CLI
-- headless support matrix — opencode has no headless path).
ALTER TABLE memory_workers
    DROP CONSTRAINT IF EXISTS memory_workers_task_check;

ALTER TABLE memory_workers
    ADD CONSTRAINT memory_workers_task_check
        CHECK (task IN (
            'gatekeeper','cleaner','gitactivity','transcript',
            'plan_drift','conflict_detector','capture',
            'blueprint','curation','loop_judge'
        ));

INSERT INTO memory_workers (task, kind) VALUES
    ('loop_judge', 'summarizer')
ON CONFLICT (task) DO NOTHING;
