# OpenDray v2 — Design Document

**Status:** Draft (north-star, replaces all v1 architecture docs)
**Author:** Linivek
**Created:** 2026-04-27
**Module path:** `github.com/opendray/opendray-v2` (will rename to `opendray` at v1.0)

---

## Table of contents

0. [Why a rewrite](#0-why-a-rewrite)
1. [Mission & vision](#1-mission--vision)
2. [Users & scenarios](#2-users--scenarios)
3. [Value propositions](#3-value-propositions)
4. [Anti-goals](#4-anti-goals)
5. [Tech stack decisions](#5-tech-stack-decisions)
6. [System architecture](#6-system-architecture)
7. [Domain model](#7-domain-model)
8. [Subsystems](#8-subsystems)
9. [API contracts (sketch)](#9-api-contracts-sketch)
10. [Data model](#10-data-model)
11. [Integration protocol](#11-integration-protocol)
12. [Auth & security](#12-auth--security)
13. [Deployment topology](#13-deployment-topology)
14. [Project structure](#14-project-structure)
15. [Coding conventions](#15-coding-conventions)
16. [Testing strategy](#16-testing-strategy)
17. [Migration from v1](#17-migration-from-v1)
18. [Roadmap & milestones](#18-roadmap--milestones)
19. [Open questions](#19-open-questions)

---

## 0. Why a rewrite

This is a rewrite, not a refactor. The v1 codebase (`/Claude_Workspace/opendray`) had a sound product surface but accumulated architectural debt that surgical extraction cannot resolve cleanly. This section documents the reasons so future-us doesn't second-guess the decision.

### What v1 got right (we will reuse, not rewrite)
- **PTY session lifecycle** (creack/pty + ring buffer + resume).
- **Manifest-driven UI contributions** for CLI providers (declarative `manifest.json` → Flutter renders activity bar / views / panels).
- **Flutter mobile app** UX investments (terminal rendering, copy mode, quick-keys strip, in-app browser, image upload).
- **Vue web frontend** session control UX.
- **PostgreSQL data layer** patterns.
- **Cloudflare Tunnel** deploy topology.
- **Channel concept** (Telegram bot for outbound notifications + reply ingestion).

### What v1 got wrong (we will not carry over)
1. **Two conflated "plugin" concepts.** The word "plugin" refers simultaneously to (A) UI extension points consumed by Flutter and (B) what would-be third-party extensions ought to be. v1 over-invested in a generic plugin SDK while only (A) was ever used.
2. **~28k lines of plugin infrastructure for use cases that never materialised.** Audit: 16/16 builtins are `form: declarative` — zero used the bridge protocol (12k lines), zero used the host sidecar supervisor (~1.5k lines), the marketplace catalog is empty, no external plugins were ever installed. The bridge / install / market / host packages are dead weight.
3. **No "integrations" system.** Other apps in the ecosystem (PetTracker, future projects) had no clean way to consume OpenDray's AI capabilities. v1 plugins point inward (extending OpenDray); we need a system that points outward (exposing OpenDray to peers).
4. **Vendor modules in `gateway/`.** `gateway/telegram`, `gateway/sourcecontrol`, `gateway/mcp`, `gateway/claude_accounts` couple feature code to the gateway router. Adding a new channel meant editing `gateway/server.go`. Channel/forge abstractions exist but only the manifest side; the runtime side is hardcoded.
5. **Marketplace, signing, consent, capability-gate** were built for a third-party plugin economy that doesn't exist for a single-maintainer self-hosted tool at the v0.x stage.

### Goal of v2
Build the smallest, cleanest codebase that delivers the actual product:

- Mobile / web remote control of multiple AI agent CLIs.
- A first-class **integration system** (RCC-style reverse-proxy, but with auth, scopes, event push, and a real SDK) that lets the user's other apps consume OpenDray's AI capabilities without their code entering this repo.
- A first-class **channel hub** abstraction (telegram + slack + imessage + others share one contract).
- Aggressive deletion of features that v1 carried but never used.

If at any point during the rewrite a v2 module exceeds 1500 lines, stop and ask whether the design has drifted. v1's bridge package was 12k lines.

---

## 1. Mission & vision

### Mission
**Let an AI agent CLI follow me wherever I am, and let any app I build use it.**

### Vision (3-year horizon)
A self-hosted personal AI infrastructure: one Mac/server runs the Claude / Codex / Gemini CLIs; OpenDray exposes them through a stable HTTP/WS API. Phone, web, and the user's other applications all consume that API. Subscription cost (Claude Pro $20/mo) replaces per-token API billing for the entire personal app ecosystem.

### What "OpenDray" is, in one sentence
**A multiplexer + integration gateway for AI agent CLIs.**

---

## 2. Users & scenarios

### Primary user: Linivek (the maintainer)

- "I'm at the airport, my agent at home is mid-task. I want to check progress and reply on my phone."
- "I have three projects open simultaneously. I want claude in project A and codex in project B running in parallel, both visible from one app."
- "I'm building a new web app. Instead of paying for API tokens, I want my web app to ask my OpenDray to run a Claude session for it."
- "Telegram pings me 'agent finished'. I tap and reply 'looks good, commit and push'."

### Secondary user: Linivek's own apps (integration consumers)

- PetTracker, MaterialScout, ShopOnlineCms, … register as integrations; call OpenDray's API for AI inference, file ops, vision tasks.
- Each integration has its own scoped API key, quota, and audit trail.

### Tertiary user: other self-hosted operators (future)

- Same archetype as primary, running their own OpenDray instance.
- Treat as eventual; do not over-engineer multi-tenancy.

### Explicit non-users
- Enterprise teams.
- Public SaaS subscribers.
- Third-party plugin marketplace developers.

---

## 3. Value propositions

In strict priority order. Decisions that trade higher VP for lower VP go to the higher.

| # | Value | Concrete win |
|---|---|---|
| 1 | **Ubiquitous CLI control** | Mobile + web work as well as desktop terminal |
| 2 | **Subscription cost arbitrage** | $20/mo Claude Pro replaces per-token API billing |
| 3 | **Multi-CLI orchestration** | Parallel claude / codex / gemini, route + compare |
| 4 | **Integration ecosystem** | Other apps register and consume AI without polluting OpenDray |
| 5 | **Async / channel-based workflow** | Telegram / Slack / iMessage (bi-directional) |
| 6 | **Reliable remote dev loop** | File browser, source control, in-app browser preview |

---

## 4. Anti-goals

The system will explicitly **not** do these:

- Direct LLM API calls (Anthropic / OpenAI / Gemini). The CLI does that work.
- Generic agent framework (the CLI is the agent).
- Plugin marketplace, store, signing infrastructure.
- Webview / host sidecar plugin runtimes.
- Multi-tenant SaaS, billing, payments.
- Per-user permission systems beyond admin / integration scopes.
- iOS-side native app store distribution beyond TestFlight (for now).
- Any feature that requires opening a public bug tracker or support channel.

If at planning time a feature pulls toward an anti-goal, drop the feature.

---

## 5. Tech stack decisions

### Locked decisions

| Layer | Tech | Rationale |
|---|---|---|
| **Backend / gateway** | Go 1.25+ | Single static binary, excellent concurrency for PTY + WebSocket fanout, ecosystem maturity (chi, pgx, gorilla/websocket, creack/pty). Hot-reload trade-off accepted: integration system runs external processes that hot-reload independently. |
| **Database** | PostgreSQL 15+ | pgx/v5 stable; JSON columns for opaque metadata; pgvector for memory embeddings. |
| **Mobile** | Flutter 3 + Riverpod | v1 investment, proven UX, single codebase iOS+Android. |
| **Web** | Vue 3 + Vite | TBD — see §19, may consolidate to Flutter Web. |
| **Internal RPC / event bus** | In-process Go channels | Keep simple; do not adopt NATS/Kafka without a real reason. |
| **Integration protocol** | HTTP/JSON + WebSocket events | Language-agnostic; integration code lives in any language the consumer prefers. |
| **SDK languages** | Go (first), TypeScript (second) | Cover server-side and Node/Bun/Deno integrations. Python deferred to V1.2. |
| **Deployment** | Single binary on Mac launchd or LXC | Embed all assets via `go:embed`. No Docker required. |
| **External access** | Cloudflare Tunnel | Already configured; no public IP / port-forwarding. |

### Explicitly rejected

| Stack | Why rejected |
|---|---|
| Elixir / Erlang full rewrite | Hot reload sounds appealing but real demand is for external-process hot reload, which Go solves. Cost: 6-12 months. |
| Node.js / TypeScript backend | Weaker concurrency for PTY workloads, harder single-binary deploy. |
| Rust | Hot reload worse than Go; rewrite cost outsized. |
| Embedded scripting layer (Lua / Starlark / QuickJS) at v1.0 | Premature optimisation. Add to v1.2+ if a concrete need surfaces. |
| GraphQL | REST + a small set of WS event topics is simpler for the integration audience. |

---

## 6. System architecture

```
┌────────────────────────────────────────────────────────────────────┐
│                          CLIENTS                                    │
│  Flutter (iOS/Android)   ·   Vue Web   ·   Integration Apps         │
└───────┬─────────────────────┬──────────────────────┬───────────────┘
        │ HTTPS / WSS          │ HTTPS / WSS         │ HTTPS / WSS
        ▼                      ▼                      ▼
┌────────────────────────────────────────────────────────────────────┐
│                      OPENDRAY GATEWAY (Go)                          │
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  HTTP Router (chi) + Middleware                              │  │
│  │  · auth   · rate-limit  · audit log  · CORS  · request id    │  │
│  └─────┬───────────┬─────────────┬──────────────┬───────────────┘  │
│        │           │             │              │                   │
│  ┌─────▼─────┐ ┌───▼───────┐ ┌───▼────────┐ ┌───▼─────────────┐   │
│  │  Session  │ │Integration│ │  Channel   │ │  Catalog +      │   │
│  │  Manager  │ │  Registry │ │    Hub     │ │  Workbench API  │   │
│  │ (PTY)     │ │ (proxy +  │ │ (telegram, │ │ (CLI provider   │   │
│  │           │ │  events)  │ │  slack…)   │ │  manifests)     │   │
│  └─────┬─────┘ └─────┬─────┘ └─────┬──────┘ └─────┬───────────┘   │
│        │             │             │               │                │
│        └─────────────┼─────────────┴───────────────┘                │
│                      │                                              │
│             ┌────────▼────────┐                                     │
│             │   Event Bus     │  (in-process pub/sub)               │
│             │  session.*      │                                     │
│             │  channel.*      │                                     │
│             │  integration.*  │                                     │
│             └────────┬────────┘                                     │
│                      │                                              │
│             ┌────────▼────────┐                                     │
│             │  Auth & Quota   │  api keys, scopes, counters         │
│             └────────┬────────┘                                     │
└──────────────────────┼──────────────────────────────────────────────┘
                       │
              ┌────────▼────────────────────┐
              │   PostgreSQL                │
              │   sessions, integrations,   │
              │   channels, audit, kv       │
              └─────────────────────────────┘

              ┌─────────────────────────────┐
              │   Local OS                  │
              │   AI CLI binaries           │
              │   (claude, codex, gemini…)  │
              └─────────────────────────────┘
```

Ordering principles:
- **HTTP router** is dumb — it routes to subsystems and never holds business state.
- **Subsystems** own their state and communicate via the event bus.
- **PostgreSQL** is the only durable store; subsystems must not invent shadow caches that diverge.
- **Auth** is middleware-level + subsystem-enforced (defence in depth).

---

## 7. Domain model

### Entities

**Session**
- `id`, `name`, `cwd`, `provider_id`, `cli_args[]`, `state` (pending/running/idle/ended), `started_at`, `ended_at`, `pid`.
- Owns a PTY, a ring buffer of recent output, and a list of subscribed clients.

**Provider** (CLI provider catalog entry)
- `id`, `display_name`, `kind` (cli / shell), `manifest`, `enabled`, `config` (per-instance values for the manifest's config schema).
- This is the v1 "builtin plugin" concept, simplified — purely a manifest + config.

**Integration**
- `id`, `name`, `base_url`, `route_prefix`, `api_key_hash`, `scopes[]`, `version`, `enabled`, `health_status`, `health_last_seen`, `created_at`.
- Represents an external app that registered to consume OpenDray.

**Channel**
- `id`, `kind` (telegram / slack / imessage / ...), `config` (kind-specific), `enabled`.
- One row per configured channel instance (e.g. one Telegram bot, one Slack workspace).

**ChannelMessage**
- `id`, `channel_id`, `direction` (inbound/outbound), `conversation_id`, `author`, `text`, `metadata`, `ts`.
- Materialised history; used for resume + audit.

**Event**
- Transient (not persisted by default): `kind`, `subject_id`, `payload`, `ts`.
- Some classes (audit, security) get persisted.

### Relationships

```
Session ───many──── Provider             (a session is launched against a provider)
Session ───many──── ChannelMessage       (a session can be steered via channel replies)
Channel ───many──── ChannelMessage
Integration ──can subscribe to── Event   (event bus, scope-gated)
Integration ──can call── Session API     (proxy + scope-gated)
```

---

## 8. Subsystems

### 8.1 Session Manager

**Responsibility:** lifecycle of PTY processes for AI CLIs.

**Owns:**
- PTY pairs (creack/pty).
- Ring buffer (configurable size, default 1MB per session).
- Goroutines: stdin pump, stdout pump, idle detector, exit detector.
- Resize handling.

**API surface (exposed by gateway):**
- `POST /api/v1/sessions` — create
- `GET /api/v1/sessions` — list
- `GET /api/v1/sessions/{id}` — detail
- `DELETE /api/v1/sessions/{id}` — terminate
- `POST /api/v1/sessions/{id}/input` — REST input (no WS needed)
- `POST /api/v1/sessions/{id}/resize` — resize PTY
- `WS  /api/v1/sessions/{id}/stream` — bidirectional terminal stream
- `GET /api/v1/sessions/{id}/buffer` — replay ring buffer

**Invariants:**
- A session terminating must publish `session.ended` exactly once.
- Resume reconnects to existing PTY without restarting CLI.
- Idle threshold configurable; idle event fires once per idle window.

### 8.2 Integration Registry (★ new in v2)

**Responsibility:** external app onboarding, reverse proxy, event push, scoped auth.

**Owns:**
- Registration table.
- API key issuance (issue once, store hash).
- Health checker goroutine (30s default).
- Reverse proxy: `/api/v1/integrations/{prefix}/*` → `{base_url}/*`.
- Event subscription WS: `/api/v1/integrations/events`.

**API surface:**
- `POST /api/v1/integrations` — register; returns API key once
- `GET /api/v1/integrations` — admin only
- `PATCH /api/v1/integrations/{id}` — update config
- `DELETE /api/v1/integrations/{id}` — deregister
- `POST /api/v1/integrations/{id}/rotate-key` — rotate API key
- `GET /api/v1/integrations/{id}/stats` — request count, error rate, p95 latency
- `WS  /api/v1/integrations/events?topics=...` — event stream (auth: API key)

**Reverse-proxy rules:**
- Strip `/api/v1/integrations/{prefix}` prefix.
- Inject `X-OpenDray-Forwarded-For`, `X-Integration-ID`, `X-OpenDray-API-Version`.
- Do not forward OpenDray's own `Authorization` header.
- Unhealthy integration → HTTP 503 immediately, no proxy attempt.

**Differences from RCC's plugin-registry (improvements):**
- API key per integration (RCC has none).
- Capability declaration (`scopes[]`) rather than opaque proxy.
- Event push WS (RCC requires polling).
- Structured health (`{status, version, busy_ratio}` instead of bare 200 OK).
- Versioned API contract via `X-OpenDray-API` header.
- Per-integration quota & observability stats.

### 8.3 Channel Hub

**Responsibility:** unified abstraction over messaging services with rich-content support, slash commands, and external (non-Go) adapters.

**Owns:**
- A small core `Channel` interface plus optional **capability interfaces** (`CardSender`, `ButtonSender`, `MessageUpdater`, `TypingIndicator`, `ImageSender`, `FileSender`, `ReplyCapable`).
- A `Card` model (Markdown / Divider / Buttons / ListItem / Select / Note) with text-fallback rendering.
- A `CommandRegistry` for slash commands (built-in `/help`, `/notify`, `/status`; app code wires session-aware ones).
- Hub: lifecycle, persistence, event-bus dispatch, command routing, mute filter.
- The `bridge` channel kind: a WebSocket protocol that lets external (Python/Node/...) adapters register at runtime — see ADR 0011 and `docs/bridge-protocol.md`.

**API surface:**
- `GET /api/v1/channels` — list configured (now reports `capabilities[]` and `muted`)
- `GET /api/v1/channels/_kinds` — available kinds (currently `telegram`, `bridge`)
- `POST /api/v1/channels` — configure new
- `PATCH /api/v1/channels/{id}` — update config / enabled
- `DELETE /api/v1/channels/{id}`
- `POST /api/v1/channels/{id}/test` — send "OpenDray channel test ✓"
- `GET /api/v1/channels/bridge/ws` — token-authenticated adapter WS (no admin auth)
- Internal: subscribes to `session.idle`, `session.ended`; emits `channel.message_received`, `channel.command_received`, `channel.command_unknown`, `channel.message_sent`.

**Channel interface (Go):**
```go
// Mandatory baseline.
type Channel interface {
    Kind() string
    ID() string
    Start(ctx context.Context, inbound InboundFunc) error
    Stop(ctx context.Context) error
    Send(ctx context.Context, msg ChannelMessage) error
}

// Optional capabilities (any subset).
type CardSender interface { SendCard(ctx, msg, *Card) error }
type ButtonSender interface { SendWithButtons(ctx, msg, [][]ButtonOption) error }
type MessageUpdater interface { UpdateMessage(ctx, msg, handle, text) error }
type TypingIndicator interface { StartTyping(ctx, msg) (stop func()) }
type ImageSender interface { SendImage(ctx, msg, ImageAttachment) error }
type FileSender interface { SendFile(ctx, msg, FileAttachment) error }
type ReplyCapable interface { SupportsReply() bool }
```

Capability methods that the underlying connector cannot fulfil should return
`channel.ErrNotSupported` so the Hub falls back to text rendering.
`ChannelMessage.ReplyCtx` is opaque to the Hub; it threads platform-specific
routing (Telegram chat+message id, Slack thread_ts, bridge adapter handle, ...)
across the inbound→outbound boundary.

Implementations live in `internal/channel/{telegram,bridge}/`. New built-in
kinds register a factory from `init()` and are wired into `app/` via blank
import. External adapters use the bridge protocol — no Go recompile required.

### 8.4 Catalog (CLI providers)

**Responsibility:** the existing v1 builtin manifest concept, simplified.

**Owns:**
- Embedded provider manifests (`go:embed` from `internal/catalog/builtin/`).
- Per-provider user config (config schema → config form values).
- Enable/disable state.

**API surface:**
- `GET /api/v1/providers` — list
- `GET /api/v1/providers/{id}` — detail (manifest + current config)
- `PATCH /api/v1/providers/{id}/config` — set config
- `PATCH /api/v1/providers/{id}/toggle` — enable/disable

**Removed from v1:**
- No install / uninstall flow (providers are bundled).
- No marketplace.
- No signing.
- No webview / host runtime forms.
- No bridge protocol.
- Manifests are read-only data; the user customises only `config`.

### 8.5 Event Bus

**Responsibility:** in-process pub/sub.

**Implementation:**
- Plain Go channels under a `Hub` struct.
- Topics namespaced: `session.*`, `channel.*`, `integration.*`.
- Subscribers are weak: a slow subscriber is dropped (with log), bus never blocks publishers.
- No persistence by default; an audit subscriber writes selected topics to DB.

### 8.6 Auth & Quota

**Responsibility:** identity, scopes, rate limits.

**Owns:**
- Admin auth (Basic auth or token, simple — single user).
- Integration auth (API key → integration row → scopes).
- Quota counters (in-memory ring buffer + periodic DB flush for persistent counters).

**Scopes:**
- `session:read`, `session:create`, `session:input`
- `channel:send`, `channel:receive`
- `event:subscribe:<topic>`
- `provider:read`

**Default scopes for a new integration:** `session:read`, `event:subscribe:session.*`. Anything else requires admin approval at registration time.

---

## 9. API contracts (sketch)

Full OpenAPI spec lives at `/api/v1/openapi.json` (generated). Key endpoints below.

### Health
- `GET /api/v1/health` → `{status, version, uptime_s, db_ok}`

### Auth
- `POST /api/v1/auth/login` → admin session cookie
- `POST /api/v1/auth/logout`

### Integration registration

```
POST /api/v1/integrations
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "name": "PetTracker",
  "base_url": "http://192.168.3.42:8080",
  "route_prefix": "pet-tracker",
  "scopes": ["session:create", "session:input", "event:subscribe:session.*"],
  "version": "0.1.0"
}

→ 201 Created
{
  "id": "int_abc123",
  "api_key": "odk_live_...",   // shown once, then hashed
  "name": "PetTracker",
  "base_url": "http://192.168.3.42:8080",
  "route_prefix": "pet-tracker",
  "scopes": [...],
  "health_status": "unknown",
  "created_at": "2026-04-27T14:00:00Z"
}
```

### Integration calling OpenDray (consumer side)

```
POST /api/v1/sessions
Authorization: Bearer odk_live_...
X-OpenDray-API: v1

{
  "provider_id": "claude",
  "cwd": "/projects/pet-tracker",
  "args": ["--continue"]
}

→ 201 Created
{ "id": "ses_...", "ws_url": "wss://.../api/v1/sessions/ses_.../stream" }
```

### Event subscription

```
WS /api/v1/integrations/events?topics=session.output,session.ended
Authorization: Bearer odk_live_...

← {"topic":"session.output", "session_id":"ses_...", "data":"..."}
← {"topic":"session.ended", "session_id":"ses_...", "exit_code":0, "reason":"normal"}
```

### Reverse proxy to integration

```
GET /api/v1/integrations/pet-tracker/api/v1/dogs/123
→ proxied to http://192.168.3.42:8080/api/v1/dogs/123
   with X-Integration-ID, X-OpenDray-Forwarded-For injected
```

---

## 10. Data model

PostgreSQL schema (sketch — refined during Phase 1).

```sql
-- Providers (CLI catalog instance + user config)
CREATE TABLE providers (
  id            TEXT PRIMARY KEY,
  manifest_hash TEXT NOT NULL,
  config        JSONB NOT NULL DEFAULT '{}',
  enabled       BOOLEAN NOT NULL DEFAULT true,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Sessions
CREATE TABLE sessions (
  id            TEXT PRIMARY KEY,
  name          TEXT,
  provider_id   TEXT NOT NULL REFERENCES providers(id),
  cwd           TEXT NOT NULL,
  args          JSONB NOT NULL DEFAULT '[]',
  state         TEXT NOT NULL,          -- pending/running/idle/ended
  pid           INT,
  started_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ended_at      TIMESTAMPTZ,
  exit_code     INT
);
CREATE INDEX sessions_state_idx ON sessions(state);

-- Integrations (★ new)
CREATE TABLE integrations (
  id              TEXT PRIMARY KEY,    -- "int_<random>"
  name            TEXT NOT NULL UNIQUE,
  base_url        TEXT NOT NULL,
  route_prefix    TEXT NOT NULL UNIQUE,
  api_key_hash    TEXT NOT NULL,       -- bcrypt
  scopes          JSONB NOT NULL DEFAULT '[]',
  version         TEXT,
  enabled         BOOLEAN NOT NULL DEFAULT true,
  health_status   TEXT NOT NULL DEFAULT 'unknown', -- healthy/degraded/unhealthy/unknown
  health_payload  JSONB,
  health_last_seen TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  rotated_at      TIMESTAMPTZ
);

-- Channels
CREATE TABLE channels (
  id          TEXT PRIMARY KEY,
  kind        TEXT NOT NULL,            -- telegram/slack/imessage
  config      JSONB NOT NULL,           -- kind-specific (e.g. bot token ref)
  enabled     BOOLEAN NOT NULL DEFAULT true,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Channel message history
CREATE TABLE channel_messages (
  id            BIGSERIAL PRIMARY KEY,
  channel_id    TEXT NOT NULL REFERENCES channels(id),
  direction     TEXT NOT NULL,          -- inbound/outbound
  conversation_id TEXT NOT NULL,
  session_id    TEXT,                   -- nullable; set when message routed to a session
  author        TEXT,
  text          TEXT NOT NULL,
  metadata      JSONB,
  ts            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX channel_messages_conv_ts_idx ON channel_messages(conversation_id, ts DESC);

-- Audit log (selected lifecycle events)
CREATE TABLE audit_log (
  id          BIGSERIAL PRIMARY KEY,
  ts          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  actor_kind  TEXT NOT NULL,            -- admin/integration/system
  actor_id    TEXT,
  action      TEXT NOT NULL,
  subject_kind TEXT,
  subject_id  TEXT,
  metadata    JSONB
);

-- Integration call log — per-call API audit, separate from audit_log
-- because traffic volume is potentially 1000x and the schema is
-- different. See ADR 0010.
CREATE TABLE integration_call_log (
  id              BIGSERIAL PRIMARY KEY,
  ts              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  integration_id  TEXT NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
  direction       TEXT NOT NULL,        -- inbound/outbound
  method          TEXT NOT NULL,
  path            TEXT NOT NULL,
  status_code     INT  NOT NULL,
  duration_ms     INT  NOT NULL,
  bytes_written   BIGINT,
  request_id      TEXT,
  resource_kind   TEXT,
  resource_id     TEXT
);
```

Migrations live in `internal/store/migrations/` and are embedded with `go:embed`. A library like `goose` or hand-rolled is acceptable; do not adopt an ORM.

---

## 11. Integration protocol

### Goals
- An external app can register, call OpenDray, and receive events with **no OpenDray code changes** on either side.
- Any HTTP-capable language works.
- Events arrive over WebSocket, not polling.
- Integration health is observable.

### Lifecycle

```
┌──────────────┐                         ┌──────────────┐
│  Integration │                         │   OpenDray   │
└──────┬───────┘                         └──────┬───────┘
       │                                        │
       │  1. operator registers (admin REST)    │
       │ ◀────────── api_key (once) ────────────│
       │                                        │
       │  2. provide /health endpoint           │
       │ ◀────── GET /health (every 30s) ───────│
       │                                        │
       │  3. call OpenDray API with api_key     │
       │ ──────────POST /sessions────────────▶  │
       │ ◀─────── { session_id, ws_url } ───────│
       │                                        │
       │  4. subscribe to events                │
       │ ───── WS /integrations/events ───────▶ │
       │ ◀──────── stream of events ────────────│
       │                                        │
       │  5. (optional) be reverse-proxied      │
       │ ◀─── GET /integrations/{prefix}/foo ───│
       │ ─────────── response ────────────────▶ │
```

### Health endpoint contract

Integration provides:
```
GET /health
→ 200 OK
{
  "status": "healthy" | "degraded" | "unhealthy",
  "version": "0.1.0",
  "busy_ratio": 0.42,           // 0..1, optional
  "queue_depth": 7              // optional
}
```

Non-2xx OR `status: unhealthy` for two consecutive checks → mark unhealthy → reverse proxy returns 503 + integration's WS subscription throttled.

### Event payload schema

```json
{
  "topic": "session.output",
  "ts": "2026-04-27T14:01:00.123Z",
  "id": "evt_...",
  "subject": { "kind": "session", "id": "ses_..." },
  "data": { ... }
}
```

Topics are dot-namespaced; integrations subscribe with prefix matching (`session.*` matches all session events).

### SDK obligations

Both Go and TS SDKs must:
- Handle registration + key persistence.
- Auto-retry on 502/503 with exponential backoff.
- Reconnect WS on disconnect, replay subscriptions.
- Surface health state to the consumer for UI.

---

## 12. Auth & security

### Identity tiers
1. **Admin** — single human (you). Authenticated via Basic auth or signed cookie, admin-only endpoints.
2. **Integration** — bearer API key (`odk_live_…`) issued at registration; scopes attached to the key.
3. **Session client** — ephemeral WS connection authenticated by an admin or integration token; scoped to one session.

### Threat model (acknowledged)
- LAN attacker: blocked by Cloudflare Tunnel + admin auth.
- Compromised integration host: blackholes events, hits proxy 503; should not affect other integrations or the admin path.
- Stolen API key: rotate via `POST /integrations/{id}/rotate-key`; old key invalidated immediately.

### Security non-goals
- Defending against the admin's own machine being compromised (out of scope; LXC/Mac + filesystem ACL is the layer).
- Audit-grade compliance.
- HSM-backed key storage (the API key hash in PG is acceptable).

### Logging / audit
Two separate tables, by design:

- **`audit_log`** — *lifecycle* events. Every integration registration, scope grant, key rotation, and admin action goes here. PII / message bodies do **not** go to audit by default. Topic allowlist enforced in `internal/audit/sink.go`.
- **`integration_call_log`** — *per-call* API audit for both inbound (third-party → opendray) and outbound (admin → proxy → integration) traffic attributable to an integration. Admin-attributed direct API calls are intentionally **not** logged here. See ADR 0010 for the full design and the deferred polish list.

---

## 13. Deployment topology

### Modes

**Mode A — Local single-binary (default)**
- `opendray serve --config /etc/opendray/config.toml`
- Mac launchd plist or LXC systemd unit.
- All assets embedded.

**Mode B — Docker Compose (optional)**
- For users who prefer container deploy.
- Single image; Postgres separate.

### Recommended layout
- `opendray` binary on Mac (PTY needs a real TTY).
- PostgreSQL 15+ reachable from the binary's host.
- Cloudflare Tunnel routes `opendray.<your-domain>` to the local port.
- LXC option deferred until a non-Mac deployment is actually needed.

### Configuration
`config.toml` contains:
- Listen address.
- DB URL.
- Admin credentials (or env var ref).
- Default channel config refs (which use Vaultwarden via env per CLAUDE.md global rules).
- Provider catalog path / discovery dir.

No runtime UI for editing config (admin restarts to apply).

---

## 14. Project structure

```
opendray-v2/
├── README.md
├── LICENSE                           (Apache 2.0, same as v1)
├── go.mod
├── go.sum
├── .gitignore
├── docs/
│   ├── design.md                     (this doc — the SSOT)
│   ├── adr/                          (Architecture Decision Records)
│   ├── api.md                        (REST + WS reference, generated)
│   ├── integration-guide.md          (for integration authors)
│   └── operator-guide.md             (deploy + config + ops)
├── cmd/
│   └── opendray/
│       └── main.go                   (entry point: parse flags → run)
├── internal/
│   ├── app/                          (composition root: wires subsystems)
│   ├── gateway/                      (HTTP router + middleware)
│   ├── session/                      (PTY manager)
│   ├── integration/                  (registry + proxy + events WS)
│   ├── channel/                      (hub + telegram/slack/imessage)
│   ├── catalog/                      (CLI provider manifests + config)
│   │   └── builtin/                  (embedded manifests: claude, codex, gemini, ...)
│   ├── eventbus/                     (in-process pub/sub)
│   ├── auth/                         (api keys, scopes, rate limit)
│   ├── store/                        (postgres data layer + migrations)
│   ├── audit/                        (audit log writer)
│   └── version/                      (build-time version info)
├── pkg/
│   └── opendrayclient/               (public Go SDK for integration authors)
├── sdk-ts/                           (public TS SDK; npm-publishable)
├── app/
│   ├── mobile/                       (Flutter; lifted from v1, modernised)
│   └── web/                          (Vue or Flutter Web; see §19)
├── deploy/
│   ├── launchd/
│   └── lxc/
├── test/
│   ├── integration/                  (real PG + real opendray binary)
│   └── e2e/                          (mobile/web flow tests)
└── scripts/
    ├── dev.sh                        (run with hot rebuild via `air` or similar)
    ├── seed.sh                       (DB seed)
    └── build_release.sh              (matches CLAUDE.md mobile-release convention)
```

**Discipline rules:**
- No file in `internal/` may exceed 500 lines without justification in a comment.
- No package may have more than 10 `.go` files; split before crossing.
- Top-level `cmd/opendray/main.go` ≤ 100 lines. All wiring in `internal/app`.
- `internal/gateway` is pure HTTP — no business logic.

---

## 15. Coding conventions

- **Error handling**: explicit. No panics outside `main` startup. Errors carry context via `fmt.Errorf("op: %w", err)`.
- **Logging**: `log/slog`, structured JSON in production, console in dev. No `fmt.Println` in committed code.
- **Context**: every external call takes `context.Context` as first arg.
- **No globals**: subsystems are structs constructed in `internal/app`. Tests construct their own.
- **Immutability**: prefer returning new structs to mutating receivers. Aligns with CLAUDE.md.
- **Comments**: only when WHY is non-obvious. No restating the WHAT.
- **Tests**: table-driven where it fits. `_test.go` files alongside source. Integration tests in `test/integration/`.
- **Imports**: stdlib, blank line, third-party, blank line, internal.
- **Naming**: subsystem packages are nouns (`session`, `integration`, `channel`); helper packages are nouns or short verbs (`auth`, `eventbus`).

---

## 16. Testing strategy

| Layer | Tool | Target coverage |
|---|---|---|
| Unit | stdlib `testing` | 70%+ in `internal/` |
| HTTP handler | `httptest` | every route happy + 1 error |
| Integration | real PG + real binary, spawned in `test/integration/` | every cross-subsystem flow |
| End-to-end (mobile) | Flutter integration tests | 5 golden flows |
| End-to-end (web) | Playwright | 5 golden flows |
| Load | `vegeta` smoke before each release | session start, integration proxy, event WS |

Hot rules:
- Don't mock the database in integration tests (CLAUDE.md feedback memory: prior incident with mocked tests passing while migration broke prod).
- E2E tests must run against a real built binary, not the Go test runner.
- A failing test in `test/integration/` blocks merge.

---

## 17. Migration from v1

### What we lift unchanged
- **Provider manifests** (`plugins/builtin/<x>/manifest.json`) → copy into `internal/catalog/builtin/`. Drop `permissions`, `host`, `bridge`, `webview` related fields.
- **PTY session core** (creack/pty usage patterns from `kernel/terminal/`).
- **Flutter app shell** (lift `app/lib/` largely as-is).
- **Vue web app** (decision pending — see §19).

### What we rebuild from scratch
- HTTP router + middleware (clean chi setup).
- Session manager (new structure with explicit subsystem boundaries).
- Channel hub (telegram first, slack second, abstracted from day one).
- Integration registry (zero v1 code).
- Auth + quota (zero v1 code; v1 had ad-hoc patterns).
- Event bus (zero v1 code; HookBus had useful concepts but no clean API).

### What we explicitly do not bring over
- `plugin/bridge/` (12k lines, never used).
- `plugin/install/` + `signing/` (marketplace empty).
- `plugin/market/` (no third-party plugins).
- `plugin/host/` (no host-form plugins).
- `gateway/llm_providers.go` + `gateway/llm_proxy/` (CLI does the LLM work; we don't proxy).
- `gateway/sourcecontrol/` and friends (rebuild as a channel-side feature later if needed).
- `gateway/marketplace.go`.
- v1's `plugin_kv` / `plugin_secret` tables (use channel/provider/integration tables instead).

### v1 repository disposition
- Keep `/Claude_Workspace/opendray` as read-only reference for one quarter after v2 reaches feature parity.
- Archive (rename branch, mark README) at v2 v1.0 release.
- v1 remains running in production until v2 is fully migrated; no big-bang switchover.

---

## 18. Roadmap & milestones

V1.0 target: **12 weeks from start of code.**

| Week | Milestone | Deliverable |
|---|---|---|
| 1 | M0 — scaffold | go.mod, gateway hello world, slog, config loading, PG migrate |
| 2 | M1 — sessions α | spawn/list/kill PTY sessions, REST + WS stream |
| 3 | M1 — sessions β | resume, ring buffer replay, idle detector, audit log |
| 4 | M2 — provider catalog | embed manifests, render config, enable/disable |
| 5 | M3 — integration α | registration, API key, scope check, basic reverse proxy |
| 6 | M3 — integration β | health checker, stats, key rotation |
| 7 | M3 — integration γ | event WS, subscription, integration SDK Go skeleton |
| 8 | M4 — channel hub α | Channel interface, telegram impl (lift from v1, behind interface) |
| 9 | M4 — channel hub β | inbound message → session routing, slack first impl |
| 10 | M5 — clients | Flutter app rewired to v2 API; Vue web rewired or replaced |
| 11 | M6 — SDK & docs | Go SDK polish, TS SDK MVP, integration-guide.md, operator-guide.md |
| 12 | M7 — release | smoke tests, deploy to prod, archive v1 |

After V1.0, V1.1+ candidates (in priority order):
- Mobile push reliability hardening
- Multi-CLI parallel session UX (claude + codex + gemini side-by-side)
- iMessage channel
- Source-control mobile flow (rebuild minimal, channel-style)
- Quota dashboard
- Embedded scripting layer (Starlark) — only if a real need surfaces

---

## 19. Open questions

These need answers before code lands. Each gets an ADR (`docs/adr/NNNN-<slug>.md`) when resolved.

1. **Web client: Vue 3 or Flutter Web?**
   - Vue is v1 code already written. Flutter Web means one client codebase.
   - Cost of switching: rewrite web flows.
   - Cost of keeping: maintain two UIs.
   - Recommendation: Flutter Web, unless mobile UX is genuinely incompatible.

2. **Single vs split repo for SDK**
   - SDK lives in `pkg/opendrayclient/` and `sdk-ts/` inside this repo (monorepo) vs. separate repos.
   - Monorepo until SDK release cadence diverges from server.

3. **Module path: `opendray-v2` or `opendray`**
   - `opendray-v2` while building; rename to `opendray` at v1.0 release.

4. **Database migration tool: hand-rolled vs goose**
   - Hand-rolled to avoid a dependency unless the migration set grows past ~20 files.

5. **Config format: TOML or YAML or env-only**
   - TOML for human editing + env-only override (12-factor). Decision: TOML.

6. **License**
   - Apache 2.0 (matches v1). Keep.

7. **opendray vs RCC long-term relationship**
   - Three options:
     a. opendray replaces RCC (RCC retired)
     b. opendray runs alongside RCC, talks to it as an integration
     c. opendray sits above RCC (RCC handles raw PTY, opendray adds product layer)
   - Recommendation pending: (a) is cleanest if the v2 PTY layer reaches RCC's reliability.

8. **Mobile build / release pipeline**
   - Per CLAUDE.md global rules: every Flutter mobile app must have `build_release.sh` doing version bump + APK to UNAS + IPA to TestFlight. Carry over from v1.

---

## Document maintenance

This document is the SSOT. Every PR that contradicts it must either:
- Update the doc + ADR with the change, or
- Be rejected.

If the doc and code disagree, the doc is wrong (in this rewrite, intentionally) — fix the doc.

Updates land via PR with `docs:` prefix; review by repo maintainer.
