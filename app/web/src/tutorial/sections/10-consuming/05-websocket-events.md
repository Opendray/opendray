# Event subscriptions

opendray publishes lifecycle events on an internal pub/sub bus.
Your integration can subscribe over a single WebSocket and react
in real time — no polling.

## Endpoint

```
GET /api/v1/integrations/_events?topics=<csv>&token=<api_key>
```

Upgrade to WebSocket. Server frames are JSON (one event per
message). Client frames are read for disconnect detection only —
opendray ignores their content.

`topics` is a CSV of glob-style patterns. Each one is checked
against your integration's `event:subscribe:<topic>` scope at
connect time; missing scope → `403 missing scope: event:subscribe:<topic>`.

## Topic catalogue

| Topic | When | Payload |
|---|---|---|
| `session.started` | Session spawned | `{session_id, provider_id, name}` |
| `session.idle` | Stdout silent for `idle_threshold` | `{session_id, last_activity}` |
| `session.activity` | Stdout active again after idle | `{session_id}` |
| `session.stopped` | Operator-initiated stop | `{session_id, ended_at, exit_code}` |
| `session.ended` | Process exited on its own | `{session_id, ended_at, exit_code}` |
| `session.restarted` | Re-spawn after stop/end | `{session_id}` |
| `channel.message_sent` | Outbound to a channel succeeded | `{channel_id, session_id?, level}` |
| `channel.message_forwarded` | Inbound chat reply forwarded to a session | `{channel_id, session_id, bytes}` |
| `channel.command_received` | Slash command from a chat platform | `{channel_id, command, args}` |
| `integration.registered` | New integration row created | `{integration_id, name}` |
| `integration.health_changed` | Reverse-proxy health status changed | `{integration_id, prev, next}` |
| `integration.key_rotated` | Operator rotated this integration's key | `{integration_id, name}` |

Subscribe to wildcards via `.*`:
- `session.*` — all `session.…` topics
- `channel.*` — all `channel.…` topics
- `integration.*` — all `integration.…` topics

The wildcard goes in the scope **and** the topic — `event:subscribe:session.*`
scope authorises subscribing to `topics=session.*`.

## Event frame shape

```json
{
  "topic": "session.started",
  "data": {
    "session_id": "ses_xLG2uLq4mX_X",
    "provider_id": "shell",
    "name": "demo-1777871927588"
  },
  "ts": "2026-05-04T05:18:00.162308+10:00"
}
```

`ts` is the publish timestamp on the gateway, RFC3339. Use it for
ordering even if your client receives frames out of order under
high load.

## Connect lifecycle

```
1. Client opens GET /integrations/_events?topics=session.*&token=…
2. Server validates scope per topic.
3. Server upgrades to WS.
4. Server pushes events for the listed topics until:
     · client closes (any code), or
     · server shuts down (close 1001).
```

There is **no replay** of past events — subscriptions only see
events published after the connect completes. If you missed
something during a reconnect, query the relevant REST endpoint
(`/sessions`, `/integrations`, etc.) to recover state.

## Browser caveat

Browser-side WebSocket can't add headers, so the bearer goes in
the query string. Anyone with shoulder access to the URL can read
the token; serve over HTTPS and treat the URL like a credential.
The Node.js reference client does the same for parity.

## Example (Node.js / `ws`)

```ts
import WebSocket from 'ws'

const url =
  'wss://opendray.example.com/api/v1/integrations/_events' +
  '?topics=session.*,integration.*' +
  `&token=${encodeURIComponent(apiKey)}`

const ws = new WebSocket(url)
ws.on('message', (raw) => {
  const ev = JSON.parse(raw.toString())
  if (ev.topic === 'session.idle') notifyOperator(ev.data.session_id)
})
ws.on('close', (code) => console.log('closed', code))
```

For a fuller reference (subscribe-replay-recover loop, reconnect
backoff, scope check), see [TypeScript SDK](#consuming-typescript-sdk)
and the `client.ts` source it walks through.
