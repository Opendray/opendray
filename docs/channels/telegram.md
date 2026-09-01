# Telegram

**Mode:** long-poll (no public URL needed)
**Capabilities:** text · card · buttons · update_message · typing · reply_to_message

## 1. Create a bot

1. In Telegram, message [@BotFather](https://t.me/BotFather).
2. Send `/newbot`, follow prompts, choose a name + username.
3. BotFather returns a **bot token** like `123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11`.
4. (Optional) `/setjoingroups Enable` if you want the bot in groups.

## 2. Find a chat ID

The simplest path:

1. Add the bot to a group (or DM it directly).
2. Send any message in the chat.
3. Hit `https://api.telegram.org/bot<TOKEN>/getUpdates` in a browser.
4. The JSON response includes `chat.id` (positive for DMs, negative for groups). Copy it.

## 3. Configure in opendray

Channels → New channel → kind `Telegram`.

- **Bot token:** the BotFather token from step 1.
- **Default chat ID:** from step 2 (optional, used as the destination
  for outbound notifications when no `ReplyCtx` is set).

Save with **Enabled = on**. The channel card flips to *running* once
the long-poll loop starts (a few seconds).

## 4. Test

- The admin **Test** button sends `OpenDray channel test ✓` to the
  default chat.
- Send `/help` to the bot, and opendray replies with the registered
  command list.
- Trigger a session.idle event, and you should see a card with Resume /
  End / Mute buttons.

## Notes

- Inline-keyboard `callback_data` is capped at 64 bytes by Telegram.
  The Hub's command system uses payloads like `cmd:/cancel <session-id>`
  which fit comfortably.
- Telegram has no built-in slash command registration, so `/help` etc.
  work as plain text. To get autocomplete, run the BotFather
  `/setcommands` flow with the same list opendray reports for `/help`.
