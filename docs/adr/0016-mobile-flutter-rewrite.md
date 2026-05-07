# ADR 0016 — Mobile platform: rewrite in Flutter, retire Capacitor

**Status**: Proposed (rewrite in progress on `feat/mobile-flutter`)
**Date**: 2026-05-08
**Supersedes**: ADR 0015

## Context

ADR 0015 picked Capacitor 6 wrapping a shared React SPA as the
mobile platform. Phases A1–A3 (workspace split), B1–B6 (Capacitor
init, mobile auth, sessions list, session detail + terminal), and
T2–T4 (Memory / Notes / Activity / Channels / Integrations /
Providers / Backups / Settings) shipped to `feat/mobile-platform`
across PRs #21–#32. After running the resulting build on a
physical / simulator iPhone for several iterations, the maintainer
identified two structural problems that the framework choice cannot
fix without leaving Capacitor:

1. **The mobile app must be a primary surface, not a companion.**
   The intended use case is operating opendray entirely from a
   phone — spawning sessions, editing memory, managing channels,
   restoring backups — without needing a laptop. ADR 0015 framed
   mobile and web as "equal-priority surfaces", but the WebView
   shell ends up feeling like a stripped-down desktop UI even when
   feature parity is achieved.

2. **A WebView ceiling makes "native-quality" out of reach.**
   Concrete pain points encountered during ADR 0015 implementation:

   - **Input method**: Chinese IME composition jitters; soft-
     keyboard-driven viewport resize causes focus loss inside
     forms and terminal input. Patchable but not fixable.
   - **Gestures**: No interactive swipe-back, no native-feeling
     list overscroll, no UIKit `UISheetPresentationController`
     detents — all approximated in CSS at lower fidelity.
   - **iOS 26.3 system surfaces**: Dynamic Island, Live
     Activities, Lock-Screen widgets, Home-Screen widgets, Quick
     Actions, App Intents, Share Sheet — every one requires a
     Capacitor plugin (most third-party, several unmaintained)
     and a separate native target. Maintenance cost compounds.
   - **Push notifications**: `@capacitor/push-notifications`
     works but configuring APNs entitlements, notification
     extensions, and rich attachments through the Capacitor
     bridge is materially harder than touching them in Swift /
     Dart directly.
   - **Background**: WebSocket connections die on app suspend.
     Recovery requires push wake-up, which is the same problem
     amplified.

The trigger for revisiting ADR 0015 was not a single technical
failure — Capacitor was producing a working app — but the
realization that **every quality bar the maintainer cares about
sits on the wrong side of the WebView boundary**, and that the
remaining work to reach those bars (push, biometrics, share sheet,
widgets, live activities) was going to be plugin-glue rather than
app development.

The sunk cost of `app/mobile/` (~3000 lines React/TS, all merged
into `feat/mobile-platform` already) was weighed against the
forward cost of **continuing** the same direction (estimated
~12-17 sessions to reach feature parity, with the WebView ceiling
intact at the end). The maintainer's other production projects
(`remote_claude_code` and others under `Obsidiannote/Projects/`)
already use Flutter; the global `CLAUDE.md` standardizes mobile
on Flutter; the `build_release.sh` template at
`remote_claude_code/mobile/build_release.sh` exists and is
reusable. Switching consolidates rather than fragments the
maintainer's mobile stack.

## Decision

### 1. Cross-platform shell — Flutter 3.27+, replace Capacitor

`app/mobile/` is **rewritten from scratch as a Flutter project**.
The existing Capacitor `app/mobile/` directory is removed in the
same PR that initializes Flutter, so the path stays canonical.

Why Flutter (revisited from ADR 0015):

- **Native rendering & gestures**: Skia / Impeller render every
  pixel; gestures route through platform-native gesture
  recognizers. Animations and IME behavior are at the same
  fidelity as a hand-written Swift / Kotlin app.
- **System integration is direct**: Push, biometrics, Share
  Sheet, widgets, Live Activities, App Intents — each is a Swift
  / Kotlin file in `ios/Runner/` or `android/app/`. No Capacitor
  bridge.
- **Stack consolidation**: Aligns with the maintainer's other
  Flutter apps (`remote_claude_code/mobile`, others). Same release
  pipeline, same Vaultwarden + Keychain credentials, same
  TestFlight / UNAS SMB target.
- **Long-term maintenance cost**: Fewer moving parts than
  React + Vite + Capacitor + plugin matrix.

