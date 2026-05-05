# Ambient Memory — providers

A **summarizer provider** is the LLM (or wrapping service) that
turns a transcript window into a JSON `{facts: [...]}` array. v1
ships five provider kinds covering both the local-first and
cloud-API paths.

## `ollama` — local

Most direct local-first option. Run `ollama pull qwen2.5:7b`
(or any chat model) and point opendray at the daemon.

```
Base URL:  http://localhost:11434
Model:     qwen2.5:7b
API key:   (none)
```

Speaks ollama's native `/api/chat` endpoint with
`format: "json"` for strict JSON output.

## `lmstudio` — local with GUI

LM Studio exposes an OpenAI-compatible HTTP server. Friendlier
GUI than ollama but otherwise interchangeable.

```
Base URL:  http://localhost:1234/v1
Model:     <model name from LM Studio's loaded list>
API key:   (none — LM Studio ignores)
```

Speaks chat completions with
`response_format: {"type": "json_object"}`.

## `anthropic` — Claude API

Use Haiku 4.5 for the cheapest / fastest extraction. The Sonnet
4.6 and Opus 4.7 entries in the cost table are there for
operators who want higher quality.

```
Base URL:  https://api.anthropic.com  (default; can override)
Model:     claude-haiku-4-5
API key:   sk-ant-…  (encrypted at rest with backup cipher)
```

Uses Anthropic's tool-use API
(`tool_choice = record_facts`) for guaranteed structured output.

## `openai` — OpenAI API + any compatible

Same OpenAICompatProvider that handles LM Studio — just
different defaults. Pricing baked in for `gpt-4o-mini`,
`gpt-4o`, `gpt-4.1`, `gpt-4.1-mini`, `gpt-4.1-nano`, `o3-mini`.

```
Base URL:  https://api.openai.com/v1  (default)
Model:     gpt-4o-mini
API key:   sk-…  (encrypted at rest)
```

Compatible with: Azure OpenAI (set base_url), Together, Groq,
Fireworks, vLLM, Anyscale, Novita, and most domestic-Chinese
gateways that re-export the OpenAI shape.

## `integration` — your local service

The "opendray CLI as summarizer" path. ANY service that speaks
the documented `/summarize` protocol can act as a summarizer
backend. You write the service in any language; opendray
forwards transcript chunks to it via reverse proxy.

Protocol (the integration must implement):

```
POST <base_url>/summarize
  body:    {"messages":[{"role","text","timestamp"}, ...],
            "scope": "session|project|global",
            "scope_key": "<cwd or session id or 'operator'>"}
  reply:   {"facts":[{"text","category","confidence"}, ...],
            "input_tokens": N,    (optional)
            "output_tokens": M}   (optional)
```

Setup:
1. Run a service somewhere (localhost, in-LAN, in your LXC) that
   POST handles `/summarize` per the protocol.
2. Register that service as an opendray integration with the
   `memory:summarize` scope.
3. Create a summarizer provider with `kind: integration`,
   referencing your integration's id.
4. Optional: provide an outbound bearer token (encrypted at rest)
   if your service requires auth.

This unlocks every imaginable backend: rule-based extractors,
self-hosted Llama clusters, internal LLM gateways, even
distillation services that hand off to a smarter model.

## `is_default` flag

Exactly one provider may be marked default. When a capture rule
doesn't pin a specific provider, the default is used. Marking a
new provider as default automatically clears the flag from any
other row (transactional).
