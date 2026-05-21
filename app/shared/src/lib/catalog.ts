import { api } from './api'
import type { Provider, ProviderRuntime } from './types'

export async function listProviders(): Promise<Provider[]> {
  const res = await api<{ providers: Provider[] }>('/api/v1/providers')
  return res.providers ?? []
}

// checkProviderUpdate probes the installed version AND the latest npm
// version (network; cached server-side). Kept separate from
// listProviders so the list render isn't blocked on the npm lookup.
export async function checkProviderUpdate(
  id: string,
): Promise<ProviderRuntime> {
  return api<ProviderRuntime>(`/api/v1/providers/${id}/update-check`)
}

export interface ProviderUpdateResult {
  package: string
  beforeVersion?: string
  afterVersion?: string
  changed: boolean
  output?: string
}

// updateProvider patches the CLI to the latest npm version (admin-only,
// audited server-side). The npm package is taken from the trusted
// manifest, not the request — no arbitrary-package vector.
export async function updateProvider(
  id: string,
): Promise<ProviderUpdateResult> {
  return api<ProviderUpdateResult>(`/api/v1/providers/${id}/update`, {
    method: 'POST',
  })
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
