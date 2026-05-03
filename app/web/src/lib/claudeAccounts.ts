import { api } from './api'
import type {
  ClaudeAccount,
  CreateClaudeAccountRequest,
  UpdateClaudeAccountRequest,
} from './types'

export async function listClaudeAccounts(): Promise<ClaudeAccount[]> {
  const res = await api<{ accounts: ClaudeAccount[] }>('/api/v1/claude-accounts')
  return res.accounts ?? []
}

export async function createClaudeAccount(
  req: CreateClaudeAccountRequest,
): Promise<ClaudeAccount> {
  return api<ClaudeAccount>('/api/v1/claude-accounts', {
    method: 'POST',
    body: req,
  })
}

export async function updateClaudeAccount(
  id: string,
  req: UpdateClaudeAccountRequest,
): Promise<ClaudeAccount> {
  return api<ClaudeAccount>(`/api/v1/claude-accounts/${id}`, {
    method: 'PUT',
    body: req,
  })
}

export async function toggleClaudeAccount(
  id: string,
  enabled: boolean,
): Promise<ClaudeAccount> {
  return api<ClaudeAccount>(`/api/v1/claude-accounts/${id}/toggle`, {
    method: 'PATCH',
    body: { enabled },
  })
}

export async function setClaudeAccountToken(
  id: string,
  token: string,
): Promise<ClaudeAccount> {
  return api<ClaudeAccount>(`/api/v1/claude-accounts/${id}/token`, {
    method: 'PUT',
    body: { token },
  })
}

export async function deleteClaudeAccount(id: string): Promise<void> {
  await api<unknown>(`/api/v1/claude-accounts/${id}`, { method: 'DELETE' })
}

export async function importLocalClaudeAccounts(): Promise<{
  created: ClaudeAccount[]
  count: number
}> {
  return api('/api/v1/claude-accounts/import-local', { method: 'POST' })
}
