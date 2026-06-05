# Memory System Unification Redesign (Milestone arc: M-U)

> **Status:** Approved blueprint — §8 decisions owner-confirmed; not yet
> implemented. Delivery proceeds locally per phase and is pushed to
> GitHub only after local tests pass.
> **Owner:** memory subsystem is a single-author design; this redesign
> supersedes the incremental M5 → M-PD architecture described in
> [`memory-system.md`](./memory-system.md). That document remains the
> operator guide for the *current* (pre-M-U) behaviour; sections it
> describes are flagged below as **superseded** where M-U changes them.
> **Scope of change:** `internal/memory/`, `internal/memquery/`,
> `internal/memconflict/`, `internal/projectdoc/`, `internal/catalog/adapter.go`,
> `internal/app/app.go`, plus new migrations `0033+`.

---

## 1. Why this redesign exists

The memory system grew through ~15 milestones (M5, M11, M13, M22, M25,
M-PA…M-PD). Each shipped a correct feature, but the accretion left
**structural redundancy** that now costs us performance, correctness,
and operator time:

- **Two parallel memory stores.** A DB-backed pgvector store (shared,
  queryable, cross-CLI) *and* the Claude Code harness's file memory
  (`~/.claude-accounts/<acct>/projects/<cwd>/memory/*.md`). The file
  store is Claude-only — Codex and Gemini cannot see it — so it
  actively undermines the system's reason to exist (cross-agent
  continuity). Today it is bridged one-way by a mirror
  (`internal/memory/mirror.go`), which is a sync band-aid, not unity.
- **Three scopes where two suffice.** `session | project | global`
  (`0011_memory.sql` CHECK). A session is always built on a project;
  session-scoped memory is a redundant partition that fragments recall
  and complicates every read/write path.
- **Two ranking formulas.** `internal/memory/ranking.go` (full
  multi-factor) vs `internal/memquery/memquery.go` `decayScore`
  (age-only). The same fact ranks differently in `memory_search` vs
  `project_search`.
- **Mixed retrieval quality.** goal/plan are matched **lexically**
  (`LIKE`) in cross-layer search, never semantically. The three
  cross-layer sub-searches run **sequentially**.
- **Manual-everything cleanup.** The cleaner is 100% operator approval
  with **hard, irreversible deletes** and no project-lifecycle signal.
  Real-world result: hundreds of pending decisions, including obviously
  stale ones (e.g. the tech-stack fact of a finished project) that the
  operator must still hand-approve.
- **A one-line write bug that disables the whole premise.** Agents are
  *intended* to write memory, but the auto-provisioned memory
  integration key is granted only `session:read`
  (`internal/app/app.go:1129`), so every `memory_store` from an agent
  returns `403`. Cross-agent contribution — the entire point — has
  never actually run from the agents themselves.

This redesign is **subtractive first**: it removes redundant systems,
scopes, and code paths, then adds a small number of better practices on
the cleaner foundation.

---

## 2. Design constraints (fixed by the system owner)

1. The memory system serves **all** cloud-agent CLIs (Claude, Codex,
   Gemini, future ones) equally. Its job: accumulate each agent's
   contributions in one project brain so that **switching agents
   preserves project familiarity**.
2. **Scopes collapse to two:** `project` (default; key = cwd) and
   `global` (written **only on explicit operator request**). The
   `session` scope is removed — session ≡ project.
3. Writes are **free** (no per-write human approval). The **only**
   human intervention point is **conflict detection**.
4. No patching. Each phase is a self-contained, coherent unit that
   leaves the system simpler than it found it.
5. **Lossless, automatic upgrade for existing users.** Operators already
   running opendray must reach the new system by a normal `opendray
   update` with **no data loss and no manual SQL**. Every migration is
   forward-only, idempotent, and **never hard-deletes existing user
   data** — rows are folded, archived, or re-embedded, never dropped.
   See §7.
6. **Ship as one change, merge after validation.** Because this is a
   large, cross-cutting redesign, the whole arc (Phase 0-6) is built on
   a single branch and lands as **one PR**, opened only when every phase
   is complete and the migration has been test-validated end-to-end on a
   copy of real data. No phase merges to `main` on its own.

