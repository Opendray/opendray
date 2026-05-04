# The Claude local-memory mirror

Claude Code v2.1+ has its own memory feature — when the model
decides something is worth remembering, it writes a markdown file
under the project's Claude directory:

```
~/.claude-accounts/<account>/projects/<encoded-cwd>/memory/<topic>.md
~/.claude/projects/<encoded-cwd>/memory/<topic>.md
```

Without intervention, those memories are **invisible** to Codex
and Gemini sessions in the same project — defeating the
cross-CLI value prop.

opendray's **mirror** closes that gap. On every session spawn (any
provider — claude, codex, gemini, shell), opendray walks the
relevant memory directories for the session's `cwd` and ingests
each `.md` file as a project-scoped memory in pgvector. The next
`memory_search` from any CLI in that cwd sees them.

## What the mirror reads

For a session in `cwd=/Users/x/myproj`, the mirror scans:

- `~/.claude/projects/-Users-x-myproj/memory/*.md`
- `~/.claude-accounts/<account>/projects/-Users-x-myproj/memory/*.md`
  (every account dir under `~/.claude-accounts`, deduped via
  `EvalSymlinks` since the multi-account setup typically symlinks
  shared/projects → each account)

Files with name `MEMORY.md` are skipped — that's Claude's index
file (a list of links to the actual memory topic files).

## What gets stored

Each `.md` becomes one memory row:

```json
{
  "id":         "mem_…",
  "scope":      "project",
  "scope_key":  "/Users/x/myproj",
  "text":       "<full file contents, frontmatter included>",
  "embedder":   "bm25",
  "metadata": {
    "source":        "claude_local_memory",
    "source_path":   "/Users/x/.claude-accounts/.../preference_pnpm.md",
    "source_mtime":  "2026-05-04T10:00:36Z",
    "source_hash":   "cb42172e3648cf56"
  },
  "created_at": "…"
}
```

The full file content (frontmatter + body) becomes the memory's
text. This is intentional — the frontmatter has structured fields
(`name`, `description`, `type`) that BM25 indexes alongside the
body, and a future structured ingestor can parse them out.

## Idempotency

The mirror runs on **every** session spawn. To avoid duplicate
ingestion, it dedupes by `metadata.source_path + source_mtime`:

- Same path, same mtime → already ingested, skip
- Same path, newer mtime → ingest as new row (we don't dedupe;
  the inspector lets you delete obsoleted ones manually)
- New path → ingest

So if Claude writes 5 files today and you spawn 10 sessions today,
each new session sees the 5 files but skips re-ingesting them.

## When it runs

Inside the catalog adapter's PrepareFunc, right after the agent
process is about to spawn:

```go
if sp.memoryMirror != nil {
    cwd := session.Cwd(prepareCtx)
    if cwd != "" {
        go func() {
            sp.memoryMirror(context.Background(), cwd)
        }()
    }
}
```

Fire-and-forget goroutine — spawn isn't blocked on filesystem
walks or embed calls. The agent might race ahead and call
`memory_search` before the mirror finishes; in practice the mirror
takes <100ms for the kinds of memory dirs Claude actually writes,
and the agent's first tool call won't fire that fast anyway.

## On-demand sync

The Memory page (left sidebar 🧠 → `g m`) has a **Sync .md** button
that runs the same ingestor on demand for the current scope_key.
Use it when:

- You edited a Claude memory file in your editor and don't want to
  spawn a fresh session just to mirror.
- An agent in an active session wrote a new `.md` (the in-flight
  session won't see it until you sync).

The button is gated to `scope = project` — that's the only scope
mirror operates on. It's idempotent (same path+mtime is a no-op),
so spamming it is harmless. The toast reports how many new files
were ingested in this call.

## What it doesn't do

- **No fsnotify**: mirror only runs on session spawn or when you
  hit the manual Sync button — never in real-time. The in-flight
  agent that wrote the file won't see its own change appear in
  search until next sync.
- **No reverse sync**: opendray-stored memories don't get written
  back to Claude's local files. Claude is a write-source; opendray
  is the unified read-source.
- **Codex / Gemini local memories** aren't mirrored yet. Their
  storage formats are less standardised (Codex rollouts are
  per-session JSONL, not per-project markdown); we'll add ingestors
  as their conventions stabilise.

## Disabling

There's no toggle — if memory is enabled, mirror runs at every
spawn. If you don't want this, your options are:

- Don't enable memory at all (`[memory] backend = "off"` — TODO,
  not exposed yet).
- Delete mirrored rows from the inspector after each spawn (not
  recommended; tedious).
- Move the directory you don't want indexed out of Claude's project
  tree.

In practice, mirror is the cross-CLI value prop — turning it off
mostly defeats the point of opendray's memory layer.

## Storage cost

Each `.md` typically becomes a 384-float BM25 vector (~1.5KB) plus
the original text (1-3KB) plus metadata (~200B). Across 50 Claude
memories per project, ~250KB. Postgres handles this fine; no
practical limit.

## Verifying

Memory page → expand any row. Memories with `source =
claude_local_memory` were mirrored from a Claude `.md`. The
metadata shows the original file path + mtime, useful when you
want to know "where did this fact come from?".

Or via SQL:

```sql
SELECT id, scope_key, metadata->>'source_path' AS path
FROM memories
WHERE metadata->>'source' = 'claude_local_memory'
ORDER BY created_at DESC LIMIT 20;
```
