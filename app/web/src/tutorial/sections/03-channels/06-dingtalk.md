# DingTalk (钉钉)

**Mode:** custom group robot (outbound only)
**Capabilities:** text · card (markdown / actionCard) — no callback buttons
**Setup time:** ~3 minutes

DingTalk's group robot is the simplest way to push notifications
into a chat. It's outbound-only — no inbound, no buttons that fire
callbacks — but pairs well with the *Notify on session.idle* +
*Once per session* mode for "tell me when work is done" alerts.

## When to use DingTalk vs other Chinese platforms

| Need | Use |
|---|---|
| Just notifications, no replies | DingTalk group robot |
| Replies / interactive buttons | Feishu or bridge |

## 1. Add a custom robot to a group

1. Open the target DingTalk group → ⋯ → **Group settings** →
   **Group bots** → **Add Robot**.
2. Pick **Custom (自定义)**.
3. Name it (e.g. `OpenDray`).
4. **Security settings** — pick at least one. Strongly recommend
   **Sign (加签)**:
   - DingTalk generates a `SEC...` secret.
   - opendray automatically appends `&timestamp=...&sign=...` to
     every webhook call so DingTalk accepts it.
5. Click **Done**. DingTalk reveals the **Webhook URL**:
   ```
   https://oapi.dingtalk.com/robot/send?access_token=abc123...
   ```

![DingTalk robot create](/tutorial/dingtalk-robot-create.png)

The other security options:

- **Custom keyword (关键词)** — every message must contain a fixed
  substring or DingTalk drops it. Less convenient (every
  notification needs to include the keyword).
- **IP allow-list** — restricts by source IP. Useful when
  opendray runs on a fixed egress IP.

## 2. Configure in opendray

Channels → **New channel** → kind **DingTalk (钉钉)**.

| Field | Value |
|---|---|
| **Webhook URL** | from step 1 |
| **Sign secret** | the `SEC...` value (only when *Sign* mode is selected in DingTalk) |

Save with **Enabled = on**.

## 3. Verify

- Hit **Test** on the card → plain-text message appears in the
  group.
- Trigger a session.idle event (let a session sit for 30s) → an
  *actionCard* with the title + markdown body lands. Buttons whose
  value is a clickable URL render as buttons; `cmd:` callbacks are
  dropped silently because group robots can't fire callbacks.

## Card rendering

opendray's Card → DingTalk message:

| Card element | DingTalk |
|---|---|
| `CardHeader.Title` | actionCard `title` |
| `CardMarkdown` | actionCard `text` (markdown) |
| `CardActions` with URL values | actionCard `btns` (each with `actionURL`) |
| `CardActions` with `cmd:` values | dropped |
| `CardDivider` / `CardNote` | inline markdown `---` / `> blockquote` |

When the card has no URL buttons, opendray downgrades to a plain
`markdown` message instead of `actionCard`.

## Limitations

- **Outbound only.** To receive replies you need the
  app-platform setup (corp_id + agent_id + secret + AES-encrypted
  callback URL) which is not yet implemented. For bidirectional
  DingTalk use a bridge channel + a Python adapter you write
  against the App Platform SDK.
- **Rate limit:** 20 messages/min per robot. Bursty notifications
  on a chatty session can hit this — the *Once per session* mode
  default keeps you well under it.
- **Payload size:** ~20 KB. Long Claude responses are still chunked
  client-side but this is closer to the cap than Telegram's 4 KB.
- **Sign timestamp tolerance:** ±1 hour. The host clock must be
  roughly NTP-synced.
