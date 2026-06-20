# Connecting the visayabai bot to opendray

Goal: let the **visayabai bot** drive opendray-managed Claude Code (or
Codex / Antigravity / shell) sessions — send a user's message in, stream the
agent's reply back out — using opendray's **Integrations** system.

> Scope note: this is the *legitimate* integration path — your own bot
> orchestrating your own interactive sessions through opendray's API.
> It is NOT a way to resell Claude-subscription inference to third
> parties (that violates Anthropic's terms). Keep visayabai as a
> front-end you control.

---

## ✅ DECISION (2026-06-17): Bridge channel (Pattern B)

Operator chose the **bridge channel** path. It's the best outcome for a
chat bot: opendray handles session routing, reply detection
(`session.turn_completed`), notification policy, `/new` provider
selection, account switching, and the voice MCP — visayabai just relays
messages in/out. Runnable reference adapter:
**`docs/integrations/visayabai_bridge_adapter.py`**.

### Bridge wire protocol (verified against `internal/channel/bridge`)

- **Connect:** WebSocket to
  `wss://<host>/api/v1/channels/bridge/ws?token=<BRIDGE_TOKEN>`
  (token also accepted via `X-Bridge-Token` / `Authorization: Bearer`
  / or inside the register frame).
- **First frame (adapter → opendray):**
  ```json
  {"type":"register","platform":"visayabai",
   "capabilities":["text","typing"],"metadata":{}}
  ```
  opendray replies `{"type":"register_ack","ok":true}` (or
  `ok:false`+`error`). Capabilities (exact strings): `text`, `card`,
  `buttons`, `image`, `file`, `typing`, `update_message`,
  `reply_to_message`. Declare only what visayabai can render — opendray
  only sends frames for declared caps; everything degrades to `text`.
- **Inbound (adapter → opendray):** a user message →
  ```json
  {"type":"message","conversation_id":"<stable per-user/chat id>",
   "user_id":"...","user_name":"...","text":"hello"}
  ```
  Button taps → `{"type":"card_action","action":"<id>","conversation_id":...}`.
  Adapter ping → `{"type":"ping"}` (opendray replies `{"type":"pong"}`).
- **Outbound (opendray → adapter):** render to the user —
  `{"type":"send",...,"text"}`, `{"type":"start_typing"/"stop_typing"}`,
  `{"type":"send_card",...,"card"}`, `{"type":"send_buttons",...,"buttons":[[…]]}`,
  `{"type":"send_image"/"send_file",...}`, `{"type":"update_message",...}`.
- **`conversation_id` is the routing key** — use a stable id per
  visayabai user/chat. opendray binds each `conversation_id` to its own
  session, so N users → N independent Claude sessions automatically.

### Operator setup (one-time)

1. opendray → **Channels** → add a channel of kind **bridge**: set a
   `name` (`visayabai`) and a shared `token`. Bind it to a session (or
   let `/new` create one).
2. Put that token in visayabai's env as `BRIDGE_TOKEN`, point the
   adapter at the gateway host, run `visayabai_bridge_adapter.py`.
3. Optional: add `buttons`/`card` to the adapter's capabilities later to
   get the rich control keyboard + provider picker instead of text
   fallback.

---

## Reference: the two ways to connect (B chosen above)

| | **A. Consumer integration** (recommended for a custom app) | **B. Bridge channel** (recommended if visayabai is "just another chat surface") |
|---|---|---|
| What it is | The bot holds an opendray API key and calls the REST/WS API directly to create + drive sessions. | The bot registers as a `kind=bridge` channel; opendray's channel machinery handles session binding, notifications, and replies (same as Telegram/Slack). |
| Bot owns | Session lifecycle + routing (full control). | Just message relay in/out. |
| Effort | More code, more control. | Less code, reuses reply/notify logic. |
| Mount | `/api/v1/sessions/*` + `/api/v1/integrations/_events` | Public webhook `/api/v1/channels/<id>/inbound` + `channel:send` |

The rest of this doc walks **Pattern A** in full (matches the
Integrations UI in the screenshots), then summarizes **Pattern B**.

---

## Pattern A — Consumer integration

### A1. Register the integration (operator, one-time, in the web UI)

opendray → **Integrations** → **Register**. A *consumer* integration
has **no Base URL** and **no route prefix** (those are only for the
reverse-proxy direction). Give it:

- **Name:** `visayabai`
- **Base URL:** *(leave blank — consumer, not reverse-proxied)*
- **Scopes** (minimum to drive a chat→Claude→chat loop):
  - `session:read` — list/get sessions, read buffer, fetch history
  - `session:create` — spawn / restart / delete sessions
  - `session:input` — **required** to forward messages into the PTY
  - `event:subscribe:session.*` — live-tail `session.idle` /
    `session.turn_completed` / `session.ended` so the bot knows when a
    reply has settled
  - `provider:read` *(optional)* — list providers (claude/codex/…)

On save, opendray shows the **API key once** — format
`odk_live_<...>`. Copy it into visayabai's secret store (env var, not
source). It's bcrypt-hashed server-side and never shown again; use
**Rotate key** to issue a new one.

