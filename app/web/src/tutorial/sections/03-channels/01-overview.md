# Channels — overview

A *channel* is one configured messaging integration. Every channel
follows the same lifecycle:

1. **Provision credentials** in the platform's admin console (bot
   token / OAuth scopes / webhook URL / etc.).
2. **Channels → New channel** in opendray, paste the credentials.
3. **Wait for `running`** in the card status pill.
4. (Optional) **Edit** the Notifications panel to tune mode, topics,
   and snippet cap.
5. (Optional) For multi-platform setups, repeat with a different kind.

Each platform has its own setup section below — read the one matching
your destination chat. The notifications panel and routing rules are
shared, so once you've wired up one channel the rest is identical.

## Bundled platforms

| Kind | Inbound | Outbound | Public URL? | Best for |
|---|---|---|---|---|
| `telegram` | long-poll | REST | no | solo dev — fastest setup |
| `slack` | Socket Mode | Web API + blocks | no | team chat, native interactivity |
| `discord` | Gateway WS | REST + embeds | no | dev/maker community |
| `feishu` | webhook | tenant API | **yes** | China / cross-org formal channels |
| `dingtalk` | (none) | group robot | no | China enterprise group rooms |
| `wecom` | (none) | group robot | no | WeCom (企业微信) group rooms |
| `bridge` | WebSocket | WebSocket | no (token-auth) | custom platforms (Line / KakaoTalk / your own) |

## Capability comparison

| Capability | telegram | slack | discord | feishu | dingtalk | wecom | bridge |
|---|---|---|---|---|---|---|---|
| Receive user replies | ✓ | ✓ | ✓ | ✓ | — | — | ✓ |
| Markdown body | ✓ HTML | ✓ blocks | ✓ embed | ✓ card | ✓ md | ✓ md | adapter |
| Inline buttons | ✓ | ✓ | ✓ | ✓ | nav-only | nav-only | adapter |
| Reply-to-message routing | ✓ | ✓ thread | ✓ ref | ✓ reply | — | — | adapter |
| Edit-in-place updates | ✓ | ✓ | ✓ | partial | — | — | adapter |

"nav-only" = group robots can't fire callback buttons, but URL links
still render as tappable rows.

## Where to next

- Pick your platform from the TOC: Telegram / Slack / Discord /
  Feishu / DingTalk / WeCom / Bridge — each has its own setup
  section.
- After at least one channel is running, read **Notifications
  panel** (mode, topics, snippet cap) and **Multi-session routing**
  (reply-to-message, `/select`, `/sessions`) — they apply to every
  channel.

![Channels page with one running telegram bot](/tutorial/channels-running.png)
