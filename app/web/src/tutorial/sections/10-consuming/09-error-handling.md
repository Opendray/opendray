# Error handling

opendray uses standard HTTP status codes. The body always has
`{"error": "<message>"}` so you can log the underlying reason
without parsing prose.

## Status code catalogue

| Status | Meaning | Retry? |
|---|---|---|
| `200` / `201` / `202` | Success | n/a |
| `204` | Success, no body (DELETE, PATCH-no-change, etc.) | n/a |
| `400` | Bad request — invalid JSON / missing required field / type mismatch | No (fix the request) |
| `401` | Bearer missing or doesn't match any integration | Maybe (rotated? see [Key rotation](#consuming-key-rotation)) |
| `403` | Authenticated but missing scope | No (re-register / edit scopes) |
| `404` | Resource doesn't exist | Maybe (race with delete?) |
| `409` | Conflict (name/prefix collision, already-ended session, route_prefix taken) | No |
| `429` | Rate-limited (not currently emitted; future-compatible) | Yes, honor `Retry-After` |
| `500` | Gateway internal error — bug or DB connectivity loss | Yes, with backoff |
| `502` | Reverse-proxy target unreachable | Yes |
| `503` | Integration disabled, unhealthy, or DB unavailable | Yes |
| `504` | Reverse-proxy target timed out | Yes |

## Specific error messages worth knowing

| Body | Status | What it means |
|---|---|---|
| `unauthorized` | 401 | No credentials or invalid bearer |
| `missing scope: <name>` | 403 | Auth ok, but the request needs a scope you don't have |
| `integration not found` | 404 | The id in the path doesn't exist |
| `session not found` | 404 | Session id missing — already deleted? |
| `session has ended` | 409 | Trying to send input to a dead PTY (use `/start` first or `/sessions` to spawn fresh) |
| `route_prefix is reserved` | 400 | Picking a prefix opendray uses internally (`auth`, `proxy`, `health`, etc.) |
| `name already in use` | 409 | Pick a different `name` when registering |
| `health check failed` | 503 | Reverse-proxy target's `/health` isn't returning 2xx |

## Retry strategy

Use exponential backoff with jitter for retryable codes:

```ts
async function withRetry<T>(fn: () => Promise<T>): Promise<T> {
  const delays = [200, 800, 2000, 5000] // ms
  for (const d of delays) {
    try {
      return await fn()
    } catch (err) {
      const status = (err as ApiError).status
      if (![500, 502, 503, 504].includes(status)) throw err
      await sleep(d + Math.random() * d * 0.3)
    }
  }
  return fn() // last attempt; let the throw bubble up
}
```

Don't retry `400 / 403 / 404 / 409` — they don't fix themselves.

For `401` specifically: a single retry path goes through your key
recovery flow (re-fetch from secret store, or self-rotate) — see
[Key rotation](#consuming-key-rotation).

## Long-lived WebSocket reconnection

The events WebSocket can drop for any reason — gateway restart,
network blip, idle timeout. A robust consumer reconnects and
reconciles state:

```ts
function startEventLoop(client: OpendrayClient, topics: string[]) {
  let backoff = 1000
  function connect() {
    const ws = client.wsEvents(topics, handleEvent, (code) => {
      // 1006 = abnormal close, often network. Reconnect.
      if (code === 1000 || code === 1001) return // clean shutdown
      setTimeout(connect, backoff)
      backoff = Math.min(backoff * 2, 30_000)
    })
    ws.raw.on('open', () => { backoff = 1000 })
  }
  connect()
}
```

opendray does **not** replay missed events on reconnect. If the
gap matters to you, query the relevant REST endpoint after
reconnecting (`GET /sessions` to learn current states) before
trusting the live stream again.

## Logging recommendations

Log every non-2xx with these dimensions:

| Field | Why |
|---|---|
| HTTP status | Lets you alert on rate of non-2xx |
| Request method + path (without IDs) | Group failures by endpoint |
| `error` body field | Human-readable cause |
| Latency | Slow 5xx = downstream issue, fast 5xx = panic |
| Bearer principal hash | If you rotate often, helps correlate failures with rotation events |

For successful calls, opendray's own integration call log
(`/integrations/_calls`, admin-only) records every request the
gateway saw with caller attribution. Cross-check there if your
client-side logs disagree with the server.

## Reporting bugs

If you see a 500 with a generic message:

1. Capture the request — method, path, body, time.
2. Check the gateway log (Settings → Logging → Live tail) for an
   error around the same timestamp.
3. File at <https://github.com/Opendray/opendray_v2/issues> with
   both the request shape and the relevant log lines.

opendray is small enough that most 500s are reproducible bugs;
expect quick turnaround.
