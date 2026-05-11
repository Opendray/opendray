# ADR 0015 — Mobile platform: Capacitor wraps a shared React SPA

**Status**: Superseded by [ADR 0017](./0017-mobile-flutter-rewrite.md) (2026-05-08)
**Date**: 2026-05-07

> **Historical note (2026-05-08).** Phases A1–A3, B1–B6, and T2–T4
> shipped to `feat/mobile-platform` per this ADR. After running the
> Capacitor build on iPhone, the WebView ceiling was judged
> unacceptable for the maintainer's primary-surface use case. The
> mobile platform was rewritten in Flutter under ADR 0017. This
> document is preserved for the rationale chain; do not start new
> work from it.

## Context

v2 needs a mobile app. Constraints established with the maintainer:

1. **No code reuse from v1** — full rewrite based on the v2 stack
2. **Web and mobile are equal-priority surfaces.** Both expose the
   full admin feature set. Neither is a "lite" version of the other.
3. **Four target viewports**: iOS phone, iOS tablet, Android phone,
   Android tablet. The implementation must cover all four well; in
   practice the phone↔tablet split inside iOS / Android is more
   significant than the iOS↔Android split.
4. **Distribution decisions are deferred** until end-to-end
   development and testing complete. No TestFlight / App Store /
   Play Store choices are made in this ADR.
5. **v2's existing web stack is locked**: React 19 + Vite + TS +
   Tailwind v4 + shadcn/ui + TanStack Router + TanStack Query +
   Zustand + xterm.js. The built bundle is embedded into the Go
   binary via `go:embed all:dist` and served at `/admin/*`.

## Decision

### 1. Cross-platform shell — Capacitor 6

We wrap a React SPA inside a native shell (Capacitor) rather than
build a separate mobile UI in Flutter, React Native, or platform
native (Swift/Kotlin).

**Why Capacitor wins for opendray specifically:**

- **xterm.js is non-negotiable.** Remote control of Claude / Codex /
  shell PTY sessions is the product. xterm.js is the only mature
  terminal renderer for the platforms we care about. Capacitor's
  WebView runs xterm.js with zero changes; every other option means
  porting / writing a terminal from scratch (months of work).
- **The web SPA already exists and must stay.** A separate Flutter
  or React Native app would mean maintaining two parallel UIs at
  full feature parity — unaffordable on this team size.
- **Tradeoff accepted**: WebView performance is below truly native;
  bundle size is larger. For an admin / dashboard / terminal app,
  this is fine. We're not building a 60fps game.

Rejected alternatives:
- **Flutter** — mature, would match the existing CLAUDE.md
  `build_release.sh` infrastructure, but no shared code with the
  React web stack and no path to xterm.js without writing native
  plugins. Maintenance cost prohibitive.
- **React Native** — shares React mental model but not React DOM;
  every web component would need rewriting. xterm.js works inside
  `react-native-webview` only as an island, not as the main UI.
- **Tauri Mobile** — architecturally close to what we want, but
  still pre-1.0 / alpha-grade as of 2026-05. Production risk too
  high.
- **PWA only** — iOS Safari constraints (background WebSocket,
  push notifications until very recently, no App Store presence)
  make this insufficient even though it costs the least.

### 2. Repository layout — pnpm workspaces monorepo

```
app/
├── shared/          ← Pure logic (no React)
│   ├── api/         ← REST client (fetch + Zod schemas)
│   ├── ws/          ← WebSocket subscription helpers
│   ├── types/       ← Session, Memory, Channel, Integration, …
│   └── hooks/       ← Business hooks: useSession, useMemory, …
├── shared-ui/       ← React components, cross-viewport
│   ├── primitives/  ← shadcn-derived base components
│   ├── patterns/    ← Composite components (DataTable, SessionCard, …)
│   ├── layouts/     ← AdaptiveSidebar, ResponsiveSheet, …
│   ├── terminal/    ← xterm.js + touch keyboard abstraction
│   └── breakpoints.ts
├── web/             ← Desktop browser entry (thin)
│   ├── src/main.tsx ← Bootstrap, mounts shared-ui Provider, routes
│   ├── src/routes/
│   └── vite.config.ts
└── mobile/          ← Capacitor entry (thin)
    ├── src/main.tsx ← Bootstrap + Capacitor plugin wiring
    ├── src/routes/  ← 95% reuse from web; a few mobile-only routes
    ├── ios/         ← Capacitor-generated Xcode project
    ├── android/     ← Capacitor-generated Gradle project
    ├── capacitor.config.ts
    └── vite.config.ts
```

