// IntegrationClient — minimal reference SDK for talking to opendray
// from a third-party application. Covers the four call shapes a
// real integration needs:
//
//   1. login()                — admin → bearer token (one-time, for setup)
//   2. createIntegration()    — register the third-party app (admin-only)
//   3. apiCall<T>()           — generic JSON REST helper, bearer-aware
//   4. wsEvents()             — subscribe to /integrations/_events
//
// Everything else (sessions, channels, history, etc.) is just
// `apiCall(path, opts)` — there's no hard-coded endpoint surface
// here on purpose, so this file doesn't need updates when the
// gateway adds new endpoints.

import WebSocket from 'ws'

export interface ClientOptions {
  /** Base URL of the gateway, no trailing slash. */
  base: string
  /** Bearer token. Either an admin token (login) or an integration API key. */
  token?: string
}

export interface LoginResponse {
  token: string
  /** ISO-8601 timestamp when the token expires. */
  expires_at: string
  username: string
}

export interface Integration {
  id: string
  name: string
  base_url: string
  route_prefix: string
  scopes: string[]
  version?: string
  enabled: boolean
  health_status: string
  created_at: string
  rotated_at?: string | null
}

export interface RegisterResult {
  integration: Integration
  api_key: string
}

export interface RegisterRequest {
  name: string
  base_url: string
  route_prefix: string
  scopes?: string[]
  version?: string
}

export interface ApiError extends Error {
  status: number
  body: unknown
}

function makeError(status: number, body: unknown, msg: string): ApiError {
  const err = new Error(msg) as ApiError
  err.status = status
  err.body = body
  return err
}

export class OpendrayClient {
  base: string
  token: string | undefined

  constructor(opts: ClientOptions) {
    this.base = opts.base.replace(/\/$/, '')
    this.token = opts.token
  }

  /**
   * POST /api/v1/auth/login — exchanges admin credentials for a
   * bearer token. Returns the token + expiry; the client stashes
   * it on `this.token` so subsequent calls don't need it explicit.
   */
  async login(username: string, password: string): Promise<LoginResponse> {
    const res = await fetch(`${this.base}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password }),
    })
    if (!res.ok) {
      throw makeError(res.status, await safeJson(res), `login failed: ${res.status}`)
    }
    const data = (await res.json()) as LoginResponse
    this.token = data.token
    return data
  }

  /** GET /api/v1/integrations — admin-only list of registered apps. */
  async listIntegrations(): Promise<Integration[]> {
    const data = await this.apiCall<{ integrations: Integration[] }>(
      '/api/v1/integrations',
    )
    return data.integrations ?? []
  }

  /** POST /api/v1/integrations — admin-only register. Returns the key ONCE. */
  async createIntegration(req: RegisterRequest): Promise<RegisterResult> {
    return this.apiCall<RegisterResult>('/api/v1/integrations', {
      method: 'POST',
      body: req,
    })
  }

  /** POST /api/v1/integrations/{id}/rotate-key — issues a new API key. */
  async rotateKey(id: string): Promise<{ api_key: string }> {
    return this.apiCall<{ api_key: string }>(
      `/api/v1/integrations/${encodeURIComponent(id)}/rotate-key`,
      { method: 'POST' },
    )
  }

  /** DELETE /api/v1/integrations/{id} — admin-only remove. */
  async deleteIntegration(id: string): Promise<void> {
    await this.apiCall<void>(
      `/api/v1/integrations/${encodeURIComponent(id)}`,
      { method: 'DELETE' },
    )
  }

  /**
   * Generic JSON REST helper. Path must start with "/api/v1/...".
   *
   * Accepts an admin token OR an integration API key in
   * `this.token` — opendray's middleware accepts either as a
   * Bearer credential.
   */
  async apiCall<T>(
    path: string,
    init: { method?: string; body?: unknown; headers?: Record<string, string> } = {},
  ): Promise<T> {
    const headers: Record<string, string> = { ...(init.headers ?? {}) }
    if (this.token) headers['Authorization'] = `Bearer ${this.token}`
    let body: string | undefined
    if (init.body !== undefined) {
      headers['Content-Type'] = 'application/json'
      body = JSON.stringify(init.body)
    }
    const res = await fetch(`${this.base}${path}`, {
      method: init.method ?? 'GET',
      headers,
      body,
    })
    if (res.status === 204) return undefined as T
    const json = await safeJson(res)
    if (!res.ok) {
      const msg =
        json && typeof json === 'object' && 'error' in json
          ? String((json as { error: unknown }).error)
          : `HTTP ${res.status}`
      throw makeError(res.status, json, msg)
    }
    return json as T
  }

  /**
   * Subscribe to the gateway event bus over WebSocket.
   *
   * `topics` is required by the server — pass each topic the
   * caller has `event:subscribe:<topic>` scope for, e.g.
   * `["session.*", "integration.*"]`. Each topic is checked
   * against the integration's scopes individually; an unscoped
   * topic causes a 403 before the upgrade.
   *
   * The current bearer token is forwarded as `?token=` because
   * browsers can't add WS headers; we use the same shape in Node
   * for parity.
   */
  wsEvents(
    topics: string[],
    onEvent: (e: BusEvent) => void,
    onClose?: (code: number, reason: string) => void,
  ) {
    const proto = this.base.startsWith('https') ? 'wss' : 'ws'
    const host = this.base.replace(/^https?:\/\//, '')
    const qs = new URLSearchParams({
      topics: topics.join(','),
      token: this.token ?? '',
    })
    const url = `${proto}://${host}/api/v1/integrations/_events?${qs}`
    const ws = new WebSocket(url)
    ws.on('message', (raw) => {
      try {
        const ev = JSON.parse(raw.toString()) as BusEvent
        onEvent(ev)
      } catch {
        // skip malformed
      }
    })
    ws.on('close', (code, reason) => onClose?.(code, reason.toString()))
    ws.on('error', (err) => onClose?.(-1, (err as Error).message))
    return {
      close: () => ws.close(),
      raw: ws,
    }
  }
}

export interface BusEvent {
  topic: string
  data: Record<string, unknown>
  ts: string
}

async function safeJson(res: Response): Promise<unknown> {
  const ct = res.headers.get('content-type') ?? ''
  if (!ct.includes('application/json')) return null
  try {
    return await res.json()
  } catch {
    return null
  }
}