### Non-goals

- Re-implementing the embedder zoo. We standardise on one dense
  embedder + a fallback; we do not add new embedder providers.
- Changing the channel/notify subsystems.
- A new UI framework. UI changes are limited to what the new data model
  requires (cleanup inbox shrinks; conflicts remain).

---

## 3. Target architecture — "one brain"

```
   ┌────────────── all cloud agents (Claude / Codex / Gemini / …) ──────────────┐
   │                                                                            │
   │   read at spawn  ◀───────────────┐               ┌──────▶  write any time  │
   └──────────────────────────────────┼───────────────┼────────────────────────┘
                                       │               │
                       (4) spawn       │               │   (2) write pipeline
                       injection       │               │   embed → semantic
                       ranked fill ────┤   ┌───────┐    ├──  dedup → consolidate
                                       └───│  DB   │────┘    or insert  (project
   (3) on-demand search                    │ pgvec │         scope default;
   one parallel pass, one ranker  ◀────────│  tor  │         global only on ask)
                                           └───┬───┘
                                               │  (5) background maintenance
                          ┌────────────────────┼─────────────────────┐
                          │  consolidation/dedup sweep  (auto)        │
                          │  staleness auto-archive     (auto, soft)  │
                          │  conflict detection         (auto) ───────┼──▶ operator
                          └───────────────────────────────────────────┘    inbox
                                                                          (conflicts ONLY)
```

Five pillars:

### Pillar 1 — One store, two scopes

- Single Postgres pgvector store. **No file memory** as a store (see
  Pillar 6 for the retirement path).
- `memories.scope ∈ {project, global}`. `project` keyed by cwd;
  `global` keyed by `''`. **`session` removed.**
- All agents read and write the same project memory. Switching agents
  is instant continuity by construction, not by sync.

### Pillar 2 — One write pipeline (free, self-consolidating)

`memory_store(text, scope=project)` — agents default to `project`;
`global` requires an explicit operator-originated instruction.

On write (the hot path — **no LLM call**, must stay millisecond-fast):

1. Embed with the active embedder.
2. Semantic dedup **within the same (scope, scope_key)**: nearest
   neighbour search, top-1.
3. If similarity ≥ `consolidate_threshold`: **fold** — keep the
   higher-confidence / newer row as canonical, bump `frequency`/
   `deduped_count`, record the absorbed id in `merged_from`, append
   provenance. Return the canonical id. **Rule-based only; no LLM in the
   write path** (a `claude --print` call here would put 5-15s of latency
   and subscription-quota cost on every near-duplicate write).
4. Else insert a new row.

**The LLM text merge does NOT run at write time.** Refining the folded
text into one better canonical statement is deferred to the background
consolidation sweep (Pillar 5), where latency and cost do not matter.
That sweep uses the pluggable-worker chain (M25): a local OpenAI-compatible
worker if configured, else the **cloud-agent worker** (`claude --print` /
`gemini --print`), which every opendray install has. Rationale: merge is
an easy LLM task (lighter than the gatekeeper/cleaner/conflict jobs these
workers already do), so capability is not the concern — keeping it off
the synchronous write path is.

No human approval on writes. The write-time gatekeeper (LLM durability
filter) stays **optional** and **off by default**; quality is maintained
by folding + ranking + the maintenance loop, not by a write gate. This
replaces "store then dedup then clean up the backlog" with "don't create
the duplicate in the first place".

### Pillar 3 — One retrieval path, one ranker

- A single scoring function, defined once, used by **both** spawn
  injection and on-demand search:

  ```
  effective_score = similarity
                  × recency(age_days)        # max(floor, 1 - age/HALF_LIFE)
                  × frequency(hit_count)     # 1 + min(hit_count*k, cap)
                  × importance(confidence)   # max(conf, conf_floor)
  ```

  `internal/memquery`'s separate `decayScore` is **deleted** and routed
  through this function.
