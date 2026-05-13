// Client for /api/v1/memory/cleanup/* — the M13 LLM librarian.
// Mirrors app/mobile/lib/core/api/memory_cleanup_api.dart.

import { api } from './api'

export type Verdict = 'keep' | 'stale' | 'duplicate'
export type DecisionStatus =
  | 'pending'
  | 'approved'
  | 'rejected'
  | 'executed'
  | 'expired'

export interface CleanupDecision {
  id: string
  memory_id: string
  memory_scope: string
  memory_scope_key: string
  memory_text_snapshot: string
  verdict: Verdict
  reason: string
  merge_into?: string
  run_id: string
  status: DecisionStatus
  summarizer_provider_id?: string
  created_at: string
  decided_at?: string
  executed_at?: string
}

export interface CleanupRunResult {
  run_id: string
  decisions: CleanupDecision[]
  scanned: number
  decided: number
  skipped: number
}

export async function runCleanup(input: {
  scope: string
  scope_key: string
}): Promise<CleanupRunResult> {
  return api<CleanupRunResult>('/api/v1/memory/cleanup/run', {
    method: 'POST',
    body: input,
  })
}

export interface ListDecisionsParams {
  status?: DecisionStatus
  scope?: string
  scope_key?: string
  limit?: number
}

export async function listCleanupDecisions(
  params: ListDecisionsParams = {},
): Promise<CleanupDecision[]> {
  const qs = new URLSearchParams()
  if (params.status) qs.set('status', params.status)
  if (params.scope) qs.set('scope', params.scope)
  if (params.scope_key) qs.set('scope_key', params.scope_key)
  qs.set('n', String(params.limit ?? 100))
  const res = await api<{ decisions: CleanupDecision[] }>(
    `/api/v1/memory/cleanup/decisions?${qs}`,
  )
  return res.decisions ?? []
}

/**
 * Approve atomically marks the decision approved + executes the
 * delete/merge. Returns the updated decision with status=executed
 * (or status=expired if the executor couldn't apply).
 */
export async function approveDecision(id: string): Promise<CleanupDecision> {
  return api<CleanupDecision>(
    `/api/v1/memory/cleanup/decisions/${id}/approve`,
    { method: 'POST' },
  )
}

export async function rejectDecision(id: string): Promise<CleanupDecision> {
  return api<CleanupDecision>(
    `/api/v1/memory/cleanup/decisions/${id}/reject`,
    { method: 'POST' },
  )
}
