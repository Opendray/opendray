# ADR 0018 — Unified cross-CLI memory layer

**Status**: Accepted (shipped via PR #56, milestones M5-M24)
**Date**: 2026-05-13
**Builds on**: ADR 0013 (ambient memory), ADR 0014 (memory subsystem)

## Context

ADR 0014 (memory subsystem) gave opendray a shared pgvector store
that any CLI can read/write via MCP. ADR 0013 (ambient memory)
added auto-injection of relevant facts at spawn time. Both are
necessary but not sufficient — operators still saw three gaps:

1. **No project-level coherence.** Each agent learnt facts in
   isolation. A new Claude session asked to "continue the auth
   refactor" had no idea which auth refactor, what was already
   done, or what the operator's plan was.

2. **No structured project state.** "Goal", "current plan",
   "tech stack", "what just happened in this session" — none of
   these had a place to live. Agents had to re-grep / re-`ls` the
   project every spawn.

3. **Memory quality erosion.** With agents storing anything they
   judged "relevant", the memory table accumulated ephemeral
   debugging state ("currently on line 412 of app.go"),
   duplicates, and "thanks bye" niceties. There was no
   maintenance system.

## Decision

Build a five-layer project-scoped memory model on top of the M5
schema, with three quality-control mechanisms, three isolation
guarantees, and per-CLI native delivery channels.

### Five layers (read at every agent spawn)

| Layer | Source | Author | Lifecycle |
|---|---|---|---|
| **L1 — Tech stack & structure** | `internal/projectscan/` walks marker files + git HEAD | scanner | refreshed when stale ≥ 6h, on spawn |
| **L2 — Project goal** | `project_docs.kind='goal'` | operator (agent can propose) | hand-edited via UI; agent edits go through `project_doc_proposals` approval |
| **L3 — Project plan** | `project_docs.kind='plan'` | operator (agent can propose) | same proposal flow |
| **L4 — Recent activity** | `internal/gitactivity/` shells git log + LLM-summarises | scanner | 24h ticker + spawn freshness check |
| **L5 — Journal + facts** | `session_logs` + `memories` (pgvector) | event-driven (session.ended) + agent MCP calls | per session-end auto-entry; agent fact stores gated |

A single `RenderForSpawn(cwd)` call (internal/projectdoc/projectdoc.go)
composes all five into a ~4-5KB markdown banner. Each provider
delivers it through its native channel:

- **Claude Code**: `--append-system-prompt <banner>` + `--session-id <uuid>`
- **Codex**: appended to `$CODEX_HOME/AGENTS.md`
- **Gemini**: `<baseDir>/GEMINI.md` + `--include-directories <baseDir>` + `--session-id <uuid>`

### Three quality controls (anti-noise)

1. **Gatekeeper (M12)**. Every `memory_store` MCP call is pre-
   judged by an LLM classifier. Outputs `user_preference` /
   `project_fact` / `feedback` / `reference` / `ephemeral`. The
   `ephemeral` bucket is rejected with a typed `ErrNotDurable`.

2. **Server-side dedup (M11)**. After classification, the
   embedder produces a vector; if cosine similarity to an
   existing entry in the same scope exceeds threshold (0.85 for
   dense embedders, 0.2 for BM25 dev setups), the new entry is
   *merged* into the existing row with `deduped_count++`. No
   client-side coordination needed.

3. **LLM librarian (M13)**. A scheduler runs every 24h over
   aged-eligible memories and asks the LLM to judge each as
   `keep` / `stale` / `duplicate`. Output goes to
   `memory_cleanup_decisions` (status=pending). Operator approves
   in the Cleanup inbox before any deletion or merge executes.

### Three isolation guarantees (M22 — anti-contamination)

Real-world testing exposed Claude Code v2.1.126 inconsistently
honoring `--session-id`. When it generates its own UUID, the
transcript reader's mtime fallback can pick up an unrelated
session's jsonl, and the LLM journal summariser produces a
plausibly-worded summary of the wrong work — silent
misinformation that would mislead every future agent in that cwd.
Three defenses, applied in transcript reading:

1. **Fail-closed on missing UUID file**. When the caller passes a
   `claude_session_id`, the reader looks for that exact file. If
   it isn't there, return nil — never substitute "latest mtime in
   dir".

2. **Time-window filter**. Every parsed turn must have a
   timestamp within `[startedAt - 30s, endedAt + 30s]`. Even when
   the right file is opened, if it accumulates content across
   multiple opendray spawns in the same cwd, only the current
   spawn's turns survive.

3. **Cwd canary**. The first jsonl entry that carries a `cwd`
   field must match the calling session's cwd exactly. One
   mismatch and the whole file is abandoned. Catches the worst
   case — a jsonl from a different project mis-routed into this
   project's dir.

### Token-budget optimisation (M23 — AI-first text)

Memory is consumed primarily by the agent at spawn, not by humans
browsing UI. The spawn banner pruned operator-facing flourishes
(intro essays, "auto-generated, do not hand-edit" disclaimers,
last-scanned timestamps) while keeping LLM-load-bearing structure
(section headers, file-path backticks, behaviour-constraint
footers). Net saving: ~15-25% spawn token cost.

## Architecture

```
                  ┌─────────────────────────────────────────┐
                  │ At spawn (catalog/adapter.go::Prepare)  │
                  │                                          │
                  │  1. Generate UUID, inject --session-id   │
                  │     (M21)                                │
                  │  2. Refresh tech_stack if stale (sync)   │
                  │  3. RefreshAsync git activity            │
                  │  4. RenderForSpawn(cwd) → ~4-5KB banner  │
                  │  5. Per-CLI inject (Claude/Codex/Gemini) │
                  └────┬────────────────────────────────────┘
                       │
                       ▼
         ┌──────────────────────────────────┐
         │ Agent runs, may call MCP:        │
         │  memory_store(text, scope=…)     │
         │   → gatekeeper.Judge → ErrNotDurable │
         │   → embedder → dedup search      │
         │   → upsert into memories table   │
         │  project_goal_set(content)       │
         │   → project_doc_proposals insert │
         │  session_log_append(...)         │
         │   → session_logs insert          │
         └──────────┬───────────────────────┘
                    │
                    ▼ session ends
         ┌──────────────────────────────────┐
         │ eventbus.session.ended           │
         │   → projectdoc/journaler.go      │
         │   → lookup.TranscriptText        │
         │     (M22 fail-closed + window +  │
         │      cwd-canary filtering)       │
         │   → summariser.LLM 1-3 paragraphs│
         │   → AppendLog(session_logs)      │
         └──────────────────────────────────┘

         Periodic (24h tick):
           ├─ gitactivity.Run → updates L4
           └─ cleaner.Run     → writes memory_cleanup_decisions

         Operator approves proposals + cleanup via mobile/web UI.
```

## Schema delta

Four migrations (forward-only, additive, no backfills):

- `0025_project_docs_journal.sql` — `project_docs`,
  `project_doc_proposals`, `session_logs`
- `0026_memory_cleanup_decisions.sql` —
  `memory_cleanup_decisions` with pending/approved/rejected/
  executed/expired status states
- `0027_project_docs_tech_stack.sql` — widens
  `project_docs.kind` CHECK to allow `tech_stack`; widens
  `updated_by` CHECK to allow `scanner`
- `0028_project_docs_recent_activity.sql` — adds
  `recent_activity` kind

`sessions` table gains `claude_session_id` column population (M21
fills it via `--session-id` flag at spawn).

## Consequences

**Positive**:
- New agent on day 1 of project = old agent on day N, modulo
  things the operator decided not to journal.
- Memory quality bounded — gatekeeper + dedup + cleaner mean the
  store doesn't degrade over time without operator intervention.
- Cross-CLI parity: Claude / Codex / Gemini see the same project
  state via their respective native injection channels.
- Project A's records can't leak into project B's spawn banner
  even when Claude Code mis-routes jsonl files (M22).

**Negative**:
- LLM-driven features (gatekeeper, cleaner, summariser, git
  activity narrative) require an OpenAI-compatible endpoint
  configured (LM Studio default; or ChatGPT-OAuth via summarizer
  registry). Without one, the system degrades to metadata-only
  journaling and skips dedup/cleanup — works, just leaner.
- Scanner cache (6h) means tech_stack and git activity can lag
  briefly after major project shifts; mitigated by spawn-time
  staleness check that triggers sync refresh when overdue.
- Operator approval queues (proposals + cleanup) require touch.
  Inactive operators leave goal/plan proposals pending forever;
  mitigated only by cleanup-staleness sweeps (TODO: M25).

## Follow-ups

- **M25 — pluggable Memory worker**. Today all four LLM touch
  points go through the same `summarizer.Registry` HTTP path.
  M25 will add an `AgentWorker` implementation that spawns a
  headless Claude/Codex/Gemini session in `--print` mode to do
  the work, with per-task-kind switching (operator picks
  summarizer vs agent per touch-point in mobile Settings).
- **Codex session UUID capture**. Codex has no `--session-id`
  flag; currently relies on cwd-based matching + M22 time-window
  filtering. Robust for single-session-per-cwd; multi-concurrent
  Codex would benefit from parsing `session_meta.id` from the
  first rollout line post-spawn.
- **Self-healing cleaner**. Extending the cleaner to detect
  cross-cwd file refs in journal summaries (hallucination signal)
  using the same provider chain — opendray would QA its own
  memory. Sketched in M-verify discussions.
- **`docs/memory-system.md`** — operator-facing guide that
  expands on this ADR with concrete UI walkthroughs and SQL recipes.
