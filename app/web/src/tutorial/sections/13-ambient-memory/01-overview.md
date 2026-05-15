# Ambient Memory — overview

opendray's **ambient memory** subsystem solves the
"model-only-stores-when-told" problem: instead of waiting for the
user to say "remember X", a background goroutine polls every live
agent session every 10 seconds, sends new transcript messages to a
configurable LLM ("summarizer provider"), receives back a JSON
list of durable facts, dedupes against existing memories, and
writes the survivors with `source_kind='summarizer'`.

The result: when you `/clear` your context or jump to a different
agent, the project memory you've built up is already there — the
new agent finds it via `memory_search` (or sees a banner of recent
memories prepended to its system prompt, depending on your
injection profile).

## Three configurable pieces

```
┌─────────────────┐    ┌──────────────────┐    ┌──────────────────┐
│   Provider      │    │   Capture rule   │    │  Injection       │
│   (which LLM)   │ +  │   (when to fire) │ +  │  profile         │
│                 │    │                  │    │  (how to inject) │
└────────┬────────┘    └────────┬─────────┘    └────────┬─────────┘
         │                      │                       │
         ▼                      ▼                       ▼
   internal/memory/      internal/memory/         internal/memory/
    summarizer/           capture/                  injector/
```

- **Provider** — which LLM extracts facts. Today supports Anthropic,
  OpenAI, LM Studio, Ollama, a passthrough Integration kind (any
  service speaking the documented `/summarize` protocol works
  as a summarizer — zero-API-cost path), **and** the M-PE Agent
  route which spawns a headless `claude --print` / `gemini --print`
  for higher-quality fact extraction at the cost of CLI quota.
- **Capture rule** — when to fire. 4 trigger kinds: `after_messages`,
  `on_idle`, `k_chars`, `manual`.
- **Injection profile** — how (or whether) to prepend memories to
  the agent's system prompt at spawn. 5 strategies:
  `none`, `top_k_recent`, `top_k_relevant`, `manual_only`, `hybrid`
  (plus `on_keyword` reserved for a future release).

Configure all three under **Memory → Workers** (sidebar) — that
page, formerly just "Workers", was rebuilt as the single Memory
configuration landing in M-PF and now hosts Providers / Workers /
Capture rules / Injection profiles / Token cost together so related
knobs sit next to each other instead of split across two pages.

## What you DON'T need

- No env vars to enable — the subsystem is always wired. It just
  doesn't do anything until you create your first provider + rule.
- No external services unless you want to use the Anthropic /
  OpenAI providers. Ollama and LM Studio cover the local-first
  path entirely.
- No explicit "memorize this" prompts — the model doesn't know
  ambient capture is happening; it just keeps answering normally.

## What costs money

Anthropic (Haiku $1/$5 per MTok) and OpenAI (gpt-4o-mini
$0.15/$0.60 per MTok) summarize via paid APIs. Every call writes
a `memory_summarizer_calls` row with token counts and an
estimated USD figure aggregated in the **Token cost** panel.

Local providers (Ollama, LM Studio) and Integration providers
are priced as $0 — you own the hardware / external service.
