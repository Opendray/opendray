import { api } from './api'
import type {
  AntigravityAccount,
  CreateAntigravityAccountRequest,
  UpdateAntigravityAccountRequest,
} from './types'

export async function listAntigravityAccounts(): Promise<AntigravityAccount[]> {
  const res = await api<{ accounts: AntigravityAccount[] }>(
    '/api/v1/antigravity-accounts',
  )
  return res.accounts ?? []
}

export async function createAntigravityAccount(
  req: CreateAntigravityAccountRequest,
): Promise<AntigravityAccount> {
  return api<AntigravityAccount>('/api/v1/antigravity-accounts', {
    method: 'POST',
    body: req,
  })
}

export async function updateAntigravityAccount(
  id: string,
  req: UpdateAntigravityAccountRequest,
): Promise<AntigravityAccount> {
  return api<AntigravityAccount>(`/api/v1/antigravity-accounts/${id}`, {
    method: 'PUT',
    body: req,
  })
}

export async function toggleAntigravityAccount(
  id: string,
  enabled: boolean,
): Promise<AntigravityAccount> {
  return api<AntigravityAccount>(`/api/v1/antigravity-accounts/${id}/toggle`, {
    method: 'PATCH',
    body: { enabled },
  })
}

export async function deleteAntigravityAccount(id: string): Promise<void> {
  await api<unknown>(`/api/v1/antigravity-accounts/${id}`, { method: 'DELETE' })
}

export async function importLocalAntigravityAccounts(): Promise<{
  created: AntigravityAccount[]
  count: number
}> {
  return api('/api/v1/antigravity-accounts/import-local', { method: 'POST' })
}
