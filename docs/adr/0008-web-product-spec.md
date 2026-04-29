# ADR 0008 — Web product spec: IA, flows, visual language

**Status:** Accepted
**Date:** 2026-04-29
**Decider:** Linivek

## Mission for the web client

> Be the primary work surface for an operator running multiple AI CLI
> sessions through opendray. Optimised for keyboard, dense
> information, and a single long-lived browser tab.

Reference inspiration: **Raycast** — dark-first, keyboard-native,
information-dense, single-color palette plus accent, fluid motion,
command palette as the central navigation device.

## 1. Information architecture

```
opendray (root layout)
├── Sessions (default landing once authed)
│   ├── List (sidebar)
│   ├── Workbench (main: terminal, tabs, transcript scrollback)
│   └── New session (dialog)
├── Providers
│   ├── Catalog list
│   └── Provider detail (manifest + config form)
├── Channels
│   ├── List
│   └── Channel detail (telegram bot config, notify_on, test send)
├── Integrations
│   ├── List
│   ├── Register (dialog with one-time API key reveal)
│   ├── Detail (scopes, health, base_url, rotate-key)
│   └── Reverse proxy console
├── Activity
│   ├── Audit log (stream + filters)
│   └── Live events (WS feed; admin-side viewer)
└── Settings
    ├── Theme (dark / light / system)
    ├── Account (admin → /auth/me, logout)
    └── About (version, commit, /health snapshot)
```

Top-level navigation lives in the **left sidebar**. Sessions is the
default route after login. Sidebar items each have a single-letter
shortcut (`g s`, `g p`, `g c`, `g i`, `g a`, `g ,`).

## 2. Primary flows

### Flow A — Day-zero (first login)

1. `/login` — single form (username + password). On success: persist
   token in localStorage, redirect to `/sessions`.
2. Empty state: "No sessions yet. Press ⌘N or Spawn to start."
3. Spawn dialog: provider selector (catalog) + cwd + args (optional).
4. New session opens as a tab in the workbench, terminal connects
   via WS, ring-buffer replays.

### Flow B — Daily driver

1. `/sessions` — sidebar shows live + ended sessions, ordered by
   last activity. Live ones bear a state pill (running / idle).
2. Click or `j/k` to focus, `Enter` opens the session in a new tab.
3. Multiple tabs across the workbench. `⌘1..⌘9` switches; `⌘W`
   closes; `⌘⇧T` reopens last closed.
4. Terminal supports paste, copy on select, link click, resize on
   container resize. Scrollback up to ring buffer capacity (1 MiB).

### Flow C — Command palette (⌘K)

Single fuzzy palette, three result groups in priority order:

1. **Sessions** (open, focus, terminate) — typing matches name + cwd
2. **Navigation** (`Sessions`, `Providers`, `Channels`, …)
3. **Actions** (`Spawn session…`, `Register integration…`,
   `Toggle theme`, `Logout`)

Palette closes on ⎋ or selection. Keyboard hints visible on every
row (e.g. ⌘K then `⌘↵` to terminate).

### Flow D — Configuration round-trip

1. Open `Providers` → select `claude` → form renders from manifest's
   `configSchema`.
2. Edit → save → `PATCH /providers/claude/config`.
3. Toast confirms; cached query refetches.
4. Same shape for `Channels` (config-as-form) and
   `Integrations` (register form + scopes multiselect).

### Flow E — Reverse proxy console

`Integrations → <name> → Reverse proxy`:

- Path input (relative to base_url), method dropdown, headers/body.
- Send → `/api/v1/proxy/{prefix}/<path>` → response panel below.
- Saved request history per integration (localStorage).

### Flow F — Live events viewer

`Activity → Live events`:

- Topic filter chips (`session.*`, `channel.*`, `integration.*`,
  `admin.*`).
- Streams to `/api/v1/integrations/_events` using a special "admin
  viewer" mechanism — for v1 we mint a synthetic admin-only event
  WS endpoint that bypasses integration auth (server-side TODO,
  added when this view ships).
- Auto-scroll, pause, JSON pretty-print on row expand.

## 3. Visual language (Raycast-inspired)

### 3.1 Color tokens

CSS variables, both modes, OKLCH where supported:

```css
--bg-base:        oklch(13% 0.012 270)   /* near-black, slight cool */
--bg-elevated:    oklch(17% 0.014 270)
--bg-overlay:     oklch(22% 0.016 270)
--border-soft:    oklch(28% 0.012 270 / 0.5)
--border-strong:  oklch(40% 0.014 270)
--fg-primary:     oklch(96% 0.005 270)
--fg-secondary:   oklch(72% 0.010 270)
--fg-muted:       oklch(52% 0.012 270)

/* Single accent — Raycast-orange leaning */
--accent-base:    oklch(72% 0.18 35)     /* warm orange */
--accent-strong:  oklch(80% 0.20 35)
--accent-muted:   oklch(60% 0.16 35 / 0.2)

/* Status pills */
--state-running:  oklch(72% 0.16 145)    /* green */
--state-idle:     oklch(72% 0.18 90)     /* yellow */
--state-ended:    oklch(60% 0.012 270)   /* neutral gray */
--state-failed:   oklch(64% 0.20 25)     /* red */
```

