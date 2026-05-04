# opendray demo client

Reference TypeScript program that proves a third-party application
can drive opendray end-to-end through the public API. It models
the exact credential lifecycle a real consumer would follow:

1. **First run** registers a new integration with the admin token
   and **persists the api_key** to a local file.
2. **Subsequent runs** load the saved key and never touch the
   admin token again.
3. **If the operator rotates the key in the UI**, the next run
   detects 401 and recovers by rotating once with the admin
   token — emulating what an automated consumer would do under a
   secret-rotation policy.
4. `pnpm reset` undoes everything (deletes the integration row
   and removes the state file) so you can replay the demo from
   scratch.

## Quick start

```bash
cd examples/integrations/demo-client
cp .env.example .env
# edit .env: OPENDRAY_BASE, ADMIN_USER, ADMIN_PASSWORD
pnpm install
pnpm dev
```

A typical first-run output:

```
 1. Connect to opendray  http://127.0.0.1:8770
   state file: …/demo-client/.demo-state.json

 2. Load credentials
   ⚠ integration "demo-client" already exists (int_…); deleting so demo owns the next key
   ✓ registered int_… and saved key to …/.demo-state.json
   ✓ fresh registration · integration int_… · key odk_live_…

 3. List active sessions (consumer auth)  /api/v1/sessions
   3 session(s) currently running.
 4. Spawn a shell session in  /tmp
 5. Send input  "echo hello from demo-client"
 6. Read terminal buffer
   │ hello from demo-client
 7. Fetch project history
 8. Wait for events  15s window
   ↳ session.started   …
 9. Cleanup spawned session
   ↳ session.stopped   …
   ✓ session deleted; 2 event(s) captured total
✓ demo finished
  state preserved at …/.demo-state.json
```

Run it again immediately and you'll see:

```
 2. Load credentials
   loaded state for integration int_…
   ✓ reused saved key · integration int_… · key odk_live_…
```

That's the demo proving the saved key still works without any
admin login. **Steps 3-9 run exactly the same way.**

## How key storage works

The first-run path persists a JSON file alongside `package.json`:

```
demo-client/
└── .demo-state.json     # mode 0600, gitignored
   {
     "integration_id":   "int_Qp8vBWT5WHiu",
     "integration_name": "demo-client",
     "api_key":          "odk_live_...",
     "registered_at":    "2026-05-04T05:55:25.571Z"
   }
```

This file replaces what a production app would put in a real
secret store. The points it illustrates:

- **The api_key is stored once.** opendray never re-displays it.
- **`mode 0600`** keeps other shell users on the host from
  reading it. The demo enforces the perms on every write.
- **`.gitignore`** stops it from leaking into commits.

A real third-party app should swap `state.ts` for whichever
mechanism fits its environment:

| Environment | Likely backing |
|---|---|
| Local dev script | env var, `~/.config/<app>/credentials` |
| macOS desktop app | Keychain (`security` CLI / `node-keytar`) |
| Linux desktop app | libsecret / GNOME Keyring |
| Linux daemon | systemd `LoadCredential=` / file with mode 0600 |
| Server | AWS Secrets Manager, GCP Secret Manager, Vault, etc. |

## The four credential branches

`authenticate()` in `src/index.ts` is the decision tree. It runs
on every demo start and chooses one of these branches:

| State file? | Saved key works? | Branch | What happens |
|---|---|---|---|
| ✗ | n/a | **Fresh registration** | Login admin, register new row, save key. |
| ✓ | ✓ | **Reuse saved key** | Skip admin login entirely. |
| ✓ | 401 | **Recover by rotate** | Login admin, rotate key for the existing row, save new key. |
| ✓ | other error | **Fall through to fresh** | Treat as unrecoverable; re-register. |

## Testing rotate-key invalidation

This is the lifecycle scenario you can't see clearly without a
persistent state:

1. `pnpm dev` — fresh registration; state saved.
2. `pnpm dev` — reuses saved key (`✓ reused saved key`).
3. Open the web UI, go to **Integrations → demo-client → Rotate key**.
4. Acknowledge the new key in the dialog (you can ignore it for
   this test — we want to see the recovery branch fire).
5. `pnpm dev` — output now shows:
   ```
   loaded state for integration int_…
   ⚠ saved key returned 401 — assume operator rotated it; recovering…
   ✓ rotated and saved new key (…)
   ✓ recovered by rotate · integration int_…
   ```
   Steps 3-9 then run with the freshly-recovered key.

That's proof that opendray actually invalidates the old key
on rotation **and** that the demo's recovery path lets a
live consumer self-heal.

## Resetting

```bash
pnpm reset
```

Deletes the DB integration row + removes the state file. The
next `pnpm dev` will go down the **Fresh registration** branch,
identical to what a brand-new operator would experience.

## File layout

```
demo-client/
├── package.json       # tsx + ws + dotenv
├── tsconfig.json      # ESNext + strict
├── .env.example       # OPENDRAY_BASE, admin creds, scopes…
├── .gitignore         # blocks .env and .demo-state.json
├── README.md          # this file
└── src/
    ├── client.ts      # OpendrayClient class (REST + WS)
    ├── state.ts       # load/save/clear .demo-state.json
    ├── reset.ts       # `pnpm reset` entry point
    └── index.ts       # `pnpm dev` — 9-step demo flow
```

## Adapting this for your own app

`OpendrayClient` has a generic `apiCall<T>()` that hits any
`/api/v1/...` endpoint with the bearer your app holds:

```ts
const client = new OpendrayClient({
  base: 'https://your-opendray.example.com',
  token: yourStoredApiKey,
})

// Any REST endpoint:
const sessions = await client.apiCall<{ sessions: Session[] }>(
  '/api/v1/sessions',
)

// POST with a body:
const spawned = await client.apiCall<Session>('/api/v1/sessions', {
  method: 'POST',
  body: { provider_id: 'shell', cwd: '/tmp' },
})

// Live event subscription:
const ws = client.wsEvents(
  ['session.*'],
  (ev) => console.log(ev.topic, ev.data),
)
```

Replace `state.ts` with your own secret-store integration. The
rest of the demo flow can be read as a checklist for what to
implement in your client: verify-on-startup, recover-on-401,
graceful subscription teardown.

## Scopes used

When registering the integration, the demo asks for:

```
session:read
session:create
session:input
event:subscribe:session.*
event:subscribe:integration.*
provider:read
```

Trim or widen depending on what your app actually needs. The
full scope list lives at `app/web/src/lib/types.ts: ALL_SCOPES`.

## Troubleshooting

**`HTTP 401: unauthorized`** on the very first call — admin
credentials in `.env` don't match `config.toml`. Log in once via
the web UI to confirm.

**Recovery loop returns 404 on rotate** — the integration row
was deleted (e.g. via the UI). The demo should fall through to
fresh registration; if it doesn't, run `pnpm reset` and try
again.

**`Cannot find module 'tsx'`** — run `pnpm install` first.

**WS closes immediately with code 1006** — the integration's
scopes don't include the topic you're subscribing to. Reset and
re-register to pick up the demo's full scope set.
