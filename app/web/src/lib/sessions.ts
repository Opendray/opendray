import { api } from './api'
import type { CreateSessionRequest, Provider, Session } from './types'

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

export async function terminateSession(id: string): Promise<void> {
  await api(`/api/v1/sessions/${id}`, { method: 'DELETE' })
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

export async function listProviders(): Promise<Provider[]> {
  const res = await api<{ providers: Provider[] }>('/api/v1/providers')
  return res.providers ?? []
}
