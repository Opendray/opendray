# Scopes reference

Every integration has a list of scopes attached to its API key.
Scopes are claim-style strings; the gateway middleware checks them
on every request and returns 403 if the requested endpoint isn't
in the integration's set.

The set is fixed at registration time and editable through the
admin UI (Integrations → Edit → Scopes). Live API keys see the
updated scope set on the **next** request after the change — no
restart, no key rotation.

## Choosing scopes for your app

The principle is **least privilege**. Start with the read-only
subset that proves your app works, then widen on demand. opendray's
default scopes for a freshly-registered row are:

```
session:read
event:subscribe:session.*
```

That's enough for a dashboard or monitor; not enough to drive a
session.

## Catalogue

### Sessions

| Scope | Endpoints unlocked |
|---|---|
| `session:read` | `GET /sessions`, `/sessions/{id}`, `/sessions/{id}/buffer`, `/sessions/{id}/stream` (WS), `/sessions/{id}/history` |
| `session:create` | `POST /sessions`, `POST /sessions/{id}/start`, `POST /sessions/{id}/stop`, `DELETE /sessions/{id}`, `PATCH /sessions/{id}/claude-account` |
| `session:input` | `POST /sessions/{id}/input`, `POST /sessions/{id}/resize` |

Common combos:

- **Dashboard** → `session:read` only.
- **Bot that drives Claude** → `session:read` + `session:create` +
  `session:input`.
- **Read-only monitor** → `session:read` + `event:subscribe:session.*`.

### Channels

| Scope | Endpoints unlocked |
|---|---|
| `channel:send` | `POST /channels/{id}/notify`, `/channels/{id}/send` |
| `channel:receive` | `POST /channels/{id}/inbound` (webhook) |

Channel CRUD (registration, deletion) is admin-only — your app
can't manage channels via these scopes.

### Event subscriptions

These are gates on the WebSocket at `/integrations/_events`. Each
topic family has its own scope:

| Scope | Topics |
|---|---|
| `event:subscribe:session.*` | session.started, .idle, .activity, .stopped, .ended, .restarted |
| `event:subscribe:channel.*` | channel.message_sent, .message_forwarded, .command_received |
| `event:subscribe:integration.*` | integration.registered, .health_changed, .key_rotated |

You can also subscribe to a single specific topic by asking for the
exact scope, e.g. `event:subscribe:session.idle` — but in practice
the wildcard variants are easier and still narrow enough.

### Misc

| Scope | Endpoints unlocked |
|---|---|
| `provider:read` | `GET /catalog`, `/catalog/providers`, `/catalog/providers/{id}` |

Useful when your app wants to know which agent CLIs the gateway
host has available before it tries to spawn one.

## What scopes don't cover

- **Admin endpoints** — `/admin/settings`, `/admin/restart`,
  `/integrations` (CRUD), `/auth/login` are gated by **admin
  identity**, not scopes. There's no scope that lets an
  integration impersonate the admin.
- **Reverse-proxy targets** — once your integration is reachable
  at `/api/v1/proxy/<prefix>/*`, callers reach it with **their own**
  bearer (admin or any integration). opendray doesn't do per-target
  scope filtering on the proxy path; your downstream service is
  responsible for whatever finer-grained authz it needs.

## Auditing

Every scope-checked request is logged in the integration call log
(see [Activity → Call log](#integrations-call-log)) with the
attributed integration_id. If you suspect a stolen key, search
for unexpected endpoints and rotate.
