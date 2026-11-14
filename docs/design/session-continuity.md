# Session Continuity ("Handoff")

Status: **Draft — RFC for review (no implementation yet)**
Owner: opendray gateway
Last updated: 2026-08-19

Make opendray's local↔remote, cross-device continuity an explicit,
surfaced experience. Prompted by [tiann/hapi](https://github.com/tiann/hapi),
which markets "seamless handoff" as a headline feature.

The thesis of this RFC: **opendray already has stronger continuity than
hapi** — it is memory-backed and survives restarts and account switches —
but it is plumbing, not a named feature. This is mostly a *surfacing* job
plus a few small gaps, not new architecture.

---

## 1. What exists today

Continuity is already the backbone, spread across several primitives:

- **Any client attaches to the same live session.** `Subscribe` (WS live
  stream) + `Buffer(ctx, id, since)` replay let web and mobile attach to a
  running session and backfill missed output from a cursor.
- **Sessions survive process restarts.** Auto-resume respawns interrupted
  sessions at startup (`--resume`, `ReconcileStartup`), bounded by
  `autoResumeConcurrency`. See `docs/design/session-state-machine.md`.
- **Meaning carries across account switches.** `carry-context-on-account-
  switch` injects the prior transcript into the fresh session's system
  prompt (`docs/specs/carry-context-on-account-switch.md`).
- **Cross-session memory.** Goal / plan / journal / decision records +
  ambient injection mean a *new* session already knows the project's state —
  continuity beyond a single conversation, which hapi has no equivalent for.

## 2. The gap

Continuity works but is invisible and slightly rough at the edges:

- No named "pick up where you left off" surface — the user can't see that a
  session is live and attachable from another device.
- No **AFK loop close**: when something needs the operator (approval, turn
  done) there's no notification that deep-links back into the exact session.
  (This is where H2 / push connects.)
- Cross-device attach has no shared cursor / "you're also open on iPhone"
  awareness.
- The strongest asset — memory-backed continuity — is never shown to the
  user as *why* a new session already understands the project.

## 3. Fundamentals compliance

| Ref | Fundamental          | Verdict                                          |
|-----|----------------------|--------------------------------------------------|
| F1  | Cross-CLI parity     | Neutral — sits on the session layer, no per-CLI code |
| F2  | Memory-first         | **Reinforces** — surfaces memory as the continuity engine |
| F5  | Multi-user isolation | Attach/notify scoped by principal                |

No architectural risk: this RFC adds UX and one small awareness signal on
top of existing primitives.

## 4. Components

- **C1 — Continuity surface (web + Flutter).** A "Live sessions" / "Resume"
  view: every attachable session, its host, last activity, and a one-tap
  attach that replays via `Buffer(since)`. Turns the existing attach model
  into a visible feature.
- **C2 — Resume banner.** On attach, a compact "here's what happened while
  you were away" header, built from the buffer backfill + the latest journal
  entry (reuses the summarizer, like the voice spoken-progress path).
- **C3 — AFK deep-link.** Notifications (via H2 push) carry a session id and
  open directly into that session's live view. Closes the leave→get-pinged→
  resume loop.
- **C4 — Presence signal (optional).** Lightweight "also attached on
  <device>" indicator from the existing Subscribe fan-out, so two clients on
  one session don't fight.

## 5. Relationship to other RFCs

- **Depends on H2 (push)** for C3's AFK deep-link — the notification is what
  makes remote handoff feel seamless. Continuity is the destination; push is
  the doorbell.
- **Complements voice** — spoken progress (voice C4) and the resume banner
  (C2) are the same summary rendered two ways (audio vs header).

## 6. Phasing

1. **P1 — Surface it.** C1 continuity/resume view + C2 resume banner on
   existing primitives. Pure UX, zero backend risk, ships immediately.
2. **P2 — Close the AFK loop.** C3 deep-link once H2 push lands.
3. **P3 — Presence.** C4 multi-client awareness. Nice-to-have.

## 7. Open decisions

- **D1 — Resume banner source.** Buffer backfill only (cheap, literal) vs
  summarizer-generated recap (richer, an LLM call). **Recommend a hybrid:**
  literal tail immediately, summary if a turn boundary was crossed.
- **D2 — Scope of "handoff" framing.** Market it as continuity (honest, ties
  to memory) rather than mimicking hapi's "handoff" wording, which undersells
  the memory advantage.

## 8. Review asks (linivek)

1. Is P1 (surfacing the existing attach + resume model as a named feature)
   worth doing on its own, ahead of push?
2. Framing: lead with "memory-backed continuity" (our differentiator) vs
   hapi's "handoff" language — agree?
3. Presence (C4) — worth it, or overkill for a single-operator install?

## 9. Non-goals

- New continuity *mechanism* — this rides the existing `Subscribe` /
  `Buffer(since)` / auto-resume / carry-context stack.
- Native mobile rebuild — the Flutter app gets the new views.
