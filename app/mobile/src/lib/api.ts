// Minimal mobile-side API client.
//
// Unlike app/shared/src/lib/api.ts (which assumes same-origin and reads
// the token from a Zustand store), this client takes both the server
// URL and the token explicitly because:
//   1. The server URL is user-entered at first launch (onboarding)
//      and stored in Preferences — not known at module load.
//   2. The token comes from Preferences (Keychain / EncryptedSharedPrefs),
//      not from a Zustand store. We read it ad-hoc per call.
//
// A4/A5 may eventually unify this with app/shared/src/lib/api.ts via
// dependency-injected baseURL + tokenGetter; for now the duplication
// is small (<60 lines) and worth the boundary clarity.

export class MobileAPIError extends Error {
  status: number
  body: unknown
  constructor(status: number, body: unknown, message: string) {
    super(message)
    this.status = status
    this.body = body
  }
}

interface MobileFetchInit extends Omit<RequestInit, 'body'> {
  body?: unknown // JSON-serialisable; we'll stringify
  token?: string | null
}

export async function mobileFetch<T = unknown>(
  serverURL: string,
  path: string,
  init: MobileFetchInit = {},
): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('Accept', 'application/json')
  if (init.token) {
    headers.set('Authorization', `Bearer ${init.token}`)
  }
  let body: BodyInit | undefined
  if (init.body !== undefined) {
    headers.set('Content-Type', 'application/json')
    body = JSON.stringify(init.body)
  }

  const res = await fetch(joinURL(serverURL, path), {
    method: init.method ?? 'GET',
    headers,
    body,
    signal: init.signal,
    credentials: 'omit',
  })

  const contentType = res.headers.get('content-type') ?? ''
  const payload: unknown = contentType.includes('application/json')
    ? await res.json().catch(() => null)
    : await res.text().catch(() => '')

  if (!res.ok) {
    const message =
      typeof payload === 'object' &&
      payload !== null &&
      'error' in payload &&
      typeof (payload as { error: unknown }).error === 'string'
        ? (payload as { error: string }).error
        : `${init.method ?? 'GET'} ${path} failed (${res.status})`
    throw new MobileAPIError(res.status, payload, message)
  }
  return payload as T
}

function joinURL(serverURL: string, path: string): string {
  const trimmed = serverURL.replace(/\/+$/, '')
  const prefixed = path.startsWith('/') ? path : `/${path}`
  return trimmed + prefixed
}

// ── Concrete endpoints used by B3 ───────────────────────────────────

export interface HealthResponse {
  status: string
  version: string
  commit: string
  uptime_s: number
  db_ok: boolean
}

export function getHealth(serverURL: string): Promise<HealthResponse> {
  return mobileFetch<HealthResponse>(serverURL, '/api/v1/health')
}

export interface MobileLoginResponse {
  token: string
  username: string
  issued_at: string
  expires_at: string
}

export function postMobileLogin(
  serverURL: string,
  username: string,
  password: string,
): Promise<MobileLoginResponse> {
  return mobileFetch<MobileLoginResponse>(serverURL, '/api/v1/auth/mobile-login', {
    method: 'POST',
    body: { username, password },
  })
}

// Subset of `Session` from app/shared/src/lib/types.ts — only the
// fields B5 actually renders. Importing the shared type would also
// work, but B5 deliberately avoids coupling the mobile data layer
// to the shared types until A4/A5 unify the API client.
export interface SessionSummary {
  id: string
  name?: string
  provider_id: string
  cwd: string
  state: 'pending' | 'running' | 'idle' | 'stopped' | 'ended'
  started_at: string
  ended_at?: string
}

export async function listSessions(
  serverURL: string,
  token: string,
): Promise<SessionSummary[]> {
  const res = await mobileFetch<{ sessions?: SessionSummary[] }>(
    serverURL,
    '/api/v1/sessions',
    { token },
  )
  return res.sessions ?? []
}

// ── Memory ──────────────────────────────────────────────────────────

export interface MemoryRecord {
  id: string
  scope: 'session' | 'project' | 'global'
  scope_key: string
  text: string
  embedder: string
  created_at: string
  updated_at: string
}

export async function listMemories(
  serverURL: string,
  token: string,
  opts: { scope?: string; scopeKey?: string; n?: number } = {},
): Promise<MemoryRecord[]> {
  const sp = new URLSearchParams()
  sp.set('scope', opts.scope ?? 'global')
  if (opts.scopeKey) sp.set('scope_key', opts.scopeKey)
  sp.set('n', String(opts.n ?? 100))
  const res = await mobileFetch<{ memories?: MemoryRecord[] }>(
    serverURL,
    `/api/v1/memory/list?${sp.toString()}`,
    { token },
  )
  return res.memories ?? []
}

export interface SearchHit {
  record: MemoryRecord
  similarity: number
}

export async function searchMemories(
  serverURL: string,
  token: string,
  query: string,
  scope: 'session' | 'project' | 'global' = 'global',
  topK = 20,
): Promise<SearchHit[]> {
  const res = await mobileFetch<{ hits?: SearchHit[] }>(
    serverURL,
    '/api/v1/memory/search',
    {
      method: 'POST',
      token,
      body: { query, scope, top_k: topK },
    },
  )
  return res.hits ?? []
}

// ── Notes ───────────────────────────────────────────────────────────

export interface NoteSummary {
  path: string
  title: string
  modified: string
  size: number
}

export interface FullNote extends NoteSummary {
  body: string
}

export async function listNotes(
  serverURL: string,
  token: string,
): Promise<NoteSummary[]> {
  const res = await mobileFetch<{ notes?: NoteSummary[] }>(
    serverURL,
    '/api/v1/notes/list',
    { token },
  )
  return res.notes ?? []
}

export async function getNote(
  serverURL: string,
  token: string,
  notePath: string,
): Promise<FullNote> {
  return mobileFetch<FullNote>(
    serverURL,
    `/api/v1/notes/get?path=${encodeURIComponent(notePath)}`,
    { token },
  )
}

// ── Audit / activity ────────────────────────────────────────────────

export interface AuditEntry {
  id: number
  ts: string
  actor_kind: string
  actor_id?: string
  action: string
  subject_kind?: string
  subject_id?: string
  metadata?: unknown
}

export async function listAudit(
  serverURL: string,
  token: string,
  opts: { action?: string; since?: string; limit?: number } = {},
): Promise<AuditEntry[]> {
  const sp = new URLSearchParams()
  if (opts.action) sp.set('action', opts.action)
  if (opts.since) sp.set('since', opts.since)
  if (opts.limit) sp.set('limit', String(opts.limit))
  const path = sp.toString()
    ? `/api/v1/audit?${sp.toString()}`
    : '/api/v1/audit'
  const res = await mobileFetch<{ entries?: AuditEntry[] }>(
    serverURL,
    path,
    { token },
  )
  return res.entries ?? []
}
