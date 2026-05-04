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
}

export interface SearchHit {
  memory: MemoryRecord
  similarity: number
}

export interface MemoryStatus {
  embedder: string
  dimensions: number
  enabled: boolean
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

export async function testEmbedder(text: string): Promise<TestEmbedResponse> {
  return api<TestEmbedResponse>('/api/v1/memory/test', {
    method: 'POST',
    body: { text },
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
