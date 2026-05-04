# Session defaults

Two knobs control session lifecycle behaviour at the global
level. Both live under `[session]` in `config.toml` and are
editable in **Settings → Server → Sessions**. UI changes write
back to `config.toml` but require a **Restart** to take effect
(the idle detector reads config once at startup).

## Idle threshold

How long stdout-silent before opendray flips a session to
`IDLE` and publishes `session.idle`.

| Field | Default | Range |
|---|---|---|
| `idle_threshold` | `30s` | `5s` – `5m` |

Lower = more notifications (every short pause counts).
Higher = miss short "I just paused to think" pauses, only get
notified for genuinely abandoned sessions.

For Claude Code: `30s` is a sweet spot — long enough that
streaming output (which has small inter-byte gaps) doesn't
cause spurious idle flaps, short enough to alert you within a
minute when the assistant is actually waiting.

For plain shell: `60s`+ is more reasonable since shell prompts
sit silent for far longer than chat.

## Idle watcher interval

How often the per-session goroutine polls for idleness. Lower =
detect transitions faster (better UX); higher = less CPU per
session.

| Field | Default | Range |
|---|---|---|
| `idle_interval` | `5s` | `1s` – `30s` |

Default is fine for most setups. If you're running 100+
sessions and the watcher is showing up in profiling, raise to
10s. Otherwise leave it.

## Per-channel cooldown is separate

Session-level idle threshold is the FLOOR — once a session is
silent for N seconds, idle fires. Whether it gets *forwarded* to
each channel depends on the channel's notification policy:

- **Once per session** (channel default) — fire one
  notification, then suppress until reply or session end. Even
  if the session goes active→idle 50 times in an hour, the
  channel only pings you once.
- **Time-window cooldown** — re-notify every N minutes while
  idle. Use when you want recurring reminders.
- **Every event** — no suppression; every idle fires a fresh
  notification.

See [Channels → Notifications panel](#channels-notifications)
for full per-channel tuning.

## Tuning workflow

If you're getting too many notifications:

1. Don't reduce idle_threshold — the underlying idle event isn't
   the noise source.
2. Switch the channel's policy to *Once per session*. This
   handles 90% of "too noisy" complaints.

If you're missing notifications:

1. Check the channel has `session.idle` checked in its
   *Notify on* topics.
2. Check the channel isn't muted (`muted: true` in config or
   `/notify off` was sent).
3. Check the cooldown isn't suppressing (Once mode + no reply
   = stays suppressed until session ends).

The **Activity** page is the fastest way to confirm what's
happening — `session.idle` events show up in real time, then
follow `channel.message_sent` events, or the lack of them.

## Other knobs (read-only via UI; edit in config.toml)

| Knob | Default | Purpose |
|---|---|---|
| `[session].ring_size` | `1 MiB` | Per-session stdout ring buffer capacity |
| `[session].terminate_grace` | `3s` | SIGTERM-to-SIGKILL gap on stop |
| `[session].vt_cols` / `vt_rows` | `120 × 40` | Initial virtual terminal size for screen snapshots |

These are config-only because changing them at runtime would
affect already-spawned sessions in surprising ways. Restart
opendray after editing.
