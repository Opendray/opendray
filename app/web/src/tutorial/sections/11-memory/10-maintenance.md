# Memory — maintenance

Once memory is wired and agents are storing facts, three operator
tools handle the day-to-day upkeep. All of them live on the
dedicated **Memory** page (left sidebar → 🧠 Memory, shortcut `g m`).

## The Memory page at a glance

A separate top-level route, deliberately split from `Settings → Memory`
so runtime browsing doesn't share screen space with a configuration
draft.

The page contains:

- **Active embedder strip** — the embedder the gateway is *currently*
  using for every `memory_search` / `memory_store`. If this disagrees
  with what's typed into Settings, you have an unsaved config — Save
  + Restart server to apply.
- **Migration banner** — only visible when memories exist under a
  different embedder than the active one. Detail in [Reembed](#reembed).
- **Scope + scope_key** with a "Pick" dropdown that surfaces both
  *previously saved* scope_keys and *active session* cwds, so you
  don't have to type long paths. **Sync .md** button (project scope
  only) re-ingests Claude's local memory files for the cwd.
- **Search bar** — semantic search; press Enter to run.
- **Rows** — each memory's id, similarity (when from a search), hit
  count, age, and edit (✏) / delete (🗑) buttons.

## Hit count

Every successful search bumps `hit_count` and `last_hit_at` for the
memories it returns (post-threshold filter). The bump is fire-and-
forget on a background goroutine, so it never slows the search
response.

What it tells you:

- **Zero hits + old `created_at`** — agents stored this fact but
  nobody ever looked it up. Common for noisy memories Claude wrote
  during exploration. Safe to delete.
- **High hits + recent `last_hit_at`** — load-bearing fact. Keep
  even if you'd otherwise prune for length.
- **High hits but stale `last_hit_at`** — the topic stopped being
  relevant. The memory is probably still correct but no longer
  driving any agent behaviour.

The inspector shows hit count next to each row. Hover the badge for
the absolute "last hit N hours ago" timestamp.

## Reembed

opendray's pgvector index is partitioned by `(embedder, dim)`. When
you switch embedder backends mid-flight (most commonly **bm25 →
http** after wiring up an ollama instance, or **http → local** to
go offline), older memories silently stop matching: they exist in
the database but have vectors the new embedder can't compare against.

The Memory page detects this and surfaces a yellow banner:

> *"NN memories won't appear in searches: NN on bm25 — current
> embedder is http:nomic-embed-text. pgvector partitions its
> similarity index by embedder, so older entries are silent until
> reembedded."*

Click **Migrate** → confirm in the dialog → opendray walks every
mismatched row, recomputes its vector with the current embedder,
and writes it back **in place** (id, scope, scope_key, metadata,
timestamps preserved). Search picks up the migrated memories
immediately — no restart needed.

Notes:

- Reembedding is synchronous over HTTP. For an HTTP backend it
  takes roughly *N rows × your embedder's per-call latency* — local
  ollama on a laptop hits ~30 rows/sec; OpenAI is bandwidth-bound
  but easily 100+/sec.
- For a local ONNX backend (`-tags local_onnx`) it's CPU-bound and
  parallelises well; budget ~10–15 ms per row.
- A failure (model down, network drop) doesn't roll back successful
  rows; the report shows `examined / reembed / failed` and lists the
  first 20 errors. Re-running picks up where it stopped.

## Manual `.md` sync

opendray's mirror normally runs at session **spawn**: each time you
start a new Claude / Codex / Gemini session in a given cwd, opendray
walks `<cwd>/.claude/projects/.../memory/*.md` and ingests anything
new (idempotent against `source_path` + `source_mtime`).

But if Claude **edits** a memory file *during* an active session,
that edit only reaches pgvector on the next spawn. To force an
immediate ingest:

1. Memory page → set **Scope** = `project` and **Scope key** = the
   project's cwd.
2. Click **Sync .md**.
3. Toast reports the number of new memories ingested. Zero is
   normal when nothing has changed.

The sync only reads files; it never writes them. There's no risk of
clobbering Claude's local state.

## When to do what

| Symptom | Tool |
|---|---|
| New facts aren't in search even after `memory_store` | Restart? Then check active embedder strip |
| Agent forgot something it knew yesterday | Yellow Migrate banner — older embedder still has it |
| Inspector full of facts no one uses | Sort by `hit_count = 0`, delete |
| Edited a Claude `.md` and want it indexed *now* | **Sync .md** button |
| Want a fresh start | Browse → select all in scope → delete (no batch button yet; manual) |

## What's *not* automated

- **Periodic re-mirror** — opendray doesn't poll the filesystem.
  Use **Sync .md** on demand, or just spawn a new session.
- **Hit-count GC** — there's no auto-deletion policy. The inspector
  surfaces stale memories; pruning is manual on purpose so you don't
  lose load-bearing facts to a bad heuristic.
- **Backups** — pgvector data lives in your existing PostgreSQL.
  `pg_dump` of the `memories` + `memory_index_state` tables is the
  recovery path; opendray adds nothing on top.
