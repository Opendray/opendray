// Client for the /api/v1/knowledge/* endpoints (M-KG knowledge graph).
// Mirrors the Go shapes in internal/knowledge. Dual-auth (admin OR
// integration); the admin web UI uses the operator's bearer token like
// every other admin call.

import { api } from './api'

export type KnowledgeKind = 'entity' | 'fact' | 'playbook' | 'skill'
export type KnowledgeScope = 'project' | 'domain' | 'global'

export interface KnowledgeNode {
  id: string
  kind: KnowledgeKind
  entity_type?: string
  title: string
  body: string
  scope: KnowledgeScope
  scope_key?: string
  maturity: string
  confidence?: number | null
  provenance?: Record<string, unknown>
  created_at: string
  updated_at: string
  archived_at?: string | null
}

export interface KnowledgeNeighbor {
  node: KnowledgeNode
  edge_type: string
  direction: 'in' | 'out'
}

export interface KnowledgeSearchHit {
  node: KnowledgeNode
  similarity: number
}

export interface KnowledgeBrain {
  project: KnowledgeNode | null
  facts: KnowledgeNode[]
}

export async function listKnowledgeNodes(
  params: { kind?: KnowledgeKind; scope?: KnowledgeScope; scopeKey?: string } = {},
): Promise<KnowledgeNode[]> {
  const q = new URLSearchParams()
  if (params.kind) q.set('kind', params.kind)
  if (params.scope) q.set('scope', params.scope)
  if (params.scopeKey) q.set('scope_key', params.scopeKey)
  const qs = q.toString()
  const res = await api<{ nodes: KnowledgeNode[] }>(
    `/api/v1/knowledge/nodes${qs ? `?${qs}` : ''}`,
  )
  return res.nodes ?? []
}

export async function getKnowledgeNode(id: string): Promise<KnowledgeNode> {
  return api<KnowledgeNode>(`/api/v1/knowledge/nodes/${encodeURIComponent(id)}`)
}

export async function getKnowledgeGraph(
  id: string,
): Promise<{ node: KnowledgeNode; neighbors: KnowledgeNeighbor[] }> {
  return api<{ node: KnowledgeNode; neighbors: KnowledgeNeighbor[] }>(
    `/api/v1/knowledge/nodes/${encodeURIComponent(id)}/graph`,
  )
}

export async function getKnowledgeBrain(cwd: string): Promise<KnowledgeBrain> {
  return api<KnowledgeBrain>(
    `/api/v1/knowledge/brain?cwd=${encodeURIComponent(cwd)}`,
  )
}

export async function searchKnowledge(
  query: string,
  cwd = '',
  topK = 20,
): Promise<KnowledgeSearchHit[]> {
  const p = new URLSearchParams({ q: query })
  if (cwd) p.set('cwd', cwd)
  if (topK) p.set('top_k', String(topK))
  const res = await api<{ hits: KnowledgeSearchHit[] }>(
    `/api/v1/knowledge/search?${p.toString()}`,
  )
  return res.hits ?? []
}

export async function promoteKnowledgeNode(
  id: string,
  scope: KnowledgeScope,
  scopeKey = '',
): Promise<void> {
  await api(`/api/v1/knowledge/nodes/${encodeURIComponent(id)}/promote`, {
    method: 'POST',
    body: { scope, scope_key: scopeKey },
  })
}

export async function skillifyKnowledgeNode(id: string): Promise<KnowledgeNode> {
  return api<KnowledgeNode>(
    `/api/v1/knowledge/nodes/${encodeURIComponent(id)}/skillify`,
    { method: 'POST' },
  )
}

// deleteKnowledgeNode removes a node — used to undo an accidental promote /
// skillify. Auto-derived facts/entities re-appear on the next anchor sweep;
// skills stay deleted (and their SKILL.md is removed server-side).
export async function deleteKnowledgeNode(id: string): Promise<void> {
  await api(`/api/v1/knowledge/nodes/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}
