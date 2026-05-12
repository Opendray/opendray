-- 0025_project_docs_journal — unified cross-agent memory layers 2-4.
--
-- Builds on the existing memory subsystem (layer 5 / discrete facts
-- in `memories`) and CLAUDE.md (layer 1 / project rules in git) to
-- close the loop on layers 2-4:
--
--   2. Project goal      — long-term project intent (cwd-scoped, single
--                          row per project).
--   3. Project plan      — current roadmap / WIP arc (cwd-scoped, single
--                          row per project).
--   4. Session journal   — append-only chronological log of what each
--                          session did (cwd-scoped, multi-row).
--
-- Why three tables not one — goal/plan are *replace-in-place* documents
-- with at most one current state per (cwd, kind); session_logs are
-- *append-only* with an unbounded number of rows per cwd. Different
-- shapes, different indexes.
--
-- Cross-agent angle: project_docs are read at spawn-time and injected
-- into the system prompt for claude / codex / gemini using each CLI's
-- native conv (--append-system-prompt, AGENTS.md, GEMINI.md). All
-- three agents see *identical* content; the bytes-on-the-wire path
-- differs per CLI but the source is the same DB row.

-- ── project_docs ───────────────────────────────────────────────
-- Goal + plan markdown bodies, one row per (cwd, kind). UPSERT-driven
-- — operators replace the whole document rather than diff-patch
-- subsections. UI surface is a single markdown textarea.
--
-- kind is constrained to 'goal' and 'plan' for now. Future kinds
-- (e.g. 'rationale', 'decision_index') would extend via CHECK
-- relaxation.
CREATE TABLE IF NOT EXISTS project_docs (
    id          TEXT PRIMARY KEY,
    cwd         TEXT NOT NULL,
    kind        TEXT NOT NULL CHECK (kind IN ('goal', 'plan')),
    content     TEXT NOT NULL DEFAULT '',
    -- updated_by records the actor that last wrote this row:
    --   'operator'  → human via UI / API
    --   'agent'     → an approved agent proposal merged in
    -- Useful for the UI to flag "auto-generated, please review".
    updated_by  TEXT NOT NULL DEFAULT 'operator'
        CHECK (updated_by IN ('operator', 'agent')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (cwd, kind)
);

CREATE INDEX IF NOT EXISTS project_docs_cwd_idx ON project_docs (cwd);

-- ── project_doc_proposals ──────────────────────────────────────
-- Agents (via MCP) can propose changes to goal/plan, but they don't
-- overwrite the live row directly — the change lands here in a
-- 'pending' state and the operator approves or rejects from the
-- mobile / web inbox. Decision 3 from the design discussion:
-- "agent through MCP changes goal/plan should require operator
-- confirm".
--
-- decided_at NULL = still pending. Indexed partially on that for the
-- inbox query.
CREATE TABLE IF NOT EXISTS project_doc_proposals (
    id                  TEXT PRIMARY KEY,
    cwd                 TEXT NOT NULL,
    kind                TEXT NOT NULL CHECK (kind IN ('goal', 'plan')),
    proposed_content    TEXT NOT NULL,
    -- session_id of the agent that proposed (may be NULL if a
    -- later integration writes proposals via direct API).
    proposed_by_session TEXT,
    -- agent's stated reason, shown in the inbox to help operator
    -- decide. May be empty.
    reason              TEXT NOT NULL DEFAULT '',
    decision            TEXT CHECK (decision IN ('approved', 'rejected')),
    decided_at          TIMESTAMPTZ,
    -- Snapshot of the prior content at approval time. Lets the UI
    -- show "before / after" diffs in the operator's history view
    -- without recomputing from logs.
    prior_content       TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS project_doc_proposals_pending_idx
    ON project_doc_proposals (cwd, created_at DESC)
    WHERE decided_at IS NULL;

CREATE INDEX IF NOT EXISTS project_doc_proposals_cwd_idx
    ON project_doc_proposals (cwd, created_at DESC);

-- ── session_logs ───────────────────────────────────────────────
-- Append-only journal entries. Each row is one bullet-point summary
-- of work done — either auto-generated at session end (M8 hook) or
-- manually appended by the operator / agent during a session.
--
-- session_id is nullable because manual entries (operator typed a
-- "we decided to rename foo to bar" line via UI) don't have one.
-- For auto-summaries it's the session that just ended.
--
-- kind:
--   'session_summary' → agent / summarizer-generated at session end
--   'manual'           → operator typed via UI
--   'decision'         → ADR-style entry (M7 decision_record tool)
CREATE TABLE IF NOT EXISTS session_logs (
    id          TEXT PRIMARY KEY,
    cwd         TEXT NOT NULL,
    session_id  TEXT REFERENCES sessions(id) ON DELETE SET NULL,
    kind        TEXT NOT NULL DEFAULT 'session_summary'
        CHECK (kind IN ('session_summary', 'manual', 'decision')),
    -- title is shown as a one-line preview in lists; content is the
    -- full markdown body shown when the operator drills in. Splitting
    -- avoids a regex/split at render time and keeps lists fast.
    title       TEXT NOT NULL DEFAULT '',
    content     TEXT NOT NULL DEFAULT '',
    -- updated_by mirrors project_docs but with two extra values for
    -- the journal-specific path:
    --   'operator' | 'agent' | 'summarizer' | 'manual'
    updated_by  TEXT NOT NULL DEFAULT 'summarizer'
        CHECK (updated_by IN ('operator', 'agent', 'summarizer', 'manual')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS session_logs_cwd_created_idx
    ON session_logs (cwd, created_at DESC);

CREATE INDEX IF NOT EXISTS session_logs_session_idx
    ON session_logs (session_id);
