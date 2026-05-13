// Client for /api/v1/project-docs/* + /project-doc-proposals/* +
// /session-logs/*. Backs the Project page in web (and mirrors
// app/mobile/lib/core/api/project_docs_api.dart shape).
//
// Powers the unified cross-CLI memory L2/L3/L4/L5 surface:
// project_docs holds the goal / plan / tech_stack / recent_activity
// markdown bodies; proposals queue agent-suggested goal/plan
// edits for operator approval; session_logs is the per-cwd journal.

import { api } from './api'

// ── project_docs ──────────────────────────────────────────────

export type DocKind = 'goal' | 'plan' | 'tech_stack' | 'recent_activity'
export type DocAuthor = 'operator' | 'agent' | 'scanner'

export interface ProjectDoc {
  id: string
  cwd: string
  kind: DocKind
  content: string
  updated_by: DocAuthor
  updated_at: string
}

export interface ListDocsResponse {
  docs: ProjectDoc[]
}

export async function listProjectDocs(cwd: string): Promise<ProjectDoc[]> {
  const res = await api<ListDocsResponse>(
    `/api/v1/project-docs?cwd=${encodeURIComponent(cwd)}`,
  )
  return res.docs ?? []
}

export async function getProjectDoc(
  cwd: string,
  kind: DocKind,
): Promise<ProjectDoc> {
  return api<ProjectDoc>(
    `/api/v1/project-docs/${kind}?cwd=${encodeURIComponent(cwd)}`,
  )
}

export async function putProjectDoc(input: {
  cwd: string
  kind: DocKind
  content: string
}): Promise<ProjectDoc> {
  return api<ProjectDoc>(`/api/v1/project-docs/${input.kind}`, {
    method: 'PUT',
    body: {
      cwd: input.cwd,
      content: input.content,
      updated_by: 'operator',
    },
  })
}

// ── proposals ─────────────────────────────────────────────────

export interface DocProposal {
  id: string
  cwd: string
  kind: 'goal' | 'plan'
  proposed_content: string
  proposed_by_session?: string
  reason: string
  /** When the proposal has been decided, the verdict. */
  decision?: 'approved' | 'rejected'
  decided_at?: string
  /** The prior live content at the time of proposal — used for diff display. */
  prior_content?: string
  created_at: string
}

export async function listPendingProposals(cwd?: string): Promise<DocProposal[]> {
  const qs = cwd ? `?cwd=${encodeURIComponent(cwd)}` : ''
  const res = await api<{ proposals: DocProposal[] }>(
    `/api/v1/project-doc-proposals/pending${qs}`,
  )
  return res.proposals ?? []
}

export async function approveProposal(id: string): Promise<ProjectDoc> {
  return api<ProjectDoc>(`/api/v1/project-doc-proposals/${id}/approve`, {
    method: 'POST',
  })
}

export async function rejectProposal(id: string): Promise<void> {
  await api(`/api/v1/project-doc-proposals/${id}/reject`, {
    method: 'POST',
  })
}

// ── session_logs (journal) ────────────────────────────────────

export type LogKind = 'session_summary' | 'manual' | 'decision'

export interface SessionLogEntry {
  id: string
  cwd: string
  session_id?: string
  kind: LogKind
  title: string
  content: string
  updated_by: DocAuthor | 'summarizer'
  created_at: string
}

export async function listSessionLogs(
  cwd: string,
  limit = 50,
): Promise<SessionLogEntry[]> {
  const res = await api<{ logs: SessionLogEntry[] }>(
    `/api/v1/session-logs?cwd=${encodeURIComponent(cwd)}&n=${limit}`,
  )
  return res.logs ?? []
}

export async function appendSessionLog(input: {
  cwd: string
  kind?: LogKind
  session_id?: string
  title?: string
  content: string
}): Promise<SessionLogEntry> {
  return api<SessionLogEntry>('/api/v1/session-logs', {
    method: 'POST',
    body: { ...input, updated_by: 'operator' },
  })
}

export async function deleteSessionLog(id: string): Promise<void> {
  await api(`/api/v1/session-logs/${id}`, { method: 'DELETE' })
}
