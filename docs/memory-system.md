# opendray memory system — operator guide

This guide walks through what opendray's unified memory system
does, how to use it day-to-day, and how to verify it's behaving.

## What it is

Every Claude / Codex / Gemini session you spawn through opendray
boots up reading the same five-layer project context. Sessions
that end automatically write back a journal entry. Long-term facts
agents discover go through a quality gate before they're stored,
and a periodic LLM librarian proposes cleanups for your review.

The cross-CLI value prop: tell **Claude** "we use pnpm here", then
the next **Codex** session in the same project sees that fact
without you saying it again.

## The five layers (what every agent sees at spawn)

| Layer | What it is | Who edits it |
|---|---|---|
| **Tech stack & structure** | Auto-detected — Flutter / Go / Node / Postgres etc. via marker files. Plus current git branch + HEAD. Plus top-level directory layout. | Scanner only (read-only in UI). Refreshes every 6h or on stale-spawn. |
| **Project goal** | What we are building, one paragraph. | You — directly in the mobile/web Project screen. Agents can *propose* edits, which queue in the Inbox for your approval. |
| **Project plan** | What we are doing right now and what's next. | Same as goal. |
| **Recent activity** | LLM narrative of `git log --since="7 days ago"` plus hot-path file list. | Scanner only. 24h refresh ticker. |
| **Recent journal** | Last 5 session-end summaries. | Auto. Each session-end appends one. |

The composite banner is ~4-5 KB and gets prepended to the agent's
system prompt via each CLI's native injection channel.

## Day-to-day workflow

### Setting goal and plan

1. Open mobile or web → **Memory** → **Project**
2. Pick your project (or land directly via Session detail → 🏁
   Project memory icon)
3. **Goal** tab: write one paragraph. "Ship unified cross-agent
   memory" is fine; the agent reads this verbatim.
4. **Plan** tab: write current state + next steps. Update as work
   progresses; this is the most-edited doc.

### Reviewing agent-proposed changes

When an agent calls `project_goal_set` or `project_plan_set` MCP
tools, the change doesn't apply immediately — it goes to the
**Inbox** tab as a proposal with a red banner warning that
"Approve REPLACES the current goal/plan entirely". A side-by-side
diff shows current vs proposed. You can Approve (with a confirm
dialog) or Reject.

This matters because agents are eager and would otherwise silently
overwrite your hand-written goal with their interpretation.

### Cleanup inbox

When the LLM librarian runs (default: every 24h), it judges aged-
eligible memories as **keep** / **stale** / **duplicate** and
writes decisions to the queue. Open **Memory → Cleanup inbox** to
triage:

- `stale` (e.g., "currently debugging session.ended event delivery"
  from last week) → Approve to delete
- `duplicate` → Approve to merge into the indicated target
- `keep` → Approve to freeze re-judgement; or Reject if you
  disagree

The cleaner's `reason` field carries the LLM's justification.
If 80% of verdicts look reasonable, the system is calibrated; if
not, tweak the cleaner provider or prompt.

## Cross-CLI verification

Quickest smoke test:

1. Mobile/web: set the project goal to a distinct sentinel, e.g.
   `"TEST_2026_05_13: validating cross-CLI"`
2. Spawn a **Codex** session in that cwd
3. First prompt: "What is the current project goal? Quote it
   exactly."
4. Codex should quote `TEST_2026_05_13` verbatim → the L2 goal
   layer reaches Codex
5. Spawn a **Gemini** session in the same cwd, repeat → confirms
   L2 reaches Gemini

If either step fails, check the spawn injection channel:
- Codex: `cat $CODEX_HOME/AGENTS.md` should contain the banner
- Gemini: `cat <baseDir>/GEMINI.md` should contain the banner

## Project isolation guarantees

opendray promises that **project A's records will never appear in
project B's agent context**. The reader implementation in
`internal/session/transcript.go` enforces three checks:

1. **Fail-closed on missing UUID file** — if the caller asked for
   a specific session UUID and the file isn't there, return empty
   rather than fall back to "latest mtime in directory" (which is
   how unrelated sessions used to leak in).
2. **Time-window filter** — every parsed turn must have a
   timestamp within `[startedAt - 30s, endedAt + 30s]`. Even when
   the right file is opened, accumulated content from other
   spawns in the same cwd is filtered out.
3. **Cwd canary** — the first jsonl entry that carries a `cwd`
   field must exactly match the calling session's cwd. One
   mismatch and the whole file is abandoned.

When any defense triggers, the journaler degrades to metadata-only
(no LLM summary appended). "We don't know what happened" is the
correct failure mode — never a confidently-wrong summary.

## Quality gates (anti-noise mechanisms)

### Gatekeeper (write-time filter)

