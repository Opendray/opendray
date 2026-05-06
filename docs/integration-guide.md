# Integration Guide

This guide is for developers building an **integration** — an external
application that talks to an opendray-v2 gateway over HTTP and
WebSockets, in any language.

If you're operating a deployment, see [`docs/operator-guide.md`](operator-guide.md).
If you're contributing to opendray itself, see [`CONTRIBUTING.md`](../CONTRIBUTING.md).

## What is an integration?

An integration is a separate process — your app — that registers with
opendray and consumes its capabilities:

- Create and drive PTY sessions backed by Claude Code, Codex, Gemini,
  or arbitrary shells
- Subscribe to event streams (session output, idle, ended, channel
  events, etc.)
- Be reverse-proxied through opendray for inbound traffic

opendray itself does not embed integration code. Every integration is
external; this keeps the gateway lean and language-agnostic.

The full design is in [`docs/design.md`](design.md) §7–§11 and
[`docs/adr/0006-integration-auth-and-paths.md`](adr/0006-integration-auth-and-paths.md).

## Quick walk-through

A 6-step concrete flow for a hypothetical Slack-style bot that posts
to Slack when a session ends:

1. **Operator registers your integration** with admin auth, picking a
   `route_prefix` and a list of scopes
2. opendray returns a **plaintext API key once** (not stored, not
   shown again — you save it securely)
3. **You expose `GET /health`** so opendray can probe you every 30s
4. **You subscribe to events** via WebSocket using the API key
5. **When `session.ended` fires**, your bot posts to Slack
6. **When the user replies in Slack**, you `POST /api/v1/sessions/{id}/input`
   to drive the conversation back into opendray

## Registration

Only an admin can register an integration. Endpoint:

```http
POST /api/v1/integrations
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "name":         "slack-bot",
  "base_url":     "http://slack-bot-server:3000",
  "route_prefix": "slack-bot",
  "scopes":       ["session:read", "session:input", "event:subscribe:session.*"],
  "version":      "1.0.0"
}
```

Required fields:

| Field | Notes |
|---|---|
| `name` | Human-readable label |
| `base_url` | Where opendray reverse-proxies inbound requests addressed to your `route_prefix`. Optional — pure consumers can omit it. |
| `route_prefix` | Unique slug. Reserved: `_events`, `_kinds`, `_internal`, `_*`. |
| `scopes` | Array of allow-listed capabilities (see below). |
| `version` | Free-form, your choice. opendray surfaces it in admin UI. |

Response (HTTP 201):

```json
{
  "id":      "int_abc123",
  "api_key": "odk_live_KJq8ne3...",   // ← shown once, save it
  "name":    "slack-bot",
  ...
}
```

The `api_key` is the **only** time the plaintext is visible. opendray
stores a bcrypt hash (cost 12) and discards the plaintext. Lose the
key and you'll need to rotate via:

```http
POST /api/v1/integrations/{id}/rotate-key
Authorization: Bearer <admin_token>
```

…which invalidates the old key immediately and returns a new one.

## Authentication

Every request your integration makes uses the API key as a bearer
token:

```http
GET /api/v1/sessions
Authorization: Bearer odk_live_KJq8ne3...
```

For WebSocket endpoints (browsers can't set custom headers on the
handshake), use the query-parameter form:

```
ws://opendray:8770/api/v1/integrations/_events?token=odk_live_KJq8ne3...&topics=session.*
```

opendray validates the bearer token by:
1. Trying admin first (in-memory token lookup)
2. Falling back to integration (bcrypt-compare against every
   integration's hash, then attaching the matched integration's scopes
   to the request principal)

Requests with no `Authorization` header / no `token` query param fail
with 401.

## Scopes

Scopes gate what your integration can do. Defined values:

| Scope | What it allows |
|---|---|
| `session:read` | List sessions, read buffers and metadata |
| `session:create` | Spawn new sessions |
| `session:input` | Send keyboard input to a session |
| `channel:send` | Post messages to channel adapters |
| `channel:receive` | Receive inbound messages from channel adapters |
| `provider:read` | List CLI providers + per-provider config |
| `event:subscribe:<topic>` | Subscribe to one event topic on the WS endpoint |

`event:subscribe:*` and `event:subscribe:session.*` work as wildcards
(prefix match).

**Today (M3)**: only `event:subscribe:<topic>` is enforced — every
other scope is _declared_ but every business endpoint accepts any
valid integration token. Per-route scope enforcement on session /
channel / provider endpoints is on the v1.1 roadmap.

Default scopes for a new integration:
`["session:read", "event:subscribe:session.*"]`.

## REST endpoints exposed for integrations

All under `/api/v1/`. Full method/route table below; the dual-auth
group accepts both admin and integration tokens.

### Admin-only

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/integrations` | Register |
| `GET` | `/integrations` | List |
| `GET` | `/integrations/{id}` | Detail |
| `PATCH` | `/integrations/{id}` | Update (`base_url`, `scopes`, `version`, `enabled`) |
| `DELETE` | `/integrations/{id}` | Deregister |
| `POST` | `/integrations/{id}/rotate-key` | Issue new API key |
| `GET` | `/integrations/_calls` | Call-log audit |

### Dual-auth (admin or integration token)

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/sessions` | Create a new PTY session |
| `GET` | `/sessions` | List sessions |
| `GET` | `/sessions/{id}` | Session detail |
| `DELETE` | `/sessions/{id}` | Terminate |
| `POST` | `/sessions/{id}/input` | Send keyboard input |
| `POST` | `/sessions/{id}/resize` | Resize PTY |
| `WS` | `/sessions/{id}/stream` | Bidirectional terminal stream |
| `GET` | `/sessions/{id}/buffer` | Replay the ring buffer |
| `GET` | `/providers` | List CLI providers |
| `PATCH` | `/providers/{id}/config` | Set per-provider config |
| `GET` | `/channels` | List channels |
| `WS` | `/integrations/_events` | Subscribe to event topics |

