# Provider-Agnostic Voice Layer

Status: **Draft — RFC for review (no implementation yet)**
Owner: opendray gateway
Last updated: 2026-08-19

Two-way voice on top of opendray's session I/O and event bus. One
implementation works across every provider (claude, grok, codex, opencode,
antigravity, …) because no agent-specific code touches the CLI.

Prompted by [tiann/hapi](https://github.com/tiann/hapi), which invests
heavily in voice while lacking opendray's memory / skills / MCP layers.
Voice is the one hapi feature that is *aligned* with opendray's
fundamentals rather than merely tolerable: it lives on the provider-
agnostic session layer and needs no per-CLI branching.

---

## 1. Principle

Voice wraps session I/O, not any agent:

- **Listen** — subscribe to the event bus (`internal/eventbus`), which
  already publishes `session.turn_completed`, `session.idle`,
  `session.output`, and structured `text` / `thinking` / `tool_use` /
  `tool_result` blocks parsed from the transcript.
- **Speak to the agent** — write to `POST /sessions/{id}/input`.

Every provider already flows through those two seams, so cross-CLI parity
holds by construction. The layer never switches on provider (except one
small keymap table in Phase 3 — data, not logic).

## 2. Fundamentals compliance

| Ref | Fundamental          | Verdict                                             |
|-----|----------------------|-----------------------------------------------------|
| F1  | Cross-CLI parity     | Reinforces — voice is a session-layer wrapper       |
| F2  | Memory-first         | Reuses `internal/memory/summarizer` for spoken progress |
| F3  | Self-hosted          | Complies **if** a local STT/TTS backend is a first-class option |
| F5  | Multi-user isolation | Complies **with** per-account/session voice config  |
| F6  | Secrets never leak   | Complies **via** server-side key proxying / short-lived tokens |

The only ways this slips are at the edges — a forced-cloud backend (breaks
F3) or provider keys reaching the browser (breaks F6). Both are design
choices controlled below, not inherent to the feature.

## 3. Architecture

```
                 ┌───────────────────────────────┐
                 │  Browser / PWA voice client    │  mic in · audio out
                 └───────────────┬───────────────┘
                        audio (WebRTC / WS)
                 ┌───────────────┴───────────────────────────┐
                 │  internal/voice service (C1)               │
                 │  ┌──────────┐ ┌───────────┐ ┌────────────┐ │
                 │  │ backend  │ │ context   │ │ voice-in   │ │
                 │  │ registry │ │ feeder C3 │ │ C5 · appr  │ │
                 │  │   C2     │ │ progress  │ │ C6 →/input │ │
                 │  └──────────┘ │   C4      │ └────────────┘ │
                 │  keys server-side; ElevenLabs/Gemini/Qwen/local
                 └───────┬───────────────────────────┬────────┘
                    subscribe                    POST /input
        ┌───────────────┴──────────  PROVIDER-AGNOSTIC SEAM  ─┴───────────┐
        │  eventbus.Hub — session.* topics   session I/O — /stream · /input │
        └───────────────┬───────┬───────┬───────┬─────────────────────────┘
                        claude   grok   codex   opencode · antigravity
              Voice touches only the seam — never a CLI. That keeps F1 intact.
```

## 4. Components

- **C1 — voice service.** New `internal/voice` package. Owns backend
  sessions, token minting, and the bridge between a browser voice socket
  and an opendray session. Stateless per-connection; zero agent knowledge.
- **C2 — backend abstraction.** `voice.Backend` interface: `Realtime`
  (two-way: ElevenLabs, Gemini Live, Qwen) and `Transcribe` / `Synthesize`
  (STT/TTS incl. local). Registry keyed by id — same shape as the MCP /
  summarizer registries.
- **C3 — context feeder.** Subscribes to the event bus for one session;
  turns `tool_use` / `text` / `turn_completed` into short context updates
  for the realtime backend. Raw `/stream` PTY is a fallback only.
- **C4 — spoken progress.** On `session.turn_completed` / `session.idle`,
  reuses `internal/memory/summarizer` to compress the turn into one spoken
  sentence. No parallel LLM path.
- **C5 — voice-in adapter.** Final transcript → `POST /sessions/{id}/input`
  (text + `\r`). Byte-identical for every provider.
- **C6 — approval bridge.** The only provider-touching piece. Detects a
  pending approval and writes the provider's accept/deny keystroke via a
  small keymap table. Isolated to Phase 3.

## 5. Data flows

1. **Speak → agent** *(F1-clean)* — mic → C2 STT → transcript → C5 →
   `POST /input`. Provider-agnostic, zero risk.
2. **Agent → speak** *(F1-clean)* — event bus `turn_completed` → C4 summary
   → C2 TTS → browser audio. Reuses existing events + summarizer.
3. **Voice approvals** *(Phase 3, provider-touching)* — new
   `session.approval_pending` (a `tool_use` with no matching `tool_result`
   while the session goes idle) → voice asks → on yes/no, C6 writes the
   provider accept/deny keystroke via a keymap table → `/input`. The one
   place true genericity is hard; the keymap is the only per-provider data.

## 6. Backend & security model

| Backend                        | Mode       | Key handling (F6)                            |
|--------------------------------|------------|----------------------------------------------|
| ElevenLabs Conversational      | Realtime   | hub mints a short-lived token; WebRTC direct |
| Gemini Live / Qwen Realtime    | Realtime   | hub proxies the WS, injects credentials server-side |
| OpenAI / Deepgram / Groq       | Transcribe | gateway-side, key from `secrets.env`         |
| Local OpenAI-compatible STT/TTS| Both       | no external call — keeps F3 (self-hosted) intact |

Keys resolve from `secrets.env` at the gateway and never reach the browser.
Cloud backends are opt-in; the local backend is a first-class option so an
air-gapped install still gets voice.

## 7. Config & API

- **`[voice]` config** — `enabled`, `default_backend`, per-backend blocks,
  `spoken_progress`; per-account/session override resolved through the
  existing account isolation (F5). Env override
  `OPENDRAY_VOICE_*` for 12-factor parity with the rest of the config.
- **`POST /sessions/{id}/voice/token`** — mint / authorize a backend
  session; never bypasses session auth.
- **`GET /sessions/{id}/voice/stream`** — WS carrying audio + context;
  proxied when the backend needs server-side keys.

## 8. Phasing

1. **P1 — MVP: dictation-in + spoken-progress-out.** Flows 1 & 2, one cloud
   backend + local fallback. Ships value with zero per-provider code. Fully
   F1-clean.
2. **P2 — Full two-way realtime.** ElevenLabs / Gemini / Qwen + context
   feeder C3. Still pure event bus + `/input` — no agent-specific code.
3. **P3 — Voice approvals.** C6 + `session.approval_pending` + keymap.
   Deferred: the only provider-touching piece; verify one CLI at a time.

## 9. Open decisions

- **D1 — Realtime backend for P2.** ElevenLabs (best quality, token-mint)
  vs Gemini Live (fits an existing Google key). **Recommend ElevenLabs
  first** — its token model is the cleanest for F6.
- **D2 — Approval detection.** A robust `session.approval_pending` event
  (more work) vs an idle-after-`tool_use` heuristic (fast, occasionally
  wrong). **Recommend the event** — it also benefits the web UI, not just
  voice.

## 10. Review asks (linivek)

1. Does the phasing hold to the governance bar — is P1 (event bus + `/input`
   only, no per-provider code) acceptable to build without waiting on P2/P3?
2. Is the `[voice]` backend policy (cloud opt-in, local required for a
   self-hosted install) the right default for F3?
3. D1 and D2 above — any objection to the recommended options?
4. `session.approval_pending` (D2) is a new event other surfaces could use.
   Should it land as its own change ahead of voice?

## 11. Non-goals

- Native mobile apps (SwiftUI/Kotlin) — opendray's Flutter + web already
  cover mobile; not worth rebuilding on hapi's turf.
- Wake-word activation — out of scope; voice starts on an explicit user
  action, matching hapi.