Every `memory_store` MCP call from an agent gets pre-judged by an
LLM. Outputs `user_preference` / `project_fact` / `feedback` /
`reference` / **`ephemeral`**. The `ephemeral` bucket is rejected:

✅ Stored: "User prefers bcrypt over argon2 for this stack"
✅ Stored: "Backup runs at 03:00 daily"
✅ Stored: "Linear board for pipeline bugs: linear.app/teams/INGEST"
❌ Rejected: "Currently debugging session.ended event delivery"
❌ Rejected: "Now editing line 412 of app.go waiting for the build"
❌ Rejected: "Thanks for the help, bye"

### Server-side dedup (M11)

After classification, the embedder produces a vector. If cosine
similarity to an existing entry in the same scope exceeds
threshold (0.85 dense / 0.2 BM25), the new entry is **merged**
into the existing row with `deduped_count++`. Two close paraphrases
become one record.

### LLM librarian (M13)

Periodic batch: scans aged-eligible memories, asks the LLM to judge
each as `keep` / `stale` / `duplicate`. Operator approval gates
all delete/merge actions through the Cleanup inbox.

## SQL recipes (for verification)

### Check spawn injection completeness

```sql
SELECT cwd, kind, length(content) AS bytes, updated_at::timestamp(0)
FROM project_docs
WHERE cwd = '/your/cwd'
ORDER BY kind;
```

You should see four rows: `goal` / `plan` / `tech_stack` /
`recent_activity`.

### Hallucination check on Recent activity

```sql
SELECT content FROM project_docs
WHERE cwd = '/your/cwd' AND kind = 'recent_activity';
```

Compare backtick-quoted file paths in the LLM summary against:

```bash
cd /your/cwd
git log --since="7 days ago" --name-only --format='' | sort -u
```

Every path the LLM mentions should be in the git log output. If
not, the LLM is hallucinating — file a bug.

### Gatekeeper categorisation distribution

```sql
SELECT COALESCE(metadata->>'type', '<no-type>') AS category, COUNT(*)
FROM memories
GROUP BY 1 ORDER BY 2 DESC;
```

`<no-type>` entries are from `~/.claude/.../memory/` mirror imports
(those bypass the gatekeeper). Manual and MCP stores should land
under one of `user_preference` / `project_fact` / `feedback` /
`reference`.

### Cleanup decision quality

```sql
SELECT verdict, status, substring(memory_text_snapshot, 1, 50) AS preview,
       substring(reason, 1, 80) AS llm_reason
FROM memory_cleanup_decisions
ORDER BY created_at DESC LIMIT 20;
```

Skim the LLM `reason` column. If ≥ 80% are sensible to you, the
librarian is calibrated. Adjust the cleaner provider if not.

### Find contamination (rare, post-M22)

```sql
SELECT id, cwd, content FROM session_logs
WHERE content LIKE '%Agent activity summary%'
ORDER BY created_at DESC LIMIT 5;
```

Read the summaries against the actual jsonl in
`~/.claude(-accounts/*)/projects/<encoded-cwd>/<session-uuid>.jsonl`.
Any file path the summary mentions should appear in that jsonl's
tool_use blocks. Pre-M22 data may have hallucinated summaries
from cross-session contamination; new entries should be clean.

## Configuration

Config is in `config.toml`. Memory-related sections:

```toml
[memory]
  enabled = true
  embedder = "bm25"                 # or "openai"
  dim = 1024                        # vector dimension (matches embedder)
  dedup_threshold = 0.85            # BM25 dev: try 0.2

  [memory.gatekeeper]
    summarizer_id = ""              # empty = use registry default
    max_latency_ms = 10000

  [memory.cleaner]
    enabled = true
    summarizer_id = ""
    interval_seconds = 86400        # 24h
    batch_size = 20
    min_age_hours = 0
    skip_if_decided_within_hours = 168
```

Each LLM touch-point (gatekeeper / cleaner / git activity /
transcript summariser) uses the same provider chain via
`summarizer_providers` table. Configure providers in
Settings → Server → Memory.

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| Session ends but no journal entry | Server didn't run M5 migration | `opendray migrate` |
| Journal entry has no "Agent activity summary" | Summariser provider not configured, or session was too sparse, or M22 filtered out the transcript | Check summariser config; review session length |
| Tech stack tab is empty | First spawn in this cwd hasn't triggered scanner | Spawn any session in the cwd |
| Tech stack tab is stale (HEAD lag, etc.) | 6h cache | Wait, or kill the cached row to force re-scan |
| Inbox proposals never appear | Agents are using `project_goal_set` correctly? | Check agent's system prompt explicitly mentions the MCP tools |
| Cleanup inbox empty | Librarian hasn't run yet; or all memories are too fresh | Lower `min_age_hours`, or wait for next 24h tick |
| Memory growing unbounded | Cleaner disabled; or operator not approving decisions | Enable cleaner, work through inbox |
| Cross-project leakage suspected | Pre-M22 contamination; or new M22 bypass | Run hallucination check SQL above; if confirmed, file issue |

