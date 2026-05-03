# ADR 0010 — Integration call log

**Status:** Accepted (minimum-viable; deferred polish — see §Deferred)
**Date:** 2026-05-03
**Decider:** Linivek
**Relates to:** ADR 0006 (integration auth), ADR 0009 (events WS extended to admin)
**Code:**
- `internal/store/migrations/0010_integration_call_log.sql`
- `internal/integration/calllog.go`
- `internal/integration/calllog_http.go`
- `internal/integration/proxy.go` (modified)
- `internal/app/app.go` (wired)
- `app/web/src/pages/Activity.tsx`
- `app/web/src/lib/integrationCalls.ts`

## Context

Through Phase 2 W4 the Activity page shipped twice and twice failed
the "is this useful day-to-day?" test:

1. **First iteration** (W4 ADR 0009): WebSocket buffer + topic chips.
   Lost on every page refresh; no historical query; drowned in
   `session.idle` polling artifacts.
2. **Second iteration** (this work, earlier pass): rewired to read
   from `audit_log`. Persistence solved (1) but the underlying data
   was still >95% noise — the events that actually matter to a
   day-to-day operator (something *broke* / something *was called by
   a third party*) were either missing or buried.

The user-facing diagnosis was sharp: opendray's *primary* product
value is being a **gateway for third-party apps to call it via an
integration API key**. The admin-UI-as-driver flow is secondary and
already self-documenting (you spawned it, you know what you did).
The traffic that's *worth observing* — and the traffic that's a
genuinely unanswered question without this — is the integration
call flow:

- Which third-party app called what endpoint?
- Did the calls succeed?
- Was an admin's reverse-proxy call to a registered integration's
  upstream healthy?

`audit_log` (added in §12) was never going to answer those
questions: it captures **lifecycle** events (registered,
key_rotated, health_changed) on a curated topic allowlist, not
**per-call** API traffic.

## Decision

Add a dedicated `integration_call_log` table separate from
`audit_log`. Records every API request whose principal is an
integration (inbound) or whose target is an integration via the
reverse proxy (outbound).

```sql
CREATE TABLE integration_call_log (
    id              BIGSERIAL PRIMARY KEY,
    ts              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    integration_id  TEXT NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
    direction       TEXT NOT NULL CHECK (direction IN ('inbound','outbound')),
    method          TEXT NOT NULL,
    path            TEXT NOT NULL,
    status_code     INT  NOT NULL,
    duration_ms     INT  NOT NULL,
    bytes_written   BIGINT,
    request_id      TEXT,
    resource_kind   TEXT,    -- nullable; future: extracted from path
    resource_id     TEXT     -- nullable; future: extracted from path
);
```

Indexes (migration `0010`):

- `idx_intgr_call_by_intgr (integration_id, ts DESC)` — per-integration timeline
- `idx_intgr_call_by_ts (ts DESC)` — global recent stream
- `idx_intgr_call_errors (ts DESC) WHERE status_code >= 400` — partial; "show me failures"

### Capture path

**Inbound** (third-party app → opendray):

A `CallLogger.Middleware` is mounted **after** `CombinedMiddleware`
in the dual-auth chi group. It reads `Principal` from request
context — if `Kind == KindIntegration`, it timestamps the request,
wraps the response writer to capture status + bytes, and after the
handler returns, enqueues a write. **Admin-attributed requests are
silently skipped** (this is what makes the page trustworthy as a
"third-party gateway view").

**Outbound** (admin → `/proxy/{prefix}/*` → integration upstream):

`ProxyHandlers.serve` calls `CallLogger.LogOutbound` explicitly
after the proxy returns. The integration is identified by the URL
prefix lookup that already happens for routing, so no extra DB hit.
Direction is always `"outbound"`; principal is admin but is not
recorded (the *integration* is the subject of interest).

### Write path

Producer is non-blocking: `record()` does a `select` against a
256-deep buffered channel and drops with a WARN log if full. A
single consumer goroutine `consume()` drains the channel and
issues one INSERT per row. This mirrors `audit.Sink` for
consistency, and keeps DB latency off the request hot path.

Graceful shutdown: `App.Run` calls `intgrCallLogger.Close()` after
the HTTP server has stopped accepting connections, which closes
the channel and lets the consumer drain remaining rows.

### Read path

Single endpoint: `GET /api/v1/integrations/_calls` (admin-only,
under the admin middleware group). Filters: `integration_id`,
`direction`, `status_class` (2/3/4/5), `since`, `until`. Cursor
pagination by `id` descending; default limit 100, max 500.

The `_calls` literal segment shadows the `{id}` wildcard in the
sibling admin handler `GET /integrations/{id}` because chi prefers
static matches over wildcards.

