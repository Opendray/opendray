# Consuming opendray — overview

This group is for **third-party developers** building applications
that call opendray's API. If you're an opendray operator wondering
how to manage integrations from the admin UI, see the
[Integrations](#integrations-overview) group instead.

## Mental model

opendray exposes a single, dual-auth REST + WebSocket surface under
`/api/v1/...`:

```
                                ┌──────────────────────────┐
                                │     opendray gateway     │
                                │                          │
   Your app ──── HTTPS/WS ────► │  /api/v1/sessions       │
   (Bearer key)                 │  /api/v1/channels       │
                                │  /api/v1/integrations   │
                                │  /api/v1/integrations/  │
                                │      _events    (WS)     │
                                └──────────────────────────┘
                                          │
                                          ▼
                                ┌──────────────────────────┐
                                │   Backed by:             │
                                │   · PTYs (claude/codex/  │
                                │     gemini/shell)        │
                                │   · channels (Telegram,  │
                                │     Slack, Discord, …)   │
                                │   · postgres (audit log) │
                                └──────────────────────────┘
```

Your app authenticates once, then reads or drives whichever surface
its **scope** allows.

## What you can do

Two complementary modes — pick whichever (or both) fits:

### 1. Consumer mode

Your app **calls** opendray's API. Examples:

- A web dashboard that lists running Claude/Codex/Gemini sessions
- A Slack bot that receives session-idle events and pings the
  operator
- A monitoring service that subscribes to the event bus and pushes
  metrics to Grafana
- A CLI utility that spawns headless agent sessions for batch jobs

**Auth**: integration API key as Bearer token.
**Network**: outbound HTTPS from your app to opendray.

### 2. Reverse-proxy mode

Your app **exposes** an HTTP service that opendray fronts. Other
opendray consumers reach it through `/api/v1/proxy/<your-prefix>/*`.
Examples:

- A custom Anthropic-API tracker
- A receipt-scanning service that takes uploaded files
- A webhook receiver that re-publishes events on opendray's bus

**Auth**: opendray verifies the caller's bearer; your service trusts
the gateway and just serves HTTP.
**Network**: opendray initiates inbound HTTP to your service.

A single integration row can be both at once (set `base_url` +
`route_prefix` to enable proxy; the same `api_key` lets the same
app consume too).

## What's in this group

| Topic | Section |
|---|---|
| 5-minute curl walkthrough | [Quickstart](#consuming-quickstart) |
| Bearer-token plumbing | [Authentication](#consuming-authentication) |
| REST endpoint reference | [REST API](#consuming-rest-api) |
| WebSocket events | [Event subscriptions](#consuming-websocket-events) |
| Scope catalogue | [Scopes reference](#consuming-scopes) |
| Persisting + recovering keys | [Key rotation](#consuming-key-rotation) |
| TypeScript starter | [TypeScript SDK](#consuming-typescript-sdk) |
| Error responses + retries | [Error handling](#consuming-error-handling) |

The whole thing maps onto the
[`examples/integrations/demo-client/`](https://github.com/Opendray/opendray_v2/tree/main/examples/integrations/demo-client)
reference — copy that and edit, or read it alongside.
