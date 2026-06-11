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
  /** Skill usage tracking: sessions whose transcript referenced this skill. */
  use_count?: number
  last_used_at?: string | null
  /** Outcome tracking: of the sessions that referenced this skill, how many
   * ended in success vs failure — the experience compiler's feedback loop. */
  success_count?: number
  failure_count?: number
  /** Skills: disabled skills keep their node but their SKILL.md is
   * removed from the vault, so no session loads them. */
  enabled?: boolean
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

// draftKB triggers a background regeneration of all curated KB pages
// (infrastructure / conventions / lessons / per-project handbooks). Returns
// immediately (202); the drafter runs detached and updates the note docs.
export async function draftKB(): Promise<void> {
  await api('/api/v1/knowledge/kb/draft', { method: 'POST' })
}

// deleteKnowledgeNode removes a node — used to undo an accidental promote /
// skillify. Auto-derived facts/entities re-appear on the next anchor sweep;
// skills stay deleted (and their SKILL.md is removed server-side).
export async function deleteKnowledgeNode(id: string): Promise<void> {
  await api(`/api/v1/knowledge/nodes/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

/** Flips a node's enabled flag. For skills this writes/removes the
 * vault SKILL.md so sessions only load enabled skills. */
export async function setKnowledgeNodeEnabled(
  id: string,
  enabled: boolean,
): Promise<KnowledgeNode> {
  return api<KnowledgeNode>(
    `/api/v1/knowledge/nodes/${encodeURIComponent(id)}/enable`,
    { method: 'POST', body: { enabled } },
  )
}

// ── retirement proposals (the closed feedback loop) ───────────

export type RetirementReason = 'never_used' | 'low_success' | 'dormant'

export interface RetirementCandidate {
  node: KnowledgeNode
  reason: RetirementReason
}

/** Skills the outcome loop proposes to retire: never referenced after 14+
 * days, repeatedly loaded into sessions that then fail, or long dormant. */
export async function listRetirementCandidates(): Promise<RetirementCandidate[]> {
  const res = await api<{ candidates: RetirementCandidate[] }>(
    '/api/v1/knowledge/skills/retirement',
  )
  return res.candidates ?? []
}

/** Workbench ranking: recurrence × manual time cost, computed by the
 * experience compiler and stored in provenance. Missing → 0 (legacy rows). */
export function candidateScore(n: KnowledgeNode): number {
  const v = n.provenance?.score
  return typeof v === 'number' ? v : 0
}

// ── impact view (the graph's production face) ─────────────────

export interface ImpactEntity {
  node: KnowledgeNode
  /** Number of live nodes connected to this entity — its blast radius. */
  degree: number
}

/** Entities ordered by blast radius: pick one (a database, a host, a
 * tool) and see everything that depends on it before you touch it. */
export async function listImpactEntities(limit = 200): Promise<ImpactEntity[]> {
  const res = await api<{ entities: ImpactEntity[] }>(
    `/api/v1/knowledge/impact?n=${limit}`,
  )
  return res.entities ?? []
}