- goal/plan become **first-class vectors** (embedded), not `LIKE`
  matches. journal is already embedded (M-PB); embedding stays
  **synchronous on append** so there is no spawn-time blind spot.
- `project_search` runs its facts/journal/docs sub-searches **in
  parallel**, then merges and ranks once.

### Pillar 4 — Spawn injection (query-less, budgeted)

At spawn there is no query yet, so injection ranks the project brain by
`importance × recency` (no similarity term), fills a **token budget**,
and assembles: goal + plan + top-N facts + recent journal. Delivery
stays per-provider (`--append-system-prompt` for Claude, `AGENTS.md`
for Codex, `GEMINI.md` for Gemini) — already provider-agnostic at the
source (`internal/catalog/adapter.go:363` `Prepare`). Once the agent has
a task it uses `project_search`/`memory_search` with a real query.

### Pillar 5 — One maintenance loop (auto; human only for conflicts)

All destructive maintenance runs on **soft-delete + grace period**
(reversible), which is what makes automation safe:

- **Consolidation/dedup sweep** — retroactively merges semantically
  equivalent facts (both the ones the write-time fold caught and any it
  missed). This is where the **LLM text merge happens** — refining
  folded near-duplicates into one canonical statement via the
  pluggable-worker chain. Off the hot path, so latency/cost are free.
- **Staleness auto-archive** — facts never retrieved + aged, OR facts
  belonging to **inactive/finished projects** (lifecycle signal: git
  inactivity, or operator "project done"), are auto-archived (soft;
  hard-purged after the grace window).
- **Conflict detection** (already wired, `internal/memconflict/`, 24h)
  is the **only** producer of operator-inbox items.

The manual cleanup-approval queue is **removed**. Routine staleness
never reaches the operator.

### Pillar 6 — One embedder strategy (availability-tiered)

**Design principle — availability tiering.** Not every operator runs a
local AI, but **every** opendray install necessarily has a cloud-agent
CLI. So each LLM/vector touch-point must name a backend that is
*guaranteed present*, and prefer better backends only when they exist.
The guaranteed-present backend differs by task type:

| Task | Best (if present) | Guaranteed-present floor |
|---|---|---|
| **Embedding** (vectors) | local dense (`bge-m3` 1024, `nomic-embed-text`, `mxbai-embed-large`, …) served by **any** local runtime, or a cloud OpenAI-compatible embedding endpoint | **BM25** — pure-Go, zero-dependency, already built |
| **LLM text** (merge, gatekeeper, cleaner, conflict) | local LLM via **any** runtime | **cloud-agent worker** (`claude --print` / `gemini --print`) |

Critical distinction: a **cloud-agent CLI cannot produce embeddings** —
Claude/Gemini `--print` are text generation, not vector models, and
Anthropic exposes no embeddings API. So the embedding floor is **BM25**,
not the cloud agent. The cloud agent is the floor for *LLM text* tasks
(Pillar 2 / Pillar 5 merge, gatekeeper, cleaner, conflict) only.

**Local-runtime neutrality (required).** "Local LLM" must not mean
"ollama only". Both the embedder and the LLM workers talk to a local
backend over the **OpenAI-compatible HTTP contract** (`/v1/embeddings`,
`/v1/chat/completions`), so any runtime that speaks it works:
**ollama, LM Studio, vLLM, LocalAI, llama.cpp server, text-generation-webui**,
etc. Configuration is a base URL + model name + (optional) API key — no
runtime-specific code paths. The existing `internal/memory/embedder_http.go`
and the summarizer's `provider_openai_compat.go` already follow this
contract; the redesign keeps that contract as the single local-integration
surface and does not special-case any one vendor. (A dedicated
`provider_ollama.go` may remain as a convenience preset, but it is one
preset over the generic contract, not the only path.)

- **Embedder selection is tiered, not fixed:** use a configured dense
  embedder (any local OpenAI-compatible runtime, or a cloud endpoint)
  when available; otherwise fall back to BM25 automatically. No hard
  requirement on local AI, and no lock-in to a single local runtime.
