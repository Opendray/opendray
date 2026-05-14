// Client for /api/v1/memory/conflicts — the M-PC cross-layer
// conflict ledger.

import { api } from './api'

export type ConflictLayer = 'fact' | 'plan' | 'goal' | 'journal'
export type ConflictSeverity = 'low' | 'medium' | 'high'
export type ConflictStatus =
  | 'pending'
  | 'accepted'
  | 'dismissed'
  | 'expired'

export interface MemoryConflict {
  id: string
  cwd: string
  layer_a: ConflictLayer
  ref_a: string
  layer_b: ConflictLayer
  ref_b: string
  evidence: string
  severity: ConflictSeverity
  status: ConflictStatus
  detected_at: string
  decided_at?: string
  decided_by?: string
}

export async function listMemoryConflicts(params: {
  cwd: string
  status?: ConflictStatus
  limit?: number
}): Promise<MemoryConflict[]> {
  const qs = new URLSearchParams({ cwd: params.cwd })
  if (params.status) qs.set('status', params.status)
  qs.set('n', String(params.limit ?? 50))
  const res = await api<{ conflicts: MemoryConflict[] }>(
    `/api/v1/memory/conflicts?${qs}`,
  )
  return res.conflicts ?? []
}

export async function decideMemoryConflict(
  id: string,
  action: 'accepted' | 'dismissed',
): Promise<void> {
  await api(`/api/v1/memory/conflicts/${id}/${action}`, { method: 'POST' })
}

export async function detectMemoryConflicts(cwd: string): Promise<number> {
  const qs = new URLSearchParams({ cwd })
  const res = await api<{ detected: number }>(
    `/api/v1/memory/conflicts/detect?${qs}`,
    { method: 'POST' },
  )
  return res.detected ?? 0
}
