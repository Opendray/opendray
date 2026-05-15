// Client for the /api/v1/memory/* endpoints. Mirrors the Go shapes
// in internal/memory.
//
// Memory is dual-auth (admin OR integration). The admin web UI hits
// these endpoints with the operator's bearer token, same as every
// other admin call.

import { api } from './api'

export type Scope = 'session' | 'project' | 'global'

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
