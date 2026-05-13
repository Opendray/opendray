# Scanner & cleaner — auto-managed memory

Three pieces of the memory system run autonomously without your
input — they're worth understanding because they shape the spawn
banner every agent sees and the cleanup queue you triage.

## Project scanner (L1 — tech_stack)

Lives in `internal/projectscan/`. Cheap, deterministic, no LLM
involvement.

**Triggers**:

- Every spawn checks the row's `updated_at`; if older than **6h**,
  triggers a sync re-scan **before** the agent boots.
- `POST /api/v1/project-scan/run` if you want to force it.

**What it captures**:

- Tech stacks via marker files: `go.mod` → Go; `package.json` →
  Node.js; `pubspec.yaml` → Flutter; `pyproject.toml` → Python;
  `Cargo.toml` → Rust; `Gemfile` → Ruby; `pom.xml` → Java;
  `Dockerfile` → Docker; `migrations/*.sql` → PostgreSQL.
- Versions when discoverable (parses the marker files lightly).
- Current git branch + HEAD short-hash.
- Top-level directory layout (depth 1, skipping `.git`, `node_modules`,
  `target`, etc.).
- Entry points heuristics (cmd binaries for Go, `lib/main.dart`
  for Flutter, etc.).

Output lands as `project_docs.kind='tech_stack'` and gets
rendered into every spawn banner verbatim under `### Tech stack
& structure`.

**Why a 6h cache**: a re-scan is cheap (~50ms) but spawn latency
matters. 6h is enough to absorb dependency adds and commit
movement within a working day. If you need newer data right
after, just spawn (next session triggers the refresh).

## Git activity summariser (L4 — recent_activity)

Lives in `internal/gitactivity/`. **LLM-driven**, more expensive,
runs less often.

**Triggers**:

- A 24h scheduler ticker in the background.
- Every spawn checks staleness; if older than **12h**, kicks an
  **async** refresh (the current spawn sees the prior summary;
  the next one sees the fresh one).

**What it does**:

1. Shells `git log --since="7 days ago" --stat --no-merges` with
   strict field separators so the output parses unambiguously.
2. Builds a deterministic preamble (commit count, file count,
   net lines, hot-path files by commit count).
3. Sends the structured data to the configured summarizer
   provider (LM Studio, ChatGPT-OAuth, or any OpenAI-compatible
   endpoint).
4. Asks for 1-3 paragraphs of narrative + a "for the incoming
   session: avoid X / focus on Y" hint section.
5. Persists as `project_docs.kind='recent_activity'`.

Result appears in the Project → **Activity** tab and in spawn
banners under `### Recent activity`.

If no summarizer provider is configured, the summariser falls
back to raw stats (window + commit count + hot paths) — still
useful, just less narrative.

## Cleanup librarian (L5 maintenance)

Lives in `internal/memory/cleaner/`. Maintains the section 11
discrete-fact store, **not** the project_docs above.

**Why**: agents that have `memory_store` access tend to write
ephemeral noise alongside durable facts. Even with the **M12
gatekeeper** rejecting "currently debugging X"-class entries at
write-time, you'll still accumulate facts that age out — old
project plans, stale infra details, duplicates the dedup
threshold barely missed.

**Triggers**:

- 24h scheduler ticker (configurable via
  `[memory.cleaner].interval_seconds`).
- Manual: Project → **Cleanup** tab → "Run cleanup now" button.

**What it does**:

1. Picks up to `batch_size` (default 20) aged-eligible memories
   for one scope at a time.
2. Asks the LLM to judge each: `keep` / `stale` / `duplicate`
   (with `merge_into` target).
3. Writes one row per judgment to `memory_cleanup_decisions`
   with status `pending`.
4. **Does NOT execute anything.** All deletions / merges wait
   for your approval.

**Triage UI**:

- Project → **Cleanup** tab — decisions for this cwd only.
- More → Memory → **Cleanup inbox** — cross-project, grouped by
  scope_key, useful when you've been off for a while and many
  decisions queued up.

Each decision row shows:

- Verdict badge (`stale` red, `duplicate` muted, `keep` outline)
- The memory text snapshot (frozen at decision time)
- The LLM's reason (read this!)
- Optional `merge_into` target (for duplicates)
- Approve / Reject buttons

**Approve**: executes the action. `stale` → memory deleted.
`duplicate` → merged into target (target's `deduped_count++`).
`keep` → memory frozen from re-judgment for `skip_if_decided_within_hours`
(default 168 = 1 week).

**Reject**: decision marked rejected; memory stays as-is. LLM
will re-judge later.

**Calibration check**: skim 10-20 decisions and ask "do the
reasons match how I'd judge these?". If ≥ 80% feel right, the
librarian is calibrated. If not, the cleaner prompt or the
summarizer model needs adjustment.

## What's NOT auto-managed

These need your hand:

- **Goal**, **Plan**: you write them. Agents propose; you
  approve in Inbox.
- **Cleanup approvals**: every delete/merge/freeze is
  operator-gated.
- **Reset**: wiping a project's memory needs the explicit
  destructive button (see section 04).

The auto / manual split is intentional — the auto pieces are
either deterministic (scanner) or operator-reviewable (cleaner
proposes, you approve). You never hit "the system silently
threw away the plan I wrote yesterday".
