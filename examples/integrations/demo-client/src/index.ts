// Demo flow: prove that a third-party application can drive
// opendray end-to-end using only the public API + an integration
// key. Output is plain text + colour codes so you can read it
// straight off the terminal.
//
// What this script proves, in order:
//
//   1. A first run registers a fresh integration with the admin
//      token and persists the api_key into .demo-state.json
//      (mode 0600). This is exactly how a real third-party app
//      ought to handle its bearer credential — store it once,
//      reuse forever.
//   2. Subsequent runs load the saved key and use it directly,
//      with NO admin login required.
//   3. If the key has been rotated externally (operator clicked
//      "Rotate key" in the UI) the demo's first authenticated
//      call returns 401. The demo recognises that, falls back to
//      the admin token to rotate again, and updates state — the
//      smallest possible recovery path.
//   4. The api_key is sufficient to drive every session-management
//      endpoint a real third-party app would need: list, spawn,
//      send-input, read-buffer, fetch-history.
//   5. The same key authenticates a long-running WebSocket on
//      /integrations/_events so the third-party can react to
//      events the gateway emits (session.idle, etc.).
//
// Run: `pnpm i && pnpm dev` from this directory after copying
// .env.example to .env and adjusting credentials.
//
// Reset: `pnpm reset` deletes the integration row + state file so
// the next run starts from scratch.

import 'dotenv/config'
import { setTimeout as sleep } from 'node:timers/promises'

import {
  OpendrayClient,
  type ApiError,
  type BusEvent,
  type Integration,
} from './client.js'
import { loadState, saveState, STATE_PATH, type DemoState } from './state.js'

const cfg = readConfig()

async function main() {
  step(1, 'Connect to opendray', `${cfg.base}`)
  console.log(`   state file: ${STATE_PATH}`)

  step(2, 'Load credentials')
  const auth = await authenticate()
  ok(
    `${auth.source} · integration ${auth.integrationId}`,
  )

  // From here on the demo stops using any admin token (if we
  // even had one). Every call below carries only the integration's
  // API key — exactly what an external app would do.
  const client = new OpendrayClient({ base: cfg.base, token: auth.apiKey })

  // Subscribe BEFORE we touch any session-mutating endpoint so
  // the WS gets the `session.started` event we're about to
  // trigger by spawning. Events accumulate in `seen` in the
  // background while the rest of the demo runs.
  const seen: BusEvent[] = []
  const ws = client.wsEvents(
    ['session.*', 'integration.*'],
    (ev) => {
      seen.push(ev)
      console.log(`   ↳ ${ev.topic.padEnd(28)} ${oneLine(ev.data)}`)
    },
    (code, reason) => {
      if (code !== 1000 && code !== 1005)
        console.log(`   ⚠ ws closed: ${code} ${reason}`)
    },
  )
  // The WS handshake is async; brief sleep so the server-side
  // bus.Subscribe completes before we publish the first event.
  await sleep(300)
  ok('event subscription open (session.* + integration.*)')

  step(3, 'List active sessions (consumer auth)', '/api/v1/sessions')
  const list = await client.apiCall<{ sessions: SessionRow[] }>(
    '/api/v1/sessions',
  )
  console.log(`   ${(list.sessions ?? []).length} session(s) currently running.`)
  for (const s of (list.sessions ?? []).slice(0, 5)) {
    console.log(`   · ${s.id}  ${s.provider_id.padEnd(7)}  ${s.state.padEnd(8)}  ${s.cwd}`)
  }

  step(4, 'Spawn a shell session in', cfg.cwd)
  const spawned = await client.apiCall<SessionRow>('/api/v1/sessions', {
    method: 'POST',
    body: {
      name: `demo-${Date.now()}`,
      provider_id: 'shell',
      cwd: cfg.cwd,
    },
  })
  ok(`spawned ${spawned.id} (pid ${spawned.pid})`)

  step(5, 'Send input', '"echo hello from demo-client"')
  await client.apiCall<void>(
    `/api/v1/sessions/${encodeURIComponent(spawned.id)}/input`,
    {
      method: 'POST',
      // Shell expects \n; Claude expects \r (raw mode).
      body: { data: 'echo hello from demo-client\n' },
    },
  )
  ok('input forwarded to PTY')

  // Give the shell a beat to print before we read the buffer.
  await sleep(800)

  step(6, 'Read terminal buffer', `/sessions/${spawned.id}/buffer`)
  const buffer = await fetchBuffer(client, spawned.id)
  for (const line of buffer.split(/\r?\n/).slice(-6)) {
    console.log(`   │ ${line}`)
  }

  step(7, 'Fetch project history', '(claude only — empty for shell)')
  const history = await client.apiCall<{
    entries: { ts: string; text: string; session_id: string }[]
    unsupported_provider?: boolean
  }>(`/api/v1/sessions/${encodeURIComponent(spawned.id)}/history?limit=5`)
  if (history.unsupported_provider) {
    console.log(`   ⓘ History API returned unsupported_provider=true (expected for shell).`)
  } else {
    console.log(`   ${history.entries.length} historic prompt(s).`)
  }

  step(8, 'Wait for events', `${cfg.eventSeconds}s window — idle / stop / end events`)
  // The shell session went silent after step 5; idle threshold
  // (default 30 s) may fire within the window. Either way, we'll
  // catch session.stopped/ended when we delete it next.
  await sleep(cfg.eventSeconds * 1000)

  step(9, 'Cleanup spawned session')
  await client.apiCall<void>(
    `/api/v1/sessions/${encodeURIComponent(spawned.id)}`,
    { method: 'DELETE' },
  )
  // Give the server a moment to publish session.stopped / ended.
  await sleep(800)
  ws.close()
  ok(`session deleted; ${seen.length} event(s) captured total`)

  console.log()
  console.log('\x1b[32m✓ demo finished\x1b[0m')
  console.log(
    `\x1b[2m  state preserved at ${STATE_PATH} — re-run to reuse the same key, or 'pnpm reset' to wipe.\x1b[0m`,
  )
}

