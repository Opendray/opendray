# Activity — overview

The Activity page is `tail -f` for opendray's **system-wide**
internal event bus. Every session lifecycle event, every channel
inbound, every notification fan-out, every integration call
lands here in real time.

> **Not to be confused with:** the **History** tab on the
> Sessions Inspector, which is a *per-project prompt log* sourced
> from each CLI's transcript on disk. Activity = events;
> History = user prompts.

## When to use

- Debug "why didn't my Telegram channel notify?" — watch
  `session.idle` fire, confirm `channel.message_sent` follows.
- Confirm a slash command was received — look for
  `channel.command_received` with your command name.
- Trace tool-forwarding — `channel.message_forwarded` shows
  channel→session input deliveries.
- Watch the integration call log scroll past in real time.
- Just observe — opendray's behaviour is highly visible from
  this page.

![Activity page](/tutorial/activity-layout.png)

## What you see

Each row is one event:

| Column | Notes |
|---|---|
| Time | server-side timestamp (UTC) |
| Topic | dotted-name event id (`session.idle`, `channel.message_sent`, …) |
| Summary | one-line synthesised from the payload |
| Expand | click row to see the full JSON payload |

The newest event is at the top; auto-scroll to follow can be
toggled (default on). Pause auto-scroll when you want to read a
specific row without it sliding away.

## Filters

The filter bar is client-side — events keep streaming over the
WebSocket regardless. Filtering is by topic prefix:

- `session.` — only session lifecycle events
- `channel.` — channel inbound, outbound, command, forward
- `integration.` — call log + auth attempts
- Any custom prefix you've added via plugins

The filter is sticky per-user; reload the page and your last
filter persists.

## Live count

The bottom bar shows events/second (smoothed over 5s). Useful
to gauge whether anything is happening — silence means the bus
is genuinely quiet, not that the page is broken.

## Read on

| Topic | Section |
|---|---|
| Every event topic + payload shape | Topics catalogue |
