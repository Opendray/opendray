# Notifications panel deep-dive

Every non-bridge channel has the same Notifications panel inside
the Edit dialog. It controls *which* events fire notifications,
*how often*, and *what content* gets included.

![Notifications panel](/tutorial/notifications-panel-detail.png)

## Notify on (topic checkboxes)

Three session-level events the channel can subscribe to:

- **`session.started`** — fires when a session is spawned
- **`session.idle`** — fires when a session has been silent for
  the configured threshold (default 30s)
- **`session.ended`** — fires when a session's process exits

Click any tag to toggle. The default state is *all three selected*
(equivalent to "any topic" — opendray omits the field from config
so future-added topics also flow through).

When you uncheck topics, only the selected ones notify. Uncheck
all and the channel goes silent for outbound (it still receives
inbound replies).

## Repeat policy

The most-asked-about control. Three modes:

### Once per session (default, recommended)

Fire one notification per `(channel, topic, session)` triple.
After the first idle notification for session A on channel X,
opendray suppresses further `session.idle` events for that triple
**until one of these happens:**

- The session ends (different topic — that fires `session.ended`)
- You reply to the channel with non-command text (forwards to
  the session's stdin and resets the suppression)
- The 24-hour safety TTL elapses

This is what you want for solo use. CLI tools that emit periodic
output (Claude's "thinking" spinner, code-streaming) cause many
active→idle transitions per actual conversational turn — *Once*
mode collapses them into one ping per turn.

### Time-window cooldown

Suppress repeats for the chosen window:

| Window | Behaviour |
|---|---|
| 1 minute | Aggressive — most edge-case re-notifications get through |
| 5 minutes | Reasonable for "long-running deploys" |
| 15 minutes | "Tell me again if it's still idle" |
| 30 / 60 minutes | Heartbeat-style |

Use this when you want periodic check-ins on a long-running
session rather than a single ping.

### Every event (noisy)

No suppression at all. Use only on channels with low natural
volume (a single per-day notification channel) or when debugging.

## Terminal snippet

When enabled (default), the idle notification embeds the recent
output. Two source paths:

- **Claude sessions:** opendray reads `~/.claude/projects/<encoded-cwd>/<latest>.jsonl`
  and renders the last conversation turn (assistant text + tool
  calls + their tool results) — clean, complete, no TUI artefacts.
- **Other CLIs:** opendray maintains a virtual terminal (vt10x)
  alongside the raw PTY stream. The snippet is what the user
  would see in the live web terminal *right now*, with Claude
  TUI chrome (model bar, "bypass permissions" hint, status
  spinners, separator runs) stripped via regex.

### Snippet cap

Default is *No cap* — Telegram automatically chunks long content
into multiple messages with the action buttons attached to the
last chunk. Other channels apply their own platform-specific
sizing (DingTalk's 20 KB limit etc.).

If you specifically want shorter notifications (e.g. on a noisy
channel where you skim), pick 1000 / 3000 / 6000 / 12000 chars.
Trimmed content shows a `[…]` prefix marker.

## Per-platform snippet rendering

| Platform | How the snippet renders |
|---|---|
| Telegram | HTML mode — bold/italic/code/blockquote/headings work; tables become "Header: value" vertical blocks |
| Slack | Block Kit `mrkdwn` — `*bold*` (single asterisk), no headings |
| Discord | Embed `description` field, capped at 4096 chars |
| Feishu | Card v2 markdown with `lark_md` content type |
| DingTalk | actionCard markdown |
| WeCom | Markdown with WeCom's limited subset |
| Bridge | Whatever the adapter decided to render |

## Mute toggle

Inside any channel chat, run `/notify off` — opendray sets a
`muted: true` flag in the channel config. The dispatch loop skips
muted channels entirely. `/notify on` clears it.

This is a faster path than opening the admin UI when you just want
to silence one channel for the day.