## WebSocket events

Subscribe by query string:

```
WS /api/v1/integrations/_events?token=odk_live_...&topics=session.output,session.ended
```

- `topics` is comma-separated; wildcards via trailing `.*` (e.g.
  `session.*`)
- The handler enforces `event:subscribe:<topic>` for each requested
  topic — admin tokens bypass the check

### Frame schema

Every event frame is a JSON object:

```json
{
  "topic": "session.output",
  "ts":    "2026-04-27T14:01:00.123Z",
  "data":  { ... topic-specific payload ... }
}
```

### Standard topics

| Topic prefix | Source |
|---|---|
| `session.*` | `output`, `idle`, `ended` from the PTY session manager |
| `channel.*` | `message_received`, `command_received`, `message_sent` from channels |
| `integration.*` | `registered`, `health_change`, `rotated` from the registry |

### Connection management

- Server pings every 20s; clients should respond with pong (handled
  automatically by most WS libraries)
- Per-message write timeout is 5s
- The connection drops on auth failure, scope violation, or topic
  parse error — reconnect with backoff

## Reverse proxy

If you set a `base_url` on registration, opendray exposes a passthrough:

```
ANY /api/v1/proxy/{your-prefix}/*
```

Example: with `route_prefix = "slack-bot"` and `base_url = "http://slack-bot-server:3000"`,
a request to:

```
GET http://opendray:8770/api/v1/proxy/slack-bot/api/v1/dogs/123
```

…is forwarded to:

```
GET http://slack-bot-server:3000/api/v1/dogs/123
```

opendray injects three headers:
- `X-OpenDray-Forwarded-For` — the original client IP
- `X-Integration-ID` — your integration's ID (so you know it's coming
  from opendray, not direct)
- `X-OpenDray-API: v1`

…and **strips** the inbound `Authorization` header before forwarding,
so admins can hit your integration through opendray without leaking
their token to your process.

If your integration is disabled or unhealthy, opendray returns HTTP
503 and never forwards the request.

## Health probe

opendray probes every registered integration's `GET /health` every
30 seconds. Expected response:

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "status":     "healthy",      // "healthy" | "degraded" | "unhealthy"
  "version":    "1.0.0",
  "busy_ratio": 0.1             // 0.0..1.0, optional
}
```

A non-200 response or a non-`healthy` status flips the integration's
health flag in opendray's registry; the next reverse-proxy request
will return 503 instead of forwarding.

## Worked example: Slack-style bot

```bash
# 1. Operator registers
curl -X POST http://opendray:8770/api/v1/integrations \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name":         "slack-bot",
    "base_url":     "http://slack-bot-server:3000",
    "route_prefix": "slack-bot",
    "scopes":       ["session:read", "session:input", "event:subscribe:session.*"],
    "version":      "1.0.0"
  }'
# → returns api_key, save it as $BOT_KEY

# 2. Bot subscribes to session events (using websocat / Python websockets / etc.)
websocat "ws://opendray:8770/api/v1/integrations/_events?token=$BOT_KEY&topics=session.output,session.ended"
# ← {"topic":"session.ended","ts":"...","data":{"session_id":"sess_42","exit_code":0}}

# 3. On session.ended, bot posts to Slack via Slack's own API
#    (this part is bot-internal, opendray doesn't see it)

# 4. User replies in Slack; bot routes back to opendray
curl -X POST "http://opendray:8770/api/v1/sessions/sess_42/input" \
  -H "Authorization: Bearer $BOT_KEY" \
  -H "Content-Type: application/json" \
  -d '{"text":"@bot: continue from where you left off"}'

# 5. Operator can hit the bot's status page through opendray's proxy
curl "http://opendray:8770/api/v1/proxy/slack-bot/status" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

## Reference implementation

A working TypeScript demo client lives at
[`examples/integrations/demo-client/`](../examples/integrations/demo-client).
It shows:

- First-run registration flow + secure local API-key storage (mode 0600)
- Key rotation recovery (auto-trigger on 401 → admin rotates → new key
  persisted)
- Session creation, input, event subscription
- The full 9-step lifecycle from `register` to `unregister`

Files to read in order:
- `examples/integrations/demo-client/src/index.ts` — top-level flow
- `examples/integrations/demo-client/src/client.ts` — `OpendrayClient`
  REST + WS abstraction (a model for your own SDK)
- `examples/integrations/demo-client/src/state.ts` — local key persistence

The demo deliberately avoids any framework dependency — it's pure
node `fetch` + the `ws` library. Port to your language of choice.

## See also

- [`docs/design.md`](design.md) §7–§12 — integration design rationale
- [`docs/adr/0006-integration-auth-and-paths.md`](adr/0006-integration-auth-and-paths.md) —
  dual-auth model, scope deferral, API-key format decisions
- [`docs/adr/0010-integration-call-log.md`](adr/0010-integration-call-log.md) —
  call-log audit trail
- [`docs/operator-guide.md`](operator-guide.md) — running opendray
- [`SECURITY.md`](../SECURITY.md) — vulnerability disclosure