Light mode mirrors these with inverted lightness and lower chroma.
The toggle flips a `data-theme="light"|"dark"` attribute on `<html>`.

### 3.2 Typography

- UI: `Inter Variable` (system fallback: `-apple-system, Segoe UI,
  sans-serif`).
- Mono / terminal: `JetBrains Mono Variable`
  (system fallback: `ui-monospace, Menlo, Consolas, monospace`).
- Sizes (rem): 11px caption / 12px body / 13px ui-default /
  14px section / 18px page-title / 24px hero.
- Line height: 1.45 for body; 1.20 for tight UI rows
  (sidebar items, table rows).

### 3.3 Spacing / radius / shadow

- Spacing scale (Tailwind): 1 / 2 / 3 / 4 / 6 / 8 / 12 / 16 / 24
  (px = rem × 16 with default base).
- Radius: `4px` for inputs, `6px` for cards, `8px` for dialogs,
  `12px` for command palette.
- Shadow: minimal, only on dialogs / popovers / palette
  (`0 4px 16px rgba(0,0,0,0.25)` dark; `0 4px 16px rgba(0,0,0,0.08)`
  light). Body content stays flat.

### 3.4 Motion

- 120 ms ease-out for hover state changes.
- 180 ms ease-out for popover / dialog enter; 120 ms for exit.
- 240 ms with cubic-bezier(0.32, 0.72, 0, 1) for command palette
  open (Raycast curve).
- Reduced-motion query disables non-functional motion globally.

### 3.5 Density

- Sidebar item: 28 px tall, 12 px horizontal padding.
- Table row: 32 px tall.
- Form field: 36 px tall, 12 px horizontal padding.
- Dialog: 480 px wide for forms, 720 px for proxy console.
- Command palette: 640 px wide, max 6 visible result rows.

## 4. Component inventory (W0-W3)

shadcn-derived: `Button`, `Input`, `Label`, `Textarea`, `Select`,
`Dialog`, `Sheet`, `DropdownMenu`, `Tabs`, `Tooltip`, `Toast`,
`Command` (palette), `Switch`, `Badge`, `Separator`, `ScrollArea`.

Custom (`src/components/`):

- `AppShell` — sidebar + topbar + main layout
- `SidebarNav` — active route highlighting + shortcut hints
- `Topbar` — page title slot + theme toggle + account menu
- `CommandPalette` — fuzzy across sessions + nav + actions
- `SessionList` — virtualised list of session rows
- `SessionRow` — name + cwd + state pill + relative time
- `SessionTabs` — workbench tab strip with close affordance
- `Terminal` — xterm.js + WS bridge + resize observer
- `ProviderCard` — manifest icon + title + capabilities
- `ConfigForm` — render `ConfigField[]` into form inputs
- `IntegrationKeyDialog` — one-time api_key reveal with copy + warning
- `HealthBadge` — colored pill for `healthy / degraded / unhealthy / unknown`
- `EventStream` — virtualised live-events list with topic filter chips
- `EmptyState` — large icon + title + body + CTA

## 5. Accessibility / keyboard

- Focus rings always visible (no `outline: none` blanket override).
- Sidebar nav: `g`-prefix shortcuts (`g s`, `g p`, …).
- Command palette: `⌘K` / `Ctrl K`.
- Terminal: `⌘W` close tab, `⌘1..9` switch, `⌘N` new session.
- Dialogs trap focus; ⎋ closes; restore focus on close.
- Color contrast AA for all text + interactive states (verified via
  the Raycast palette which already meets it).

## 6. State / data layer rules

- **TanStack Query** owns every list/detail fetch. Cache keys:
  `["sessions"]`, `["sessions", id]`, `["providers"]`,
  `["channels"]`, `["integrations"]`, `["audit", filters]`,
  `["health"]`. Mutations invalidate the relevant key.
- **Zustand stores**:
  - `useAuth` — token, username, expires_at; persisted via
    `persist` middleware to localStorage; cleared on 401.
  - `useTheme` — `'dark' | 'light' | 'system'`; effects update
    `document.documentElement` data-theme.
  - `useSessionTabs` — `[{id, name}]` open in workbench;
    `currentId`; persisted (so refresh restores tab order).
- **WebSocket**: one socket per session terminal; reconnect with
  exponential backoff + `?since=<cursor>`; pong every 20s.

## 7. Error / empty / loading patterns

- Empty state: large lucide icon (40 px), title 14 px, body 12 px,
  CTA button.
- Loading: skeleton rows (no spinners on lists). Dialog forms: spin
  icon inside the submit button.
- Error: red `Toast` for transient (5 s), red banner above content
  for persistent (e.g. "DB unreachable", drives off `/health`).
- 401: useAuth.clear → redirect to `/login` with `?next=<path>`.

## 8. Out of scope for v1.0 web

Captured here so they do not creep in:

- Internationalisation (English-only per operator decision).
- Mobile layouts (< 768 px). Tablet ≥ 768 px is a stretch goal.
- Plugin marketplace UI (anti-goal per design §4).
- Live collaboration / multi-admin presence indicators.
- Screen recording / session replay video.
- Custom themes beyond dark/light system.
- Push notifications (browser PNs deferred until M5+ mobile).