> Registration itself is admin-only (`POST /api/v1/integrations` needs
> the admin token), so the operator does this in the UI and hands the
> key to the bot. The bot never self-registers.

### A2. Authenticate every request

Send the key as a Bearer token (or `?token=` for WebSocket URLs, since
browsers/WS can't set headers):

```
Authorization: Bearer odk_live_xxxxxxxx
```

Base path for everything below: `https://<opendray-host>/api/v1`.

### A3. The drive loop (REST + WS)

**1. Create a session** (one per conversation, or reuse one):

```bash
curl -sX POST https://HOST/api/v1/sessions \
  -H "Authorization: Bearer $OPENDRAY_KEY" \
  -H 'Content-Type: application/json' \
  -d '{
        "name": "visayabai · user-123",
        "provider_id": "claude",
        "cwd": "/var/lib/opendray/projects/visayabai",
        "args": []
      }'
# → 201 { "id": "ses_...", "state": "running", "pid": ..., ... }
```

Optional: pin a specific Claude account with
`"claude_account_id": "cla_..."`.

**2. Send the user's message into the PTY:**

```bash
curl -sX POST https://HOST/api/v1/sessions/$SES/input \
  -H "Authorization: Bearer $OPENDRAY_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"data": "summarize the latest sales report\n"}'
# → 204 No Content
```

`data` is raw bytes written to stdin. Include the trailing `\n` to
"press enter". You can send control bytes too (e.g. `` = Ctrl-C).

**3. Read the reply.** Two options:

- **WebSocket stream (live PTY bytes):**
  `wss://HOST/api/v1/sessions/$SES/stream?token=$OPENDRAY_KEY`
  On connect it replays the recent buffer, then streams live output as
  binary frames. Send binary frames back to write to stdin (alternative
  to the `/input` POST). This is the raw terminal stream (ANSI included)
  — strip ANSI for a chat surface.

- **Event bus (know when the turn is done):**
  `wss://HOST/api/v1/integrations/_events?topics=session.*&token=$OPENDRAY_KEY`
  JSON frames like:
  ```json
  { "topic": "session.turn_completed",
    "ts": "2026-06-17T...Z",
    "data": { "session_id": "ses_...", "recent_output": "..." } }
  ```
  `session.turn_completed` fires a few seconds after the agent stops
  emitting — the right signal to grab `recent_output` and post it back
  to the visayabai user, and to stop any "typing…" indicator.

**Recommended bot pattern:** open the `_events` WS once (not per
session), filter by `session_id`. On a user message → `POST /input`.
On `session.turn_completed` for that session → read `recent_output`
(or `GET /sessions/{id}/buffer`), strip ANSI, send to the user. Use
`session.idle` as a "still thinking / nudge" cue and `session.ended`
to recreate a session if the CLI exited.

### A4. Useful extras

- `GET /api/v1/sessions` — list (filter to visayabai's own by name).
- `GET /api/v1/sessions/{id}/buffer?since=<cursor>` — pull output
  since a byte cursor (headers `X-OpenDray-Buffer-Cursor`) instead of
  the WS, if the bot prefers polling.
- `GET /api/v1/sessions/{id}/history?limit=N` — past user prompts.
- `POST /api/v1/sessions/{id}/stop` / `POST /.../start` /
  `DELETE /.../{id}` — lifecycle.
- `POST /api/v1/sessions/{id}/uploads` — attach a file (multipart);
  returns a server path to reference in a prompt.

---

## Pattern B — Bridge channel (alternative)

If visayabai is fundamentally a chat surface (user texts, Claude
replies) and you'd rather not write the session-routing yourself, model
it on the built-in channels (Telegram/Slack/Discord) via the **bridge
adapter** (`kind=bridge`, registered in opendray's channel system).

- Inbound: visayabai forwards user messages to opendray's public
  webhook `POST /api/v1/channels/<id>/inbound` (needs `channel:receive`
  to verify traffic). opendray binds the message to the channel's
  active session and writes it to the PTY.
- Outbound: opendray pushes replies/notifications back out through the
  registered channel; the bot relays them to the user (needs
  `channel:send`).

This reuses opendray's reply detection, notification policy, and
`/new`-style session management for free — at the cost of less
low-level control than Pattern A. The bridge adapter WS endpoint is
mounted publicly (`bridgeHandlers.Mount`, see `internal/channel/bridge`).

---

## Reverse-proxy direction (for completeness — NOT this use case)

The "Base URL + route prefix" fields you saw on the PDAweb integration
are the *other* direction: opendray reverse-proxies
`/api/v1/proxy/<prefix>/*` **to** the integration's Base URL (opendray
→ your app). Use that when you want opendray to be the authenticated
front door to a service. visayabai driving sessions does **not** need
it — leave Base URL blank.

---

## Quick reference

| Thing | Value |
|---|---|
| Base path | `https://<host>/api/v1` |
| Auth | `Authorization: Bearer odk_live_...` (or `?token=` on WS) |
| Register (admin/UI) | `POST /integrations` → returns key once |
| Create session | `POST /sessions` `{provider_id, cwd, name, args}` |
| Send input | `POST /sessions/{id}/input` `{"data":"...\n"}` → 204 |
| Live output WS | `wss://…/sessions/{id}/stream?token=…` (binary) |
| Events WS | `wss://…/integrations/_events?topics=session.*&token=…` |
| Turn-done signal | event topic `session.turn_completed` → `data.recent_output` |
| Min scopes | `session:read`, `session:create`, `session:input`, `event:subscribe:session.*` |

## Open questions to resolve next session

- Is visayabai a chat relay (→ lean Pattern B / bridge channel) or a
  full app needing programmatic control (→ Pattern A / consumer)?
- One long-lived session per user, or ephemeral per-conversation?
- Where should session `cwd` point for visayabai's work?
- Does it need `channel:send`/`channel:receive` (only if Pattern B)?

## Source references (verified in repo)

- Integration model + register: `internal/integration/integration.go`,
  `internal/integration/service.go`, `internal/integration/handler.go`
- Auth + scope enforcement: `internal/integration/middleware.go`
  (`CombinedMiddleware`, `bearerFromRequest`), `service.go` (`HasScope`)
- Session routes: `internal/session/handler.go` (`Handlers.Mount`)
- Events WS: `internal/integration/events.go`,
  `internal/app/app.go:759` (`/integrations/_events`)
- Reverse proxy: `internal/integration/proxy.go`
  (`/proxy/{prefix}/*`)
- Scope catalog (UI strings): web bundle `QO{...}` —
  session:read/create/input, channel:send/receive,
  event:subscribe:{session,channel,integration}.*, provider:read,
  memory:read/write
