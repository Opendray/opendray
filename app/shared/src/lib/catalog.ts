import { api } from './api'
import type { Provider } from './types'

export async function listProviders(): Promise<Provider[]> {
  const res = await api<{ providers: Provider[] }>('/api/v1/providers')
  return res.providers ?? []
}

export async function getProvider(id: string): Promise<Provider> {
  return api<Provider>(`/api/v1/providers/${id}`)
}

export async function updateProviderConfig(
  id: string,
  config: Record<string, unknown>,
): Promise<Provider> {
  return api<Provider>(`/api/v1/providers/${id}/config`, {
    method: 'PATCH',
    body: config,
  })
}

export async function toggleProvider(
  id: string,
  enabled: boolean,
): Promise<Provider> {
  return api<Provider>(`/api/v1/providers/${id}/toggle`, {
    method: 'PATCH',
    body: { enabled },
  })
}
