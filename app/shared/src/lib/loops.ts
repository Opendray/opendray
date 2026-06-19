import { api } from './api'
import type { CreateLoopRequest, Loop, LoopRun } from './types'

// The Loop Engine REST surface. The gateway returns a bare array for the
// list/runs endpoints and a bare object for a single loop (see
// internal/autoloop/handler.go), so — unlike sessions — there is no wrapper
// envelope to unwrap.

export async function listLoops(): Promise<Loop[]> {
  return (await api<Loop[]>('/api/v1/loops')) ?? []
}

export async function getLoop(id: string): Promise<Loop> {
  return api<Loop>(`/api/v1/loops/${id}`)
}

export async function listLoopRuns(id: string): Promise<LoopRun[]> {
  return (await api<LoopRun[]>(`/api/v1/loops/${id}/runs`)) ?? []
}

export async function createLoop(req: CreateLoopRequest): Promise<Loop> {
  return api<Loop>('/api/v1/loops', { method: 'POST', body: req })
}

export async function pauseLoop(id: string): Promise<Loop> {
  return api<Loop>(`/api/v1/loops/${id}/pause`, { method: 'POST' })
}

export async function resumeLoop(id: string): Promise<Loop> {
  return api<Loop>(`/api/v1/loops/${id}/resume`, { method: 'POST' })
}

export async function stopLoop(id: string): Promise<Loop> {
  return api<Loop>(`/api/v1/loops/${id}/stop`, { method: 'POST' })
}