main().catch((err) => {
  console.error('\x1b[31m✗ demo failed\x1b[0m')
  console.error(err)
  process.exit(1)
})

// ----- credential lifecycle -----

interface AuthResult {
  apiKey: string
  integrationId: string
  /** Human-readable description of which branch produced the key. */
  source: string
}

/**
 * authenticate is the four-branch decision tree that gets the demo
 * a working api_key, mirroring how a real third-party app would
 * handle its bearer credential.
 *
 *   1. State file exists  → verify the key  → use it
 *   2. State file exists  → key returns 401 → recover by rotating
 *                                              (admin auth needed)
 *                                            → save new key
 *   3. State file absent  → register fresh  → save key
 *   4. (Edge) Existing row with same name but no state file →
 *      delete + re-register so we own a fresh credential.
 */
async function authenticate(): Promise<AuthResult> {
  const state = loadState()

  if (state) {
    console.log(`   loaded state for integration ${state.integration_id}`)
    const verdict = await verifyKey(state.api_key)
    if (verdict === 'ok') {
      return {
        apiKey: state.api_key,
        integrationId: state.integration_id,
        source: 'reused saved key',
      }
    }
    if (verdict === 'rotated') {
      console.log(
        `   ⚠ saved key returned 401 — assume operator rotated it; recovering…`,
      )
      const fresh = await rotateAndSave(state)
      return {
        apiKey: fresh.api_key,
        integrationId: fresh.integration_id,
        source: 'recovered by rotate',
      }
    }
    // Other failure (network, integration deleted, etc.) — fall
    // through to the registration path below.
    console.log(
      `   ⚠ saved key probe failed (${verdict}); attempting fresh registration`,
    )
  }

  return await registerFreshAndSave()
}

/**
 * verifyKey hits a small read endpoint and inspects the result:
 * `ok` for 2xx, `rotated` for 401 (means the saved key no longer
 * authenticates), `error` for anything else.
 */
async function verifyKey(apiKey: string): Promise<'ok' | 'rotated' | string> {
  const probe = new OpendrayClient({ base: cfg.base, token: apiKey })
  try {
    await probe.apiCall('/api/v1/sessions')
    return 'ok'
  } catch (err) {
    const status = (err as ApiError).status
    if (status === 401) return 'rotated'
    return `HTTP ${status ?? '???'}`
  }
}

