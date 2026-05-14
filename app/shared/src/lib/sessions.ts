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

export interface HistoryEntry {
  ts: string
  text: string
  session_id: string
}

export interface HistoryResponse {
  entries: HistoryEntry[]
  unsupported_provider?: boolean
}

// fetchSessionHistory pulls every user prompt the operator has sent
// in this session's project (cwd), pooled across every Claude
// session ever spawned in that directory. Non-Claude providers
// return `unsupported_provider: true` with empty entries.
export async function fetchSessionHistory(
  id: string,
  limit = 200,
): Promise<HistoryResponse> {
  return api<HistoryResponse>(
    `/api/v1/sessions/${id}/history?limit=${limit}`,
  )
}

// resendInput re-submits a prompt to the live session. Claude runs
// the PTY in raw mode where Enter is `\r`, not `\n` — using `\n`
// produces a literal newline in the prompt instead of submitting.
export async function resendInput(
  id: string,
  text: string,
): Promise<void> {
  await api(`/api/v1/sessions/${id}/input`, {
    method: 'POST',
    body: { data: text + '\r' },
  })
}

export interface UploadResponse {
  // Absolute path of the file on the gateway host's tempdir.
  // The caller writes this into the PTY so the CLI (Claude / Codex /
  // Gemini) can resolve it as an image attachment.
  path: string
  size: number
  original_name?: string
}

// uploadSessionFile multipart-uploads a file to the gateway and
// returns the absolute server-side path where it was saved. The
// caller is expected to write the returned path into the live PTY
// (e.g. via TerminalHandle.sendInput) so the running CLI can attach
// the file as context.
export async function uploadSessionFile(
  id: string,
  file: File,
): Promise<UploadResponse> {
  const form = new FormData()
  form.append('file', file, file.name || 'upload')
  return api<UploadResponse>(`/api/v1/sessions/${id}/uploads`, {
    method: 'POST',
    body: form,
  })
}

