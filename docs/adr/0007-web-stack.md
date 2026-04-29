# ADR 0007 — Web frontend stack

**Status:** Accepted
**Date:** 2026-04-29
**Decider:** Linivek

## Context

Phase 2 of v2 is a production web frontend that becomes the user's
primary work surface — not a debug console. Choices (locked here):

- **Framework**: React 18 + TypeScript
- **Build / dev**: Vite
- **Styling**: Tailwind CSS v4 with CSS variables for theme switching
- **Component primitives**: shadcn/ui (Radix under the hood)
- **Server state**: TanStack Query (`@tanstack/react-query`)
- **Routing**: TanStack Router (`@tanstack/react-router`)
- **Client state**: Zustand
- **Terminal renderer**: xterm.js + xterm-addon-fit + xterm-addon-web-links
- **HTTP**: native `fetch` wrapped in `lib/api.ts`
- **WebSocket**: native `WebSocket` wrapped in `lib/ws.ts` (reconnect + cursor)
- **Icons**: lucide-react
- **Form state / validation**: react-hook-form + zod (only where needed)
- **Date / time**: native `Intl.*` + `date-fns` for relative formatting

design §5 left web client as TBD (Vue 3 was the v1 choice). ADR 0007
overrides that — Vue 3 was rejected to align with the "v2 doesn't
inherit v1 by default" principle (ADR 0001).

## Rationale

| Choice | Why |
|---|---|
| React 18 | Largest ecosystem for admin / workbench UIs. Most reusable patterns for the kind of UX we need (terminal, dashboards, command palette). Future contributor pool is broadest. |
| TypeScript | Non-negotiable at this scope. Catches the integration-shape mismatches (REST schemas, WS event payloads) at edit time. |
| Vite | Sub-second HMR for the iteration speed Phase 2 needs. No SSR requirement (Go gateway serves the SPA shell, no Node runtime). |
| Tailwind v4 + CSS vars | Tailwind for utility-first speed; CSS variables cleanly support the dark/light toggle without a re-render of every styled component. |
| shadcn/ui | Source-first components — they live IN the repo (not as a `node_modules` dependency), so we can mutate any primitive when the Raycast-inspired design needs it. Radix gives accessibility for free. |
| TanStack Query | Server-state management is the dominant complexity in this app (4 REST surfaces × CRUD × cache invalidation). Query handles cache, retry, focus refetch, optimistic updates. |
| TanStack Router | Type-safe routes with inferred search params. Same generation/lifecycle model as Query, lower mental overhead than React Router v6 for this team size. |
| Zustand | Tiny client-state store for `currentTheme`, `currentSessionTabs`, `commandPaletteOpen`. No Redux boilerplate; no Context perf headaches. |
| xterm.js | Industry-standard terminal renderer used by VS Code. Plugin ecosystem (fit, web-links) handles the obvious quality-of-life bits. |
| react-hook-form + zod | Used selectively (login, channel/integration register dialogs). Provider config form schema is rendered dynamically from manifest, not a static zod schema, so most of the app does not import either. |

## Rejected

| Stack | Why not |
|---|---|
| Vue 3 + Element Plus / Naive UI | Carries v1 baggage; ecosystem for AI-style workbenches smaller; admin pattern reference material thinner. |
| SvelteKit | Smaller ecosystem; non-trivial hire / contribution friction; SSR features wasted on a Go-served SPA. |
| Next.js | App Router brings React Server Components and Node runtime requirements we don't need. Static export would work but adds layers without payoff. |
| Material UI | Visual identity locked to Material Design — wrong for Raycast-inspired tone. Hard to deeply restyle. |
| Ant Design Pro | Excellent admin templates but reads as "enterprise back office", not "AI agent workbench". |
| Redux Toolkit / RTK Query | Overkill. TanStack Query already covers RTK Query's surface with less ceremony. |
| Chakra UI | Smaller momentum vs shadcn/Radix in 2026; less raw-CSS access. |

## Consequences

- `app/web/` is the canonical web client; if Flutter Web is ever
  added it will be a parallel app, not a replacement.
- Bundle size budget: ≤300 kB gzipped for first paint (login route);
  ≤600 kB gzipped for the session workbench (xterm.js dominates).
- Every shadcn primitive lands in `src/components/ui/`. We never
  publish them; the repo *is* the registry.
- Web tests use Playwright (deferred to a later milestone — manual
  smoke is fine for Phase 2 W0–W4).

## Trigger to revisit

- React adoption flips and a contributor with stronger Vue/Svelte
  background takes over the web client.
- Cursor / Linear-class native shells displace web for AI workbench
  use cases broadly enough that we have to follow.
- shadcn/ui is abandoned upstream — at which point we already own
  the source so nothing breaks; we just stop pulling new components.
