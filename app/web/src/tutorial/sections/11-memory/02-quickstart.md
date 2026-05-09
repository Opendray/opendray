# Quickstart

Memory is **on by default** when you start opendray. There's no
flag, no extra service, no API key. This section walks through the
five-minute experience.

## What you don't need to do

- ❌ Install qdrant or chromadb
- ❌ Run mem0 as a subprocess
- ❌ Sign up for OpenAI / Voyage / Cohere
- ❌ Configure any environment variables

## Step 1 · Start opendray

```bash
go run ./cmd/opendray serve -config config.toml
```

You should see a log line like:

```
INFO memory ready  embedder=bm25  dimensions=384
INFO memory MCP auto-attach enabled
```

If memory init fails (e.g. pgvector extension missing), you'll see
the error here and the rest of opendray still boots — just without
memory tools attached to sessions.

## Step 2 · Spawn any session

Open the web UI → Sessions → Spawn → pick Claude/Codex/Gemini and a
working directory. opendray writes a per-session `mcp.json` that
includes `opendray-memory` automatically.

You can verify by inspecting the rendered file:

```bash
ls /var/folders/.../opendray-sess-<id>/
cat /var/folders/.../opendray-sess-<id>/claude-mcp.json
```

You'll see an `opendray-memory` entry alongside any other MCP
servers you've registered.

## Step 3 · Tell the agent something to remember

In a Claude session:

```
me: my preferred frontend frameworks are vue and react
```

The agent does either:

- **Calls `opendray-memory.memory_store(text)`** — preferred path,
  installs into the shared store immediately, or
- Writes `<project>/.claude/.../memory/<topic>.md` — the local-only
  Claude path. opendray's mirror picks it up on the next session
  spawn (see [Mirror](#memory-mirror) for details).

Either way, the fact lands in pgvector under
`scope=project, scope_key=<your cwd>`.

## Step 4 · Verify in the UI

Two pages, deliberately split:

- **Settings → Server → Memory** is the *configuration draft* —
  embedder choice, ports, dim, restart-required fields. Click
  **Test embedder** here; toast confirms the active backend is alive.
- **Memory** (left sidebar 🧠, shortcut `g m`) is the *runtime
  inspector* — browse, search, edit, delete the actual stored
  memories.

On the Memory page you should see:

- Status badge: `bm25 · 384-dim · enabled`
- Inspector lists the memory you just stored
- Search "vue" or "react" returns the row with similarity > 0
- Each row shows hit-count once searches have hit it

When you switch embedder backends later (e.g. wire up ollama), a
yellow Migrate banner appears on this page to reembed older
memories — see [Maintenance](#memory-maintenance).

## Step 5 · Cross-CLI test

Spawn a Codex session in the **same cwd** and ask:

```
me: what frontend framework do I usually use?
```

Codex should call `opendray-memory.memory_search` and get the same
fact back. That's the cross-CLI value prop, working.

## What's NOT happening

- The agent's response isn't going through opendray; only its tool
  calls.
- opendray isn't reading agent stdout for memory; nothing is
  scraped from the conversation. Only explicit tool calls (or
  Claude's local memory files via the mirror) end up in pgvector.
- Other operators on the same opendray instance can't see your
  memories — `scope_key` is your cwd; if their cwd is different,
  the rows are invisible.

## Troubleshooting at a glance

| Symptom | First thing to check |
|---|---|
| Agent never calls memory tools | Did you spawn the session AFTER the system-prompt guidance was added? Restart opendray, restart session. |
| `tool error: 401 unauthorized` | The mcp.json has a stale API key. Restart the session — opendray re-renders mcp.json with the cached key. |
| Search returns no hits | BM25 only matches exact tokens. Try a query word that literally appears in the stored text. |
| `connection refused` from mcp-memory | opendray gateway crashed. Check `tail -f /tmp/opendray.log`. |

Deep dive: [Troubleshooting](#memory-troubleshooting).
