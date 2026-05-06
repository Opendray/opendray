# ADR 0013 — Ambient Memory (auto-capture + auto-inject)

**Status**: Accepted, 2026-05-05
**Authors**: Linivek, Claude (planner + impl)
**Builds on**: ADR 0014 (memory subsystem)

## Context

ADR 0014 established opendray's persistent memory layer (pgvector
+ MCP bridge + 3-CLI mirror). It works perfectly for facts the
user explicitly tells the model to remember — but in practice
99% of "valuable durable context" surfaces in conversation
*without* explicit "remember xxx" prompts. Users who clear
context or jump between Claude / Codex / Gemini lose this
context.

Three observations drove the design:

1. opendray sessions are PTY byte streams, not message-discrete
   — there's no "fire after every user message" hook on the
   eventbus.
2. Anthropic's Pro subscription token cannot be reused as an
   API key, so "use my Claude Pro plan as the summarizer" is
   impossible without a paid Console API key.
3. Users have wildly different cost / privacy / latency
   tolerances — one provider doesn't fit all.

## Decision

Ship **ambient memory** as three orthogonal first-class objects:

1. **Provider** — what LLM does the extraction. Five built-in
   kinds (anthropic, openai, ollama, lmstudio, integration);
   the integration kind passes through to any HTTP service
   speaking the documented `/summarize` protocol, unlocking
   zero-cost local-only paths.
2. **Capture rule** — when to fire. Four trigger kinds
   (`after_messages`, `on_idle`, `k_chars`, `manual`);
   per-session or global default.
3. **Injection profile** — what (if anything) goes into the
   agent's system prompt at spawn. Five strategies (`none`,
   `top_k_recent`, `top_k_relevant`, `manual_only`, `hybrid`)
   plus `on_keyword` reserved.

Storage: 4 new tables (`memory_summarizer_providers`,
`memory_capture_rules`, `memory_injection_profiles`,
`memory_summarizer_calls`) + 4 columns on `memories`
(`source_kind`, `source_ref`, `summarizer_session`,
`confidence`).

## Implementation highlights

### PTY-bypass via transcript polling

Instead of hooking PTY (which has no message boundary), the
capture engine polls every 10s, calls
`session.Manager.History(id)` which already wraps the
provider-native transcript JSONLs (~/.claude/projects/*.jsonl
etc.), and decides whether to fire based on cursor state.

Worst-case latency is 10s, but the PTY hot path is untouched
and we get the same quality of message boundary that the
underlying CLI sees.

### Cipher reuse from backup subsystem

API keys for paid providers (Anthropic, OpenAI) are encrypted
at rest using the same AES-GCM cipher backup uses (env-derived
master passphrase via `OPENDRAY_BACKUP_KEY`). This avoids
introducing a second secrets envelope and means any operator
using backup automatically has encrypted summarizer keys.

Local providers (ollama, lmstudio, integration without
outbound token) require no cipher.

### Failure backoff

Three consecutive failures of a (rule, session) pair pauses
that pair for 1h. Prevents a misconfigured rule from burning
cycles every tick.

### Dedup before store

Each extracted fact runs through `memory.Search` against the
target scope; matches above `dedup_threshold` (default 0.85)
are skipped. Prevents the summarizer from re-extracting the
same fact every cycle.

### Spawn-time injection — same dispatch as memory guidance

`catalog.SessionProvider.WithAmbientInjector` plumbs the
injector through the existing per-CLI dispatch
(`injectMemoryGuidanceFor`'s sibling `injectAmbientMemoryFor`).
Claude gets another `--append-system-prompt`, Codex appends to
`AGENTS.md`, Gemini to `GEMINI.md`.

### Token cost telemetry

Every provider call writes a `memory_summarizer_calls` row
with `input_tokens`, `output_tokens`, and a snapshot
`estimated_usd` computed from the in-process pricing table.
The Settings UI panel aggregates per-provider via SUM.

Pricing table (`internal/memory/summarizer/cost.go`) is
hand-curated per Anthropic / OpenAI public pricing. Local
providers cost $0.

## Phases

- **Phase A (10 steps, 47 tests)** — schema, summarizer types
  + 2 providers (Anthropic, ollama), capture engine + rule
  store, injector + profile store, admin handlers, app wiring.
- **Phase B (1337 LOC)** — added LM Studio + OpenAI providers
  (shared OpenAICompatProvider), IntegrationProvider with
  /summarize protocol, full trigger kinds (on_idle, k_chars,
  manual), full injection strategies (top_k_relevant,
  manual_only, hybrid, on_keyword reserved), spawn-time
  injection wiring.
- **Phase C (this commit)** — `/run-now` endpoint, MCP
  `memory_load_context` + `memory_get_provenance` tools,
  Settings → Memory · Ambient UI control panel with token cost
  table, tutorial.

## Consequences

**Good:**
- Zero-config opt-in: subsystem ships always-on but does
  nothing until at least one rule + provider are configured.
- Local-first by default: ollama and LM Studio cover the
  privacy-sensitive deployments without paid APIs.
- Cost-transparent: the UI shows estimated USD per provider
  on aggregated call log.
- Cross-CLI continuity: facts captured during a Claude
  session are visible to Codex / Gemini on next spawn.

**Trade-offs:**
- 10s polling latency means the very latest message in an
  active conversation may not be summarized for ≤10s. Fine
  for ambient capture; not suitable as a "live response
  grounder."
- API key encryption depends on `OPENDRAY_BACKUP_KEY`;
  ollama-only or lmstudio-only deployments work without it,
  but anthropic/openai providers require backup feature
  enabled.
- `on_keyword` injection strategy ships its UI but the actual
  message-stream hook is v1.1. Selecting it is functionally
  equivalent to `none` until then.
- The summarizer adds a per-tick LLM call cost; operators
  must size their dedup threshold and trigger frequency for
  their cost budget.

## Out of scope (Phase v1.1+)

- Spawn-time prompt injection per-session UI override.
- on_keyword's actual message-stream hook.
- Session toolbar "Load context" button (manual injection
  trigger from the session UI).
- Web UI for per-session capture rule / injection profile
  overrides.
- Cost forecasting (predict next month's spend from current
  trigger frequency × avg call cost).
