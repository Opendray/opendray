-- 0066 — kb_integrations: ship the Third-Party Integration Guide as a
-- first-class GLOBAL knowledge page in the Cortex knowledge base.
--
-- The four classic KB pages (infrastructure/conventions/lessons/reusable,
-- migration 0050) are AI-drafted. Integrations is a new, HUMAN-AUTHORED
-- knowledge domain: the canonical, forward-looking contract every
-- third-party app/website follows to integrate with opendray. The
-- knowledge base is extensible (0046 made project_docs.kind a slug, not a
-- fixed enum; ValidGlobalKBSlug accepts any kb_* slug), so no CHECK change
-- is needed here.
--
-- Two data inserts, no Go change:
--   1. Register the blueprint section so the page is a first-class member
--      of the global blueprint (ListSections / knowledge UI / on-demand
--      search see it). maintainer_mode='human' (the KB drafter only
--      redrafts the classic four, so it never touches this page); pinned
--      so it ships undeletable; inject=FALSE because the guide is large —
--      it stays out of every spawn banner and is reached on demand via
--      cross-layer search (memory_search / project_search). Flip to
--      inject=TRUE + nature='foundational' only to make it a standing
--      guardrail injected into every spawn.
--   2. Seed the page content with updated_by='operator' (a human edit), so
--      the AI drafter never overwrites it (human-edit = lock). The
--      operator can still edit it live via the doc API/UI.
--
-- Source of truth for the prose: docs/integrations/INTEGRATION_GUIDE.md.
-- This migration seeds the same content; keep them in sync when the
-- contract changes (see the guide's "Versioning" section).

INSERT INTO doc_blueprint_sections
    (cwd, slug, title, description, position, maintainer_mode, prompt_hint, pinned, inject, nature)
VALUES
    ('__global__', 'kb_integrations', 'Integrations',
     'Canonical contract every third-party app or website follows to integrate with opendray: registration & API key, authentication, scopes, the unified spawn profile (provider/model + MCP/system-prompt/permission), driving sessions, memory & privacy, reverse proxy / event stream / health, observability, security, multi-tenancy, upgrades & offboarding, and a scenarios quick-reference.',
     4, 'human', '', TRUE, FALSE, 'emergent')
ON CONFLICT (cwd, slug) DO NOTHING;

INSERT INTO project_docs (id, cwd, kind, content, updated_by)
VALUES (
    'doc_global_kb_integrations',
    '__global__',
    'kb_integrations',
    $ODGUIDE$
# opendray Third-Party Integration Guide

This is the canonical, forward-looking contract that **every** third-party app or website MUST follow to integrate with opendray. opendray is a self-hosted gateway that drives AI coding CLIs (Claude Code, Codex, Gemini, OpenCode, antigravity) over a PTY behind a unified REST + WebSocket API. A third-party app integrates by being **registered by an operator** (admin-only), receiving a **one-time scoped API key**, and then spawning and driving agent sessions over REST while optionally proxying its own UI and subscribing to a live event stream. This document states the rules as explicit **MUST / SHOULD / NEVER**, documents the current reality honestly (including what is *not* enforced yet), and gives you a runnable end-to-end path.

## Who this is for

External developers building an app, bot, website, or service that drives opendray agent sessions. You have never seen opendray's internals and you do not need to. You will be given an API key by the opendray operator. This guide tells you exactly what you may send, what you will receive, what is guaranteed, and what you MUST tolerate.

## Table of contents

1. [Current reality — read this first](#1-current-reality--read-this-first)
2. [The integration model](#2-the-integration-model)
3. [Registration & the API key](#3-registration--the-api-key)
4. [Authentication on every request](#4-authentication-on-every-request)
5. [Scopes & authorization](#5-scopes--authorization)
6. [The unified spawn profile — configuring what runs](#6-the-unified-spawn-profile--configuring-what-runs)
7. [Driving a session & getting replies](#7-driving-a-session--getting-replies)
8. [Memory & privacy](#8-memory--privacy)
9. [Reverse proxy, event stream & health](#9-reverse-proxy-event-stream--health)
10. [Observability & call audit](#10-observability--call-audit)
11. [Security & safety](#11-security--safety)
12. [Operating many integrations (multi-tenancy)](#12-operating-many-integrations-multi-tenancy)
13. [Upgrades, restarts & offboarding](#13-upgrades-restarts--offboarding)
14. [HTTP status & error-shape reference](#14-http-status--error-shape-reference)
15. [Data formats](#15-data-formats)
16. [Endpoint reference](#16-endpoint-reference)
17. [End-to-end worked example](#17-end-to-end-worked-example)
18. [Onboarding checklist](#18-onboarding-checklist)
19. [Scenarios & edge cases (quick reference)](#19-scenarios--edge-cases-quick-reference)
20. [DO / DON'T summary](#20-do--dont-summary)
21. [Versioning of this guide](#21-versioning-of-this-guide)

---

## 1. Current reality — read this first

Calibrate before reading the deep dives. These are verified properties of the current build (`/api/v1`, "M3"):

- **NO rate limiting, quota, or concurrency cap exists.** There is no `429`, no `Retry-After`, no per-integration session cap anywhere in the integration or session HTTP paths. An integration can spawn unbounded sessions and exhaust the host. Respecting limits is an **operator** responsibility (process/OS isolation) and an integrator courtesy (self-throttle). NEVER write client code that expects a `429` or a `Retry-After` header from opendray today.
- **Scope enforcement is partial.** Only `event:subscribe:<topic>` (on the event WebSocket) and `memory:read` / `memory:write` (on `/memory/*`) are actually checked. `session:*`, `channel:*`, `provider:read` are **declared but not enforced** in M3. See [§5](#5-scopes--authorization).
- **Pure-API consumers receive `session.idle`, NOT `session.turn_completed`.** `session.turn_completed` is only emitted for sessions armed by the channel layer. A REST `POST /input` does NOT arm turn detection. If your code blocks on `session.turn_completed` for a pure-API session, **it will hang forever.** See [§7](#7-driving-a-session--getting-replies).
- **No event replay.** If your event WebSocket disconnects, every event published during the gap is lost. There is no catch-up buffer. You MUST reconcile state via REST after reconnect.
- **`POST /sessions` is not idempotent and `POST /input` is at-least-once.** Retrying a timed-out create leaks a session; retrying an input double-sends bytes to the PTY. There is no idempotency key. See [§13](#13-upgrades-restarts--offboarding).
- **`permission_mode` is forthcoming.** Today the spawn profile carries a boolean `bypass_permissions`. The intended contract reshapes this into a provider-agnostic `permission_mode` field (documented in [§6](#6-the-unified-spawn-profile--configuring-what-runs) and clearly marked **FORTHCOMING**). Build against `bypass_permissions` now; the reshape will be backward-compatible at the boolean level.

---

## 2. The integration model

### Two patterns

Every integration is **one of two shapes**. Pick deliberately.

| | **Pattern A — Consumer (pure API)** | **Pattern B — Bridge channel** |
|---|---|---|
| Suitable for | Custom apps needing full session control | Chat bots (Telegram, Slack, visayabai) |
| Ownership | Your app creates, routes, drives, terminates sessions | opendray owns session binding + reply detection |
| Endpoints | `POST /sessions`, `POST /sessions/{id}/input`, `wss .../sessions/{id}/stream`, `wss .../integrations/_events` | Bridge WebSocket `wss .../channels/bridge/ws?token=<BRIDGE_TOKEN>` |
| Reply signal | `session.idle` (or the agent pushing via its own MCP tool) | opendray detects the turn for you (`session.turn_completed`) |
| Complexity | Higher — you build the loop | Lower — opendray handles reply/typing |
| Reference | This guide | `docs/integrations/connecting-visayabai-bot.md` + `visayabai_bridge_adapter.py` |

This document focuses on **Pattern A**. Pattern B is fully covered in the visayabai bridge guide; the bridge wire protocol is summarized in [§17](#17-end-to-end-worked-example).

### The lifecycle

1. **Register** (operator, admin-only, one-time) → you receive a one-time `odk_live_…` key.
2. **Configure the spawn profile** (operator, on the integration row) → which agent + what MCP/prompt/bypass to inject.
3. **Spawn sessions** (your app, per user/conversation) → `POST /sessions` with the key.
4. **Drive** (your app) → `POST /input`, listen on the event WS for `session.idle` / `session.ended`, read output, post back.

### Isolation: `origin=integration`

Every session your integration creates is stamped server-side with `origin=integration` and your `integration_id`. **These fields are derived from your authenticated principal, NEVER from the request body** — you cannot spoof them. This gives three guarantees:

- **Visibility isolation.** Integration sessions are hidden from the operator's session list. The operator console (admin token, or no integration principal) never sees `origin=integration` sessions. An integration token sees **only** sessions whose `integration_id` matches its own.
- **Memory isolation.** Integration sessions default to `memory_policy=none`: nothing is captured into shared memory, and the cross-project memory MCP is never attached. See [§8](#8-memory--privacy).
- **Spawn-profile injection.** Your declared MCP servers / system prompt / bypass flag apply **only** to your sessions, never to operator or CLI-direct sessions.

---

## 3. Registration & the API key

### Registration is admin-only

The operator (admin bearer token) registers you. **Your app NEVER self-registers.** You ask the operator for a key; they create the row and hand you the plaintext out-of-band.

```http
POST /api/v1/integrations
Authorization: Bearer <admin_token>
Content-Type: application/json
```

```json
{
  "name": "my-bot",
  "base_url": "",
  "route_prefix": "",
  "scopes": ["session:read", "event:subscribe:session.*"],
  "version": "1.0.0",
  "memory_policy": "none",
  "default_provider_id": "claude",
  "default_model": "opus",
  "default_claude_account_id": "",
  "mcp_servers": [],
  "system_prompt": "",
  "bypass_permissions": false
}
```

| Field | Required | Default | Rule |
|---|---|---|---|
| `name` | **MUST** | — | Unique display label. DB-unique. A `409` may mean a name **or** prefix collision (see edge cases). |
| `base_url` | paired | `""` | Full URL (`http://…`/`https://…`) for a reverse-proxy integration, or empty for consumer-only. Trailing slash is stripped. MUST be set together with `route_prefix` or both empty. |
| `route_prefix` | paired | `""` | URL slug for the proxy. MUST NOT contain `/?#`. Reserved (rejected `409`): `""` (only valid when `base_url` empty), `_events`, `_kinds`, `_internal`, `_`. |
| `scopes` | SHOULD | `["session:read","event:subscribe:session.*"]` | Empty array → default. Unknown strings are stored as-is (forward-compatible). |
| `version` | optional | `""` | YOUR version string. Informational. (Not opendray's version.) |
| `memory_policy` | optional | `none` | `none` \| `quarantine` \| `full`. Validated. See [§8](#8-memory--privacy). |
| `default_provider_id` | optional | `""` | Spawn default. See [§6](#6-the-unified-spawn-profile--configuring-what-runs). |
| `default_model` | optional | `""` | Spawn default. |
| `default_claude_account_id` | optional | `""` | Spawn default. |
| `mcp_servers` | optional | `[]` | Injection. JSON array. See [§6](#6-the-unified-spawn-profile--configuring-what-runs). |
| `system_prompt` | optional | `""` | Injection. Markdown. |
| `bypass_permissions` | optional | `false` | Injection. See [§6](#6-the-unified-spawn-profile--configuring-what-runs) and [§11](#11-security--safety). |

> **The default `memory_policy` is `none`.** (An internal code comment says "quarantine" — it is stale and wrong; the registration path sets `none`.)

**Response (201):**

```json
{
  "integration": {
    "id": "int_Qp8vBWT5WHiu",
    "name": "my-bot",
    "base_url": "",
    "route_prefix": "",
    "scopes": ["session:read", "event:subscribe:session.*"],
    "enabled": true,
    "health_status": "unknown",
    "created_at": "2026-06-17T14:30:22.123Z",
    "memory_policy": "none",
    "bypass_permissions": false,
    "is_system": false
  },
  "api_key": "odk_live_KJq8ne3Tyz42X9k8m2L7pQ0uRsT_aBcDeFgH"
}
```

### Consumer-only integration

Set both `base_url` and `route_prefix` to `""`. Consumer-only integrations cannot be reverse-proxied, are never health-probed (stay `health_status: "unknown"`), but can fully drive sessions and subscribe to events. Internally opendray stores a synthetic `_consumer_<id>` prefix to satisfy a DB constraint; it is blanked in every JSON response, so you always see `"route_prefix": ""`.

### The API key

- Format: `odk_live_` + base64url payload (~56 chars total).
- It is shown **exactly once**, in the registration (or rotation) response's `api_key` field. opendray stores only a bcrypt hash (cost 12) and **discards the plaintext**.
- It is NEVER shown again — not in `GET /integrations/{id}`, not in `GET /integrations`, not in logs, not in the UI, not in backups.
- It is a first-class **Principal**, equivalent to an admin bearer for the endpoints it can reach.

**You MUST:**
- Store the plaintext in a secret manager or encrypted env var (mode `0600`), NEVER in source control or logs.
- Treat it like a password; reference it in logs only by integration ID/name.

### Rotation, disable, delete (admin-only)

| Action | Endpoint | Effect |
|---|---|---|
| Rotate | `POST /api/v1/integrations/{id}/rotate-key` | New key returned **once**; old key invalidated **immediately** (no grace period); token cache cleared. |
| Disable | `PATCH /api/v1/integrations/{id}` `{"enabled": false}` | All auth with the key returns `401`; row preserved; running sessions keep running but you lose the ability to drive them. |
| Delete | `DELETE /api/v1/integrations/{id}` | Row removed; cannot be undone; running sessions become orphaned. |

System integrations (`is_system: true`, e.g. the opendray-memory MCP bridge) **cannot** be deleted or rotated by the operator (`403 ErrSystemIntegration`) — destroying their key would orphan running sessions whose `mcp.json` references it.

---

## 4. Authentication on every request

Send the key as a bearer token on **every** request.

**HTTP:**
```http
Authorization: Bearer odk_live_KJq8ne3Tyz42X9k8m2L7pQ0uRsT
```

**WebSocket** (browsers can't set custom WS headers — use the query param):
```
wss://HOST/api/v1/integrations/_events?topics=session.*&token=odk_live_…
```
A `Bearer` header is also accepted on the WS handshake if your client can set it; otherwise use `?token=`.

**Validation order** (combined middleware):
1. Admin bearer (in-memory map, no bcrypt).
2. Integration key fallback: scans **all enabled** integrations' bcrypt hashes; first match wins. The `token → integration_id` result is cached in an in-memory map to skip bcrypt on repeat calls.

> The token cache is a plain map cleared **wholesale** on disable / rotate / delete (not a per-entry LRU). On a cache hit, the row is re-read and re-checked for `enabled`, so a disabled key is rejected even if cached.

On failure: `401` with `WWW-Authenticate: Bearer realm="opendray"` and body `{"error":"unauthorized"}`. All invalid keys are treated identically (no existence side-channel).

---

## 5. Scopes & authorization

### Principals

| Kind | Scopes | Auth |
|---|---|---|
| `admin` | none (bypasses all scope checks) | admin bearer |
| `integration` | the array granted at registration | `odk_live_…` key |

### The canonical scope list

| Scope | Purpose | Enforced in M3? |
|---|---|---|
| `session:read` | List/get sessions, read buffer/history | **No** (declared only) |
| `session:create` | Spawn sessions | **No** |
| `session:input` | Send input / resize / stop / delete | **No** |
| `channel:send` | Post to channel adapters | **No** |
| `channel:receive` | Receive inbound channel messages | **No** |
| `provider:read` | List providers / read config | **No** |
| `memory:read` | `GET /memory/status,list,archived,scope-keys`, `POST /memory/search` | **YES** |
| `memory:write` | `POST /memory/store` | **YES** |
| `event:subscribe:<topic>` | Subscribe to an event topic on the event WS | **YES** |

**Wildcards:** prefix match only, suffix `.*`.
- `event:subscribe:session.*` grants `session.idle`, `session.turn_completed`, `session.ended`, etc.
- `event:subscribe:*` grants all topics.
- There is **no hierarchy**: `memory:*` does NOT expand to `memory:read`/`memory:write` — grant each explicitly.

### Enforcement reality (M3) and what you MUST do

- **Enforced today:** `event:subscribe:<topic>` (event WS), `memory:read`, `memory:write`.
- **Declared but unenforced:** `session:*`, `channel:*`, `provider:read`. Any valid integration token can call `POST /sessions`, `GET /providers`, etc., regardless of declared scopes (per ADR 0006, deferred to v1.1).

**You MUST:**
- Request the **minimal** scopes you need and document why.
- Design your client so that **any** endpoint can start returning `403` in a future build (when enforcement lands). Handle `403` gracefully (log, alert operator, do NOT retry-loop).

**You MUST NOT:**
- Rely on scope enforcement as an access-control boundary in M3. The operator is the trust boundary; isolation between mutually-untrusting integrations is achieved at the OS/instance level (see [§12](#12-operating-many-integrations-multi-tenancy)).

The `403` body text differs per surface (not one canonical message):
- Event WS: `missing scope: event:subscribe:<topic>`
- Memory: `requires admin or the "memory:read" scope` (or `memory:write`)

---

## 6. The unified spawn profile — configuring what runs

The integration row carries one **unified spawn profile** in two halves on the same row. The whole point: **declare intent ONCE, stay decoupled from any single CLI.** opendray translates your intent per-provider at spawn time.

### Half 1 — Default agent (identity; overridable per-request)

| Field | Meaning |
|---|---|
| `default_provider_id` | Which CLI to spawn (`claude`, `codex`, `gemini`, `opencode`, `antigravity`). |
| `default_model` | Model within that provider (e.g. `opus`). |
| `default_claude_account_id` | Claude account binding (Claude provider only). |

These are **defaults**. A `POST /sessions` request that supplies its own `provider_id`/`model`/`claude_account_id` always wins; an omitted field falls back to the integration default.

### Half 2 — Injection (tools/prompt/permission; integration-only)

| Field | Type | Meaning |
|---|---|---|
| `mcp_servers` | JSON array | Provider-agnostic MCP server descriptors injected into every session you spawn. |
| `system_prompt` | string | Boot system prompt injected into every session you spawn. |
| `bypass_permissions` | boolean | Auto-approve all tool calls (unattended). **FORTHCOMING:** reshaped into a provider-agnostic `permission_mode` — see below. |

### The hard rules

- **MUST** declare all MCP / system-prompt / bypass intent on the **integration row**, never in `POST /sessions` `args`.
- **NEVER** hand-build per-CLI flags (`--mcp-config`, `--append-system-prompt`, `--dangerously-skip-permissions`, `--yolo`, …) into the request `args`. That hard-locks you to one CLI and breaks the moment the operator switches the default provider.
- **Injection fields are integration-only and are NOT per-request overridable.** `CreateRequest` has no `mcp_servers`/`system_prompt`/`bypass_permissions` fields, so any such keys in the request body are **silently dropped** by the JSON decoder (there is no `400` rejection today). This is by design: an end-user message routed through your bot MUST NOT be able to mutate the tools, prompt, or bypass of an unattended session.

### Precedence

**Identity fields** (`provider_id`, `model`, `claude_account_id`):

```
request body  >  integration default  >  provider-config / platform default
```

For `model` specifically there is a **higher** layer: a literal `--model X` in the request `args` wins over the `model` field and over the integration default (the manager dedups overridden flags). Prefer the `model` field; don't fight it with `args`.

### MCP server descriptor

```json
{
  "name": "invoicing",
  "transport": "stdio",
  "command": "/usr/bin/invoicing-mcp",
  "args": ["--db=/data/invoices.sqlite"],
  "env": {"API_KEY": "${INVOICE_API_KEY}"},
  "url": null,
  "headers": null
}
```

- `name` (required): unique; integration entries **win** on name collision against vault/provider-config entries.
- `transport` (default `stdio`): `stdio` | `sse` | `http`.
- `command`/`args` for `stdio`; `url`/`headers` for `sse`/`http`.
- `${KEY}` placeholders in `command`/`args`/`env`/`headers` are resolved at spawn time from the operator's secrets dotenv file (`secretsFile`). **A missing key passes the literal `${KEY}` through** so the agent surfaces a clear "credential not set" error rather than failing silently. Secrets live only in memory + the per-session scratch dir; they are NEVER written to the DB or logs. NEVER hardcode credentials — use `${KEY}` and tell the operator which keys to add.

### Per-provider translation (verified)

| Intent | claude / antigravity | gemini | codex |
|---|---|---|---|
| MCP | `--mcp-config <file>` | `<cwd>/.gemini/settings.json` | config (manifest) |
| System prompt | `--append-system-prompt` | `GEMINI.md` | `AGENTS.md` |
| Bypass | `--dangerously-skip-permissions` | `--yolo` | `--dangerously-bypass-approvals-and-sandbox` |

Switching providers requires **zero** code changes on your side: the operator patches `default_provider_id`, and the next session renders the same intent through the new CLI's surfaces.

### FORTHCOMING: `permission_mode`

The current schema column is `bypass_permissions BOOLEAN`. The **intended contract** (being reshaped on `feat/integration-spawn-profile`) replaces the raw boolean with a provider-agnostic **`permission_mode`** that opendray maps to each CLI's permission surface (the manifest's bypass flag for the auto-approve mode). Build against `bypass_permissions` today. When `permission_mode` lands it will be backward-compatible: `bypass_permissions: true` ≡ the auto-approve mode. Do not hand-build permission flags into `args` regardless.

---

## 7. Driving a session & getting replies

> **This section overrides any "subscribe to `session.turn_completed`" advice for Pattern A.** Read it before the worked example.

### Create a session

```http
POST /api/v1/sessions
Authorization: Bearer odk_live_…
Content-Type: application/json
```
```json
{
  "name": "my-bot · user-456",
  "provider_id": "claude",
  "model": "opus",
  "claude_account_id": "",
  "cwd": "/var/lib/opendray/projects/my-bot",
  "args": []
}
```

- `provider_id` and `cwd` are **required** (omitting `cwd` → `400`). If `provider_id` is omitted it is filled from your integration default; if there is no default the create will fail downstream.
- `args` are raw CLI args. **Keep them empty** and let your MCP tools carry the work. NEVER put MCP/prompt/bypass flags here ([§6](#6-the-unified-spawn-profile--configuring-what-runs)).
- `origin` and `integration_id` are stamped server-side from your principal; any values you send are ignored.
- Response `201` with the session (`id`, `state`, `pid`, `origin: "integration"`, `integration_id`).

### Send input

```http
POST /api/v1/sessions/{id}/input
```
```json
{"data": "what is 2+2?\n"}
```
- `data` is raw bytes to the PTY stdin. **You MUST include the trailing `\n`** ("press enter").
- Control bytes allowed: `\x03` = Ctrl-C, `\x04` = Ctrl-D.
- Returns `204 No Content` (fire-and-forget; asynchronous).
- **NOT idempotent in the dedup sense:** a retried `/input` writes the bytes again. NEVER blindly retry an input you believe may have landed.

### Getting the reply — the correct signal

Subscribe to the event WS **before** sending, and key off the right event:

| Topic | When | Use it for |
|---|---|---|
| `session.idle` | The agent went active→idle. **Always fires** for any active session (armed or not). Carries `recent_output`. | **This is your reply signal for Pattern A.** Grab `recent_output`, strip ANSI, post to the user. |
| `session.turn_completed` | Only for sessions **armed** by the channel layer (`ExpectTurn`). A REST `POST /input` does **NOT** arm it. | Pattern B (bridge) only. **Pure-API consumers will never receive this.** |
| `session.ended` | The CLI process exited (clean or crash). Terminal. | Recreate / `POST /start` if the user continues. |

**Three correct strategies for Pattern A — pick one:**

1. **`session.idle`** — subscribe `event:subscribe:session.idle` (or `session.*`), wait for `idle` matching your `session_id`, read `recent_output`. Simple; the idle threshold is operator-configurable (default ~5 minutes for unattended sessions, so tune it down for interactive bots or use strategy 3).
2. **Agent-push via your own MCP tool** — give the agent a `reply_to_user`-style MCP tool in your spawn profile and a system prompt that says "reply ONLY via `reply_to_user`." The tool calls back into your service. This is the most reliable, latency-tight pattern for unattended bots and is what the PDA secretary uses. It does not depend on idle timing.
3. **Buffer polling** (fallback when you can't hold a WS) — after `POST /input`, poll `GET /sessions/{id}/buffer?since=<cursor>` and advance the cursor from the `X-OpenDray-Buffer-Cursor` response header.

**NEVER** scrape the PTY buffer in a tight 100ms loop, and **NEVER** block forever on `session.turn_completed` for a pure-API session.

### Session state machine

```
create ──► running ──(/stop, SIGTERM)──► stopped ──(/start)──► running
                │                                          ▲
                └──(process exits / crash)──► ended ───────┘ (/start re-spawns: FRESH PTY, context lost)
```
- `POST /stop` SIGTERMs the process but **keeps the row** → restartable with `POST /start`. Returns `200` + session.
- `POST /start` re-spawns under the original provider/cwd/args/account. For a terminal (ended/stopped) row this is a **fresh PTY — prior in-process context is lost** (the row/history is preserved). Returns `200` + session. It does **not** resume a live process.
- `DELETE /sessions/{id}` SIGTERMs then drops the row. Irreversible.
- Driving an already-ended session returns `409` (`ErrAlreadyEnded`).

### File uploads

`POST /api/v1/sessions/{id}/uploads` (multipart). Capped at **16 MiB** per request. Returns the saved path under `os.TempDir()/opendray-uploads/{session_id}/`. Reference it in a message ("analyze the file at /…"). Paths are **session-scoped** and may be garbage-collected after the session ends; do not share them between sessions. NEVER stream large files into stdin.

---

## 8. Memory & privacy

### The policy (integration-scoped, NOT per-request)

| `memory_policy` | Behavior | Use for |
|---|---|---|
| `none` (default) | Sessions produce **zero** memory: no transcript read, no capture, no store. `POST /memory/store` returns `403`. | Third-party / sensitive / compliance-bound apps. |
| `quarantine` | Facts captured to the **quarantine tier**: excluded from search/injection/KB, operator-reviewable, **auto-expire after 30 days** (`DefaultQuarantineTTL`). | Trusted-but-unreviewed integrations. |
| `full` | Durable facts, same as operator sessions; injected at spawn, searchable, consolidated. | Fully-trusted internal integrations. |

**MUST NOT** put `memory_policy` in a `POST /sessions` body — it is set on the integration row by the operator and applies to all your sessions. **MUST NOT** try to disable capture by hand-crafting `args`.

### Isolation guarantees (current build)

- **Read-side (verified):** the cross-project opendray-memory MCP is **not attached** to `origin=integration` sessions. The agent cannot read the operator's or other integrations' memory.
- **Write-side (verified):** the capture pipeline resolves your policy up front; `none` → the session is skipped entirely (no history read, no summarizer call). Direct `POST /memory/store` from an integration key applies the same tier routing: `none` → `403`, `full` → durable, anything else → quarantine (30-day TTL). Global-scope writes always require admin (`403` for integration keys).
- **Today's partitioning is cwd + tier based.** Captured project facts are written with `scope_key = <session cwd>`. So if your sessions share a cwd with the operator's project, facts land in that shared partition. **Give each integration its own stable `cwd`** (e.g. `/var/lib/opendray/integrations/<name>`) for isolation. Ephemeral cwds (`/tmp/*`, `/var/folders/*/T`) are **never** captured, regardless of policy.
- **FORTHCOMING (PR #380): the `integration:<id>` memory zone.** The intended contract routes integration capture into a dedicated `scope_key = "integration:<integration_id>"` zone (and guards the KB / git-activity / mirror enumerators) so a third-party's facts are isolated by *principal*, not just by cwd. Until #380 merges, rely on `memory_policy=none` (default) + distinct cwds for isolation. Document this zone name as the target so you do not depend on cwd collisions.

**SHOULD:** default to `none`; escalate to `quarantine`/`full` only after the operator vets your data quality and trust. For PII/regulated data, stay `none`.

---

## 9. Reverse proxy, event stream & health

These three HTTP surfaces are for Pattern A's optional extras and Pattern B. Consumer-only integrations use only the event WS.

### Reverse proxy — `/api/v1/proxy/{prefix}/*` (admin-only)

Lets the operator reach **your** HTTP service through opendray. A request to `GET /api/v1/proxy/acme/api/v1/x?p=1` is forwarded to `<base_url>/api/v1/x?p=1`.

- opendray **strips** the caller's `Authorization` header and injects:
  - `X-OpenDray-Forwarded-For: <client-ip:port>`
  - `X-Integration-ID: <integration-id>`
  - `X-OpenDray-API: v1`
- **NEVER** rely on an inbound `Authorization` header to identify the caller — it has been removed. Use `X-Integration-ID`.
- `403`/disabled → `503 {"error":"integration disabled"}`. `health_status == unhealthy` → `503 {"error":"integration unhealthy"}`. Upstream unreachable/timeout → `502 {"error":"upstream: …"}`. `base_url` that parses at registration but fails at proxy time → `500`.

### Event WebSocket — `/api/v1/integrations/_events`

```
wss://HOST/api/v1/integrations/_events?topics=session.idle,session.ended&token=odk_live_…
```

- `topics` is a **required** CSV (missing → `400 topics query param required (CSV)`; all-blank → `400 no valid topics`).
- **Admins MAY also connect** (the route is wired under the combined middleware). Admins subscribe with no per-topic scope check (per ADR 0009). Integration principals MUST hold `event:subscribe:<topic>` for **every** requested topic — if **any** one is missing the **whole** subscription is rejected `403` before the upgrade (no partial subscription).
- Frame shape:
  ```json
  {"topic":"session.idle","ts":"2026-06-17T14:32:45.123456789Z","data":{"session_id":"ses_…","recent_output":"…"}}
  ```
- **Heartbeat:** opendray sends a WS ping every 20s; respond with pong (most libraries auto-pong). Per-frame write deadline is 5s.
- **Backpressure = disconnect, NOT silent drop.** Each subscriber has a 64-event buffer; if a write fails or exceeds the 5s deadline, opendray **tears down the entire connection** (it does not drop individual events and keep going). A slow consumer loses the whole stream. You MUST reconnect with exponential backoff and re-subscribe.
- **No replay.** Events published while you were disconnected are gone. After reconnect, reconcile via `GET /sessions/{id}`.

Topic families: `session.*` (`idle`, `turn_completed`, `ended`), `integration.*` (`registered`, `deregistered`, `health_changed`, `key_rotated`), `channel.*`.

### Health probes — `GET {base_url}/health`

Only integrations with a non-empty `base_url` are probed (every **30s**, **5s** timeout; also immediately on registration). Consumer-only integrations are never probed (stay `unknown`).

Expected response (all fields optional):
```json
{"status":"healthy","version":"1.0.0","busy_ratio":0.1,"queue_depth":2}
```
`status` ∈ `healthy` | `degraded` | `unhealthy`. Any 2xx with empty/missing status is treated as healthy.

**Transition rules (verified — note the asymmetry):**
- Transport error or non-2xx → 1st time `degraded`, 2nd consecutive `unhealthy` (consecutive-failure counter).
- Body `status:"degraded"` → `degraded` but does **NOT** touch the failure counter.
- Body `status:"unhealthy"` → `unhealthy` (does not touch the counter).
- Any 2xx with `healthy`/empty status → resets the counter to `0` and sets `healthy`.
- **Unhealthy AUTO-RECOVERS:** a single healthy 2xx probe flips you back to `healthy` regardless of prior state. There is **no** sticky-unhealthy latch.

When `unhealthy`, only the **reverse proxy** returns `503`. Your session/event API calls still work. A status change emits `integration.health_changed`.

**SHOULD:** implement `/health` as a millisecond-scale in-memory check (no DB calls). Use `degraded` under load (proxy stays open) and `unhealthy` only when truly broken.

---

## 10. Observability & call audit

opendray records integration traffic for the operator. As an integrator, know that:

- **Inbound REST calls** authenticated as an integration are logged (method, path, status, duration, bytes, request id, `integration_id`) — not just proxy calls.
- **Outbound proxied calls** are logged by the proxy handler.

The operator reads this via the **admin-only** audit endpoint:

```http
GET /api/v1/integrations/_calls
   ?integration_id=<id>&direction=inbound|outbound&status_class=2..5
   &since=<RFC3339>&until=<RFC3339>&cursor=<int>&limit=1..500
Authorization: Bearer <admin_token>
```
Response: `{"entries":[…], "next_cursor": "<int>" | null}` (keyset pagination: rows have `id < cursor`; default `limit=100`).

A per-integration summary endpoint (`/integrations/_calls/summary`) is a **TODO**, not yet wired — do not call it.

**You SHOULD** log `integration_id` + `session_id` for every user interaction on your side so support can correlate with the operator's audit.

---

## 11. Security & safety

### Injection is one-way and integration-only

Already stated in [§6](#6-the-unified-spawn-profile--configuring-what-runs); restated as a security rule: an end-user message routed through your bot MUST NOT be able to inject MCP servers, rewrite the system prompt, or flip bypass. opendray enforces this structurally — those fields don't exist on `CreateRequest`. **You MUST** keep it that way: never invent a path that lets request data reach `args` as per-CLI injection flags.

### Bypass permissions = you own the safety policy

When `bypass_permissions: true`, every tool call auto-approves with no human at the TUI.

- **MUST**, if `bypass_permissions: true`, validate/sanitize/rate-limit every user input **before** `POST /input`. Treat the agent as an unauthenticated executor of whatever it's told.
- If your bot is a **pure relay** with no policy of its own, set `bypass_permissions: false` so the operator-attended TUI gates tool calls.
- The `cwd` is a **trust signal, not a sandbox** — opendray does not chroot/jail the agent. A tool that runs `rm -rf /` will run. Real isolation is the operator's job (OS sandboxing, containers, separate instances — see [§12](#12-operating-many-integrations-multi-tenancy)).

### SSRF via `base_url`

opendray does **not** validate `base_url` host/range; the proxy will forward to `127.0.0.1:6379`, internal APIs, etc. **Operators MUST** only register integrations whose `base_url` they trust and SHOULD apply egress controls. **Integrators MUST** document a stable, controlled `base_url` so the operator can audit it; never derive it from user input.

### Key hygiene

Store in a secret manager; never commit/log; rotate on a schedule and immediately on suspected leak (ask the operator to `rotate-key`). In-flight sessions survive rotation (the old key is baked into the running session's config); only new requests need the new key.

---

## 12. Operating many integrations (multi-tenancy)

When one gateway hosts N integrations, integrators and operators MUST understand:

- **Shared host UID.** All spawned CLIs run as the opendray OS user. `cwd` is not a security boundary, so integration A can read integration B's files. To isolate mutually-untrusting integrations, run **separate opendray instances / containers / VMs** or apply OS-level sandboxing. Do not rely on scopes or cwd alone.
- **No per-integration quota.** Any integration can spawn unbounded sessions ([§1](#1-current-reality--read-this-first)). Operators MUST cap resources at the OS/container level; integrators SHOULD self-throttle and reuse sessions per conversation.
- **Shared cold-auth cost.** Verifying an integration key on a cache miss scans **all enabled** integrations' bcrypt hashes (O(enabled rows)). Many enabled integrations + many distinct fresh tokens = more bcrypt work. Reuse a single key and let the token cache absorb repeats.
- **Shared event bus.** Each WS subscriber has a 64-event buffer and is torn down on write stall ([§9](#9-reverse-proxy-event-stream--health)). A misbehaving subscriber affects only its own connection, but you MUST reconnect on disconnect.
- **Memory partition sharing.** Until the `integration:<id>` zone (#380) ships, capture isolation is cwd-based — give each integration a distinct `cwd` ([§8](#8-memory--privacy)).

---

## 13. Upgrades, restarts & offboarding

### What you MUST tolerate across an opendray restart/upgrade

- **API keys survive.** Restarting/upgrading the gateway does not change your key.
- **Session rows survive; processes may not.** On restart, opendray reconciles sessions; a previously-running session may come back `running` (auto-resume) or land in a non-`ended` interrupted state. Session **IDs survive** in the DB.
- **Event WS connections do NOT survive** and there is **no replay.** You MUST reconnect with backoff and reconcile state via `GET /sessions/{id}` after the gateway comes back.
- **Treat 5xx / connection-refused as transient.** Use a circuit breaker + exponential backoff (e.g. 1s→2s→4s… capped at 60s, with jitter), queue user messages locally, and drain on recovery. There is no `429`; backoff is for `503`/network only.

### Retry safety

- `POST /sessions` is **not idempotent** — a retry after a timeout may leak a second session. Prefer to **read back** (`GET /sessions` filtered to your integration) before retrying a create, or accept and reap duplicates.
- `POST /input` is **at-least-once** — a retry double-sends bytes. NEVER auto-retry an input that may have landed.
- There is no idempotency-key mechanism.

### Offboarding runbook (operator)

To fully retire an integration:
1. **Disable** — `PATCH /integrations/{id} {"enabled": false}` (blocks new auth, clears token cache).
2. **Stop orphaned sessions** — disabling/deleting does NOT terminate running sessions; `DELETE` each of the integration's sessions (visible to an admin by id, or while the key is still valid via the integration's own list).
3. **Purge captured memory** — for `quarantine`/`full` integrations, the operator wipes the zone with `POST /memory/delete-by-scope` (admin-only) for the integration's `scope_key` (today the cwd; the `integration:<id>` zone once #380 ships). `none` integrations have nothing to purge.
4. **Delete the row** — `DELETE /integrations/{id}` (irreversible) — or leave it disabled for the audit trail.

---

## 14. HTTP status & error-shape reference

The error body is uniformly:
```json
{"error": "<message>"}
```

| Status | When | Notes |
|---|---|---|
| `200` | GET / list / `stop` / `start` / account-switch | Body is the resource. |
| `201` | `POST /integrations`, `POST /sessions` | Body includes the one-time `api_key` (registration only). |
| `204` | `POST /sessions/{id}/input` | No body. |
| `400` | Validation: bad JSON, missing `cwd`, `base_url`/`route_prefix` not paired, invalid `memory_policy`, `route_prefix` contains `/?#`, invalid `since`/`cursor`/`limit`, missing/empty event `topics`, unknown provider | Message is specific. |
| `401` | Missing/invalid/rotated/disabled key | `WWW-Authenticate: Bearer realm="opendray"`, body `{"error":"unauthorized"}`. |
| `403` | Missing enforced scope; integration global-memory write; `none`-policy `memory/store`; mutating a system integration | Message differs per surface ([§5](#5-scopes--authorization), [§8](#8-memory--privacy)). |
| `404` | Unknown integration/session id; proxy prefix not found | `ErrNotFound`. |
| `409` | Registration name/prefix/reserved-prefix conflict; driving an ended session (`ErrAlreadyEnded`); provider unavailable | A `409 "integration name already in use"` may actually be a **prefix** collision (the DB unique violation maps both to one error) — check your `route_prefix` too. |
| `422` | Memory store rejected by the gatekeeper (`ErrNotDurable`) | Distinct from validation `400`. |
| `500` | Internal error; malformed `base_url` at proxy time | — |
| `502` | Proxy upstream unreachable/timeout | `{"error":"upstream: …"}`. |
| `503` | Proxy to a disabled or unhealthy integration | Session/event API still works. |

There is **no** `429` and **no** `Retry-After`.

---

## 15. Data formats

- **Timestamps:** all server times are **UTC**. Event `ts` is **RFC 3339 Nano** (`2026-06-17T14:32:45.123456789Z`). Integration timestamps (`created_at`, `rotated_at`, `health_last_seen`) are UTC. Parse with any standard library.
- **ANSI:** `recent_output` and `GET /sessions/{id}/buffer` contain terminal escape codes. **MUST** strip them before rendering to a chat/email surface — e.g. `re.sub(r'\x1b\[[0-9;]*m', '', text)` (Python) or `sed 's/\x1b\[[0-9;]*m//g'`.
- **Buffer cursor:** `GET /sessions/{id}/buffer?since=<int>` takes a **byte offset**. The response sets `X-OpenDray-Buffer-Cursor` (next offset to read from) and `X-OpenDray-Buffer-Start` (the ring-buffer eviction floor — if your `since` is below it, scrollback was evicted). Use the header values; never hand-compute cursors. Invalid `since` → `400 invalid since: must be non-negative integer`.
- **List envelopes (no pagination):** `GET /integrations` → `{"integrations":[…]}`, `GET /sessions` → `{"sessions":[…]}` — **full, unbounded** lists. `GET /sessions/{id}/history?limit=N` takes a limit but no offset/cursor. Only `/integrations/_calls` is paginated (`cursor`/`limit`).

---

## 16. Endpoint reference

Base path: `https://<host>/api/v1`. Auth: `Authorization: Bearer odk_live_…` (or `?token=` on WS). Admin-only rows are marked.

| Method | Path | Purpose | Auth |
|---|---|---|---|
| POST | `/integrations` | Register (returns one-time key) | **admin** |
| GET | `/integrations` | List | **admin** |
| GET | `/integrations/{id}` | Get | **admin** |
| PATCH | `/integrations/{id}` | Update (incl. `enabled`, spawn profile) | **admin** |
| DELETE | `/integrations/{id}` | Delete | **admin** |
| POST | `/integrations/{id}/rotate-key` | Rotate key (returns new key once) | **admin** |
| GET | `/integrations/_calls` | Call audit (paginated) | **admin** |
| GET | `/integrations/_events` | Event WebSocket (subscribe `?topics=`) | admin OR integration |
| ANY | `/proxy/{prefix}/*` | Reverse proxy to integration `base_url` | **admin** |
| POST | `/sessions` | Spawn a session | integration / admin |
| GET | `/sessions` | List (filtered to your `integration_id`) | integration / admin |
| GET | `/sessions/{id}` | Get session | integration / admin |
| POST | `/sessions/{id}/input` | Write bytes to PTY stdin (204) | integration / admin |
| GET | `/sessions/{id}/buffer?since=` | Read output delta | integration / admin |
| GET | `/sessions/{id}/history?limit=` | Read message history | integration / admin |
| WS | `/sessions/{id}/stream` | Raw terminal stream (ANSI) | integration / admin |
| POST | `/sessions/{id}/start` | Re-spawn (fresh PTY) | integration / admin |
| POST | `/sessions/{id}/stop` | SIGTERM, keep row | integration / admin |
| DELETE | `/sessions/{id}` | SIGTERM + drop row | integration / admin |
| POST | `/sessions/{id}/resize` | `{"cols","rows"}` | integration / admin |
| POST | `/sessions/{id}/uploads` | Multipart file upload (≤16 MiB) | integration / admin |
| PATCH | `/sessions/{id}/claude-account` | Switch Claude account | integration / admin |
| GET | `/providers` | List providers (validate `provider_id`) | integration / admin |
| POST | `/memory/store` | Store a fact | integration (`memory:write`) / admin |
| POST | `/memory/search` | Search | integration (`memory:read`) / admin |
| GET | `/memory/list,status,archived,scope-keys` | Read surfaces | integration (`memory:read`) / admin |
| POST | `/memory/delete-by-scope` | Purge a scope zone | **admin** |

---

## 17. End-to-end worked example

### Pattern A (consumer) — register, drive, reply via `session.idle`

**1. Operator registers (admin, one-time):**
```bash
curl -X POST https://HOST/api/v1/integrations \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d '{
    "name": "my-bot",
    "memory_policy": "none",
    "default_provider_id": "claude",
    "default_model": "opus",
    "mcp_servers": [
      {"name":"invoicing","command":"/usr/bin/invoicing-mcp","args":["--db=/data/inv.sqlite"],
       "env":{"API_KEY":"${INVOICE_API_KEY}"}}
    ],
    "system_prompt": "# Invoicing secretary\nYou manage invoicing via the invoicing MCP. Reply ONLY via the reply_to_user tool.",
    "bypass_permissions": true,
    "scopes": ["session:read","session:create","session:input","event:subscribe:session.*"]
  }'
# → {"integration":{"id":"int_…"},"api_key":"odk_live_…"}   ← store securely, shown once
```

**2. App drives the loop (Python, reply via `session.idle`):**
```python
import asyncio, json, re, httpx, websockets

API_KEY = "odk_live_…"
HOST = "opendray.example.com"
ANSI = re.compile(r"\x1b\[[0-9;]*m")

async def run():
    # Subscribe FIRST. Pattern A keys off session.idle (NOT turn_completed).
    async with websockets.connect(
        f"wss://{HOST}/api/v1/integrations/_events?topics=session.idle,session.ended&token={API_KEY}"
    ) as ws:
        async with httpx.AsyncClient(base_url=f"https://{HOST}/api/v1",
                                     headers={"Authorization": f"Bearer {API_KEY}"}) as http:
            ses = (await http.post("/sessions",
                   json={"provider_id":"claude","cwd":"/var/lib/opendray/projects/my-bot"})).json()
            sid = ses["id"]

            r = await http.post(f"/sessions/{sid}/input", json={"data":"summarize open invoices\n"})
            assert r.status_code == 204  # async; not idempotent — do not blind-retry

            while True:
                frame = json.loads(await ws.recv())
                d = frame.get("data", {})
                if d.get("session_id") != sid:
                    continue
                if frame["topic"] == "session.idle":
                    reply = ANSI.sub("", d.get("recent_output", ""))
                    return reply           # ← post to the end user
                if frame["topic"] == "session.ended":
                    raise RuntimeError("session ended; recreate to continue")

print(asyncio.run(run()))
```
> Best practice for unattended bots: instead of `session.idle`, give the agent a `reply_to_user` MCP tool (in `mcp_servers`) and force it via the system prompt; the tool calls back into your service directly — tighter latency, no idle-threshold tuning.

**3. Operator later switches provider — your code unchanged:**
```bash
curl -X PATCH https://HOST/api/v1/integrations/int_… -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"default_provider_id":"gemini"}'
# next session renders the same MCP into .gemini/settings.json, the prompt into GEMINI.md, bypass via --yolo
```

### Pattern B (bridge) — wire protocol

For chat bots, register a `kind=bridge` channel (operator UI) and connect:
```
wss://HOST/api/v1/channels/bridge/ws?token=<BRIDGE_TOKEN>
```
- Register: `{"type":"register","platform":"my-bot","capabilities":["text","typing"],"metadata":{}}` → `{"type":"register_ack","ok":true}`.
- Inbound (you→opendray): `{"type":"message","conversation_id":"user-123","user_id":"456","user_name":"Alice","text":"hello"}`.
- Outbound (opendray→you): `{"type":"send","conversation_id":"user-123","text":"…"}`.
- `conversation_id` is the stable routing key; opendray binds each to its own session and handles reply detection for you. Full details: `docs/integrations/connecting-visayabai-bot.md`.

---

## 18. Onboarding checklist

**Pre-launch**
- [ ] Choose your pattern (A consumer vs B bridge).
- [ ] List your MCP tools and decide your reply strategy (`session.idle` vs agent-push MCP tool).
- [ ] Decide `memory_policy` (default `none`; escalate only with operator trust).

**Registration**
- [ ] Ask the operator to register you; verify scopes, default provider/model, MCP servers, `bypass_permissions`.
- [ ] Store the `odk_live_…` key in a secret manager (never source/logs).
- [ ] Smoke test: `curl -H "Authorization: Bearer odk_live_…" https://HOST/api/v1/sessions` → `200 {"sessions":[]}`.

**Development**
- [ ] Create sessions with a `name` prefix; let integration defaults fill empty fields; keep `args` empty.
- [ ] Open ONE persistent event WS; subscribe `session.idle,session.ended`; reconnect with backoff; reconcile via REST after reconnect.
- [ ] Strip ANSI before rendering.
- [ ] Handle `session.ended` (recreate) and `409 ErrAlreadyEnded`.
- [ ] Test with Claude AND at least one other provider — verify MCP + system prompt translate.

**Deployment**
- [ ] Rotate the key before go-live; use the fresh token in prod.
- [ ] Log `integration_id` + `session_id` per user interaction.
- [ ] Circuit-breaker + backoff on `503`/network; never expect `429`.
- [ ] Alert on `session.ended` and on sustained idle latency.

---

## 19. Scenarios & edge cases (quick reference)

| Situation | opendray behavior | Integrator action |
|---|---|---|
| Invalid / rotated / disabled key | `401 {"error":"unauthorized"}`, `WWW-Authenticate` | Re-read key from secret store; ask operator to rotate; do not retry-loop. |
| Pure-API code waits on `session.turn_completed` | Never fires for REST-driven sessions | Key off `session.idle` or an agent-push MCP tool. |
| Event WS drops / gateway restart | No replay; connection closed | Reconnect with backoff; reconcile via `GET /sessions/{id}`. |
| Slow event consumer (>64 buffered or >5s write) | Whole WS connection torn down | Reconnect with backoff; re-subscribe. |
| Missing scope on one of several `topics` | Entire subscription `403` before upgrade | Request the scope from the operator; subscribe only granted topics. |
| `POST /sessions` times out, you retry | A second session may spawn (not idempotent) | Read back via `GET /sessions` before retrying, or reap duplicates. |
| `POST /input` retried | Bytes double-sent (at-least-once) | Never blind-retry input. |
| Drive an ended session | `409 ErrAlreadyEnded` | `POST /start` (fresh PTY, context lost) or create a new session. |
| `cwd` is `/tmp/*` | Memory never captured | Use a stable project cwd. |
| `memory_policy=none` + `POST /memory/store` | `403 memory writes disabled…` | Change policy (operator) or drop memory calls. |
| Integration shares operator's cwd | Facts land in operator's partition | Give each integration a distinct cwd. |
| MCP `${KEY}` missing from secrets file | Literal `${KEY}` passed through; tool errors at use | Operator adds the key to the dotenv; restart gateway. |
| MCP server fails to start | Session spawns without that tool | Test the command standalone; check path/perms/env. |
| Provider switched mid-flight | Running sessions keep old provider; new spawns use new | Inform user; new provider applies to the next session. |
| Health `/health` down but app up | Status→unhealthy; proxy `503`; session/event API still works | Fix `/health`; it auto-recovers on next healthy probe. |
| First `/health` failure | `degraded` (proxy still forwards) | Investigate; second consecutive failure → `unhealthy`. |
| Operator disables integration | New auth `401`; running sessions keep running | Degrade gracefully; ask operator. |
| Operator deletes integration | Key dead (rotate won't help); sessions orphaned | Re-registration required; clean up. |
| Registration `409 "name already in use"` | Name **or** prefix collision (DB-unique) | Change both `name` and `route_prefix`. |
| Proxy to disabled/unhealthy integration | `503` | Re-enable / fix health. |
| Buffer `since` below `X-OpenDray-Buffer-Start` | Scrollback evicted | Re-sync (omit `since`); accept the gap. |

---

## 20. DO / DON'T summary

| DO | DON'T |
|---|---|
| ✅ Declare MCP / system prompt / bypass on the integration row. | ❌ Hand-build `--mcp-config` / `--append-system-prompt` / `--dangerously-skip-permissions` / `--yolo` into `POST /sessions` `args`. |
| ✅ Key off `session.idle` (or an agent-push MCP tool) for Pattern A. | ❌ Block on `session.turn_completed` for a pure-API session — it never fires. |
| ✅ Subscribe to the event WS once; reconnect with backoff; reconcile via REST. | ❌ Poll `/buffer` in a tight loop, or assume the WS replays missed events. |
| ✅ Strip ANSI from `recent_output` / `/buffer` before display. | ❌ Render raw terminal escapes to users. |
| ✅ Let integration defaults fill empty `provider_id`/`model`/`account`. | ❌ Try to override `mcp_servers`/`system_prompt`/`bypass` per request — silently dropped. |
| ✅ Use a distinct stable `cwd` per integration; default `memory_policy=none`. | ❌ Share a cwd with the operator or another integration if you need isolation. |
| ✅ Use `${KEY}` placeholders for MCP secrets. | ❌ Hardcode credentials in `mcp_servers`. |
| ✅ Validate/sanitize user input before `/input` when `bypass_permissions=true`. | ❌ Treat the agent as a trusted executor; the cwd is not a sandbox. |
| ✅ Store the `odk_live_…` key in a secret manager; rotate on leak. | ❌ Commit or log the plaintext key. |
| ✅ Circuit-break + backoff on `503`/network. | ❌ Expect a `429` / `Retry-After` (they don't exist). |
| ✅ Read back before retrying a create; never blind-retry input. | ❌ Assume `POST /sessions` is idempotent or `POST /input` is dedup'd. |
| ✅ Accept that your sessions are hidden from the operator UI. | ❌ Expect your sessions to appear in the operator console. |
| ✅ Request minimal scopes; handle a future `403` on any endpoint gracefully. | ❌ Rely on M3 scope enforcement as an access-control boundary. |

---

## 21. Versioning of this guide

This guide describes the opendray `/api/v1` integration contract as of the `feat/integration-spawn-profile` line (2026-06-17, M3). It is verified against the current code in `internal/integration`, `internal/session`, `internal/catalog`, and `internal/memory`.

- The API path `/api/v1` is the stable surface; proxied requests carry `X-OpenDray-API: v1`. There is no opendray-version discovery endpoint; the `version` field on an integration is **yours**, not opendray's.
- **Forthcoming and explicitly marked** in this guide: the `permission_mode` reshape of `bypass_permissions` ([§6](#6-the-unified-spawn-profile--configuring-what-runs)), and the `integration:<id>` memory zone (PR #380, [§8](#8-memory--privacy)). Build against the current `bypass_permissions` boolean and `memory_policy=none` + distinct cwds; both forthcoming changes are designed to be backward-compatible.
- **Roadmap (v1.1):** per-route scope enforcement for `session:*` / `channel:*` / `provider:read`. Design your client now to handle `403` on any endpoint without breaking.

When the code changes (new enforced scopes, the `permission_mode` field, the `integration:<id>` zone, or any endpoint addition), this guide MUST be updated in lockstep and re-seeded into the `kb_integrations` knowledge page.
$ODGUIDE$,
    'operator'
)
ON CONFLICT (cwd, kind) DO NOTHING;
