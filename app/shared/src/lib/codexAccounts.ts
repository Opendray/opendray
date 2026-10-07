import { api } from './api'
import type {
  CodexAccount,
  CreateCodexAccountRequest,
  UpdateCodexAccountRequest,
} from './types'

export async function listCodexAccounts(): Promise<CodexAccount[]> {
  const res = await api<{ accounts: CodexAccount[] }>('/api/v1/codex-accounts')
  return res.accounts ?? []
}

export async function createCodexAccount(
  req: CreateCodexAccountRequest,
): Promise<CodexAccount> {
  return api<CodexAccount>('/api/v1/codex-accounts', {
    method: 'POST',
    body: req,
  })
}

export async function updateCodexAccount(
  id: string,
  req: UpdateCodexAccountRequest,
): Promise<CodexAccount> {
  return api<CodexAccount>(`/api/v1/codex-accounts/${id}`, {
    method: 'PUT',
    body: req,
  })
}

export async function toggleCodexAccount(
  id: string,
  enabled: boolean,
): Promise<CodexAccount> {
  return api<CodexAccount>(`/api/v1/codex-accounts/${id}/toggle`, {
    method: 'PATCH',
    body: { enabled },
  })
}

// setCodexAccountAPIKey logs the account in with an OpenAI API key. The
// gateway hands the key to `codex login --with-api-key` on stdin; it is
// never stored by opendray.
export async function setCodexAccountAPIKey(
  id: string,
  apiKey: string,
): Promise<CodexAccount> {
  return api<CodexAccount>(`/api/v1/codex-accounts/${id}/api-key`, {
    method: 'POST',
    body: { api_key: apiKey },
  })
}

export async function deleteCodexAccount(id: string): Promise<void> {
  await api<unknown>(`/api/v1/codex-accounts/${id}`, { method: 'DELETE' })
}

export async function importLocalCodexAccounts(): Promise<{
  created: CodexAccount[]
  count: number
}> {
  return api('/api/v1/codex-accounts/import-local', { method: 'POST' })
}
