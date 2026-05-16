# Welcome to OpenDray

OpenDray is a self-hosted control plane for AI coding CLIs (Claude Code,
Codex, Gemini CLI, …). It lets you spawn long-running CLI sessions on
your server, drive them from any device, and bridge them to messaging
platforms like Telegram or Discord so an idle session can ping you when
it needs attention.

## What this admin lets you do

| Page | What it's for |
| --- | --- |
| **Sessions** | Spawn / attach to / drive CLI sessions in a multi-tab terminal. The day-to-day workbench. |
| **Channels** | Wire up Telegram / Slack / Discord / Feishu / DingTalk / WeCom so you get notified when sessions go idle, and reply from your phone. |
| **Providers** | Configure CLI providers (Claude, Codex, …) — paths, env vars, default args. |
| **Integrations** | Expose opendray itself as an HTTP gateway to third-party tools (managed reverse proxy, signed integration tokens). |
| **Activity** | Live tail of every event on the bus — useful for debugging notification flow, channel inbound, etc. |
| **Notes** | Obsidian-compatible markdown vault with wiki-link backlinks. |
| **Plugins** | Skills and MCP server registries — the tool catalogue your sessions can invoke. |
| **Settings** | Auth, theme, keyboard shortcuts. |

![Sidebar overview](/tutorial/sidebar-overview.png)

## First-run checklist

If you've just installed opendray, do these in order:

1. **Log in** — admin credentials live in `config.toml` or env vars.
2. **Configure a Provider** — at minimum, point `claude` at the
   binary path. Without this, the spawn dialog has nothing to launch.
3. **Spawn your first session** — open Sessions → **New session** →
   pick a provider and working directory.
4. **Wire up a channel** (optional) — most users register a Telegram
   bot first since it works without a public URL.

The rest of this guide walks through each page in detail. You can
read it linearly or jump from the table of contents on the left.