**`app/web/dist/` continues to be the path Go's `go:embed` reads
from.** The new structure does not break the existing build /
deploy story for the web binary.

`app/mobile/` produces its own bundle inside `app/mobile/dist/`,
which Capacitor `cap sync` copies into `ios/App/App/public/` and
`android/app/src/main/assets/public/`.

### 3. Viewport strategy — 5-tier breakpoints, single design language

A single `app/shared-ui/breakpoints.ts` defines:

| Tier | Width | Devices | Layout paradigm |
|------|-------|---------|-----------------|
| `xs` | 0–479pt   | iPhone SE / compact | Single column, bottom tabs, drawer |
| `sm` | 480–767pt | iPhone Pro Max / phone landscape | Single column optimized, bottom tabs |
| `md` | 768–1023pt | iPad mini / 11" portrait | Sidebar + main, 2-column tables |
| `lg` | 1024–1279pt | iPad 12.9" / laptop | Sidebar + main + detail panel |
| `xl` | ≥1280pt | Desktop browser | Sidebar + main + detail + tools |

The mobile build (`app/mobile/`) renders fluidly from `xs` through
`lg` (12.9" iPad in landscape). The web build (`app/web/`) renders
fluidly from `sm` through `xl`. Both reuse the same components
from `shared-ui`; viewport adaptation lives inside those
components, not branched at the entry level.

### 4. Tablet routing — tablets ship the mobile build

A 12.9" iPad in landscape has 1366pt width — close to a desktop.
There are two ways to serve this:

- Tablets visit the web SPA in Safari (responsive design)
- Tablets install the mobile app from App Store / Play Store

We pick the second. Reason: tablet users get push notifications,
biometric auth, deep links, secure token storage — all of which
are valuable on a tablet too (e.g. iPad as primary work device),
and none of which the responsive web app provides.

This decision means the mobile build's tablet layout must be
genuinely good, not "phone-sized UI stretched". The 5-tier
breakpoint system enforces this: `lg` mode in the mobile build
matches `lg` mode in the web build.

### 5. Authentication — password + biometric (phase 1)

Mobile login flow:

1. First launch: server-URL onboarding screen — user enters their
   opendray gateway URL (e.g. `https://opendray.example.com`),
   the app verifies `/api/v1/health`
2. Login screen: admin username + password (same `/api/v1/auth/login`
   endpoint as web)
3. Bearer token returned by the server is stored via
   `@capacitor/preferences` (which maps to iOS Keychain and
   Android EncryptedSharedPreferences)
4. On subsequent launches:
   - If biometric is enabled → prompt Face ID / Touch ID, on
     success unlock the stored token, no password re-entry
   - If biometric is disabled → require password again

Server-side: the existing `/api/v1/auth/login` works as-is. We add
a new optional `mobile_token_ttl` field to the `[admin]` section
of `config.toml` (default 30d, vs 24h for browser tokens), so
mobile users don't re-login daily.

Future enhancements (out of scope for phase 1):

- **QR-code pairing**: desktop web shows a one-time QR; mobile
  scans, exchanges a short-lived code for a long-lived token over
  WSS. Better UX but ~2 weeks of work to do safely.
- **Per-device tokens** with revocation list (currently a single
  admin can have multiple active mobile tokens, all invalidated
  by password rotation).

### 6. Backend protocol — shared with web; mobile-specific extensions are opt-in

The mobile app uses the existing `/api/v1/*` endpoints. Mobile is
"just another HTTP/WS client." No protocol fork.

Mobile-specific additions (each behind its own migration / config
field, all backwards compatible):

- `[admin].mobile_token_ttl` — separate TTL for tokens issued via
  the mobile login path
- `device_tokens` table — schema for push-notification tokens.
  Created in the same migration that adds `mobile_token_ttl` even
  though push notifications are deferred to phase C, so the schema
  exists before any client tries to populate it.
- `POST /api/v1/auth/mobile-login` — sibling endpoint to
  `/api/v1/auth/login` that returns a longer-TTL token.

Phase C extensions (deferred):
- `POST /api/v1/devices/register` / `DELETE /api/v1/devices/:id`
- `POST /api/v1/notifications/test` (operator-side: send a push
  to all registered devices)

## Consequences

### Positive

- **Single design language**: web and mobile feel like one product;
  no "we have two apps" branding mismatch
- **API guaranteed in sync**: types live in `app/shared/`, both
  entries import them; no protocol drift
- **Feature work pays off twice**: a new feature page lives in
  `shared-ui` and works on both web and mobile out of the box
- **xterm.js continues to be the primary terminal renderer**;
  no second implementation to maintain
- **Backend stays Go-only**: no Node / Dart / Swift / Kotlin in
  the gateway; backend reviewers don't need new skills
- **Distribution-decision optionality preserved**: Capacitor
  supports App Store, Play Store, AdHoc IPA via UNAS SMB, OTA
  via `@capgo/capacitor-updater`, etc. — all reachable from the
  same source. We keep the choice for later.

### Negative

- **WebView performance ceiling**: animations / scroll perf is
  not native-quality. Acceptable for admin UIs; a deal-breaker for
  game / 60fps creative work, neither of which we're building.
- **iOS background limits hit hard**: WebSocket connections die
  when the app suspends. Real-time event delivery in background
  requires push notifications, which are explicitly deferred.
- **App size larger than native**: a Capacitor app starts at
  ~15 MB ipa baseline (WebView chrome + system frameworks
  references) before our bundle. For an admin app this is fine.
- **Two build flows to maintain**: `app/web/` (Vite → Go binary)
  and `app/mobile/` (Vite → Capacitor → native shells). The
  monorepo + shared dependencies dampen this; it is not zero
  cost.

### Compatibility

- **No breaking change to v2 backend or web**. The
  `app/web/dist/` path that `internal/web/embed.go` reads is
  preserved. Existing `pnpm build` from the v2 web package keeps
  producing the same artifact at the same location.
- **CLAUDE.md global rule**: the existing "all Flutter mobile
  apps must have `build_release.sh`" rule applies in spirit — a
  Capacitor `build_release.sh` will live in `app/mobile/` with
  the same UNAS SMB / TestFlight upload contract (using the same
  ASC API key + Keychain SMB credential).
- **CI**: `feat/mobile-platform` long-lived branch is gated by
  the same checks as main (added in #16). Each sub-PR must pass
  Frontend + Backend + Lint before integration.
- **`docs/integration-guide.md`** is unaffected — integrations
  speak to the same gateway whether the human is on web or
  mobile.

## Implementation references

The implementation lands as a series of sub-PRs into the
`feat/mobile-platform` long-lived branch:

| Phase | PR(s) | Outcome |
|-------|-------|---------|
| A1 | TBD | pnpm workspaces; existing web continues to build |
| A2 | TBD | `app/shared/` extracted (api + types + hooks) |
| A3 | TBD | `app/shared-ui/` extracted (components + layouts + terminal) |
| A4 | TBD | `app/web/` reduced to a thin entry |
| A5 | TBD | 5-tier breakpoints + adaptive layout primitives |
| B1 | TBD | Capacitor initialization + iOS / Android dev builds |
| B2 | TBD | Backend mobile-login + device_tokens schema |
| B3 | TBD | Mobile auth flow (server URL, password, secure storage) |
| B4 | TBD | Biometric unlock |
| B5 | TBD | Sessions list (first real screen) |
| B6 | TBD | Session detail + mobile-optimized terminal view |
| C  | TBD | Push notifications, remaining admin pages, deep links, OTA |

PR numbers will be filled in as each lands.

## Future work

- **Push notifications**: APNs + FCM, fired from the existing
  event bus on `session.idle` / `session.ended` / errors
- **QR-code pairing**: alternative to password login
- **Per-device token revocation list**: invalidate a single
  device without touching the admin password
- **OTA web-bundle updates**: `@capgo/capacitor-updater` self-hosted
  on the gateway; lets us ship hot fixes without store review
- **Universal links / deep linking**: `opendray://session/sess_42`
- **App Store / Play Store packaging**: deferred until end-to-end
  testing completes
- **Tablet split-view multitasking**: iPadOS lets two apps share a
  screen; ensure layouts behave at non-standard widths
- **Watch companion** (Apple Watch / Wear OS): one-tap send
  predefined commands. Speculative.
