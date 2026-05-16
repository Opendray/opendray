# Tutorial screenshots

Drop PNG / JPG files here. The Tutorial page references them as
`/tutorial/<filename>` (Vite serves `public/` at the site root).

When an image referenced by a tutorial markdown file is missing,
the page renders a placeholder card showing the alt text and the
expected path so half-finished sections still read sensibly.

## Currently referenced (34 files)

### Overview
- `sidebar-overview.png` — sidebar nav with all routes
- `channels-running.png` — Channels page with one running telegram bot

### Sessions group
- `sessions-layout.png` — full Sessions page (tabs + terminal + inspector)
- `spawn-dialog.png` — New session dialog
- `sessions-inspector.png` — Inspector panel sub-tabs (Outline / Notes / Context / Activity)
- `sessions-tab-strip.png` — multi-tab strip with running + ended tabs

### Channels — per platform
- `telegram-botfather-token.png` — BotFather chat showing the token reveal
- `telegram-new-channel.png` — opendray Create channel form filled for telegram
- `slack-app-create.png` — api.slack.com/apps "Create New App" flow
- `slack-channel-id.png` — Slack channel details panel showing Channel ID
- `discord-token.png` — Discord developer portal Bot tab with Reset Token
- `discord-card-buttons.png` — Discord embed with action row buttons
- `feishu-credentials.png` — Feishu open platform Credentials & Basic Info
- `feishu-channel-webhook.png` — opendray channel card displaying the webhook URL row
- `dingtalk-robot-create.png` — DingTalk group robot create dialog with Sign secret
- `wecom-robot-url.png` — WeCom group robot URL reveal
- `bridge-create-form.png` — Bridge channel create form (token + capabilities)
- `bridge-adapter-setup.png` — Bridge adapter setup dialog with Python/Node tabs

### Channels — notifications + routing
- `notifications-panel-detail.png` — Notifications panel inside Edit dialog
- `routing-reply-to-message.png` — Telegram long-press reply demonstration

### Providers group
- `providers-layout.png` — Providers page
- `providers-claude-accounts.png` — Claude accounts panel below the provider list

### Integrations group
- `integrations-layout.png` — Integrations page
- `integration-key-reveal.png` — modal showing a freshly-created integration key
- `integrations-proxy-mount.png` — reverse-proxy mount form

### Activity group
- `activity-layout.png` — Activity event tail

### Notes group
- `notes-layout.png` — Notes page
- `notes-wiki-link-suggest.png` — wiki-link suggestion popup
- `notes-source-preview.png` — Source / Preview tabs on the editor

### Plugins group
- `plugins-layout.png` — Plugins page
- `plugins-mcp-add.png` — MCP server registration form

### Settings group
- `settings-layout.png` — Settings page
- `settings-shortcuts.png` — keyboard shortcuts editor

## Capture tips

- Use **Cmd+Shift+5** (macOS) for region screenshots; pick "Selected
  Window" for clean borders.
- Crop tightly — wide screenshots scroll on the tutorial page.
- Dark mode if your admin runs in dark mode. The tutorial doesn't
  re-tint images.
- Anonymise tokens / chat ids: BlurMate or paint over them with a
  dark rectangle in Preview before saving.
- Recommended file size ≤ 500 KB per screenshot — the build embeds
  them in the prod bundle when served from `public/`.

## Quick capture workflow (macOS)

1. Open the relevant admin page on the live opendray.
2. **Cmd+Shift+5** → *Selected Window* → click the window → it
   saves to Desktop with a `Screen Shot ...` filename.
3. Drop into Preview, crop with **Cmd+K** if needed.
4. Rename to one of the filenames above (matches what the
   tutorial expects).
5. Move into `app/web/public/tutorial/`.
6. Reload the Tutorial page — the placeholder swaps for the
   real image.
