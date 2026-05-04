# REST API reference

This is the consumer-facing endpoint surface — everything an
integration's API key can drive. Endpoints not listed here are
admin-only.

All paths are relative to the `/api/v1` base. All responses are
JSON unless noted.

## Auth

| Method | Path | Purpose | Body shape |
|---|---|---|---|
| POST | `/auth/login` | Mint admin token | `{username, password}` → `{token, expires_at, username}` |

## Sessions

| Method | Path | Scope | Notes |
|---|---|---|---|
| GET    | `/sessions` | `session:read` | List all sessions in any state. Returns `{sessions: [...]}` |
| GET    | `/sessions/{id}` | `session:read` | Single session view |
| POST   | `/sessions` | `session:create` | Spawn. Body: `{name, provider_id, cwd, args?, claude_account_id?}` |
| POST   | `/sessions/{id}/start` | `session:create` | Re-spawn an ended/stopped row |
| POST   | `/sessions/{id}/stop` | `session:create` | SIGTERM the PTY but keep the row |
| DELETE | `/sessions/{id}` | `session:create` | Stop + delete the row |
| POST   | `/sessions/{id}/input` | `session:input` | Body: `{data: string}`. Bytes go straight into PTY stdin |
| POST   | `/sessions/{id}/resize` | `session:input` | Body: `{cols, rows}` |
| GET    | `/sessions/{id}/buffer?since=N` | `session:read` | Raw bytes from the PTY ring buffer |
| GET    | `/sessions/{id}/stream` | `session:read` | WS upgrade — live stdout |
| GET    | `/sessions/{id}/history?limit=N` | `session:read` | Project-level prompt history (Claude/Codex/Gemini only) |
| PATCH  | `/sessions/{id}/claude-account` | `session:create` | Switch the Claude account binding |

### `provider_id` values

| Value | Spawns |
|---|---|
| `shell` | The user's `$SHELL` (zsh, bash, …) |
| `claude` | Claude Code CLI |
| `codex` | OpenAI Codex CLI |
| `gemini` | Google Gemini CLI |

For Claude/Codex/Gemini opendray has to know the CLI is installed
on the gateway host. See [Providers](#providers-overview) for
discovery + per-provider catalog.

## Channels

| Method | Path | Scope | Notes |
|---|---|---|---|
| GET    | `/channels` | (none — admin-only listing) | Listed for completeness — channel CRUD is admin-only |
| POST   | `/channels/{id}/notify` | `channel:send` | Body: `{title, body, level?}` |
| POST   | `/channels/{id}/send` | `channel:send` | Send a free-form message |
| POST   | `/channels/{id}/inbound` | `channel:receive` | Webhook receiver (provider-specific signature verification) |

## Catalog + providers

| Method | Path | Scope |
|---|---|---|
| GET | `/catalog` | `provider:read` |
| GET | `/catalog/providers` | `provider:read` |
| GET | `/catalog/providers/{id}` | `provider:read` |

## Integrations (admin-only — listed for reference)

| Method | Path | Notes |
|---|---|---|
| GET    | `/integrations` | List own + all when admin |
| POST   | `/integrations` | Register, returns plaintext key |
| GET    | `/integrations/{id}` | One row |
| PATCH  | `/integrations/{id}` | Edit `base_url`/`scopes`/`version`/`enabled` |
| DELETE | `/integrations/{id}` | Drop row |
| POST   | `/integrations/{id}/rotate-key` | New key, old invalidated |

Note that you cannot manage your **own** integration through these
endpoints with your integration key — admin only. Have your
operator do it through the web UI.

## WebSockets

| Path | Scope | Purpose |
|---|---|---|
| `/sessions/{id}/stream` | `session:read` | Live PTY stdout (binary frames) |
| `/integrations/_events?topics=…` | `event:subscribe:<topic>` per topic | Live event bus |

See [Event subscriptions](#consuming-websocket-events) for the WS
event contract.

## Reverse-proxy mode

If your integration is registered with `base_url` + `route_prefix`,
**any** path under `/api/v1/proxy/<route_prefix>/*` is forwarded
to your service after auth + scope checks. Path stripping leaves
the request path inside your service unprefixed.

Example: integration `route_prefix=pet-tracker`,
`base_url=http://192.168.3.42:8080`. Caller hits
`/api/v1/proxy/pet-tracker/api/dogs?breed=corgi` → your service
sees `GET /api/dogs?breed=corgi`.

## Response envelopes

opendray doesn't wrap successful responses in a generic envelope —
the body **is** the resource (or `{<key>: [array]}` when listing).
Errors always come back as:

```json
{ "error": "human-readable reason" }
```

paired with an HTTP status code. `4xx` = caller's fault, `5xx` = a
gateway fault (or a downstream service in proxy mode).
