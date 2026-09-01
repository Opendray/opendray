# Feishu (飞书 / Lark)

**Mode:** event webhook (requires public URL) + REST outbound
**Capabilities:** text · card (interactive) · buttons · reply_to_message

## 1. Create an app

1. Visit https://open.feishu.cn/app (or https://open.larksuite.com/app
   for international Lark).
2. **Create a custom app** → enter name + icon → confirm.
3. Sidebar → **Credentials & Basic Info** → copy:
   - **App ID** (starts with `cli_`)
   - **App Secret**

## 2. Add bot capability

Sidebar → **Add features** → **Bot** → enable.

## 3. Configure permissions

Sidebar → **Permissions & Scopes** → enable:

- `im:message` (read messages)
- `im:message:send_as_bot` (send as bot)
- `im:chat` (read chat metadata)
- `im:chat:readonly` (list chats the bot is in)

Apply for permissions, wait for approval (instant for self-built apps
in your own tenant).

## 4. Set up the webhook

Sidebar → **Event Subscriptions** → enable.

You need opendray's **public webhook URL**. Open the channel card in
opendray after creating the channel (next step) and copy the
`webhook:` row. It will look like:

```
https://your-opendray-host/api/v1/channels/<channel_id>/webhook
```

Paste it as the **Request URL** in the Feishu console. Feishu will
immediately fire a verification challenge and opendray echoes it back,
so you should see ✅ in the console.

(Optional) Copy the **Verification Token** shown on the same page and
paste it into opendray's `verification_token` field for additional
request authentication.

Then under **Subscribe to events**, add:

- `im.message.receive_v1` (Bot receives a message)

## 5. Find the chat ID

Easiest path: in the Feishu chat, type `/查询 群ID`(or use the bot's
`/help` once it's running). Or list groups via the API:

```bash
curl -X POST https://open.feishu.cn/open-apis/im/v1/chats \
  -H "Authorization: Bearer <tenant_access_token>"
```

The chat ID looks like `oc_xxxxxxxxxxxxxxxxxxxx`.

## 6. Configure in opendray

Channels → New channel → kind `Feishu`.

- **App ID** + **App Secret** from step 1.
- **Verification token** (optional, recommended) from step 4.
- **Default chat ID** (optional) from step 5.

Save with **Enabled = on**. After save, **open the channel card and
copy the webhook URL** into the Feishu console (step 4) if you have
not already.

## 7. Add the bot to a chat

Feishu chat → ⋯ → **Settings** → **Bots** → **Add bot** → pick yours.

## 8. Test

- Send any message in the chat, and opendray logs it.
- Send `/help` and opendray replies with the command list.
- Trigger session.idle and an interactive card with action buttons
  appears.

## Notes

- Card schema: opendray emits Card v2 (`schema: "2.0"`). Feishu's old
  card v1 is not used.
- AES encryption of event payloads is **not** supported in v1. Leave
  the Lark *Encrypt Key* setting blank, or expect inbound to fail.
  Verification token alone is sufficient for most setups.
- The webhook URL must be HTTPS in production. For local dev behind
  Cloudflare Tunnel, the tunnel's https URL works.
