# Optional Encrypted Relay + Mobile Push

Status: **Draft — RFC for review (no implementation yet)**
Owner: opendray gateway
Last updated: 2026-08-19

Zero-config remote access and app-closed notifications, without forcing the
operator to stand up their own tunnel. Prompted by
[tiann/hapi](https://github.com/tiann/hapi), whose `relay` package
(WireGuard+TLS + APNs) makes `--relay` print a QR code and be reachable
from a phone immediately.

This RFC covers two separable sub-features — **push delivery** (higher
value, lower risk) and the **relay transport** (bigger infra) — phased so
push can ship first.

---

## 1. What exists today

- **Registration half of push is already built.** `device_tokens`
  (migration `0024_device_tokens.sql`) stores APNs/FCM/web push tokens per
  device + principal, with labels, app version, and revocation support.
- **No push sender.** There is no APNs/FCM client in the Go tree — nothing
  ever delivers a notification to a registered device.
- **Remote access is bring-your-own-tunnel.** The gateway is reached via
  Cloudflare Tunnel / Tailscale / a reverse proxy the operator configures;
  there is no built-in broker.
- **Auth is ready.** `/auth/mobile-login` issues 30d bearer tokens for the
  Flutter app; the WS stream (`Subscribe`) + `Buffer(since)` replay already
  back the live session view.

So the gap is precise: **wire a push sender to the existing token registry**,
and (separately) **offer an optional relay** so remote works with no tunnel.

## 2. Fundamentals compliance

| Ref | Fundamental          | Verdict                                                    |
|-----|----------------------|-----------------------------------------------------------|
| F1  | Cross-CLI parity     | Neutral — infra, triggered by provider-agnostic events    |
| F3  | Self-hosted          | **Condition:** relay MUST be optional; a mandatory hosted relay breaks this |
| F5  | Multi-user isolation | Push scoped by `principal` (already in `device_tokens`)   |
| F6  | Secrets never leak   | APNs/FCM keys stay gateway-side; push tokens already stored server-side |

The single compliance risk is F3: the relay must never become a required
hosted dependency. It ships **off by default**, self-hostable, and every
existing tunnel path keeps working unchanged.

## 3. Push delivery (Phase 1)

Event-driven notifications to registered devices, so the operator gets the
AFK loop (approve from the lock screen) even with the app closed.

```
 eventbus.Hub ──▶ internal/push dispatcher ──▶ APNs / FCM / WebPush
   session.idle          │  filter by principal        │
   session.turn_completed │  (device_tokens)           ▼
   session.approval_pending└─────────────▶ phone / browser notification
                                            deep-links into the session
```

- **C1 — push dispatcher (`internal/push`).** Subscribes to the event bus,
  resolves target devices from `device_tokens` by principal, renders a
  short title/body, sends via a `push.Sender`.
- **C2 — sender abstraction.** `push.Sender` with APNs, FCM, and WebPush
  implementations; registry keyed by platform. Keys from `secrets.env`.
- **C3 — notification policy.** Per-account choice of which events notify
  (idle / turn done / approval pending) — reuses account isolation (F5).
- **Triggers.** `session.approval_pending` (needs the same new event as the
  voice RFC — shared dependency), `session.turn_completed`, `session.idle`.

## 4. Relay transport (Phase 2)

An optional broker so a device reaches the gateway without a
user-configured tunnel.

- **C4 — relay client (in-gateway).** When `[relay] enabled`, the gateway
  dials out to a relay endpoint and registers; prints a URL + QR code.
- **C5 — relay server.** A small self-hostable service (the operator runs
  it, or points at a community/opendray-hosted one **by choice**). Brokers
  encrypted device↔gateway streams; sees ciphertext only.
- **Encryption.** E2E device↔gateway (TLS to the gateway's own cert /
  a WireGuard-style key exchange); the relay is a blind pipe.
- **Fallback.** Tunnel/reverse-proxy paths stay first-class; relay is one
  more option, never the only one.

## 5. Config & API

- **`[push]`** — `enabled`, per-platform sender blocks (APNs key id / team
  id / p8; FCM service account; WebPush VAPID), default notify policy.
- **`[relay]`** — `enabled` (default false), `endpoint`, `key`. Off = today's
  behavior exactly.
- **Existing** `POST /devices` (register), `DELETE /devices/{id}` (revoke)
  back the `device_tokens` table; Phase 1 adds `PATCH /devices/{id}/policy`.

## 6. Phasing

1. **P1 — Push delivery.** `internal/push` + APNs/FCM/WebPush senders wired
   to `device_tokens`, driven by event bus. Ships the AFK-notification value
   with no new transport. Depends on `session.approval_pending` (shared with
   voice).
2. **P2 — Relay transport.** Optional broker for zero-config remote. Larger;
   gated behind `[relay] enabled=false` so F3 is never at risk.

## 7. Open decisions

- **D1 — Relay hosting model.** Self-host-only (safest for F3, more setup)
  vs an opt-in opendray-hosted convenience relay (easier, but must stay
  optional and blind). **Recommend self-host-first**, add hosted-opt-in later.
- **D2 — WebPush in P1?** Covers browsers with no app install; small extra
  surface. **Recommend yes** — it makes push useful before native apps.
- **D3 — Sequencing vs voice.** Both this and voice want
  `session.approval_pending`. **Recommend landing that event on its own**
  ahead of both (also raised in the voice RFC).

## 8. Review asks (linivek)

1. Is P1 (push sender on the existing `device_tokens`) worth doing
   independent of the relay?
2. Relay hosting model (D1) — is self-host-first the right F3 posture?
3. Should `session.approval_pending` land as a standalone change that both
   this and the voice RFC build on?

## 9. Non-goals

- Replacing existing tunnel/reverse-proxy access — the relay is additive.
- A mandatory hosted service of any kind.
