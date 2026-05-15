# Memory workers — overview

M25 lets you pick **which LLM** powers each of opendray's
memory-system touchpoints — independently per touchpoint.

The touchpoints (originally 4 in M25; grown to 7 across Phase A /
C / E):

| Touchpoint | Fires | What it does |
|---|---|---|
| **Gatekeeper** | every `memory_store` | Decides whether the agent's proposed fact is durable enough to store. |
| **Cleaner** | every 24h | LLM librarian — proposes keep/stale/duplicate verdicts on aged memories. |
| **Git activity** | every 24h | Turns 7 days of `git log` into a 2-3 paragraph narrative for the spawn banner. |
| **Transcript** | every session end | "What did the agent actually do this session" — 1-3 paragraph summary. |
| **Plan drift** *(Phase A)* | every session end | Looks at the new transcript summary + current plan; files a proposal if plan needs updating. |
| **Conflict detector** *(Phase C)* | every 24h | Cross-layer sweep — surfaces contradictions between facts / plan / goal / journal. |
| **Capture engine** *(Phase E)* | per capture-rule trigger | Per-trigger fact extraction from session transcripts; lands rows in `memories`. |

## Two worker types

| Worker | What it is | Latency | Cost | Quality |
|---|---|---|---|---|
| **Summarizer** | HTTP POST to your local LM Studio / OpenAI-compat endpoint | ~0.5-2s | free (local) | medium (3-13B) |
| **Agent** | Spawns `claude --print` or `gemini --print` headlessly | ~5-15s | Claude/Gemini quota | frontier-model |

Default deployments seed every touchpoint to **summarizer**.
Nothing changes until you flip a row.

## Why per-touchpoint config

The touchpoints have very different profiles. One global switch
would force a bad compromise:

- Gatekeeper runs hundreds of times a day. Even +5s/call makes
  `memory_store` feel broken. ⇒ summarizer only.
- Git activity summariser fires once a day, takes 60-150s anyway,
  and produces a banner every agent reads. Claude Opus here pays
  back the cost in better agent priming. ⇒ agent (Claude) makes
  sense.
- Cleaner / Conflict detector are somewhere in between — 24h batch
  but you'd run them manually sometimes. Either works.
- Transcript / Plan drift run after every session end (could be
  many per day). Latency is OK (background) but cost adds up.
- Capture engine fires multiple times per session if you set a
  short trigger (every 6 messages). Agent mode here gives the
  best fact quality if your CLI quota allows it.

The config table lets you pick per row, with per-row metrics so
you can validate the tradeoff afterwards.

## Where it lives

Open **Memory → Workers** (web sidebar) — or `/memory/workers`
directly. As of M-PF this page is the unified **Memory
configuration** landing with five sections in one place:

1. **Providers** — HTTP endpoint registry
2. **Workers** — per-task summarizer/agent routing (this section)
3. **Capture rules** — trigger config
4. **Injection profiles** — spawn-time strategy
5. **Token cost** — all-time call audit

Each Worker card (in section 2) shows:

- **Worker selector**: `Summarizer` ↔ `Agent` dropdown (gatekeeper
  is locked to Summarizer — see "Why gatekeeper stays put" below).
- **Provider selector**:
  - For summarizer: pick which `memory_summarizer_providers` row
    (the same dropdown you see on the Memory configuration page).
    Empty = registry default.
  - For agent: pick `claude` or `gemini`; for claude, pick which
    of your multi-account rows.
- **Enabled** checkbox: toggles the whole touchpoint off (degrades
  to no-LLM behaviour — metadata-only journal, raw-stats git
  activity, etc.).
- **Test** button: runs a synthetic ping ("reply with OK").
  Surfaces the round-trip latency + any auth / network error.
- **Save** button: persists. Effective on the next call — no
  restart.
- **Recent calls** expander: last 25 invocations with
  per-row timestamp, worker kind, duration in ms, success flag.

## Why gatekeeper stays put

The gatekeeper fires on every `memory_store` MCP call — that's
on the hot path between the agent saying "remember X" and the
agent's next prompt. Anything over 500ms feels broken.

Local summarizer endpoints (LM Studio with a 3B model) typically
do it in 100-300ms. Agent spawn round-trips are 5-15s. The
math doesn't work, so M25 doesn't offer the toggle for this row.

The row still exists in the config table — you can pin a specific
summarizer provider for the gatekeeper that's different from the
one your cleaner uses. Just not an agent.

## What to read next

- **02 — Picking a worker for each task** walks through the
  tradeoffs concretely with sample latency / cost numbers from
  a local setup.
- **03 — Verification & metrics** covers what to look for in
  the 24h rollup, when to trust the metrics, and how to roll
  back a bad switch.
