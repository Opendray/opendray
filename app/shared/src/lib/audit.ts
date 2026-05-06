import { api } from './api'

// Mirrors the Go audit.Entry struct. Metadata is the raw JSON payload
// of the originating event — callers parse it as needed.
export interface AuditEntry {
  id: number
  ts: string
  actor_kind: string
  actor_id?: string
  action: string
  subject_kind?: string
  subject_id?: string
  metadata?: unknown
}

export interface AuditQuery {
  subject_kind?: string
  subject_id?: string
  /** Exact (e.g. "session.idle") or prefix (e.g. "session.*") */
  action?: string
  /** RFC3339 lower bound (inclusive) */
  since?: string
  /** RFC3339 upper bound (exclusive) */
  until?: string
  /** Last id from previous page; rows returned all have id < cursor */
  cursor?: string
  /** 1..500, default 100 */
  limit?: number
}

export interface AuditPage {
  entries: AuditEntry[]
  next_cursor: string | null
}

export async function listAudit(q: AuditQuery = {}): Promise<AuditPage> {
  const sp = new URLSearchParams()
  if (q.subject_kind) sp.set('subject_kind', q.subject_kind)
  if (q.subject_id) sp.set('subject_id', q.subject_id)
  if (q.action) sp.set('action', q.action)
  if (q.since) sp.set('since', q.since)
  if (q.until) sp.set('until', q.until)
  if (q.cursor) sp.set('cursor', q.cursor)
  if (q.limit) sp.set('limit', String(q.limit))
  const qs = sp.toString()
  return api<AuditPage>(`/api/v1/audit/log${qs ? `?${qs}` : ''}`)
}
