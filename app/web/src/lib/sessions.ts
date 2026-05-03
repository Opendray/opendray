import { api } from './api'
import type { CreateSessionRequest, Session } from './types'

export async function listSessions(): Promise<Session[]> {
  const res = await api<{ sessions: Session[] }>('/api/v1/sessions')
  return res.sessions ?? []
}

export async function getSession(id: string): Promise<Session> {
  return api<Session>(`/api/v1/sessions/${id}`)
}

export async function createSession(
  req: CreateSessionRequest,
): Promise<Session> {
  return api<Session>('/api/v1/sessions', {
    method: 'POST',
    body: req,
  })
}

export async function removeSession(id: string): Promise<void> {
  await api(`/api/v1/sessions/${id}`, { method: 'DELETE' })
}

export async function stopSession(id: string): Promise<Session> {
  return api<Session>(`/api/v1/sessions/${id}/stop`, { method: 'POST' })
}

export async function startSession(id: string): Promise<Session> {
  return api<Session>(`/api/v1/sessions/${id}/start`, { method: 'POST' })
}

export async function resizeSession(
  id: string,
  cols: number,
  rows: number,
): Promise<void> {
  await api(`/api/v1/sessions/${id}/resize`, {
    method: 'POST',
    body: { cols, rows },
  })
}

// switchClaudeAccount terminates the running CLI process and respawns
// it under a new account binding. The session id (and therefore the
// UI tab) is preserved; only the underlying child process changes.
// `accountId === ''` clears the binding (CLI uses its system default).
export async function switchClaudeAccount(
  id: string,
  accountId: string,
): Promise<Session> {
  return api<Session>(`/api/v1/sessions/${id}/claude-account`, {
    method: 'PATCH',
    body: { account_id: accountId },
  })
}