What we lose (honest accounting — these *were* the ADR 0015
arguments for Capacitor):

- **No code reuse with web.** `app/shared-ui/` no longer feeds
  mobile. Web continues using it; mobile builds its own widget
  library in Dart. Two UI codebases to maintain.
- **`shared/` (TS types + REST client + WS helpers) cannot be
  imported from Dart.** Types are re-declared in Dart with
  `freezed` + `json_serializable`; the REST client is rewritten
  with `dio`. The wire format (the OpenAPI-shaped JSON / WS
  protocol) is the contract — both stacks observe it but neither
  owns it.
- **xterm.js → xterm.dart**. xterm.dart is community-maintained
  and less mature. Acceptable risk; if we hit a wall we can fall
  back to a `WebView` with xterm.js inside, scoped to the
  terminal tab only.

Rejected alternatives (revisited):

- **Stay on Capacitor + push hard on UX polish.** Rejected: the
  ceiling problems above are inherent to WebView, not patchable.
- **React Native.** Rejected: no React DOM (rewrite anyway), no
  meaningful advantage over Flutter for the maintainer who
  already owns Flutter expertise, and Hermes / Metro tooling is
  more brittle than Flutter's.
- **Native iOS + native Android (separate Swift / Kotlin apps).**
  Rejected: 2× implementation cost, no upside over Flutter for
  the kinds of UI surfaces opendray needs.
- **Tauri Mobile.** Rejected for the same reason as ADR 0015 —
  still pre-1.0, production risk too high.

### 2. Repository layout — `app/mobile/` is a Flutter project

```
app/
├── shared/          ← unchanged; TS lib for web only
├── shared-ui/       ← unchanged; React components for web only
├── web/             ← unchanged; React/Vite SPA, embedded by Go
└── mobile/          ← REWRITTEN — Flutter project
    ├── lib/
    │   ├── main.dart
    │   ├── api/             ← dio client + freezed types mirroring shared/
    │   ├── auth/            ← onboarding, login, secure storage, biometric
    │   ├── routing/         ← go_router + deep links
    │   ├── theme/           ← Material 3 + custom dark theme
    │   ├── widgets/         ← cross-screen widgets
    │   ├── features/
    │   │   ├── sessions/    ← list, detail, terminal, spawn, actions
    │   │   ├── memory/      ← list, search, scope filter, ambient rules
    │   │   ├── notes/       ← list, view, edit, git push/pull
    │   │   ├── activity/    ← audit feed
    │   │   ├── channels/    ← per-kind config forms
    │   │   ├── integrations/← register, rotate-key, scope edit, call-log
    │   │   ├── providers/   ← per-provider config, Claude account browser
    │   │   ├── backups/     ← targets, schedules, restore flow
    │   │   └── settings/    ← admin / system / git / themes / log levels
    │   └── push/            ← APNs / FCM token registration + handlers
    ├── ios/                 ← Flutter-generated Xcode project
    ├── android/             ← Flutter-generated Gradle project
    ├── pubspec.yaml
    ├── analysis_options.yaml
    └── build_release.sh     ← UNAS SMB + TestFlight (template from rcc)
```

### 3. Stack — concrete package choices

| Concern | Package | Notes |
|---|---|---|
| Routing | `go_router` 14+ | Declarative, deep-link friendly |
| State | `flutter_riverpod` 2.x | AsyncValue + family providers |
| HTTP | `dio` 5.x + `pretty_dio_logger` | Interceptors for auth + retry |
| Types | `freezed` + `json_serializable` | Code-gen mirrors of TS types |
| Secure storage | `flutter_secure_storage` | iOS Keychain, Android EncryptedSharedPrefs |
| Biometric | `local_auth` | Face ID / Touch ID / fingerprint |
| WebSocket | `web_socket_channel` | For session event subscriptions |
| Terminal | `xterm` (Dart) | The xterm.dart package |
| Markdown | `flutter_markdown_plus` | For Notes view |
| Code editor | `re_editor` | For Notes edit (lightweight) |
| Push | `firebase_messaging` + `flutter_local_notifications` | FCM Android + APNs iOS |
| Share Sheet | `share_plus` | Outbound only; inbound via App Intent later |
| Linting | `very_good_analysis` | Strict ruleset |

### 4. Backend — zero changes

Every contract added in ADR 0015 phase B2 is reused:

- `[admin].mobile_token_ttl` config field
- `device_tokens` table
- `POST /api/v1/auth/mobile-login`
- All `/api/v1/*` endpoints

