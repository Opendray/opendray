# ADR 0019 — Pluggable memory worker (M25)

**Status**: Accepted (shipped via PR #_TBD_, milestones M25)
**Date**: 2026-05-13
**Builds on**: ADR 0014 (memory subsystem), ADR 0018 (unified cross-CLI memory)

## Context

After ADR 0018 (M5-M24) shipped, the unified memory subsystem had
four LLM touchpoints, all hardcoded to the same path:
`summarizer.Registry` → OpenAI-compatible HTTP → typically a local
LM Studio model.

The four touchpoints have very different cost / latency / quality
profiles, but the architecture treated them identically:

| Touchpoint | Frequency | Quality budget | Latency budget |
|---|---|---|---|
| **Gatekeeper** | every `memory_store` | low (binary judgement) | hot path, <500ms |
| **Cleaner** | 24h batch | medium (keep/stale/duplicate) | non-critical |
| **Gitactivity** | 24h | high (multi-paragraph narrative) | non-critical |
| **Transcript** | every session_end | high (multi-paragraph narrative) | background |

Operators using opendray for serious cross-CLI memory wanted the
two narrative tasks (gitactivity, transcript) to use a frontier
model (Claude, Gemini) but had no way to do that without
running a separate proxy. Meanwhile gatekeeper / cleaner were
fine on local LM Studio.

## Decision

Introduce a `Worker` abstraction at `internal/memory/worker` with
two concrete implementations, and a per-task config table that
lets operators pick which one each touchpoint uses.

### Worker interface

```go
type Worker interface {
    Kind() WorkerKind
    Run(ctx context.Context, req Request) (Response, error)
}

type Request struct {
    Task         TaskKind
    SystemPrompt string
    UserInput    string
    MaxTokens    int
    Timeout      time.Duration
    ResponseFormatJSONSchema string // optional
}
```

Generic enough that all four touchpoints can use it without
losing their bespoke prompts / response parsers.

### Two implementations

**SummarizerWorker** wraps the existing `summarizer.Registry`
HTTP path. When `Request.ResponseFormatJSONSchema` is set, it
emits `response_format=json_schema` per the OpenAI 2024 spec
(LM Studio supports it; older OpenAI-compat endpoints fall back
gracefully).

**AgentWorker** spawns a headless CLI:

- `claude --print --append-system-prompt <prompt> --session-id <uuid>`
- `gemini --print --session-id <uuid> --include-directories <scratch>`

Input goes to stdin; stdout is captured until EOF; Request.Timeout
caps the process. Each call uses a fresh scratch CWD so the agent
doesn't pull in unrelated CLAUDE.md / GEMINI.md context. JSON-mode
tasks (cleaner) prepend the schema to the system prompt instead
of using `response_format` (agent CLIs don't natively support it).
Worker sessions deliberately don't create a `sessions` table row —
they're out-of-band invocations, invisible to the journaler.

**Why not `--bare`?** The Claude CLI's `--bare` flag (skip hooks /
plugins / CLAUDE.md auto-discovery) is tempting but forces auth
through `ANTHROPIC_API_KEY` only — opendray's multi-account flow
stores OAuth tokens (`CLAUDE_CODE_OAUTH_TOKEN`) which `--bare`
silently ignores, causing exit 1 "Not logged in". The isolation
properties `--bare` would give us are already provided by the
scratch CWD (CLAUDE.md auto-discovery has nothing to find) and
the `--print` mode (no tool use means PostToolUse / Stop hooks
don't fire). Plugin sync cost is accepted in exchange for OAuth
auth working.

Codex is unsupported because it has no `--print` equivalent.

### Per-task config table (memory_workers)

```sql
CREATE TABLE memory_workers (
    task          TEXT PRIMARY KEY,  -- gatekeeper|cleaner|gitactivity|transcript
    kind          TEXT NOT NULL,     -- summarizer|agent
    summarizer_id TEXT,              -- when kind='summarizer'; NULL = default
    provider_id   TEXT,              -- claude|gemini when kind='agent'
    account_id    TEXT,              -- claude multi-account id
    enabled       BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Seeded with `('<task>', 'summarizer')` for all four tasks so
existing deployments behave identically until an operator changes
the config — which they do via the new web/mobile **Memory →
Workers** settings page (`PUT /api/v1/memory/workers/{task}`).
No restart needed: `Registry.WorkerFor` reads the row on every
call.

### Metrics

A `memory_worker_calls` audit table captures every invocation:
task, worker_kind, provider_id, account_id, started_at,
duration_ms, success, error_message, input_bytes / output_bytes,
tokens_in/out (when known). The settings UI renders a 24h
rollup (count + avg latency + error count) per task so operators
can validate the cost / latency tradeoff before / after switching.

### What does NOT switch

**Gatekeeper stays on summarizer-only** even though the row
exists for visibility. M12's gatekeeper fires on every
`memory_store`, a hot path with a <500ms budget. Agent spawn is
5-15s; using it here would tank UX. The UI flags this row as
"summarizer-only — agent unsupported for high-frequency tasks".

The row exists because operators may want to pin a specific
summarizer provider for the gatekeeper independently of the
cleaner / gitactivity / transcript choices.

## Consequences

**Positive**:

- Per-touchpoint quality vs. cost dial. Operator can run a 3B
  local model for the hot-path gatekeeper, a 13B for the cleaner,
  and Claude Opus for the once-a-day narrative summaries — all
  configured in one settings page.
- No restart needed for config changes (read per-call).
- Metrics surface the actual cost of each choice — no need to
  guess.
- The agent worker reuses opendray's existing multi-account
  Claude plumbing (CLAUDE_CONFIG_DIR + CLAUDE_CODE_OAUTH_TOKEN),
  so the operator's existing accounts are immediately available.
- Codex's lack of `--print` is documented, not silent — the
  settings UI doesn't offer Codex as an agent provider.

**Negative**:

- Two code paths to maintain (summarizer HTTP + CLI spawn).
  Mitigated by keeping the per-touchpoint prompt + parser local
  to the touchpoint (gatekeeper, cleaner, gitactivity,
  transcript); only the LLM call dispatch is shared.
- AgentWorker swallows token counts: agent CLIs don't reliably
  expose usage. The metrics row records `tokens_in=tokens_out=0`
  for agent calls; the UI falls back to byte counts.
- New deployment-time concern: if the operator picks an agent
  worker for a task and the matching CLI isn't installed on the
  host, the call fails. The settings UI's `Test` button surfaces
  this before the next live tick.
- Agent calls run in a scratch CWD without project context. That's
  intentional (avoids contamination) but means the agent can't
  read the project's CLAUDE.md or files. Workers see only what's
  in the system prompt + user input. Each touchpoint's prompt was
  written to be self-contained, so this is fine in practice.

## Follow-ups

- **Codex support**. Codex has `resume`/`fork` subcommands but no
  one-shot `--print`. A future ADR could explore a "session manager
  shim" that spawns codex interactively, sends one prompt, captures
  output, kills — but the complexity isn't justified by demand yet.
- **Cost estimation**. With provider price tables, the metrics
  table's byte counts could approximate $/call. Useful when
  comparing local vs. agent for the same task.
- **Per-cwd worker overrides**. Some operators may want a more
  expensive worker for important projects and a cheap one for
  experiments. Not in M25; would extend `memory_workers` with
  optional `cwd` and per-cwd priority logic in `Registry.WorkerFor`.
- **Rate limiting / cost caps**. M25 has no per-task budget cap.
  An over-eager operator could rack up Claude API bills if their
  agent spawns become wedged. Worth adding a "daily-spend cap"
  knob in a follow-up.
