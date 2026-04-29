# ADR 0009 — Events WS extended to admin

**Status:** Accepted
**Date:** 2026-04-30
**Decider:** Linivek
**Amends:** ADR 0006 §1 ("integration WS event endpoint accepts the
integration's own API key only")

## Context

Phase 2 W4 needs an Activity page that streams events to the
operator's web client in real time (`session.started`,
`session.ended`, `session.idle`, `channel.message_received`,
`integration.health_changed`, etc.). The bus already publishes these.

ADR 0006 originally restricted `/api/v1/integrations/_events` to
integration API keys, on the assumption admins would consume events
via "other means". In practice the only admin client is the same
web app that runs everything else; provisioning an internal
`__admin-viewer` integration just to read its own bus is awkward
and creates a long-lived API key the admin cannot rotate without
self-locking the UI.

## Decision

Allow admin bearers in addition to integration API keys on the
existing endpoint:

```
GET /api/v1/integrations/_events?topics=…
Authorization: Bearer <admin_or_integration_token>
# OR
?token=<admin_or_integration_token>
```

The handler now:

- Accepts `Principal.Kind == KindAdmin` or `KindIntegration`.
- Skips per-topic `event:subscribe:<topic>` scope checks for admin
  (admin is super, same posture as the rest of the API).
- Continues to enforce scope checks for integration principals.

Wiring: the route moves from the integration-only group to the
dual-auth (combined) group in `internal/app/app.go`. The
`IntegrationOnlyMiddleware` helper stays available for any future
integration-only endpoint.

## Rationale

- **Single source of truth.** Both admin and integration consumers
  share the same fan-in implementation (the existing `EventsHandler`).
  No duplicate code, no two protocols to keep aligned.
- **No long-lived self-key.** The admin viewer just rides the
  current admin session token. Rotation behaviour matches the rest
  of admin auth.
- **Scope semantics preserved.** Integrations still get scope
  enforcement; only admin bypasses, which is consistent with the
  rest of the gateway (admin endpoints in §3 are likewise unscoped).

## Consequences

- ADR 0006 §1 is partially superseded — events WS is no longer
  "integration only." Updated wording: "events WS accepts admin
  bearer or integration API key; scope checks apply only to the
  integration path."
- The web Activity page (W4) connects with the admin token via the
  `?token=` query (browsers cannot set Authorization on the WS
  handshake).
- Future work: when scope enforcement lands per-handler (v1.1 per
  ADR 0006), audit topics that admin-side viewers should never
  see (e.g. raw `channel.message_received` content) need a
  redaction step at publish time, not at subscribe time. Adding
  another auth tier here is the wrong fix.

## Trigger to revisit

- An admin viewer use case appears that **shouldn't** see all
  topics (e.g. multi-operator deployment with role-segmented
  audit). Then admin gets a sub-role that limits topics, and we
  re-add scope-style filtering on the admin path.
- Bus event payload starts including PII. The right fix is
  redacting at publish, not blocking subscribe.