- On embedder change, a **background re-embed** runs automatically; the
  search query no longer silently drops rows whose `embedder` differs
  (today `WHERE embedder=$2` in `store_pgvector.go` hides them).

---

## 4. What shrinks vs what is new

| Removed / simplified (less redundancy)            | Added (more function / performance)                   |
|---|---|
| File-memory store + harness "write a memory file" loop | Write-time semantic consolidation (upsert-by-meaning) |
| `session` scope + all its read/write plumbing     | Soft-delete + grace window + auto-archive worker      |
| `memquery.decayScore` (second ranking formula)    | Project-lifecycle signal (git/done) driving archive   |
| Lexical `LIKE` doc search                         | One unified scorer (two → one)                        |
| Manual cleanup-approval queue                     | Retroactive consolidation sweep                       |
| Silent multi-embedder row dropping                | Automatic re-embed on embedder change                 |

Performance: fewer rows (consolidate + archive) → faster vector scans;
synchronous journal/doc embedding → no spawn blind spot; parallel
cross-layer search → lower latency; one canonical embedder + maintained
HNSW → stable recall.

---

## 5. Data model changes

New migrations (`0033+`). Exact column names to be finalised in each PR.

1. **Scope collapse** (`0034_memory_drop_session_scope.sql` — 0033 was
   already taken by `0033_memory_workers_capture.sql`)
   - Migrate existing `scope='session'` rows → `scope='project'`,
     setting `scope_key` to the session's cwd
     (`UPDATE … SET scope='project', scope_key=(SELECT cwd FROM sessions
     WHERE id = memories.scope_key) WHERE scope='session'`).
   - Adds the `archived_at` / `archived_reason` columns up front (the
     soft-delete primitive, see item 2) so the fold has a lossless place
     to put orphans.
   - Orphans (session row gone): recover cwd from `session_logs` if
     possible; otherwise set `archived_at = now()` (lossless — restorable,
     not dropped). **No `DELETE` in this migration.**
   - Replace the CHECK constraint with `scope IN ('project','global')`
     only after the fold, so no row violates it mid-migration.

2. **Soft delete** (`0034_memory_soft_delete.sql`) — *column may be
   co-introduced in 0033 (item 1); 0034 then only adds the behaviour.*
   - `archived_at TIMESTAMPTZ NULL`, `archived_reason TEXT NULL`.
   - All search/injection queries gain `AND archived_at IS NULL`.
   - A purge job hard-deletes rows where
     `archived_at < now() - grace_interval` (grace = 30 days, §8.2).

3. **Consolidation accounting** — *no migration (done in Phase 3).* The
   fold count (`deduped_count`) and the lossless audit (`merged_from`)
   live in the existing `metadata` JSONB; a dedicated `frequency` column
   was dropped from the plan as redundant with `deduped_count`, which
   already backs the UI.

   > Migration numbering note: the filenames in this section were
   > indicative. Actuals so far: `0034` = scope collapse (Phase 1),
   > `0035` = project_docs embedding (Phase 2). Phase 4's soft-delete +
   > lifecycle land at `0036`+ when written.

4. **Lifecycle signal** (`00NN_project_lifecycle.sql`, Phase 4)
   - `project_docs` (or a small `project_state` table): add
     `last_git_activity_at TIMESTAMPTZ`, `status TEXT CHECK (active |
     archived) DEFAULT 'active'`. Drives staleness auto-archive.

`memory_cleanup_decisions` (0026) is **deprecated** by Pillar 5 — kept
read-only for audit history, no new rows after Phase 4.

---

## 6. Phased delivery

Phases are **commits on one arc branch**, not separate PRs (constraint
§2.6). Each is a self-contained, tested unit so the branch stays green
commit-by-commit, but the whole arc lands as a **single PR** opened only
after every phase is complete and the migration is validated end-to-end
on a copy of real data. Order is dependency-driven.

### Phase 0 — Turn the write path on  *(unblocks everything)* — DONE

