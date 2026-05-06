# Ambient Memory — injection profiles

An **injection profile** controls what — if anything — gets
prepended to the agent's system prompt at spawn time. Without a
profile, the model still has access to memories via the
`memory_search` MCP tool; injection just makes recent context
visible without the model having to think to look.

## Strategies

### `none` (default if no profile exists)

Don't inject anything. Model uses `memory_search` /
`memory_load_context` on demand if it wants context.

Use this when:
- You don't want any memory in the context window
- You trust the model to look things up itself
- You're testing — confirm capture works without injection
  affecting outputs

### `top_k_recent`

Inject the K most recently created project-scoped memories at
spawn. Format:

```
## Recent project memory
opendray injected the following durable facts from prior
sessions in this project:
- User prefers pnpm over npm for package management
- DB at db.example.com:5432, dev_user role
- ...
End of memory preface.
```

Default K = 5, max 50.

This is the "give the agent freshest context" mode — appropriate
for fast-moving projects where last week's context is stale.

### `top_k_relevant`

Like `top_k_recent` but ranked by semantic similarity instead of
recency. Uses the cwd's basename as the search query, so it
roughly translates to "memories most relevant to *this* project."

Default K = 5.

This is the "give the agent the most useful context" mode —
appropriate for projects with deep memory histories where you
want the agent to surface knowledge regardless of when it was
captured.

### `manual_only`

Don't auto-inject at spawn. Operator triggers injection via UI
button or API (Phase v1.1).

Use when:
- The session toolbar's "Load context" button (Phase v1.1) is
  preferred over auto
- You want human-in-the-loop curation

### `hybrid`

Inject a single ultra-short summary line (top-1 most recent
project memory, ≤80 chars). Output looks like:

```
Project memory hint: User prefers pnpm over npm for package management
```

For tight context budgets where the multi-line banner is too
expensive but you still want some bridge-of-context.

### `on_keyword` (Phase v1.1)

Reserved. UI lets you save the profile but the actual hook into
the message stream is deferred to v1.1. Spawn-time injection is
disabled when this strategy is selected (functionally equivalent
to `none` until v1.1 ships).

## Per-session vs global

Like capture rules, profiles can be session-scoped or global. v1
UI manages the global default; per-session overrides are
DB-direct only for now.

## How injection actually wires

At spawn time the catalog adapter calls `Injector.Render()` and
prepends the result to the agent's system prompt:

- **Claude:** appended via `--append-system-prompt` arg
- **Codex:** written to `<CODEX_HOME>/AGENTS.md`
- **Gemini:** written to `<baseDir>/GEMINI.md`

Empty render → silent skip (no banner). Non-empty render → the
banner appears at the top of the agent's system prompt before
any other guidance.

## How it interacts with capture

Independent — capture writes memories, injection reads them.
A common deployment is:
- Capture rule: `after_messages` n=6, target=project
- Injection profile: `top_k_recent` k=5, global

Result: every 6 messages, durable facts from this conversation
flow into the project memory pool. When you start a new session
in the same project, the 5 most recent facts appear at the top
of the agent's system prompt — instant continuity.
