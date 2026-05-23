import { toast } from 'sonner'

import { useAuth } from '@/stores/auth'

export class APIError extends Error {
  status: number
  body: unknown
  constructor(status: number, body: unknown, message: string) {
    super(message)
    this.status = status
    this.body = body
  }
}

interface APIOptions extends Omit<RequestInit, 'body'> {
  body?: unknown
  raw?: boolean // when true, return raw Response (used by /buffer)
  skipAuthRedirect?: boolean
}

export async function api<T = unknown>(
  path: string,
  options: APIOptions = {},
): Promise<T> {
  const token = useAuth.getState().token
  const headers = new Headers(options.headers)

  let body: BodyInit | undefined
  if (options.body !== undefined) {
    if (options.body instanceof FormData) {
      body = options.body
    } else if (typeof options.body === 'string') {
      body = options.body
    } else {
      headers.set('Content-Type', 'application/json')
      body = JSON.stringify(options.body)
    }
  }
  if (token) headers.set('Authorization', `Bearer ${token}`)

  const res = await fetch(path, { ...options, headers, body })

  if (res.status === 401 && !options.skipAuthRedirect) {
    const wasAuthed = useAuth.getState().isAuthed()
    useAuth.getState().clear()
    if (typeof window !== 'undefined' && !path.endsWith('/auth/login')) {
      if (wasAuthed) {
        toast.error('Session expired', {
          description: 'Please sign in again.',
        })
      }
      // The SPA is mounted under Vite's base path (e.g. "/admin/"), so a
      // bare "/login" hits the server's 404 instead of the login route.
      // Build the redirect under the base, and make `next` router-relative
      // (strip the base) so post-login restore doesn't double-prefix it.
      const base = import.meta.env.BASE_URL.replace(/\/$/, '') // "" dev, "/admin" prod
      const path = window.location.pathname
      const rel =
        base && path.startsWith(base) ? path.slice(base.length) || '/' : path
      const next = encodeURIComponent(rel + window.location.search)
      window.location.assign(`${base}/login?next=${next}`)
    }
  }

  if (options.raw) {
    if (!res.ok) throw new APIError(res.status, null, `HTTP ${res.status}`)
    return res as unknown as T
  }

  let parsed: unknown = null
  const ct = res.headers.get('Content-Type') || ''
  if (ct.includes('application/json')) parsed = await res.json()
  else if (res.status !== 204) parsed = await res.text()

  if (!res.ok) {
    const msg =
      parsed && typeof parsed === 'object' && 'error' in parsed
        ? String((parsed as { error: unknown }).error)
        : `HTTP ${res.status}`
    throw new APIError(res.status, parsed, msg)
  }
  return parsed as T
}
