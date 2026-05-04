# TypeScript SDK

opendray ships a **reference client** at
`examples/integrations/demo-client/` that you can copy as a
starting point. It's not a published SDK package — it's plain
source you fork and adapt.

## What the reference shows

The demo runs nine numbered steps end-to-end:

```
 1. Connect to opendray
 2. Load credentials  (state file → reuse or fresh registration)
 3. List active sessions
 4. Spawn a shell session
 5. Send input
 6. Read terminal buffer
 7. Fetch project history
 8. Subscribe to event bus over WebSocket
 9. Cleanup spawned session
```

Run it locally:

```bash
cd examples/integrations/demo-client
cp .env.example .env             # edit ADMIN_USER / PASSWORD
pnpm install
pnpm dev                         # first run registers + saves key
pnpm dev                         # second run reuses saved key
pnpm reset                       # delete integration + state file
```

## File layout

```
demo-client/
├── package.json       # tsx + ws + dotenv only, zero framework
├── tsconfig.json      # ESNext + strict
├── .env.example       # OPENDRAY_BASE, admin creds, scopes…
├── .gitignore         # blocks .env and .demo-state.json
├── README.md          # full walkthrough
└── src/
    ├── client.ts      # OpendrayClient class (REST + WS)
    ├── state.ts       # load/save/clear .demo-state.json (mode 0600)
    ├── reset.ts       # `pnpm reset` entry point
    └── index.ts       # `pnpm dev` — 9-step demo flow
```

## The OpendrayClient class

`client.ts` is intentionally minimal — about 200 lines of TS.
Three primitives that cover everything:

```ts
const client = new OpendrayClient({ base, token: apiKey })

// 1. Login flow (only used at setup time)
await client.login('admin', 'pass')

// 2. Generic JSON REST helper. Path starts with /api/v1/.
//    Bearer is automatic from constructor.
const list = await client.apiCall<{ sessions: Session[] }>(
  '/api/v1/sessions',
)
const created = await client.apiCall<Session>('/api/v1/sessions', {
  method: 'POST',
  body: { provider_id: 'shell', cwd: '/tmp' },
})

// 3. WebSocket events.
const ws = client.wsEvents(
  ['session.*', 'integration.*'],   // topics
  (ev) => console.log(ev.topic, ev.data),
  (code, reason) => console.log('closed', code, reason),
)
// later
ws.close()
```

The class doesn't enumerate endpoints. Whatever opendray exposes
under `/api/v1/...`, your code reaches via `apiCall<T>(path, ...)`
with the right TypeScript shape — define those shapes in your own
app based on what you actually consume.

## Credential lifecycle

`index.ts::authenticate()` is the four-branch decision tree:

```
state file?    saved key works?    branch
─────────────────────────────────────────────────
   no               n/a             Fresh registration
   yes              yes (200)       Reuse saved key
   yes              401             Recover by rotating
   yes              other error     Fall through to fresh
```

Take this exact pattern into your own consumer, swapping
`state.ts` for whatever secret-store integration fits your
deployment (see [Authentication →
Where to store your API key](#consuming-authentication)).

## What's NOT in the reference

- **Retry / backoff** — `apiCall` throws on the first non-2xx.
  In production you want exponential backoff with jitter on
  5xx + idempotent 4xx. See [Error handling](#consuming-error-handling).
- **Connection pooling** — Node's `fetch` reuses connections under
  the hood, so this rarely matters; a heavy consumer might want
  to swap to `undici` directly.
- **Telemetry** — no metrics or traces. Add OpenTelemetry around
  `apiCall` if you want endpoint timing.
- **Rate-limit handling** — opendray doesn't rate-limit
  integrations today. If that changes (429 will show up in the
  catalogue) the consumer pattern is to honor `Retry-After`.

## Porting to other languages

The contract is plain HTTP + WebSocket; nothing TypeScript-specific.
The patterns translate directly:

- **Python**: `requests` for REST + `websockets` library for the
  WS subscription. State file with `os.chmod(path, 0o600)`.
- **Go**: `net/http` + `gorilla/websocket`, identical structure.
- **Rust**: `reqwest` + `tokio-tungstenite`.
- **bash + curl**: works for everything except event subscription
  (no usable WS client in pure bash; pipe through `websocat`).

The README inside the demo-client directory has a fuller
adaptation table.
