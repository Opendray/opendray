# opendray web

Production web frontend for opendray. Per ADR 0007 (stack) and
ADR 0008 (IA / flows / visual language).

## Stack

- React 19 + TypeScript + Vite (rolldown)
- Tailwind CSS v4 + tw-animate-css (CSS-variable theme tokens, dark + light)
- shadcn/ui-style primitives (Radix under the hood) sourced into
  `src/components/ui/`
- TanStack Router + TanStack Query
- Zustand for client state (auth, theme, session tabs)
- xterm.js for terminal rendering
- cmdk for the command palette
- sonner for toasts
- date-fns for relative timestamps

## Develop

```bash
pnpm install
pnpm dev          # http://localhost:5173 (proxies /api → :8770)
pnpm build        # writes ../../internal/web/dist for go:embed
pnpm preview      # serve the prod bundle locally
```

The dev server proxies `/api/*` to the Go gateway so login + REST + WS
work without CORS plumbing. Run `opendray serve -config config.toml`
from the repo root in another terminal.

`pnpm build` writes to **`../../internal/web/dist/`** (not `./dist`).
The Go binary embeds that directory via `//go:embed dist` and serves
it at `/admin/*` after `go build`. See [the root README](../../README.md)
for the combined build flow.

## Layout

```
src/
├── components/
│   ├── ui/                      shadcn primitives (Button, Dialog, Tabs, …)
│   ├── sessions/                SessionList / Tabs / Terminal / SpawnDialog
│   ├── providers/ConfigForm     dynamic configSchema renderer
│   ├── channels/                (handled inline in pages/Channels.tsx)
│   ├── integrations/            ProxyConsole + APIKeyRevealDialog
│   ├── AppShell.tsx             sidebar + topbar + suspense outlet + health banner
│   ├── SidebarNav.tsx           primary navigation
│   ├── Topbar.tsx               title + search button + theme + account
│   ├── CommandPalette.tsx       ⌘K palette (cmdk)
│   └── HealthBanner.tsx         /health-driven destructive banner
├── lib/
│   ├── api.ts                   fetch wrapper, bearer auth, 401 redirect
│   ├── ws.ts                    BinaryWS (reconnecting WS for terminal stream)
│   ├── catalog.ts | channels.ts | integrations.ts | sessions.ts | types.ts
│   └── utils.ts                 cn() helper
├── pages/                       one component per route
├── stores/
│   ├── auth.ts                  persisted token + username + expiry
│   ├── theme.ts                 persisted theme mode
│   └── sessionTabs.ts           open tabs + currentId
├── router.tsx                   TanStack Router tree (basepath driven by Vite base)
├── main.tsx                     entry: QueryClientProvider + RouterProvider + Toaster
└── index.css                    Tailwind + Raycast tokens
```

## Theme

Dark by default. `<html>` carries the `dark` class; mode persists to
`opendray.theme`. Toggle in the topbar (Light / Dark / System).

## Auth

Login posts to `/api/v1/auth/login`. Token persists in `opendray.auth`
(localStorage). The router's protected parent route redirects to
`/login?next=...` when the token is missing or expired. `lib/api.ts`
clears the store + raises a "Session expired" toast + redirects on
any 401. WebSocket upgrades carry the bearer in `?token=` since
browsers cannot set Authorization on WS handshake.

## Production

In production, `pnpm build` outputs to `../../internal/web/dist`
which the Go binary embeds. Vite is configured to set `base: '/admin/'`
for builds; Vite dev keeps `base: '/'`. The TanStack Router reads
`import.meta.env.BASE_URL` so client-side routing works in both modes.

Code splits (Vite rolldown manualChunks):
- `react`: react / react-dom / scheduler (~60 kB gzip)
- `tanstack`: query + router (~37 kB gzip)
- `xterm`: xterm.js + addons (~88 kB gzip, **only loaded on
  `/sessions`** via `React.lazy`)
- `index`: entry + login + remaining pages (~90 kB gzip)

First-paint at `/login` ≈ 187 kB gzip; session workbench adds the
xterm chunk on-demand.

## Milestones (delivered)

- W0: scaffold, tokens, login, sidebar
- W1: command palette, topbar, settings page, toasts
- W2: sessions workbench: list, multi-tab, xterm.js, spawn dialog
- W3: providers / channels / integrations CRUD pages
- W4: Activity live event viewer + Integrations reverse-proxy console
- W5: code-split, /admin/ basepath, health banner, go:embed
