import { api } from './api'

// Mirrors the Go integration.CallEntry struct.
export interface IntegrationCall {
  id: number
  ts: string
  integration_id: string
  direction: 'inbound' | 'outbound'
  method: string
  path: string
  status_code: number
  duration_ms: number
  bytes_written?: number
  request_id?: string
  resource_kind?: string
  resource_id?: string
}

export interface IntegrationCallsQuery {
  integration_id?: string
  direction?: 'inbound' | 'outbound'
  /** HTTP status family: 2 | 3 | 4 | 5 */
  status_class?: 2 | 3 | 4 | 5
  since?: string
  until?: string
  cursor?: string
  /** 1..500, default 100 */
  limit?: number
}

export interface IntegrationCallsPage {
  entries: IntegrationCall[]
  next_cursor: string | null
}

export async function listIntegrationCalls(
  q: IntegrationCallsQuery = {},
): Promise<IntegrationCallsPage> {
  const sp = new URLSearchParams()
  if (q.integration_id) sp.set('integration_id', q.integration_id)
  if (q.direction) sp.set('direction', q.direction)
  if (q.status_class) sp.set('status_class', String(q.status_class))
  if (q.since) sp.set('since', q.since)
  if (q.until) sp.set('until', q.until)
  if (q.cursor) sp.set('cursor', q.cursor)
  if (q.limit) sp.set('limit', String(q.limit))
  const qs = sp.toString()
  return api<IntegrationCallsPage>(
    `/api/v1/integrations/_calls${qs ? `?${qs}` : ''}`,
  )
}