## Pluggable LLM workers (M25, shipped)

Each of the four memory touch-points (gatekeeper / cleaner /
gitactivity / transcript) can independently be served by either:

- **SummarizerWorker** — the original OpenAI-compat HTTP path
  (LM Studio / OpenAI / ollama). ~1s latency.
- **AgentWorker** — a headless `claude --print` or `gemini --print`
  spawn keyed to one of your multi-account OAuth tokens. ~5-15s
  latency, frontier-model quality, draws from your Claude / Gemini
  subscription quota.

Operators flip per task on the **Memory → Workers** page (web +
mobile). Default rows are all `summarizer` — zero behavioural
change for existing installs. Per-call latency + outcome lands in
`memory_worker_calls`; a 24h rollup is surfaced on each card.

Gatekeeper stays summarizer-only by design — agent spawn (5-15s)
violates its <500ms latency budget. Codex is unsupported (no
`--print` mode).

See the **Memory → Workers** section in the admin UI for operator
workflow, and the implementation under `internal/memory/worker/`.

## Plan-drift auto-proposals (M-PA, shipped)

The plan document used to update only when an agent explicitly
called `project_plan_set` — which agents rarely did in practice,
so plans drifted out of date as projects iterated.

M-PA adds a fifth memory worker task — **`plan_drift`** — that
runs after every `session.ended` event. Given the session's
transcript summary, the current plan, and the last few journal
entries, it asks the configured worker LLM: "does this plan need
updating?" If yes, it files a proposal into the operator's inbox
with a one-line reason. Same approval flow as a manual
`project_plan_set` — operators always have final say.

Worker selection lives at **Memory → Workers → plan_drift**;
default is `summarizer`. Disable per task if you want the
historical behaviour back. The drift detector is a no-op when:

- the plan document is empty (refuses to seed an initial plan)
- the transcript summariser produced no narrative
- no worker is configured for `plan_drift`

Per-fire metrics land in `memory_worker_calls` alongside the
other four touch-points.

## Memory health dashboard (M-PA, shipped)

Each project's **Health** tab (`/memory/project` → first tab)
surfaces a single-page snapshot of "is the memory system
actually working for this project?":

- New facts / journal entries this week + cumulative totals
- Capture engine fire count, stored vs deduped, failure count
- Plan / goal last-updated relative times
- Pending proposal queue depth + oldest age
- Plan-drift proposals filed this week
- Top-hit fact (most retrieved) + count of zero-hit stale facts

Backed by `GET /api/v1/memory/health?cwd=<cwd>` — one aggregate
read that crosses both subsystems. No polling; refreshes on tab
view.

## Cross-layer search (M-PB, shipped)

Phase B closes the second gap from Phase A: the journal was
write-only — `session_log_append` shoved entries in but agents had
no way to ask "what did we decide about X last month". M-PB makes
journal entries first-class semantic-search citizens alongside
memory facts.

### What changed under the hood

- Migration 0031 adds `embedding`, `embedder`, `embedding_at`
  columns to `session_logs`.
- Every new journal entry is embedded synchronously at
  `AppendLog` time using the same embedder the memory subsystem
  already runs (BM25 / bge-m3 / OpenAI). Vector lives in the same
  space as `memories.embedding`, so cosines are directly
  comparable.
- A background goroutine catches up pre-feature journal rows in
  batches of 50; runs idle when caught up.

### The new search tool

`project_search` MCP tool + `GET /api/v1/project-search?cwd=&q=&top_k=`
take a natural-language query and return the top-K hits across
**five layers** in one ranked list:

- `fact` — memory_search results (layer 5)
- `journal` — semantic match against session_logs (layer 4)
- `goal` — lexical match against project_docs.goal (layer 2)
- `plan` — lexical match against project_docs.plan (layer 3)

Each hit carries `similarity`, `effective_score` (similarity with
time-decay applied: 1.0 today, 0.5 floor past 180 days), and the
`source` label so agents and UI can render layer badges.

Agent guidance is rolled into the MCP `instructionsBlurb` so models
use `project_search` for "where might this context live" queries
instead of guessing between `memory_search` and reading journal
pages by hand.

### Banner token budget

`RenderForSpawn` now has a sibling `RenderForSpawnWithBudget`
(`maxBytes int`). Operators dial it via
`SessionProvider.WithProjectDocBudget(N)` — 0 keeps today's
unconstrained behaviour. When set, sections are appended in
priority order (plan → tech_stack → goal → recent_activity →
journal) and rendering stops once the budget is exhausted, with
a trailing truncation note so the agent knows to visit
`/memory/project` for the full set.

