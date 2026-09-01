# Slack

**Mode:** Socket Mode (no public URL needed)
**Capabilities:** text · card (blocks) · buttons · update_message · reply_to_message

## 1. Create a Slack app

1. Visit https://api.slack.com/apps → **Create New App** → *From scratch*.
2. Name it (e.g. `opendray`), pick the target workspace.

## 2. Enable Socket Mode

1. Sidebar → **Socket Mode** → toggle on.
2. It will prompt to create an **App-Level Token**:
   - Name: `opendray-socket`
   - Scope: `connections:write`
   - Generate. Copy the **xapp-…** token (this is the *App-Level Token*).

## 3. Add bot scopes

Sidebar → **OAuth & Permissions** → **Bot Token Scopes** → add at
least:

- `chat:write` (post messages)
- `channels:history` + `groups:history` (receive messages from channels the bot is in)
- `im:history` (receive DMs)
- (optional) `chat:write.public` to post into channels the bot isn't a member of

Then **Install to Workspace**. Copy the **Bot User OAuth Token** (`xoxb-…`).

## 4. Subscribe to events

Sidebar → **Event Subscriptions** → toggle on.

Under **Subscribe to bot events** add:

- `message.channels`
- `message.groups`
- `message.im`

(Socket Mode delivers these over the WebSocket, so there's no Request URL to enter.)

## 5. Enable interactivity

Sidebar → **Interactivity & Shortcuts** → toggle on. (No URL needed in
Socket Mode.) This is required for button clicks to flow back.

## 6. Invite the bot

In Slack: `/invite @opendray` in any channel where you want the bot to
post.

Right-click the channel → *Copy link* → the trailing `/archives/Cxxxx`
segment is the **Channel ID**. Or use `/copy channel id` if your
workspace has the *Slack Channel ID* shortcut enabled.

## 7. Configure in opendray

Channels → New channel → kind `Slack`.

- **Bot token (xoxb-…):** from step 3.
- **App-level token (xapp-…):** from step 2.
- **Default channel ID:** from step 6.

Save with **Enabled = on**. The channel card shows *running* once the
Socket Mode WS handshakes (look for `slack socket-mode connected` in
the server log).

## 8. Test

- Admin **Test** button posts to the default channel.
- DM the bot `/help`: opendray replies in-thread.
- Trigger session.idle: a card with Resume/End/Mute appears in the
  default channel.

## Notes

- Slack mrkdwn ≠ standard Markdown: `*bold*`, `_italic_`,
  `<https://link|label>`. Headings (`#`) and tables don't render.
  Cards use the Block Kit `mrkdwn` text type, so the same conventions
  apply.
- Only members of a channel see the bot's messages. For a "broadcast"
  channel, invite the bot first.