/** Login admin → rotate key for the saved integration → save new state. */
async function rotateAndSave(prev: DemoState): Promise<DemoState> {
  const admin = new OpendrayClient({ base: cfg.base })
  await admin.login(cfg.adminUser, cfg.adminPassword)
  const { api_key } = await admin.rotateKey(prev.integration_id)
  const next: DemoState = {
    ...prev,
    api_key,
    registered_at: new Date().toISOString(),
  }
  saveState(next)
  console.log(
    `   ✓ rotated and saved new key to ${STATE_PATH}`,
  )
  return next
}

/**
 * registerFreshAndSave is used the first time the demo runs (no
 * state file) AND when the saved state file points at a row that
 * has been deleted server-side (the rotate recovery loop above
 * would otherwise also fall here).
 *
 * Side effect: if a row with the configured name already exists
 * (e.g. created via the UI), it's deleted first so this demo
 * always owns the fresh credential.
 */
async function registerFreshAndSave(): Promise<AuthResult> {
  const admin = new OpendrayClient({ base: cfg.base })
  await admin.login(cfg.adminUser, cfg.adminPassword)

  const existing = await admin.listIntegrations()
  const match = existing.find((i: Integration) => i.name === cfg.integrationName)
  if (match) {
    console.log(
      `   ⚠ integration "${cfg.integrationName}" already exists (${match.id}); deleting so demo owns the next key`,
    )
    await admin.deleteIntegration(match.id)
  }

  const result = await admin.createIntegration({
    name: cfg.integrationName,
    // Consumer-only integration — no reverse proxy, no health
    // probe. opendray treats blank base_url + route_prefix as
    // "this is just an API consumer".
    base_url: '',
    route_prefix: '',
    scopes: [
      'session:read',
      'session:create',
      'session:input',
      'event:subscribe:session.*',
      'event:subscribe:integration.*',
      'provider:read',
    ],
    version: '0.1.0',
  })
  saveState({
    integration_id: result.integration.id,
    integration_name: result.integration.name,
    api_key: result.api_key,
    registered_at: new Date().toISOString(),
  })
  console.log(
    `   ✓ registered ${result.integration.id} and saved key to ${STATE_PATH}`,
  )
  return {
    apiKey: result.api_key,
    integrationId: result.integration.id,
    source: 'fresh registration',
  }
}

// ----- helpers -----

interface SessionRow {
  id: string
  name: string
  provider_id: string
  cwd: string
  state: string
  pid?: number
}

/**
 * fetchBuffer pulls the PTY ring snapshot. Endpoint returns
 * raw bytes (octet-stream) — we use fetch directly so we can
 * read text instead of going through apiCall's JSON path.
 */
async function fetchBuffer(client: OpendrayClient, sessionId: string): Promise<string> {
  const headers: Record<string, string> = {}
  if (client.token) headers['Authorization'] = `Bearer ${client.token}`
  const res = await fetch(
    `${client.base}/api/v1/sessions/${encodeURIComponent(sessionId)}/buffer`,
    { headers },
  )
  if (!res.ok) return `<HTTP ${res.status}>`
  return res.text()
}

function readConfig() {
  const env = process.env
  return {
    base: env.OPENDRAY_BASE ?? 'http://127.0.0.1:8770',
    adminUser: env.OPENDRAY_ADMIN_USER ?? 'admin',
    adminPassword: env.OPENDRAY_ADMIN_PASSWORD ?? '',
    integrationName: env.INTEGRATION_NAME ?? 'demo-client',
    eventSeconds: parseInt(env.EVENT_LISTEN_SECONDS ?? '15', 10),
    cwd: env.SESSION_CWD ?? '/tmp',
  }
}

function step(n: number, title: string, detail?: string) {
  console.log(
    `\n\x1b[36m${String(n).padStart(2, ' ')}.\x1b[0m \x1b[1m${title}\x1b[0m${
      detail ? `  \x1b[2m${detail}\x1b[0m` : ''
    }`,
  )
}

function ok(msg: string) {
  console.log(`   \x1b[32m✓\x1b[0m ${msg}`)
}

function oneLine(o: Record<string, unknown>): string {
  const keys = Object.keys(o)
  if (keys.length === 0) return ''
  const head = keys.slice(0, 3).map((k) => `${k}=${stringify(o[k])}`).join(' ')
  return keys.length > 3 ? `${head} (+${keys.length - 3} more)` : head
}

function stringify(v: unknown): string {
  if (typeof v === 'string') return v.length > 40 ? `${v.slice(0, 37)}…` : v
  return String(v)
}
