# Ambient Memory — capture rules

A **capture rule** is "for these sessions, fire the summarizer
when this trigger condition is met."

## Per-session vs global default

`session_id` controls scope:
- **NULL = global default** — applies to every live session that
  doesn't have a specific override.
- **non-NULL = session-scoped** — applies only to that one session,
  shadows the global default.

Most operators have one global rule and never set per-session
ones. The override exists for cases like: "this session is a
slow hand-curated review, summarize it more aggressively."

## Trigger kinds

### `after_messages` (default)

Fire after N new user messages have arrived since the last
capture. The most common configuration.

```
trigger_config: {"n": 6}
```

Default n = 6. Lower numbers (3) catch more facts but cost more
tokens; higher (10-20) is cheaper but riskier (the model may
forget mid-conversation context that gets interrupted).

### `on_idle`

Fire when the session has been idle (no new user messages) for
at least N seconds AND new messages exist since the last fire.

```
trigger_config: {"seconds": 60}
```

Useful when conversations come in bursts — capture the whole
burst once it settles down rather than mid-burst.

### `k_chars`

Fire when the cumulative character count of new user messages
crosses K.

```
trigger_config: {"k": 4000}
```

Default 4000 (≈ 1000 tokens). Length-aware alternative to
`after_messages` for chats where prompts vary widely (long
prompt = one trigger, ten short prompts = one trigger).

### `manual`

Never auto-fires. Only triggered via the **Run now** button or
`POST /api/v1/memory-capture-rules/{id}/run-now`.

Useful for batch / on-demand workflows where you don't want
ongoing capture but periodically want to dump the current
transcript into memory.

## Dedup threshold

Each extracted fact is run through `memory.Search` against the
target scope before insertion. If any existing memory comes back
with similarity > `dedup_threshold` (default 0.85), the fact is
skipped.

- 0.95 = strict (only skip near-identical phrasings)
- 0.85 = recommended (catches paraphrases reliably)
- 0.70 = loose (may drop facts that should be kept; appropriate
  for very narrow projects)

The skipped/stored split is recorded in
`memory_summarizer_calls.facts_skipped_dedup` /
`facts_stored` so you can tune over time.

## Target scope

Where the extracted fact gets stored:
- `session` — visible only to this session
- `project` (default) — visible to every session in the same cwd
- `global` — visible everywhere

Project is right for almost every workflow. Global is useful for
operator-level preferences that aren't tied to any specific
project.

## Failure backoff

When a provider fails 3 consecutive times for a given
(rule, session) pair, the engine pauses that pair for 1 hour.
This prevents a misconfigured rule from burning cycles every
tick. The pause is in-memory — restarting opendray clears it.

## Run now

Every rule has a **Run now** button (UI) /
`POST /memory-capture-rules/{id}/run-now` (API) that fires the
rule immediately, bypassing trigger evaluation and pause state.
Useful for:
- Manual triggers
- Testing a freshly-created rule
- Forcing a fact extraction before ending a session
