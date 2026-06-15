// Client for the /api/v1/memory/* endpoints. Mirrors the Go shapes
// in internal/memory.
//
// Memory is dual-auth (admin OR integration). The admin web UI hits
// these endpoints with the operator's bearer token, same as every
// other admin call.

import { api } from './api'

// 'session' was retired in the M-U unification (session ≡ project).
export type Scope = 'project' | 'global'

export interface MemoryRecord {
  id: string
  scope: Scope
  scope_key: string
  text: string
  embedder: string
  metadata?: Record<string, unknown>
  created_at: string
  updated_at: string
  /** Number of times this memory has been returned by a search (post-threshold). */
  hit_count: number
  /** Timestamp of the most recent hit, or null if never used. */
  last_hit_at?: string | null
  /** Optional summarizer-supplied confidence in [0,1]; nil means "unknown". */
  confidence?: number | null
  /**
   * Soft-delete timestamp. Set by the auto-cleaner / lifecycle pass.
   * Archived rows are excluded from search/list and are restorable
   * until the grace window purges them. Only populated by the
   * Archived view (`listArchived`); normal reads never return them.
   */
  archived_at?: string | null
  /** Why the row was archived (e.g. "duplicate", "stale", "dormant-project"). */
  archived_reason?: string
  /** Memory tier (Cortex): 'durable' | 'quarantine'. Standard reads only
   * return durable rows; populated by the quarantine review queue + Get. */
  tier?: string
  /** TTL deadline for quarantined rows; absent on durable rows. */
  quarantine_expires_at?: string | null
}

export interface SearchHit {
  memory: MemoryRecord
  similarity: number
  /**
   * M-PC ranking signal — similarity dampened by age and lifted by
   * hit_count + stored confidence. Used by Service.Search to order
   * results. Omitted when the backend doesn't compute it
   * (legacy callers).
   */
  effective_score?: number
}

export interface ProbeResult {
  base_url: string
  reachable: boolean
  status_code?: number
  models?: string[]
  error?: string
  /** "ollama" | "lmstudio" | "openai-compatible" */
  detected?: string
}

export interface MemoryStatus {
  embedder: string
  dimensions: number
  enabled: boolean
  auto_detected?: ProbeResult[]
  /** Configured backend: "auto" | "bm25" | "http" | "local". */
  backend?: string
  /** Embedder actually serving (alias of `embedder`). */
  effective_embedder?: string
  /** True when the BM25 keyword floor is active (no dense/semantic retrieval). */
  is_floor?: boolean
  /** The configured dense endpoint, if any (null when none configured). */
  configured_dense?: { base_url: string; model: string } | null
  /** Live probe of the configured dense endpoint (null when none configured). */
  dense_reachable?: boolean | null
  /** A dense endpoint is configured but is not the healthy serving tier right now. */
  degraded?: boolean
  /** Rows not yet on the active embedder (the background converge backlog). */
  drift?: number
}

export interface TestEmbedResponse {
  dim: number
  embedder: string
  vector_preview: number[]
}

export async function fetchMemoryStatus(): Promise<MemoryStatus> {
  return api<MemoryStatus>('/api/v1/memory/status')
}

export async function listMemories(
  scope: Scope,
  scopeKey: string,
  n = 100,
): Promise<MemoryRecord[]> {
  const q = new URLSearchParams({ scope, n: String(n) })
  if (scopeKey) q.set('scope_key', scopeKey)
  const res = await api<{ memories: MemoryRecord[] }>(
    `/api/v1/memory/list?${q.toString()}`,
  )
  return res.memories ?? []
}

export interface SearchRequest {
  query: string
  scope: Scope
  scope_key?: string
  top_k?: number
  /** -1 = no threshold (return everything ranked); >0 = override service default. */
  min_similarity?: number
}

export async function searchMemories(req: SearchRequest): Promise<SearchHit[]> {
  const res = await api<{ hits: SearchHit[] }>(
    '/api/v1/memory/search',
    { method: 'POST', body: req },
  )
  return res.hits ?? []
}

export async function deleteMemory(id: string): Promise<void> {
  await api(`/api/v1/memory/${encodeURIComponent(id)}`, { method: 'DELETE' })
}

// Fetch a single memory by id. Used by the Conflicts panel's delete
// confirmation dialog so the operator can see the actual fact text
// before yanking it. Returns 404 → throws so caller's useQuery flips
// to error state.
export async function getMemory(id: string): Promise<MemoryRecord> {
  return api<MemoryRecord>(`/api/v1/memory/${encodeURIComponent(id)}`)
}

