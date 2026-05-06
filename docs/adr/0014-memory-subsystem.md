# ADR 0014 — Built-in cross-CLI memory subsystem

**Status**: Accepted (phase 1 + phase 2 step 1 shipped)
**Date**: 2026-05-04

## Context

Each agent CLI in opendray has its own private memory:
- Claude Code v2.1+ writes markdown files under
  `<project>/.claude/.../memory/`
- Codex stores rollouts under `~/.codex/sessions/`
- Gemini logs under `~/.gemini/tmp/<sha>/`

These don't talk to each other. A user who tells Claude "I prefer
pnpm" finds the next Codex session in the same project doesn't
know. Cross-CLI continuity is one of opendray's core value props
("multiplexer for AI agent CLIs"), so this gap is structurally
important.

We considered:

1. Wrapping qdrant + mem0 as services the operator runs alongside
   opendray. Rejected — violates the "single binary, single
   Postgres" deploy story.
2. Shipping a Python subprocess for mem0. Rejected — drags 30+
   pypi deps into a Go-only stack.
3. Building it ourselves. Accepted.

## Decision

opendray ships a built-in memory subsystem with two replaceable
layers:

```
internal/memory/
├── Embedder interface
│   ├── BM25Embedder            (default, pure-Go hash trick)
│   ├── OpenAICompatibleEmbedder (HTTP /v1/embeddings — ollama,
│   │                             OpenAI, LocalAI, vLLM)
│   └── LocalONNXEmbedder       (phase 2 — bge-m3 INT8 via cgo
│                                onnxruntime)
└── Store interface
    └── PgvectorStore            (uses opendray's existing PG with
                                  the vector extension)
```

Agents access it through an in-process MCP server (`opendray
mcp-memory` subcommand), auto-attached to every spawned session
via the existing catalog adapter. Three tools: `memory_search`,
`memory_store`, `memory_list`.

## Why pgvector over chromem-go / sqlite-vec / lancedb / qdrant

- pgvector reuses opendray's existing Postgres — zero new
  infrastructure
- HNSW + IVFFlat indexes scale comfortably to ~10M vectors (well
  past anyone's individual operator scale)
- Operator already manages PG backups, monitoring, etc.
- Mature ecosystem (Supabase / Neon use it), updates frequent

chromem-go was the runner-up — single-file, pure Go, ideal for
operators without PG. We may add it as an alternative `Store`
implementation later. v1 picks pgvector because the typical
opendray operator already runs PG for the rest of opendray's data.

## Why MCP over PTY input rewriting

We considered intercepting the agent's PTY input and prepending
recalled memories before forwarding. Rejected because:

- Breaks raw-mode terminator semantics (Claude needs `\r`, shell
  needs `\n`)
- Breaks alt-screen TUI apps
- The agent doesn't *know* it's getting injected memory, so it
  can't reason about freshness or relevance

MCP makes memory a tool the agent decides to call. The agent gets
to choose when search is worth the round-trip and when storage is
worth committing. Far better recall behaviour than blind
prepending.

## Why BM25 default instead of a real embedder

For phase 1 we needed something with **zero external dependencies**:
no model files, no GPU, no API keys, no extra services. BM25
hash-bucketed cosine fits exactly. Quality is poor compared to a
dense embedder, but:

- It works for code-shaped memories (file names, library names,
  exact identifiers — exact-match cases where keyword wins anyway)
- Lets phase 1 ship without cgo + 600MB binary inflation
- Operators with embedding services can flip `backend = "http"`
  immediately to get semantic recall

Phase 2 will add `LocalONNXEmbedder` with bge-m3 INT8 (~600MB)
behind a build tag for operators who want offline dense
embeddings without HTTP.

## Why scope = project by default

Three options were on the table:

| Default | Pros | Cons |
|---|---|---|
| `session` | Strict isolation | Defeats cross-CLI sharing |
| `project` | Cross-CLI within same cwd | Harder for cross-project recall |
| `global` | Maximum sharing | Privacy collapse, conflicting facts |

Project is the most common useful sharing boundary in real use
(operators work in one project at a time), and matches the
"unified memory across CLIs in this codebase" mental model the
feature exists to provide. Per-call override is always available.

## Why mirror Claude's local memory files

Despite system-prompt guidance, Claude often defaults to writing
its own markdown memory files instead of calling the MCP store.
The Mirror reads those files on every session spawn and ingests
them into pgvector — so even when Claude doesn't call our MCP
tools, the resulting facts are still available to Codex / Gemini
in the same project.

One-way (Claude writes → opendray reads). No reverse sync, no
risk of feedback loops.

## Why an integration row + cached plaintext key

The MCP subprocess (`opendray mcp-memory`) talks to opendray over
HTTP — same as any third-party app. It needs a bearer token. We
considered:

- Admin token (full power, security overkill)
- Internal-only auth path (changes auth middleware)
- A dedicated integration with limited scopes (chosen)

We mint an integration named `opendray-memory` at startup, scope
it minimally (`session:read` for future cwd visibility), and
cache the plaintext key in `~/.opendray/memory.key` (mode 0600,
same threat model as `secrets.env`). Cached key survives across
restarts so existing sessions' mcp.json keep working — rotating
on every startup would silently 401 every active session.

Recovery for stolen key: delete the cache, restart opendray.
Forced rotation = fresh plaintext.

## Token reduction expectations

Realistic numbers for the BM25 baseline:

- Long sessions (50K-200K tokens of history): RAG-via-memory
  saves 20-40% on context window because the agent fetches only
  relevant past facts instead of replaying everything.
- Short sessions (<10K tokens): no win or slightly worse —
  embed + retrieve overhead exceeds savings.

The real gain isn't raw tokens, it's **persistence**: the agent
remembers your decisions across sessions and across CLIs. We
frame the user-facing pitch that way.

## Consequences

### Positive

- Zero-config persistence works out of the box
- Cross-CLI memory continuity unlocked
- Replaceable embedder lets ops trade quality vs cost vs deps
- Replaceable store leaves the door open for chromem-go / sqlite-vec

### Negative

- BM25 quality is meh; users expecting "smart memory" with the
  default backend will be disappointed until they wire ollama or
  switch on phase 2 ONNX
- Memory rows accumulate; no consolidation / eviction in v1
- Mirror only handles Claude — Codex / Gemini local memories
  (when they stabilise) need their own ingestors
- Scope is the only privacy boundary; multi-user opendray needs
  per-operator filtering

### Compatibility

- New `[memory]` config section, all fields optional
- New migration `0011_memory.sql` requires pgvector extension
- New subcommand `opendray mcp-memory`
- New routes under `/api/v1/memory/*` (dual-auth)
- New automatic integration row `opendray-memory` (visible in
  Integrations UI; deleting it cripples memory until next restart)

## Implementation references

- `internal/memory/` — Embedder, Store, Service, Mirror
- `cmd/opendray/mcp_memory.go` — stdio MCP subcommand
- `internal/catalog/adapter.go::injectMemoryGuidanceFor` —
  per-CLI system-prompt guidance
- `internal/store/migrations/0011_memory.sql` — schema
- `app/web/src/components/settings/MemoryInspector.tsx` — UI

## Future work

- LocalONNXEmbedder (bge-m3 INT8) behind a build tag
- chromem-go alternative Store
- Codex / Gemini local-memory mirrors
- LRU / LLM-driven eviction
- Per-operator scope filtering
- Consolidation: when a new fact contradicts an old one, decide
  ADD / UPDATE / DELETE / NOOP via a small LLM call (mem0's
  pattern)
