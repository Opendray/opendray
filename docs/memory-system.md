# opendray memory system — operator guide

This guide walks through what opendray's unified memory system
does, how to use it day-to-day, and how to verify it's behaving.
For the architectural rationale see
[ADR 0018](adr/0018-unified-cross-cli-memory.md) and
[ADR 0014](adr/0014-memory-subsystem.md).

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

## Roadmap

- **M25 — pluggable Memory worker**. Today all four LLM touch
  points hit a single HTTP endpoint. M25 will let you route some
  touch-points through a headless Claude / Codex / Gemini agent
  session for higher quality (at higher cost). Per-task-kind
  switching in mobile Settings → Memory → Workers.
- **Codex session UUID capture**. Codex lacks `--session-id`;
  M25 may parse `session_meta.id` from the first rollout line
  post-spawn to get the same isolation guarantees Claude/Gemini
  already have.
- **Self-healing cleaner**. Extend the cleaner to detect cross-cwd
  file references in journal summaries (hallucination signal)
  using opendray's own LLM provider chain.
