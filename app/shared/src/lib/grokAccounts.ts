import { api } from './api'
import type {
  GrokAccount,
  CreateGrokAccountRequest,
  UpdateGrokAccountRequest,
} from './types'

export async function listGrokAccounts(): Promise<GrokAccount[]> {
  const res = await api<{ accounts: GrokAccount[] }>('/api/v1/grok-accounts')
  return res.accounts ?? []
}

export async function createGrokAccount(
  req: CreateGrokAccountRequest,
): Promise<GrokAccount> {
  return api<GrokAccount>('/api/v1/grok-accounts', {
    method: 'POST',
    body: req,
  })
}

export async function updateGrokAccount(
  id: string,
  req: UpdateGrokAccountRequest,
): Promise<GrokAccount> {
  return api<GrokAccount>(`/api/v1/grok-accounts/${id}`, {
    method: 'PUT',
    body: req,
  })
}

export async function toggleGrokAccount(
  id: string,
  enabled: boolean,
): Promise<GrokAccount> {
  return api<GrokAccount>(`/api/v1/grok-accounts/${id}/toggle`, {
    method: 'PATCH',
    body: { enabled },
  })
}

export async function deleteGrokAccount(id: string): Promise<void> {
  await api<unknown>(`/api/v1/grok-accounts/${id}`, { method: 'DELETE' })
}

export async function importLocalGrokAccounts(): Promise<{
  created: GrokAccount[]
  count: number
}> {
  return api('/api/v1/grok-accounts/import-local', { method: 'POST' })
}
