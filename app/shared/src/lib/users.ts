import { api } from '@/lib/api'
import type { Role } from '@/stores/auth'

export interface Me {
  username: string
  role: Role
  primary: boolean
  active_source: string
}

export interface UserRow {
  id: string
  username: string
  display_name: string
  role: Role
  disabled: boolean
  primary?: boolean
  created_at: string
  last_login_at?: string
}

export const getMe = () => api<Me>('/api/v1/auth/me')

export const listUsers = () =>
  api<{ users: UserRow[] }>('/api/v1/users').then((r) => r.users)

export const createUser = (body: {
  username: string
  password: string
  role: Role
  display_name?: string
}) => api<UserRow>('/api/v1/users', { method: 'POST', body })

export const updateUser = (
  id: string,
  body: Partial<{ display_name: string; role: Role; disabled: boolean; password: string }>,
) => api<UserRow>(`/api/v1/users/${encodeURIComponent(id)}`, { method: 'PATCH', body })

export const deleteUser = (id: string) =>
  api<void>(`/api/v1/users/${encodeURIComponent(id)}`, { method: 'DELETE' })