- Granted the `opendray-memory` integration key `memory:read` +
  `memory:write`; reconcile scopes on existing installs so the fix
  reaches users who registered the key before these scopes existed
  (`scopesCover` in `internal/app/app.go`). `global` writes stay
  admin-only, enforced at the store handler (`globalWriteAllowed`).
- **Acceptance:** an agent `memory_store` succeeds (no 403); the row
  lands in `memories` with `scope='project'`, `scope_key=cwd`. Verified
  the diagnosis live (read path 403s under pre-fix code).
- **Risk:** low. Behaviour change = agents can now write (intended).
  Unit tests for `scopesCover` + `globalWriteAllowed`; packages pass
  `-race`.

### Phase 1 — Scope unification — DONE

- Migration `0034`. `session` removed from the scope model; a
  `normalizeScope` boundary coerces the legacy literal to project
  (lossless) so old config / API callers / pre-migration rows never
  error. Capture rules coerce `target_scope='session'` to project.
  Web + mobile UI and i18n drop the session option; slang strings
  regenerated.
- **Lossless fold:** every `scope='session'` row becomes `scope='project'`
  keyed by the session's cwd (joined via `sessions`). Orphans (session
  gone) are **soft-archived** (`archived_at`, restorable) rather than
  deleted. **No `DELETE`.**
- **Validated:** the fold ran on an ephemeral Postgres — 4 rows in, 4 out
  (zero loss), 0 session rows remain, live-session row folded to its cwd,
  orphan archived, new CHECK rejects session. Go `normalizeScope` /
  `Scope.Validate` table tests; web `tsc -b && vite build` clean; mobile
  `dart analyze lib` clean; i18n parity 100%.
- **Risk:** medium (data migration), retired by the no-hard-delete rule +
  the ephemeral-Postgres validation above; full-data replay still happens
  at arc-final validation.

### Phase 2 — Retrieval unification — DONE

- One scorer: `memory.RankingScoreFields` (M-PC formula over raw fields)
  now serves both `memory.Service` and `memquery`; `decayScore` deleted.
  A fact ranks identically in `memory_search` and `project_search`.
- goal/plan are first-class vectors: migration `0035` adds embedding
  columns to `project_docs` (copy of 0031); `PutDoc`/`ApproveProposal`
  embed synchronously and null the vector on content change;
  `RunDocEmbedBackfill` catches up history; `searchDocs` scores by cosine
  with a lexical 0.6 fallback only for not-yet-embedded docs.
- `project_search` fans its three sub-searches out over a WaitGroup
  (wall-clock = slowest layer, not the sum).
- **Backend-only.** The web/mobile ranking mirrors already track
  `ranking.go` (unchanged) and the Hit DTO additions are additive, so no
  client change was required — verified.
- **Validated:** `-race` across memory/memquery/projectdoc/app; the 0035
  index predicates exercised on an ephemeral Postgres (full vector replay
  is the arc-final step on the real pgvector DB).
- **Risk:** low/medium, retired by the ranking table tests + race pass.

### Phase 3 — Write-time fold (cheap, no LLM) — DONE

- **No migration.** The fold accounting (`deduped_count` + the new
  `merged_from` audit) lives in the existing `metadata` JSONB rather than
  a redundant `frequency` column — `deduped_count` already backs the
  mobile "merged ×N" badge.
- Semantic dedup is now the **default** write path: an unset
  `dedup_threshold` resolves to an embedder-relative default (~0.85
  dense, ~0.2 BM25); a **negative** value is the explicit off switch. On
  a hit the write **folds** rule-based (no LLM): newer text becomes
  canonical, the superseded text is preserved in `merged_from` (capped
  20) so the fold is **lossless**, `deduped_count` bumps. LLM refinement
  of the canonical is deferred to Phase 4's background sweep.
- **Acceptance (met):** two paraphrases yield one row with
  `deduped_count=1` and `merged_from` carrying the superseded text;
  negative threshold disables; no LLM on the write path.
- **Deferred to Phase 4:** the *cleaner's* duplicate-merge metadata bug
  (it deletes a dup without recording `merged_from` on the survivor) is
  fixed when Phase 4 overhauls the cleaner.
