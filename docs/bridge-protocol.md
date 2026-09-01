# Bridge Protocol Specification

> **Version:** 1.0
> **Status:** Implemented (M5)
> **Code:** `internal/channel/bridge/`

The Bridge Protocol lets external messaging adapters (written in any
language) connect to opendray at runtime via WebSocket. This means new
platforms (WeChat, DingTalk, Discord variants, custom chat backends) can
ship as standalone adapter scripts without recompiling the Go binary.

The opendray side ("server") is the `bridge` channel kind. Each `bridge`
channel row in the database holds one adapter slot: name, shared
token, optional capability allow-list. The adapter ("client") opens a
WebSocket, presents the token, declares its capabilities, and is then
treated as a regular `Channel` implementation by the Hub.

---

## 1. Provisioning

In the admin UI (Channels → New channel → kind=bridge):

1. Set a name (`wechat`, `discord-custom`, ...).
2. Copy the auto-generated token (or paste your own).
3. Optionally tick which capabilities the adapter is allowed to claim
   (empty = accept whatever it declares).

The created row is an empty slot. The bridge starts listening for an
adapter connection immediately if `enabled=true`.

---

## 2. Connection

### Endpoint

```
ws://<opendray-host>:<port>/api/v1/channels/bridge/ws
```

The endpoint is **public** (not behind admin bearer auth); the per-bridge
token is the only authentication.

### Authentication

The token must be supplied via one of:

| Method            | Example                                 |
|-------------------|-----------------------------------------|
| Query parameter   | `?token=YOUR_TOKEN`                     |
| Header            | `X-Bridge-Token: YOUR_TOKEN`            |
| Header            | `Authorization: Bearer YOUR_TOKEN`      |
| Inside `register` | `{"type":"register", "token":"..."}`    |

If the token does not match any bridge channel row, the server replies
with a `register_ack{ok:false, error:"invalid token"}` and closes the
connection.

### Lifecycle

```
adapter                                 opendray
  │                                         │
  │── WS upgrade ──────────────────────────▶│
  │── register frame ──────────────────────▶│ (capabilities + platform name)
  │◀── register_ack {ok:true} ──────────────│
  │                                         │
  │◀── send/send_card/send_buttons/... ─────│  (outbound from opendray)
  │── message/card_action ─────────────────▶│  (inbound from adapter)
  │◀── ping (every ~54s) ───────────────────│  (WS ping/pong; adapter must reply)
  │                                         │
  │── (close) ─────────────────────────────▶│
```

Frames are JSON objects, one per WebSocket text frame. Maximum payload
size is 256KB.

---

## 3. Adapter → opendray frames

### `register` (required first frame)

```json
{
  "type": "register",
  "token": "abc123...",
  "platform": "wechat",
  "capabilities": ["text", "card", "buttons", "image", "typing"],
  "metadata": { "version": "1.0.0" }
}
```

| Field          | Type      | Required | Description |
|----------------|-----------|----------|-------------|
| `type`         | string    | yes      | `"register"` |
| `token`        | string    | (yes)    | Bridge token. Optional if supplied via header/query. |
| `platform`     | string    | yes      | Adapter identity (`wechat`, `discord-custom`, ...). |
| `capabilities` | string[]  | yes      | One or more of: `text`, `card`, `buttons`, `image`, `file`, `typing`, `update_message`, `reply_to_message`. The bridge filters this against its `accept_capabilities` allow-list (when set). |
| `metadata`     | object    | no       | Free-form, used only for logs. |

### `message`

User sent a text message.

```json
{
  "type": "message",
  "session_key": "wechat:gid42:user123",
  "conversation_id": "gid42",
  "user_id": "user123",
  "user_name": "Alice",
  "text": "Hello opendray",
  "reply_ctx": "wx-msg-001"
}
```

| Field             | Type   | Required | Description |
|-------------------|--------|----------|-------------|
| `session_key`     | string | rec.     | Stable triple `{platform}:{conversation}:{user}` (the adapter chooses the format). |
| `conversation_id` | string | rec.     | Logical chat identifier. Stored on `channel_messages.conversation_id`. |
| `user_id`         | string | no       | Platform user ID. |
| `user_name`       | string | no       | Display name. Wins over `user_id` for `Author`. |
| `text`            | string | yes      | Message body. |
| `reply_ctx`       | any    | yes      | Opaque routing handle. opendray echoes it back on every outbound frame so the adapter can deliver replies to the right thread. Often a string pointing at the platform's message id. |

### `card_action`

User clicked a button on a card opendray previously sent.

```json
{
  "type": "card_action",
  "session_key": "wechat:gid42:user123",
  "conversation_id": "gid42",
  "user_id": "user123",
  "user_name": "Alice",
  "action": "cmd:/cancel sess1",
  "reply_ctx": "wx-msg-002"
}
```

The `action` is whatever was set as `ButtonOption.Value` on the original
card. Built-in cards produce values like `cmd:/resume <sid>` (slash
command) or `nav:/sessions/<sid>` (UI navigation hint).

opendray's Hub recognises `cmd:/...` actions, including those wrapped
in `act:`, and dispatches them through the slash-command registry.

### `ping`

Application-level keepalive. The server replies with `{"type":"pong"}`.

