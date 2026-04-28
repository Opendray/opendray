# ADR 0005 — Channel inbound routing: event-only, no hardcoded session routing in M4

**Status:** Accepted
**Date:** 2026-04-29
**Decider:** Linivek

## Context

Design §8.3 describes the Channel Hub as owning two responsibilities:

1. **Outbound** — subscribe to event-bus topics (`session.idle`,
   `session.ended`, ...) and translate them into channel notifications.
2. **Inbound** — receive a message from telegram / slack / imessage
   and route it "to a session".

(2) is the harder half. v1 hard-coded the routing inside
`gateway/telegram/forwarder.go` (~800 LOC) plus `commands.go` (~824 LOC)
plus questions/multiselect/sessions helpers — together ~3000 LOC of
session-aware logic living inside the telegram package. Adding slack
or imessage in v1 would have meant duplicating most of this.

For M4 we need to deliver telegram outbound notifications and a
shape that makes adding slack later cheap. We do not need to ship
v1's full inbound feature set on day one.

## Decision

M4 implements **inbound only as far as the persistence + event layer**:

- The Channel impl receives a message from its upstream service.
- The Hub writes one `channel_messages` row (direction=`inbound`).
- The Hub publishes `channel.message_received` on the event bus with
  `{channel_id, conversation_id, author, text, channel_message_id}`.

The Hub itself does **not** look up a target session, does not run
slash-command parsing, does not implement multiselect / question /
quick-reply state machines, and does not write to any session's stdin.

Routing is delegated to whoever subscribes to
`channel.message_received`. Three foreseen consumers:

- **Future M3 integration** that has its own command grammar (e.g.
  PetTracker subscribes to channel events and decides when to call
  `POST /sessions`).
- **Future v1.x SDK utility** that ships canonical "reply to last
  active session" routing as opt-in code, not as gateway behaviour.
- **A scriptable rules engine** (Starlark, deferred per design
  §19) — only if (1) and (2) prove insufficient.

Outbound stays in Hub: each channel config carries `notify_on: [...]`
listing the topics it wants pushed (`session.idle`, `session.ended`),
and the Hub dispatches matching events as text messages.

## Consequences

- **M4 ships in ~800–1000 LOC** instead of v1's ~3000.
- **No automatic "telegram → session.input" round trip in v1.0.**
  Operator can still see the inbound message in `channel_messages`
  and the audit log; they cannot reply to a session by typing in
  telegram until a routing layer exists.
- **Adding slack later is cheap**: implement `Channel`, register a
  factory, no Hub changes.
- **Forward-compatible**: when a routing layer lands (M5+ /
  integration / script), it subscribes to `channel.message_received`
  and uses the existing session API. No Hub refactor.

## Trigger to revisit

Bring routing back into Hub (or build a dedicated `internal/router/`)
when **any** of:

1. Two or more consumers independently re-implement the same
   "reply to last active session" rule. Then the rule is canon and
   belongs upstream.
2. Telegram round-trip latency from event to session.input becomes
   a real bottleneck (event-bus + integration hop > 200ms typical).
3. Operator explicitly asks for "I want to type in telegram and see
   it in the active claude session" without writing custom code.

Until then, channel events are data on a bus; routing is policy and
policies live in consumers.

## Out of scope for M4 (carried forward as future tickets)

- Quick-reply / multiselect state machines.
- Per-conversation session-mapping table (`channel_routes`).
- Telegram inline keyboards / callback queries.
- Slash-command grammar.
- File / image upload from inbound message into session.
- Slack / iMessage implementations (Slack first scheduled in M4 if
  bandwidth permits, otherwise post-v1.0).
