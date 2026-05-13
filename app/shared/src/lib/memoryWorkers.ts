// Client for /api/v1/memory/workers/* — M25 per-task worker
// configuration + metrics. Operators flip individual memory
// touchpoints between the summarizer HTTP path (LM Studio /
// OpenAI-compat) and the agent path (Claude / Gemini `--print`).

import { api } from './api'

export type TaskKind = 'gatekeeper' | 'cleaner' | 'gitactivity' | 'transcript'
export type WorkerKind = 'summarizer' | 'agent'
export type AgentProviderID = 'claude' | 'gemini'

export interface WorkerConfig {
  task: TaskKind
  kind: WorkerKind
  /** Pinned summarizer_provider row id; empty = use registry default. */
  summarizer_id?: string
  /** When kind === 'agent': which CLI. */
  provider_id?: AgentProviderID | ''
  /** When provider_id === 'claude': which multi-account row. */
  account_id?: string
  enabled: boolean
  updated_at: string
}

export interface ListResponse {
  workers: WorkerConfig[]
}

export async function listMemoryWorkers(): Promise<WorkerConfig[]> {
  const res = await api<ListResponse>('/api/v1/memory/workers/')
  return res.workers ?? []
}

export async function getMemoryWorker(task: TaskKind): Promise<WorkerConfig> {
  return api<WorkerConfig>(`/api/v1/memory/workers/${task}`)
}

export interface UpsertWorkerInput {
  kind: WorkerKind
  summarizer_id?: string
  provider_id?: AgentProviderID | ''
  account_id?: string
  enabled?: boolean
}

export async function upsertMemoryWorker(
  task: TaskKind,
  input: UpsertWorkerInput,
): Promise<WorkerConfig> {
  return api<WorkerConfig>(`/api/v1/memory/workers/${task}`, {
    method: 'PUT',
    body: input,
  })
}

export interface TestResult {
  task: TaskKind
  ok: boolean
  duration_ms: number
  worker_kind?: WorkerKind
  provider_id?: string
  preview?: string
  error?: string
}

export async function testMemoryWorker(task: TaskKind): Promise<TestResult> {
  return api<TestResult>(`/api/v1/memory/workers/${task}/test`, {
    method: 'POST',
  })
}

export interface CallSummary {
  id: number
  task: TaskKind
  worker_kind: WorkerKind
  provider_id: string
  account_id: string
  started_at: string
  duration_ms: number
  success: boolean
  error_message: string
  input_bytes: number
  output_bytes: number
  tokens_in: number
  tokens_out: number
}

export interface ListCallsResponse {
  calls: CallSummary[]
}

export async function listMemoryWorkerCalls(opts: {
  task?: TaskKind
  limit?: number
} = {}): Promise<CallSummary[]> {
  const qs = new URLSearchParams()
  if (opts.task) qs.set('task', opts.task)
  qs.set('n', String(opts.limit ?? 100))
  const res = await api<ListCallsResponse>(
    `/api/v1/memory/workers/calls?${qs.toString()}`,
  )
  return res.calls ?? []
}

/** Display label for the UI. */
export function taskLabel(t: TaskKind): string {
  switch (t) {
    case 'gatekeeper':
      return 'Gatekeeper'
    case 'cleaner':
      return 'Cleaner librarian'
    case 'gitactivity':
      return 'Git activity summariser'
    case 'transcript':
      return 'Session transcript summariser'
  }
}

/** One-line description shown beneath the title. */
export function taskDescription(t: TaskKind): string {
  switch (t) {
    case 'gatekeeper':
      return 'Pre-write filter on every memory_store. High frequency (<500ms target) — summarizer-only.'
    case 'cleaner':
      return 'Periodic LLM librarian. Judges aged memories as keep / stale / duplicate.'
    case 'gitactivity':
      return 'git log → 2-3 paragraph narrative every 24h. Naturally fits an agent worker.'
    case 'transcript':
      return 'Session-end "what did the agent do" summary. Naturally fits an agent worker.'
  }
}

/** Per-task agent support flag — gatekeeper is summarizer-only by
 *  design (latency budget) even though the row exists. */
export function taskAgentSupported(t: TaskKind): boolean {
  return t !== 'gatekeeper'
}