WebSocket-level ping/pong is also sent automatically by the server every
~54s. Adapters need only respond to `Ping` control frames per RFC 6455
(most WebSocket libraries do this transparently).

---

## 4. opendray → adapter frames

All outbound frames echo the originating `reply_ctx` so the adapter can
route replies. `session_key` mirrors the inbound format.

### `register_ack`

```json
{ "type": "register_ack", "ok": true }
```

On failure: `{"type":"register_ack","ok":false,"error":"invalid token"}`.

### `send`

Plain text outbound.

```json
{
  "type": "send",
  "session_key": "wechat:gid42:user123",
  "conversation_id": "gid42",
  "reply_ctx": "wx-msg-001",
  "text": "Acknowledged."
}
```

### `send_card`

Structured card with markdown + buttons. Only sent when the adapter
declared `card` on register.

```json
{
  "type": "send_card",
  "session_key": "wechat:gid42:user123",
  "conversation_id": "gid42",
  "reply_ctx": "wx-msg-001",
  "card": {
    "header": { "title": "Session idle", "color": "yellow" },
    "elements": [
      { "Content": "Session `abc` went idle (silent for 60s)." },
      {
        "Buttons": [[
          { "text": "Resume", "value": "cmd:/resume abc", "style": "primary" },
          { "text": "End", "value": "cmd:/cancel abc", "style": "danger" }
        ]]
      }
    ]
  }
}
```

The element shape currently mirrors the Go `channel.CardElement` types
verbatim (`CardMarkdown` → `{Content}`, `CardActions` → `{Buttons}`,
`CardListItem` → `{Text, Button}`, `CardSelect` → `{Placeholder,
Options, ...}`, `CardNote` → `{Text}`). Adapters typically translate
the union into their native renderer.

### `send_buttons`

Text plus an inline button row, when the adapter does not implement
full cards.

```json
{
  "type": "send_buttons",
  "session_key": "...",
  "reply_ctx": "...",
  "text": "Pick one:",
  "buttons": [[
    { "text": "Yes", "value": "cmd:/confirm" },
    { "text": "No",  "value": "cmd:/abort" }
  ]]
}
```

### `update_message`

Edit a previously-sent message in place.

```json
{
  "type": "update_message",
  "session_key": "...",
  "conversation_id": "...",
  "preview_handle": "<adapter-supplied-handle>",
  "text": "Updated content"
}
```

The `preview_handle` is whatever the adapter put in
`channel_messages.metadata.preview_handle` (or returned via a future
`message_ack` frame). Today opendray simply uses the upstream platform
message id when known.

### `send_image`, `send_file`

```json
{
  "type": "send_image",
  "session_key": "...",
  "reply_ctx": "...",
  "image": {
    "path": "/abs/local/path.png",   // OR
    "url":  "https://...",
    "caption": "optional"
  }
}
```

`send_file` follows the same shape with a `file` object that may also
carry `filename`.

### `start_typing`, `stop_typing`

```json
{ "type": "start_typing", "session_key": "...", "reply_ctx": "..." }
```

Sent when the agent begins a long-running turn; the matching
`stop_typing` follows when work completes.

### `pong`

Reply to an adapter `ping`.

---

## 5. Capabilities

| Capability         | Required to receive                              |
|--------------------|--------------------------------------------------|
| `text`             | always (implicit)                                |
| `card`             | `send_card`                                       |
| `buttons`          | `send_buttons`                                    |
| `update_message`   | `update_message`                                  |
| `typing`           | `start_typing` / `stop_typing`                    |
| `image`            | `send_image`                                      |
| `file`             | `send_file`                                       |
| `reply_to_message` | (informational; set if adapter honours `reply_ctx` for thread-level replies) |

When the Hub tries a capability the adapter did not claim, the server
returns `channel.ErrNotSupported` to the caller, and the Hub
automatically falls back to a plain `send` frame with text rendered
from `Card.RenderText()`.

---

## 6. Reconnection

The bridge channel keeps its broker registration even after the WS
connection closes. Adapters may freely reconnect with the same token.
A new register frame replaces any prior connection. Outbound frames
attempted while no adapter is attached return `ErrNotSupported`.

---

## 7. Reference adapter (Python pseudocode)

```python
import asyncio, json, websockets

TOKEN = "..."

async def main():
    async with websockets.connect(
        "ws://localhost:8080/api/v1/channels/bridge/ws",
        additional_headers={"X-Bridge-Token": TOKEN},
    ) as ws:
        await ws.send(json.dumps({
            "type": "register",
            "platform": "my-platform",
            "capabilities": ["text", "card", "buttons"],
        }))
        ack = json.loads(await ws.recv())
        assert ack["ok"], ack

        async for raw in ws:
            frame = json.loads(raw)
            if frame["type"] == "send_card":
                # render frame["card"] to your platform's UI
                ...
            elif frame["type"] == "send":
                # plain text
                ...

asyncio.run(main())
```

Sending an inbound message:

```python
await ws.send(json.dumps({
    "type": "message",
    "session_key": f"my-platform:{chat_id}:{user_id}",
    "conversation_id": chat_id,
    "user_id": user_id,
    "user_name": user_name,
    "text": user_text,
    "reply_ctx": platform_message_id,
}))
```
