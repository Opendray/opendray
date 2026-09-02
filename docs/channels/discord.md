# Discord

**Mode:** Gateway WebSocket (no public URL needed)
**Capabilities:** text · card (embeds) · buttons (components) · update_message · reply_to_message

## 1. Create a Discord application

1. Visit https://discord.com/developers/applications → **New Application**.
2. Name it (e.g. `opendray`).
3. Sidebar → **Bot** → **Add Bot** → confirm.
4. **Reset Token** to reveal the bot token. Copy it (you only see it once).

## 2. Enable required intents

Same Bot page, scroll to **Privileged Gateway Intents**:

- ✅ **Message Content Intent** (required: without this, message
  events arrive with `content=""` and opendray cannot read them).
- ✅ **Server Members Intent** (only needed if you ever want to read
  member info; safe to leave on).

Save changes.

## 3. Invite the bot to your server

1. Sidebar → **OAuth2** → **URL Generator**.
2. Scopes: ✅ `bot`, ✅ `applications.commands`.
3. Bot Permissions:
   - `Send Messages`
   - `Embed Links`
   - `Read Message History`
   - `Use External Emojis` (nice-to-have)
4. Copy the generated URL → open it in a browser → pick the server.

## 4. Find the channel ID

1. Discord → User Settings → Advanced → enable **Developer Mode**.
2. Right-click the target channel → **Copy ID**.

## 5. Configure in opendray

Channels → New channel → kind `Discord`.

- **Bot token:** from step 1.
- **Default channel ID:** from step 4.

Save with **Enabled = on**. The channel goes *running* after the
Gateway handshake completes (`READY` event in logs).

## 6. Test

- Admin **Test** button posts a message to the default channel.
- Send `/help` in any channel where the bot can read messages.
- Trigger session.idle and a coloured embed with Resume/End/Mute
  buttons appears.

## Notes

- Embed colours are 24-bit RGB. Header colour names map to:
  `green→#22c55e`, `red→#ef4444`, `yellow→#eab308`, etc. Custom hex
  not supported (yet).
- `custom_id` on a button is what arrives back as the inbound action.
  The Hub uses values like `cmd:/cancel <sid>`, well below Discord's
  100-char limit.
- The bot token is highly sensitive: if leaked, anyone can act as the
  bot (DM users, edit channels). Reset it from the dev portal if
  exposure is suspected.