## Smarter ranking (M-PC, shipped)

`memory_search` used to rank purely by cosine similarity. That
let one-shot 10-month-old memories crowd out fresh high-quality
matches, and never benefited from the fact that some memories
get retrieved 30× while others sit at zero hits forever.

The new formula:

```
effective_score = similarity
                × age_decay(age_days)        # max(0.5, 1 - age/180d)
                × hit_boost(hit_count)       # 1 + min(hit_count*0.02, 0.5)
                × confidence_floor(conf)     # max(conf, 0.3)
```

Threshold filtering still uses raw `similarity` so an explicit
`MinSimilarity` from the caller behaves like always; only the
**ordering** changes. The result: a popular 6-month-old fact at
0.8 similarity (score 0.60) outranks a brand-new mediocre 0.5
match (score 0.50), which is what operators were already doing
mentally.

Tuning knobs live in `internal/memory/ranking.go` — recompile to
change them.

## Cross-layer conflict detection (M-PC, shipped)

A new daily worker (`conflict_detector` TaskKind) scans each
project's plan + top-hit facts + recent journal and asks the
configured LLM: "do any of these claims contradict each other?"
Findings land in `memory_conflicts` (migration 0032) with the
two conflicting refs + the LLM's evidence + a severity tag.

Operators review the inbox at `/memory/project` → **Conflicts**
tab. Two buttons per row:

- **Accept** — operator agrees a contradiction exists and will
  apply the fix manually (delete a stale fact, update the plan,
  etc.). Status flips to `accepted` and the row stays in the
  audit history.
- **Dismiss** — the detector got it wrong; status flips to
  `dismissed` and an identical pair won't be re-flagged in
  future sweeps.

The scheduler tick is 24h with a per-cwd 10-minute LLM budget.
Operators can force a sweep with the "Detect now" button on the
tab or `POST /api/v1/memory/conflicts/detect?cwd=…`.

Worker selection lives at **Memory → Workers → conflict_detector**;
default is `summarizer`. Disable to skip detection entirely.

## Journal cleanup helper (M-PC, shipped)

`GET /api/v1/session-logs/stale?cwd=&days=` returns
`session_summary` journal entries older than `days` (default 90)
that aren't referenced by any pending conflict. Operators use
this list to prune accumulated noise without losing any
journaling that's still doing work via the conflict detector.

## Operator UX polish (M-PD, shipped)

The Phase A-C plumbing surfaced new signals (effective_score,
journal staleness, cross-layer conflicts); Phase D wires them
into the operator UI so they're actionable in two clicks.

### Ranking math visible in the inspector

Every row in **Memory** (`/memory`) now shows a `rank` badge next
to `sim`. Hover for a tooltip that spells out the formula:

```
effective 0.43 = sim 0.78 × age 0.72 (50d) × hits 1.08 × conf 1.00
```

Operators can see at a glance *why* a row sits where it does and
which factor dominates. Backed by `app/shared/src/lib/memoryRanking.ts`
which mirrors `internal/memory/ranking.go` so the explanation
matches what the backend ranker actually computed.

### Journal bulk-prune

The Journal tab (`/memory/project` → Journal) gains a collapsed
"Prune stale entries" panel. Expand it, optionally tune the age
threshold (default 90 days), select entries with the checkboxes,
and bulk-delete in one shot. Entries currently referenced by a
pending conflict are hidden from the list (server-side filter)
so operators don't accidentally lose context the detector still
considers load-bearing.

### Conflict quick-actions

Each conflict card on the **Conflicts** tab now grows a
"Fix:" row at the bottom with context-aware shortcuts:

- `fact` side → **Delete fact** (yanks the memory row + auto-
  accepts the conflict; one click clears the row from the
  inbox and the offending fact from search results).
- `plan` / `goal` side → **Open plan/goal editor** (jumps to
  the corresponding ProjectScreen tab so the operator can
  rewrite the doc; the conflict stays pending until they hit
  Accept after editing).

The "Open editor" jumps work because `ProjectScreen`'s Tabs
component is controlled (`activeTab` state) — the panel exposes
an `onJumpTab(tab: string)` callback that flips it.

## Roadmap

- **Codex session UUID capture**. Codex lacks `--session-id`; a
  future patch could parse `session_meta.id` from the first
  rollout line post-spawn to get the same isolation guarantees
  Claude/Gemini already have.
- **Self-healing cleaner**. Extend the cleaner to detect cross-cwd
  file references in journal summaries (hallucination signal)
  using opendray's own LLM provider chain.
- **Per-cwd worker overrides**. Today the worker config is global
  per task; a future revision may let operators pin a specific
  cwd to a heavier worker (e.g. "use Claude only for project X").
