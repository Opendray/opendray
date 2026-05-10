# Memory — overview

opendray ships a built-in **persistent memory layer** so the agents
you spawn (Claude / Codex / Gemini) can remember things across
sessions and across CLIs without external services or extra API
keys.

## What problem it solves

Each agent CLI has its own private memory:

- Claude Code v2.1+ writes markdown into `<project>/.claude/.../memory/`
- Codex stores rollouts under `~/.codex/sessions/`
- Gemini logs under `~/.gemini/tmp/<project>/`

These don't talk to each other. If you tell Claude "I prefer pnpm",
the next Codex or Gemini session in the same project doesn't know.
opendray's memory subsystem is the **shared** layer they all read
and write through.

## How it works

```
                          opendray gateway
                          ┌────────────────────────┐
   Claude/Codex/Gemini    │                         │
   (agent process)        │  /api/v1/memory/*       │
        │                 │       │                 │
        │ MCP             │       ▼                 │
   ┌────┴───────┐         │  Embedder (BM25 / ONNX) │
   │ opendray   │ HTTP    │       │                 │
   │ mcp-memory │─────────┤       ▼                 │
   │ subprocess │         │  pgvector store         │
   └────────────┘         │       │                 │
                          └───────┼─────────────────┘
                                  ▼
                          PostgreSQL
```

Every spawned session gets a `opendray-memory` MCP server
auto-attached to its `mcp.json`. The agent's tool list grows
three entries:

- `memory_search(query)` — find facts relevant to the query
- `memory_store(text)` — persist a durable fact
- `memory_list(limit)` — dump recent facts

## Default behaviour

- **Backend**: BM25 (pure-Go keyword retrieval, ~384-dim sparse
  vectors). No model files, no GPU, no API key.
- **Store**: pgvector on opendray's existing PostgreSQL.
- **Scope**: project — every session in the same `cwd` shares
  memories. Different projects are isolated.
- **Mirror**: on each session spawn, opendray reads any
  `~/.claude-accounts/.../<encoded-cwd>/memory/*.md` files Claude
  has written and ingests them so cross-CLI search picks them up.

## When you'll see it work

1. Open a Claude session, say "remember I prefer pnpm"
2. Either Claude calls `opendray-memory.memory_store(...)` directly,
   OR it writes a local `.claude/.../memory/preference_pnpm.md` —
   either way, opendray ends up with the fact in pgvector.
3. Open a Codex session in the same `cwd`, ask "what package
   manager do I prefer?"
4. Codex calls `opendray-memory.memory_search("package manager")`
   and gets the pnpm fact back.

## Read on

| Topic | Section |
|---|---|
| Quick start, agent tool calls | Quickstart |
| session vs project vs global scope | Scopes |
| Settings UI, embedder choice, HTTP backend | Configuration |
| The Claude local-memory mirror, advanced | Mirror |
| Common errors and recovery | Troubleshooting |

For the third-party developer view (calling `/memory/*` directly
from your own app), see
[Consuming opendray → REST API](#consuming-rest-api).