Deferred phase-C endpoints (push register / send) are still
deferred; this ADR doesn't change the timeline for those, only
which client implements them.

### 5. Authentication flow — same as ADR 0015 §5, ported

1. First launch → server-URL onboarding → `GET /api/v1/health`
2. Login screen → `POST /api/v1/auth/mobile-login` (username +
   password)
3. Token stored via `flutter_secure_storage`
4. Subsequent launches:
   - Biometric enabled → `local_auth` prompt → unlock token
   - Biometric disabled → password again

The decision logic is identical to ADR 0015 — only the platform
APIs change (Keychain access via `flutter_secure_storage` instead
of `@capacitor/preferences`; biometric via `local_auth` instead
of `@capgo/capacitor-native-biometric`).

### 6. Viewport strategy — adaptive layouts in Flutter

Flutter's `LayoutBuilder` + `MediaQuery` replace ADR 0015's
5-tier CSS breakpoint system. The same five tiers (xs / sm / md /
lg / xl) are encoded as Dart constants and used by an `Adaptive`
widget that picks single-column vs split-view layouts at runtime.

The phone↔tablet distinction matters more than iOS↔Android, same
as ADR 0015 §3.

### 7. Tablet — ships the same Flutter app

Tablets get the Flutter app, not the web SPA in Safari. Same
reasoning as ADR 0015 §4: push, biometric, deep links, secure
token storage are valuable on tablet too.

### 8. Distribution — deferred, same as ADR 0015 §1.4

App Store / TestFlight / Play Store / AdHoc IPA / OTA decisions
are all still open. Flutter supports every option Capacitor did.

The `build_release.sh` template at
`remote_claude_code/mobile/build_release.sh` is the starting point
for `app/mobile/build_release.sh`:

1. Read version → bump build number → build APK + IPA
2. APK → `smb://192.168.9.8/Claude_Workspace/opendray-v2/android/`
3. IPA → TestFlight via App Store Connect API
4. UNAS password from Keychain (`unas-smb`); ASC key at
   `~/.appstoreconnect/private_keys/AuthKey_BPL8QFJ8M2.p8`

## Consequences

### Positive

- **Native quality**: gestures, animations, IME, keyboard
  handling, system integration — at the level of any flagship
  iOS / Android app.
- **Direct system surfaces**: push, biometrics, Share Sheet,
  widgets, Live Activities, deep links, App Intents — each a
  one-Swift-file or one-Kotlin-file addition.
- **Stack consolidation**: same toolchain as the maintainer's
  other production Flutter apps; same release pipeline.
- **No more WebView surprises**: no plugin-version drift, no
  CapacitorHttp CORS workarounds, no `safe-area-inset` CSS, no
  IME-resize layout glitches.
- **Strong typing across the wire**: `freezed` + null-safety give
  Dart-side schema safety equivalent to (arguably stronger than)
  the TS Zod schemas on the web side.

### Negative

- **Sunk cost**: ~3000 lines of React/TS in `app/mobile/`
  (Capacitor) are deleted. Snapshot preserved on local branch
  `mobile/deep-v1-sessions` for archaeology; never lands.
- **Dual UI stacks**: web stays React, mobile becomes Flutter.
  A new admin feature must now be built twice — once in
  `shared-ui` for web, once in `app/mobile/lib/features/` for
  Flutter. ADR 0015's "feature work pays off twice" claim is
  inverted.
- **Two type systems for one wire format**: TS types in
  `app/shared/types/` and Dart types in `app/mobile/lib/api/`
  must be kept in sync manually. Mitigated by deriving both from
  the gateway's OpenAPI spec (future work) — until then, drift is
  caught at runtime.
- **xterm.dart maturity risk**: less battle-tested than
  xterm.js. If a blocker emerges, fallback is a scoped WebView
  inside the terminal screen only (single-screen Capacitor /
  flutter_inappwebview pattern).
- **Push-deferred WebSocket recovery still required**: same as
  ADR 0015. Switching framework doesn't fix iOS background
  WebSocket suspension.
- **Two ADRs to keep in mind**: ADR 0015 stays on file as the
  superseded record; readers must follow the chain to 0016.

### Compatibility

- **Backend zero-change**: `/api/v1/*`, `/admin/*`,
  `internal/web/embed.go`, `app/web/dist/` are all untouched.
  Web continues to deploy via `go:embed all:dist`.
- **Web continues to use** `app/shared/` and `app/shared-ui/`.
  These packages stay in the monorepo, only mobile no longer
  imports them.