// deleteMemoriesByScope wipes every memory under (scope, scope_key)
// in one server-side SQL operation. Returns the row count actually
// removed. Server enforces:
//   - non-global scopes require a non-empty scope_key (fat-finger
//     guard against "delete every memory in this scope")
//   - global scope must have an empty scope_key (the only valid
//     value there)
export async function deleteMemoriesByScope(
  scope: Scope,
  scopeKey: string,
): Promise<number> {
  const res = await api<{ deleted: number }>(
    '/api/v1/memory/delete-by-scope',
    {
      method: 'POST',
      body: { scope, scope_key: scope === 'global' ? '' : scopeKey },
    },
  )
  return res.deleted ?? 0
}

export async function updateMemory(
  id: string,
  text: string,
  metadata?: Record<string, unknown>,
): Promise<void> {
  await api(`/api/v1/memory/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: { text, metadata },
  })
}

export async function listScopeKeys(scope: Scope): Promise<string[]> {
  const q = new URLSearchParams({ scope })
  const res = await api<{ scope_keys: string[] }>(
    `/api/v1/memory/scope-keys?${q.toString()}`,
  )
  return res.scope_keys ?? []
}

export interface EmbedderStats {
  current: string
  counts: Record<string, number>
}

export interface ReembedReport {
  examined: number
  reembed: number
  skipped: number
  failed: number
  errors?: string[]
  started_at: string
  ended_at: string
  from: string[]
  to: string
}

export async function fetchEmbedderStats(): Promise<EmbedderStats> {
  return api<EmbedderStats>('/api/v1/memory/embedder-stats')
}

export async function reembedAll(batch?: number): Promise<ReembedReport> {
  const q = batch && batch > 0 ? `?batch=${batch}` : ''
  return api<ReembedReport>(`/api/v1/memory/reembed${q}`, { method: 'POST' })
}

export interface MirrorResult {
  ingested: number
  cwd: string
}

export async function mirrorCwd(cwd: string): Promise<MirrorResult> {
  return api<MirrorResult>('/api/v1/memory/mirror', {
    method: 'POST',
    body: { cwd },
  })
}

export async function testEmbedder(text: string): Promise<TestEmbedResponse> {
  return api<TestEmbedResponse>('/api/v1/memory/test', {
    method: 'POST',
    body: { text },
  })
}

export async function probeEmbeddingEndpoint(
  baseURL: string,
  apiKey = '',
): Promise<ProbeResult> {
  return api<ProbeResult>('/api/v1/memory/probe', {
    method: 'POST',
    body: { base_url: baseURL, api_key: apiKey },
  })
}

// listArchived backs the read-only "Archived (restorable)" view: the
// soft-archived memories the auto-cleaner / lifecycle pass removed,
// which the operator can restore until the 30-day grace window purges
// them. Pass an empty scopeKey to list every archived row under the
// scope (cross-project). Mirrors GET /api/v1/memory/archived.
export async function listArchived(
  scope: Scope,
  scopeKey = '',
  n = 200,
): Promise<MemoryRecord[]> {
  const q = new URLSearchParams({ scope, n: String(n) })
  if (scopeKey) q.set('scope_key', scopeKey)
  const res = await api<{ memories: MemoryRecord[] }>(
    `/api/v1/memory/archived?${q.toString()}`,
  )
  return res.memories ?? []
}

// restoreMemory un-archives a soft-deleted memory (admin only).
// Mirrors POST /api/v1/memory/{id}/restore.
export async function restoreMemory(id: string): Promise<void> {
  await api(`/api/v1/memory/${encodeURIComponent(id)}/restore`, {
    method: 'POST',
  })
}

/** Soft-archives one memory by hand (admin only) — reversible from the
 * Archived view until the grace window purges it. */
export async function archiveMemory(id: string): Promise<void> {
  await api(`/api/v1/memory/${encodeURIComponent(id)}/archive`, {
    method: 'POST',
  })
}

/** Moves a durable memory into the quarantine review queue (admin
 * only) — release it from Cortex → Quarantine, or the TTL expires it. */
export async function quarantineMemory(id: string): Promise<void> {
  await api(`/api/v1/memory/${encodeURIComponent(id)}/quarantine`, {
    method: 'POST',
  })
}

export async function storeMemory(
  text: string,
  scope: Scope,
  scopeKey: string,
): Promise<{ id: string }> {
  return api<{ id: string }>('/api/v1/memory/store', {
    method: 'POST',
    body: { text, scope, scope_key: scopeKey },
  })
}