### Frontend

`pages/Activity.tsx` — table view with three filters (integration
dropdown, direction, status class) and a 5-second poll. Empty
state explains explicitly that admin-UI calls are NOT recorded so
the absence of rows isn't confusing for first-time users.

## Rationale

### Why a separate table from `audit_log`?

Different access patterns. `audit_log` records ~10–100 rows/day
(config changes); `integration_call_log` could be 1000s/min in a
busy gateway. Mixing them means:
- Indexes on `audit_log` get bloated with call rows.
- Retention policies have to handle both shapes (audit wants
  90+ days; call traffic wants 7–30).
- The `metadata JSONB` column on `audit_log` is a poor fit for
  status_code / duration_ms which are first-class query targets.

Costs of separation: two writers, two read paths. Acceptable —
the writers are independent goroutines, and the read paths serve
fundamentally different UI surfaces.

### Why log outbound proxy calls in the same table?

Both directions are about *integration traffic visibility*. From
the operator's view, "fake-app called us 12 times" and "we
called fake-app 12 times via proxy" are the same kind of question.
A `direction` column keeps them queryable as one stream when
useful, and as two streams when not.

### Why not log admin-attributed requests?

Three reasons:
1. Volume — every page nav fires several /api calls; the admin UI
   would dominate the table.
2. Self-documenting — you triggered it from the UI, you already
   know what happened.
3. Trust contract — the page promises "this is the third-party
   gateway view"; mixing in admin requests breaks that promise.

If admin auditing is later needed, it belongs in `audit_log` (with
a curated topic allowlist), not here.

## Consequences

- New migration `0010` must be applied (`opendray migrate`) before
  the gateway boots with this code, otherwise the FK target is OK
  but inserts fail. The wired `intgrCallLogger` only logs warnings
  on insert failure — it does not crash the request hot path.
- Existing `fake-app` integration (disabled, unhealthy at time of
  writing) will not produce rows until enabled and called.
- The Activity page no longer reads `audit_log` at all.
  `audit_log` data is still queryable via `GET /api/v1/audit/log`
  (admin-only, added in the previous pass) and is consumed by the
  Sessions Inspector "Activity" tab — that view is intentionally
  left untouched because per-session lifecycle is a different and
  still-valid use case.

## Deferred

Built only the minimum-viable slice. The following are intentional
deferrals — pick them up when one of the trigger conditions below
fires.

| Item | Why deferred | Trigger to build |
|------|--------------|------------------|
| Stats KPI cards (calls/min, error rate, p95 latency) | Decorative when there are zero rows | First real integration accumulates ≥100 calls/day for a week |
| Per-integration "View activity" deep link from Integrations page | No detail page exists yet on Integrations | When an Integration detail page is built, OR when 2+ active integrations exist |
| "Load older" pagination button | Initial 100 covers ≥7 days at current traffic projections | When `next_cursor` is consistently present after first page refresh |
| Resource extraction (resource_kind/id from path or response) | Path parsing is brittle; doing it now risks getting it wrong | When users want "show me all calls that touched session X" |
| Retention worker (prune rows older than N days) | Table empty; no urgency | Row count crosses 100k OR table size > 100MB |
| Outbound proxy call body capture | Privacy + storage cost | Explicit user request for "show me what the proxy sent/got back" |
| Per-integration rate limit / quota view | Requires telemetry tier on integrations table | When abuse is observed OR when multi-tenant integrations land |
| WebSocket live tail (instead of 5s poll) | Polling is good enough for low volumes | When p95 perceived latency from a call to it appearing > 3s annoys someone |
| Sparkline / call-volume chart | Only meaningful with sustained traffic | After stats cards land |

Code-side anchors (search for `TODO(adr-0010)` to find them):

- `internal/integration/calllog.go` — extension points for resource
  extraction, retention, and stats helpers
- `internal/integration/calllog_http.go` — extension points for the
  per-integration scoped endpoint
- `app/web/src/pages/Activity.tsx` — extension points for stats
  cards and per-integration filter UX

## Trigger to revisit

Revisit the *whole architecture* (separate-table decision included)
if any of these become true:

- Call volume sustains >10 writes/sec for >1h: per-row INSERT
  becomes a bottleneck — switch to batched writes (50–200 rows per
  transaction with a 1s flush).
- A second observability surface lands (e.g. session-level call
  log, channel-level call log) and shares >70% of the schema:
  consider extracting a generic `event_log` table with a discriminator
  column.
- We add multi-tenant integrations: `integration_call_log` will
  need an `actor_id` (which user-of-integration made the call) in
  addition to `integration_id`.
