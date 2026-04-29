# opendray web

Production web frontend for opendray. Per ADR 0007/0008.

## Stack

- React 19 + TypeScript + Vite
- Tailwind CSS v4 (CSS-variable theme tokens; dark + light)
- shadcn/ui-style primitives (Radix under the hood) sourced into `src/components/ui/`
- TanStack Router + TanStack Query
- Zustand for client state (auth, theme, session tabs)
- xterm.js for terminal rendering (W2)

## Develop

```bash
pnpm install
pnpm dev          # → http://localhost:5173 (proxies /api → http://127.0.0.1:8770)
pnpm build        # type-check + production bundle into ./dist
pnpm preview      # serve the prod bundle locally
```

The dev server proxies `/api/*` to the Go gateway so login + REST work
without CORS plumbing. Run `opendray serve -config config.toml` from
the repo root in another terminal.

## Layout

```
src/
├── components/
│   ├── ui/             shadcn primitives (Button, Input, Label, …)
│   ├── AppShell.tsx    sidebar + topbar + outlet layout
│   ├── SidebarNav.tsx  primary navigation
│   └── Topbar.tsx      title + theme toggle + account
├── lib/
│   ├── api.ts          fetch wrapper, bearer auth, 401 redirect
│   └── utils.ts        cn() helper for tailwind class merging
├── pages/              one component per route
├── stores/
│   ├── auth.ts         zustand persisted token + username + expiry
│   └── theme.ts        zustand persisted theme mode (dark / light / system)
├── router.tsx          TanStack Router tree (root → login | protected)
├── main.tsx            entry: QueryClientProvider + RouterProvider
└── index.css           Tailwind + Raycast-inspired CSS-variable tokens
```

## Theme

Dark by default. The `<html>` element carries the `dark` class; theme
mode is persisted to `opendray.theme`. Toggle in the topbar cycles
dark → light → system.

## Auth

Login at `/login` posts to `/api/v1/auth/login`. Token is persisted in
`opendray.auth` (localStorage). The router's `protected` parent route
redirects to `/login?next=...` whenever the token is missing or
expired. `lib/api.ts` clears the store and redirects on any 401.

## Roadmap (per ADR 0008)

- W0 (this commit) — scaffold, tokens, login, sidebar, placeholders
- W1 — full auth flow polish, command palette skeleton, toast
- W2 — sessions workbench: list, terminal (xterm.js), tabs, buffer replay
- W3 — providers + channels + integrations CRUD
- W4 — reverse proxy console, live event stream viewer
- W5 — polish, keyboard shortcuts, embed into Go binary at `/admin`
