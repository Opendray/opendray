# Ollama walkthrough

For most operators this is the **easiest path to high-quality
semantic memory**. Ollama runs a local embedding model in a
side-car daemon, opendray talks to it over HTTP, no API key
needed. Total setup time: 5 minutes. No cgo, no rebuild.

## Why this over LocalONNX

| Path | Setup | Build | Pros |
|---|---|---|---|
| **HTTP + ollama** *(this section)* | 5 min | none | Trivial, swap models with one CLI command, hosts daemon |
| LocalONNX | 30 min | rebuild with `-tags local_onnx` | Single binary deploy, air-gapped, no extra daemon |
| BM25 | 0 | none | Default; keyword-only |

If you don't have a hard requirement to skip the ollama daemon,
pick HTTP + ollama. Switch to LocalONNX only when you need
"embed everything into one binary".

## Step 1 · Install ollama

macOS:
```bash
brew install ollama
brew services start ollama         # background daemon, persists across reboots
```

Linux:
```bash
curl -fsSL https://ollama.com/install.sh | sh
sudo systemctl enable --now ollama
```

Verify:
```bash
curl http://localhost:11434/api/tags
# {"models":[]}  (or your existing models)
```

## Step 2 · Pull an embedding model

Three good choices for semantic memory; pick one:

```bash
ollama pull nomic-embed-text       # 137 MB, 768-dim, English-strong, fast
ollama pull mxbai-embed-large      # 670 MB, 1024-dim, multilingual, top quality
ollama pull bge-m3                 # 1.2 GB, 1024-dim, multilingual SOTA
```

Recommendation:

- **nomic-embed-text** — start here. Smallest, fastest, very
  capable for English code/notes.
- **bge-m3** — switch when you've got significant Chinese / mixed
  content and the extra latency is OK (~80ms per call vs ~30ms).

You can have multiple models pulled and swap between them in
opendray's config without touching ollama.

## Step 3 · Configure opendray

Settings → Server → Memory:

```
Backend:               http
Default scope:         project
Similarity threshold:  0.5     (raise from 0.1 — dense embeddings score higher)
```

Then under **HTTP backend (used when backend=http)**:

```
Base URL:    http://localhost:11434/v1
Model:       nomic-embed-text       (or whichever you pulled)
API key:     (leave blank — ollama doesn't need one)
```

Click **Save changes**, then **Restart server**.

After restart, the Inspector status strip should show:

```
http:nomic-embed-text · 768-dim · enabled
```

Click **Test embedder**. You should see a toast like:

```
Embedder OK: http:nomic-embed-text · 768 dimensions
vector_preview = [0.041, -0.013, 0.027, …]
```

That confirms opendray → ollama → embedding round-trip works.

## Step 4 · Verify cross-CLI memory

Spawn a Claude session in any project (cwd):

```
me: "I prefer pnpm as my package manager"
claude: Called opendray-memory.memory_store("...")
        → stored as mem_xxx
```

Now spawn a Codex session in the **same cwd**:

```
me: "what's my package manager preference?"
codex: Called opendray-memory.memory_search("package manager preference")
       → Found 1 memory hit(s)
       Based on memory: you prefer pnpm.
```

The cross-CLI link works because:
1. Claude wrote the fact via opendray-memory MCP
2. opendray embedded it with nomic-embed-text → 768-dim vector → pgvector
3. Codex's search query went through the same embedder
4. Cosine similarity found the match (semantic, not just keyword)

If you'd done the same flow with `backend=bm25`, "package manager
preference" wouldn't have matched "pnpm" because there's no token
overlap.

## Tuning

### Threshold

`similarity_threshold` defaults to 0.1 (BM25-friendly). For dense
embeddings raise it:

| Model | Suggested threshold |
|---|---|
| nomic-embed-text | 0.5 |
| mxbai-embed-large | 0.55 |
| bge-m3 | 0.6 |

Too low → noisy hits. Too high → relevant facts get filtered out.
Walk it up or down by 0.05 and watch what comes out of `memory_search`.

### Top-K

Default 5 is generous. Drop to 3 if you find the agent stuffing
too much memory into context. Raise to 10 for "broad recall"
sessions.

### Switching models

Pull the new model, edit `[memory.http].model`, save, restart.
Existing memories embedded under the old model **stay in pgvector
but won't be returned** — opendray filters by embedder name to
keep cosine math honest. To reuse them, either:

1. Stay on the old model, OR
2. Delete + re-store via the Inspector (small datasets), OR
3. Wait for phase 2.B.3's re-embedding utility (planned).

## Troubleshooting

**Connection refused on Test embedder**

```
ollama is probably not running.

  brew services restart ollama        # macOS
  sudo systemctl restart ollama       # Linux
  curl http://localhost:11434/api/tags  # should return JSON
```

**`model not found`**

You configured a model that hasn't been pulled. Pull it:

```bash
ollama list                          # see what you have
ollama pull <model-name>
```

**Embeddings returned but search hits never appear**

Threshold too high. Settings → Memory → Similarity threshold → 0.3,
restart. Raise once you've seen a few hits and feel out the
distribution.

**Slow embedding latency (>500ms per call)**

Check `ollama ps` — if the model isn't in memory ollama loads it
on first call (~2s for nomic, ~5s for bge-m3). After that it
stays warm. The `ollama ps` keeps it loaded for ~5 minutes by
default; raise via `OLLAMA_KEEP_ALIVE=24h` in the systemd unit
or `launchctl setenv`.

## What you NOW have

- Cross-CLI semantic memory shared across Claude / Codex / Gemini
- Zero vendor API keys
- Models swappable with one CLI command
- All inference local — your data never leaves the machine
- Sub-100ms latency per call after warm-up

This is the configuration most operators settle on after trying
BM25 (too keyword-y) and weighing LocalONNX (too much rebuild
hassle).