- **Risk:** low. Default-on is behaviour-compatible (lossless) and the
  threshold is tunable; covered by boundary tests.

### Phase 4 — Self-maintaining cleanup

- Migrations `0034`, `0036`. Soft-delete + grace purge job. Auto-archive
  worker (never-hit+aged, and lifecycle-driven). Retroactive
  consolidation sweep — this is where the **LLM text merge** runs
  (pluggable worker: any local OpenAI-compatible runtime → cloud-agent
  worker), off the hot path. **Remove** the manual cleanup-approval
  queue; operator inbox = conflicts only.
- **Acceptance:** a finished project's stale facts (e.g. tech_stack)
  auto-archive without operator action and are restorable within the
  grace window; the cleanup-approval inbox is empty by design; conflict
  inbox still works.
- **Risk:** medium. Soft-delete + grace makes it reversible; ship behind
  a config flag with conservative defaults (long grace, conservative
  archive predicate) and widen after observation.

### Phase 5 — Retire the file layer

- One-time mirror import of existing file memories → DB. Suppress the
  now-redundant file-memory injection for Claude (DB injection already
  covers it). Update agent instructions to use `memory_store` instead of
  writing memory files.
- **Acceptance:** a Claude session with no local memory files still
  receives full project memory at spawn (from DB); agent-authored
  memories land in DB and are visible to a subsequent Codex/Gemini
  session in the same cwd.
- **Risk:** medium (operator-visible behaviour change). Keep the
  one-way mirror as a legacy capture net during transition.

### Phase 6 — Embedder unification

- Implement the availability-tiered embedder (dense when configured,
  BM25 floor otherwise — decision §8.1). Add the automatic background
  re-embed on embedder change; drop the silent `embedder`-mismatch
  exclusion from the search path once the active embedder is settled.
- **Acceptance:** changing the configured embedder triggers a background
  re-embed; no row becomes unsearchable mid-migration.
- **Risk:** low/medium. Re-embed is I/O heavy; rate-limit + resumable.

---

## 7. Migration & backward compatibility

Constraint §2.5 is a hard requirement: an existing operator reaches the
new system by a normal `opendray update`, with **no data loss and no
manual SQL**. Concretely:

**Lossless.** No M-U migration runs a `DELETE` against user data. The
session→project collapse *folds* rows (orphans are archived, not
dropped); cleanup uses *soft-delete + 30-day grace* (restorable); the
file layer is *imported before* it is retired; an embedder change
*re-embeds*, never discards. Every destructive-looking step is a
reversible state change, not a deletion. Each data migration ships with
a counted dry-run and a pre-migration `opendray` memory export (the
backup/restore path already exists, `backup/service_import.go`) so even
operator error is recoverable.

**Automatic on upgrade — the gap to close.** Migrations today are
**not** auto-applied on startup; they run only via the explicit
`opendray migrate` (`internal/store/migrate.go` is invoked solely from
`cmd/opendray/migrate.go`). So an operator who runs `opendray update`
gets the new binary against the **old schema** until they separately run
`opendray migrate` — unacceptable for a "smooth upgrade". The arc must
therefore make M-U migrations apply automatically on the upgrade path:
either the service runs pending migrations on start (fail-closed if they
error), or `opendray update` chains `opendray migrate`. Migrations are
already idempotent, forward-only, and transactional (tracked in
`schema_migrations`), so auto-apply is safe. **This is a required
deliverable of the arc, not optional.**

**Compatibility staging.** Phases 0–3 are behaviour-compatible for the
operator (writes start working; rankings unify; dedup tightens). Phases
4–5 are the operator-visible changes (cleanup inbox disappears; file
memory retires) and ship behind flags with conservative defaults so the
first upgraded run behaves like the old one until the operator opts in.

`memory-system.md` is updated in the same PR to keep the operator guide
truthful; the **superseded** sections are: five-layer/three-scope model
(§"The five layers", §"Project isolation"), Gatekeeper as the primary
quality gate (§"Quality gates"), and the manual Cleanup inbox
(§"Cleanup inbox").

---

