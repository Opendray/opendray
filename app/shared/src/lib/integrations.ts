import { api } from './api'
import type {
  Integration,
  RegisterIntegrationRequest,
  RegisterIntegrationResult,
} from './types'

export async function listIntegrations(): Promise<Integration[]> {
  const res = await api<{ integrations: Integration[] }>('/api/v1/integrations')
  return res.integrations ?? []
}

export async function getIntegration(id: string): Promise<Integration> {
  return api<Integration>(`/api/v1/integrations/${id}`)
}

export async function registerIntegration(
  req: RegisterIntegrationRequest,
): Promise<RegisterIntegrationResult> {
  return api<RegisterIntegrationResult>('/api/v1/integrations', {
    method: 'POST',
    body: req,
  })
}

export async function rotateIntegrationKey(
  id: string,
): Promise<RegisterIntegrationResult> {
  return api<RegisterIntegrationResult>(
    `/api/v1/integrations/${id}/rotate-key`,
    { method: 'POST' },
  )
}

export async function updateIntegration(
  id: string,
  patch: {
    base_url?: string
    scopes?: string[]
    version?: string
    enabled?: boolean
  },
): Promise<Integration> {
  return api<Integration>(`/api/v1/integrations/${id}`, {
    method: 'PATCH',
    body: patch,
  })
}

export async function deleteIntegration(id: string): Promise<void> {
  await api(`/api/v1/integrations/${id}`, { method: 'DELETE' })
}
