# Configuration

Memory is configured under `[memory]` in `config.toml` and
mirrored in **Settings → Server → Memory**. All fields are optional
— the zero-value config gives BM25 + pgvector + project scope,
which is the documented default.

## config.toml shape

```toml
[memory]
backend              = "auto"      # auto | bm25 | http
store                = "pgvector"  # only option in v1
default_top_k        = 5
similarity_threshold = 0.1
chromem_path         = ""          # phase 2 placeholder

[memory.local]
model = "bge-m3"                   # phase 2 placeholder

[memory.http]
base_url   = "http://localhost:11434/v1"
model      = "nomic-embed-text"
api_key    = ""                    # blank for ollama; required for OpenAI etc.
dimensions = 0                     # 0 = autodetect

[memory.scope]
default = "project"                # session | project | global
```

## Backend choices

| Value | What it does | When to pick |
|---|---|---|
| `auto` | BM25 today; future ONNX (bge-m3) when phase 2 ships | The default. Lets opendray upgrade you transparently. |
| `bm25` | Pure-Go keyword retrieval, 384-dim hash-bucket vectors | When you want zero external dependencies and accept keyword-only matching. |
| `http` | OpenAI-compatible `/v1/embeddings` endpoint | When you have an embedding service running (ollama, OpenAI, LocalAI, vLLM) and want true semantic search now. |

### BM25 limitations

BM25's hash-bucket vectors only match **exact token overlap**. If
the user wrote "我喜欢 pnpm" and you search "package manager", BM25
returns zero hits — no shared words. For demos this is the fastest
and most reproducible path; for production-quality recall, switch
to `http` with a real embedding model.

### HTTP backend examples

**ollama (local, free, no key)**

```toml
[memory.http]
base_url = "http://localhost:11434/v1"
model    = "nomic-embed-text"
```

Make sure ollama is running and you've pulled the model:
`ollama pull nomic-embed-text`. Opendray autodetects the
dimension on first call.

**OpenAI**

```toml
[memory.http]
base_url = "https://api.openai.com/v1"
model    = "text-embedding-3-small"
api_key  = "sk-…"
```

Cheapest official option (~$0.02 per million tokens). 1536
dimensions, multilingual.

**Voyage AI**

```toml
[memory.http]
base_url = "https://api.voyageai.com/v1"
model    = "voyage-3-lite"
api_key  = "voyage-…"
```

Anthropic's recommended embedding partner. Better quality on code
than OpenAI's small models.

## Threshold + Top-K

`similarity_threshold` is the floor below which `memory_search`
hits are dropped. Defaults to `0.1` because BM25 sparse vectors
rarely break 0.5 even on related text. **Bump to 0.5+ when you
switch to a dense embedder** — false positives are harmless with
sparse, painful with dense.

`default_top_k` is how many hits the MCP tool returns when the
agent doesn't specify. 5 is the sweet spot for context-window
budget; agents rarely use more than the top 3.

## Scope defaults

`memory.scope.default` controls what `memory_store` uses when the
agent doesn't pass a scope explicitly. **Project is recommended** —
it's the most useful sharing boundary in practice. Read more at
[Scopes](#memory-scopes).

## Restarts

Changing **any** `[memory]` field requires a server restart — the
embedder + store are wired once at `app.New`. The Settings UI
flags the Memory section with a yellow "restart required" badge
the moment you edit a field.

## What you can edit live (no restart)

- Inspector pane — browse, search, delete memories: instant
- `memory_store` / `memory_search` from agents: instant (they use
  whatever embedder is currently bound)

What needs restart:

- Embedder backend (auto → http or vice versa)
- HTTP base_url, model, api_key
- Threshold, default_top_k (read once at app.New into the Service
  options bag)
- Default scope

## Verifying the live config

Settings → Server → Memory → Inspector → status strip:

- `bm25 · 384-dim · enabled` → BM25 backend is bound
- `http:nomic-embed-text · 768-dim · enabled` → http backend, model
  name appended after the colon, dimensions discovered
- `unavailable` → memory subsystem not initialised (check
  `tail /tmp/opendray.log`)

Plus the **Test embedder** button — round-trips a small text
through the configured embedder and shows the first 4 dimensions
of the resulting vector. Catches "I configured ollama wrong"
before the agent finds out.
