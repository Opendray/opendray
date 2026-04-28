# ADR 0006 — Integration Registry: dual auth, scope-deferred, proxy path

**Status:** Accepted
**Date:** 2026-04-29
**Decider:** Linivek

## Context

Design §8.2 calls for an Integration Registry: external apps register,
get an API key, can call OpenDray, can subscribe to events, and can be
reverse-proxied. Three sub-decisions need to lock for M3:

1. **How does an integration authenticate**, given the gateway already
   has an admin bearer scheme (M2.5)?
2. **How are the design's scopes (`session:read`, `event:subscribe:...`,
   etc.) enforced**?
3. **What URL hierarchy** hosts CRUD vs. WS vs. reverse proxy without
   colliding in chi's router?

## Decisions

### 1. Dual-auth on business endpoints

Sessions / providers / channels endpoints accept **either**:

- An admin bearer token (`Authorization: Bearer <admin_token>`), as
  M2.5 already issues.
- An integration API key (`Authorization: Bearer odk_live_<b64u>`).

A single combined middleware tries admin first, then integration. The
request context records the resolved Principal (`{Kind, ID, Scopes}`).

`/integrations` CRUD endpoints stay admin-only — integrations cannot
register or modify other integrations. The reverse-proxy entry point
also stays admin-only for M3 (operator-driven access to peer apps).

The integration WS event endpoint accepts the integration's own API
key only (admin can read events via the same bus through other means).

### 2. Scope strings stored, not enforced (deferred to v1.1)

Each integration's `scopes` JSONB column holds the design strings:

- `session:read`, `session:create`, `session:input`
- `channel:send`, `channel:receive`
- `event:subscribe:<topic>` — supports prefix wildcards like
  `event:subscribe:session.*`
- `provider:read`

Default at registration: `["session:read", "event:subscribe:session.*"]`
unless admin specifies otherwise.

**Enforcement is partial in M3**:

- The event WS handler **does** check `event:subscribe:<topic>` before
  attaching a subscription, since events are the only thing an
  integration uniquely consumes.
- Other business endpoints (`POST /sessions`, `PATCH /providers/...`)
  do **not** check scopes in M3 — any valid integration token has
  the same surface as admin.

Reason: scope enforcement adds checks to every handler in three
subsystems for a feature only one (hypothetical) consumer benefits
from. v1.1 layers the checks in once a real integration arrives that
needs the constraint.

### 3. URL paths

Design §11 originally writes:

- CRUD: `/api/v1/integrations/{id}`
- Proxy: `/api/v1/integrations/{prefix}/*`

These collide in chi's trie — `{id}` and `{prefix}` are both 1-segment
URL parameters at the same level. Choices:

- (a) Move CRUD to `/integrations/_/{id}` (underscore separator).
- (b) Move proxy to `/api/v1/proxy/{prefix}/*`.
- (c) Validate that integration IDs and route prefixes can never
  overlap, then dispatch by inspection.

We pick **(b)**. Reasons:

- The proxy is semantically distinct ("traverse OpenDray to reach a
  registered peer"); a top-level `/api/v1/proxy/...` namespace is
  more discoverable than burying it inside `/integrations`.
- (a) leaves an ugly `_` segment in normal admin URLs forever.
- (c) requires opaque validation rules across two tables and breaks
  if the rules drift.

Final routing under `/api/v1`:

```
POST   /integrations                         register
GET    /integrations                         list
GET    /integrations/{id}                    detail
PATCH  /integrations/{id}                    update
DELETE /integrations/{id}                    deregister
POST   /integrations/{id}/rotate-key         rotate
GET    /integrations/_events                 events WS (integration auth)

ANY    /proxy/{prefix}/*                     reverse proxy (admin auth)
```

The leading `_` on `_events` keeps it clear of any future integration
named `events`, mirroring `/channels/_kinds` and `/catalog`'s
conventions.

### 4. API key format

`odk_live_` + 32 random bytes, base64url-encoded, no padding. Final
shape `odk_live_<43 chars>`, total 51 chars. Stored as bcrypt hash
in `integrations.api_key_hash`.

Prefix mirrors the GitHub / Stripe convention so:
- Operators visually distinguish keys from admin tokens (which are
  raw 43-char b64 strings, no prefix).
- Future GitGuardian-style scanners can detect leaks from this prefix.
- Future "test" keys (`odk_test_...`) are easy to differentiate.

## Consequences

- **M3 ships in ~1500 LOC** instead of ~2000 (no per-handler scope
  rewrites in session / catalog / channel).
- **Every existing endpoint stays usable** with the same admin token;
  integration tokens just become a parallel valid principal.
- **Path migration cost if (b) is wrong**: integrations can be moved
  from `/proxy/...` back under `/integrations/...` with a server-side
  alias and a deprecation header. Clients have to update once.
- **Scope enforcement at v1.1**: when added, it's a single-handler
  helper `auth.RequireScope(ctx, "session:create")` plus the matching
  call site edits. Admin principals always pass.

## Trigger to revisit

- **Re-enable per-handler scope enforcement** when at least one
  integration ships and the operator wants asymmetric capability
  between integrations (e.g. PetTracker can read sessions but cannot
  send to channels).
- **Move proxy back under `/integrations/`** when chi (or a routing
  refactor) supports a clean dispatch and external clients ask for it.
- **Rotate API key prefix to `odk_test_`** for non-prod when env
  matters.
