import { api } from './api'
import type {
  OpenCodeAccount,
  CreateOpenCodeAccountRequest,
  UpdateOpenCodeAccountRequest,
} from './types'

export async function listOpenCodeAccounts(): Promise<OpenCodeAccount[]> {
  const res = await api<{ accounts: OpenCodeAccount[] }>(
    '/api/v1/opencode-accounts',
  )
  return res.accounts ?? []
}

export async function createOpenCodeAccount(
  req: CreateOpenCodeAccountRequest,
): Promise<OpenCodeAccount> {
  return api<OpenCodeAccount>('/api/v1/opencode-accounts', {
    method: 'POST',
    body: req,
  })
}

export async function updateOpenCodeAccount(
  id: string,
  req: UpdateOpenCodeAccountRequest,
): Promise<OpenCodeAccount> {
  return api<OpenCodeAccount>(`/api/v1/opencode-accounts/${id}`, {
    method: 'PUT',
    body: req,
  })
}

export async function toggleOpenCodeAccount(
  id: string,
  enabled: boolean,
): Promise<OpenCodeAccount> {
  return api<OpenCodeAccount>(`/api/v1/opencode-accounts/${id}/toggle`, {
    method: 'PATCH',
    body: { enabled },
  })
}

export async function deleteOpenCodeAccount(id: string): Promise<void> {
  await api<unknown>(`/api/v1/opencode-accounts/${id}`, { method: 'DELETE' })
}

// importLocalOpenCodeAccount registers the gateway user's current on-disk
// opencode credentials (auth.json) as an account (default name "local").
export async function importLocalOpenCodeAccount(
  name?: string,
): Promise<OpenCodeAccount> {
  return api<OpenCodeAccount>('/api/v1/opencode-accounts/import-local', {
    method: 'POST',
    body: name ? { name } : {},
  })
}
