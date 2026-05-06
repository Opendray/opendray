import { api } from './api'
import type { Channel, CreateChannelRequest } from './types'

export async function listChannels(): Promise<Channel[]> {
  const res = await api<{ channels: Channel[] }>('/api/v1/channels')
  return res.channels ?? []
}

export async function getChannel(id: string): Promise<Channel> {
  return api<Channel>(`/api/v1/channels/${id}`)
}

export async function createChannel(
  req: CreateChannelRequest,
): Promise<Channel> {
  return api<Channel>('/api/v1/channels', { method: 'POST', body: req })
}

export async function updateChannel(
  id: string,
  patch: { config?: Record<string, unknown>; enabled?: boolean },
): Promise<Channel> {
  return api<Channel>(`/api/v1/channels/${id}`, {
    method: 'PATCH',
    body: patch,
  })
}

export async function deleteChannel(id: string): Promise<void> {
  await api(`/api/v1/channels/${id}`, { method: 'DELETE' })
}

export async function testChannel(id: string): Promise<void> {
  await api(`/api/v1/channels/${id}/test`, { method: 'POST' })
}

export async function listChannelKinds(): Promise<string[]> {
  const res = await api<{ kinds: string[] }>('/api/v1/channels/_kinds')
  return res.kinds ?? []
}