## 8. Resolved decisions (owner-confirmed)

All decided. Two governing principles: **availability tiering** (Pillar
6) — prefer the best backend present, fall back to a guaranteed-present
floor, and the floor differs by task type because a cloud-agent CLI can
do LLM text but cannot produce embeddings — and **keep LLM work off the
write hot path** (decisions 5–6).

1. **Embedder** — *tiered and runtime-neutral.* Prefer a dense embedder
   when available — a local model (`bge-m3` 1024, `nomic-embed-text`,
   `mxbai-embed-large`, …) served by **any** OpenAI-compatible local
   runtime (**ollama, LM Studio, vLLM, LocalAI, llama.cpp**, …), or a
   cloud embedding endpoint. Fall back to **BM25** automatically when
   none is configured. BM25 is the guaranteed-present floor (pure-Go,
   zero-dep). The cloud-agent CLI is **not** an embedding backend (no
   vector API). `consolidate_threshold` is embedder-relative (≈0.85 dense
   / ≈0.2 BM25, matching today's dedup thresholds).
2. **Grace window** — **30 days** soft-delete → hard-purge.
3. **Write-time gatekeeper** — **kept as an optional knob, off by
   default.** Quality is carried by consolidation + ranking +
   maintenance, not a write gate.
4. **Lifecycle "project done" trigger** — **both**: git-inactivity
   threshold of **60 days**, and an explicit operator "archive project"
   action.
5. **Consolidation merge** — *split across two paths.* Write-time = a
   cheap **rule-based fold** (keep canonical + bump frequency + record
   `merged_from`), **no LLM**, to keep writes millisecond-fast.
   Background sweep = the **LLM text merge** via the pluggable-worker
   chain (M25): any local OpenAI-compatible runtime if configured,
   otherwise the **cloud-agent worker** (`claude --print` /
   `gemini --print`), which every install has. The cloud agent is a
   valid backend here because merge is an LLM *text* task (unlike
   embedding in decision 1).
6. **LLM merge placement** — *background only, never the write hot
   path.* A `--print` call costs 5-15s + subscription quota; putting it
   on every near-duplicate write is unacceptable. Write-time folds
   cheaply; the background consolidation sweep refines text when latency
   and cost are free.

---

## 9. Code reference map (entry points touched)

| Concern | Location |
|---|---|
| Write-scope bug (Phase 0) | `internal/app/app.go:1127–1160` (`ensureMemoryIntegration`) |
| Scope definitions / CHECK | `internal/store/migrations/0011_memory.sql` |
| Memory service (store/search/dedup/rank apply) | `internal/memory/service.go` |
| pgvector store (insert/search/delete/RecordHits) | `internal/memory/store_pgvector.go` |
| Unified ranker target | `internal/memory/ranking.go` |
| Cross-layer search (second formula to delete) | `internal/memquery/memquery.go` |
| Spawn injection (provider-agnostic) | `internal/catalog/adapter.go:363` (`Prepare`) |
| Project docs (goal/plan/journal) | `internal/projectdoc/` |
| Cleaner (to be replaced by auto + soft-delete) | `internal/memory/cleaner/` |
| Conflict detection (kept; the only human gate) | `internal/memconflict/` |
| File-memory mirror (one-way; retire in Phase 5) | `internal/memory/mirror.go` |
| MCP memory tool surface | `cmd/opendray/mcp_memory.go` |

---

## 10. Success criteria for the arc

- One store, one scope default, one ranker, one embedder.
- An agent of any kind can write project memory; a different agent in
  the same project reads it at spawn — verified by the cross-CLI smoke
  test in `memory-system.md` extended to a write→switch→read loop.
- The operator's recurring inbox work is **conflicts only**; routine
  staleness and duplicates are handled automatically and reversibly.
- An existing operator upgrades via a single `opendray update` and lands
  on the new system with **zero data loss and zero manual SQL** —
  validated by replaying the full migration on a copy of real
  pre-upgrade data and reconciling row counts to no loss.
- Net negative line count in `internal/memory*` after the arc (the
  redesign removes more than it adds).