- **CI**: `feat/mobile-flutter` long-lived branch is gated by
  the same checks as `feat/mobile-platform` was. A new Flutter
  CI job is added (`flutter analyze` + `flutter test`); existing
  Frontend / Backend / Lint jobs run unchanged.
- **CLAUDE.md global rule**: now applies *literally* — opendray
  joins the maintainer's Flutter mobile app fleet, so the
  `build_release.sh` requirement is met natively rather than in
  spirit.

## Implementation references

The rewrite lands as a series of sub-PRs into the
`feat/mobile-flutter` long-lived branch (note: a *new* long-lived
branch, not `feat/mobile-platform`):

| Phase | Outcome |
|-------|---------|
| F0 | This ADR + branch + remove Capacitor `app/mobile/` |
| F1 | `flutter create` scaffold + theme + go_router + dio + auth onboarding/login/secure storage + biometric |
| F2 | Sessions list + spawn + actions + state filters |
| F3 | Sessions detail + xterm.dart terminal view |
| F4 | Memory (list / search / CRUD / scope filter / ambient rules) |
| F5 | Notes (list / view / edit + git push/pull) |
| F6 | Activity (audit feed + filters) |
| F7 | Channels (per-kind config forms) |
| F8 | Integrations (register / rotate-key / scope / call-log) |
| F9 | Providers (per-provider config / Claude accounts) |
| F10 | Backups (targets / schedules / restore flow) |
| F11 | Settings (admin / system / git / themes / log levels) |
| F12 | Push (APNs/FCM register, handlers, deep-link to session) |
| F13 | Share Sheet outbound + inbound App Intent (cwd file pick) |
| F14 | iOS Live Activities + Home Screen widget (active sessions) |
| F15 | `build_release.sh` + TestFlight + UNAS SMB upload |
| F16 | Final PR — `feat/mobile-flutter` → `main` |

PR numbers will be filled in as each lands.

Estimated wall-clock: 8-9 weeks of focused work; faster if
ADR 0015 phase-B2 backend work continues to hold (it should —
that work is platform-agnostic).

## Future work

Same backlog as ADR 0015 §"Future work", scoped to the new stack:

- **OTA updates**: Flutter doesn't have an equivalent of
  `@capgo/capacitor-updater`. Codepush-style hotfixes are not a
  first-class feature on Flutter; if needed, ship via App Store /
  Play Store updates only.
- **Universal links / deep linking**: `opendray://session/sess_42`
  via `uni_links` or App Links + Universal Links configuration.
- **App Store / Play Store packaging**: deferred until end-to-end
  testing completes (same posture as ADR 0015).
- **Tablet split-view multitasking**: same concern, same goal.
- **Watch companion**: speculative, unchanged.
- **OpenAPI-derived type generation**: emit both TS types (for
  web) and Dart types (for mobile) from the same OpenAPI spec to
  end the manual-sync risk noted in §"Negative".

## Appendix — what to do with `feat/mobile-platform` and its PRs

`feat/mobile-platform` (PRs #21–#32, plus polish PR #32) carries
~3000 lines of Capacitor code that this ADR retires. Two paths:

1. **Merge `feat/mobile-platform` → `main` first**, then start
   `feat/mobile-flutter` on top of the merged main, deleting
   `app/mobile/` and replacing with Flutter in F0/F1.
2. **Abandon `feat/mobile-platform`** without merging; start
   `feat/mobile-flutter` from `main` directly; the Capacitor
   work never lands on `main`.

**Decision**: option (2). The Capacitor `app/mobile/` would be
removed by the very next commit anyway; merging it just to delete
it pollutes `main` history with code nobody runs. The
`feat/mobile-platform` branch is preserved locally as
archaeology; the snapshot of in-flight Sessions deep v1 work is
on `mobile/deep-v1-sessions`. Neither is pushed to `main`.

Backend-side work from phase B2 (mobile-login endpoint, device
tokens schema, mobile_token_ttl config) **is not Capacitor-
specific** and should be cherry-picked from `feat/mobile-platform`
into `feat/mobile-flutter` (or the equivalent backend changes
re-applied) so Flutter can hit the existing endpoints from F1
onward.

The shared-package extraction work (A1/A2/A3 — pnpm workspaces +
`app/shared/` + `app/shared-ui/`) is also retained on
`feat/mobile-flutter`: web continues to benefit from it even
though mobile no longer consumes it.
