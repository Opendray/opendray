import { api, APIError } from './api'

export interface FsEntry {
  name: string
  path: string
  is_dir: boolean
}

export interface FsListResponse {
  path: string
  parent?: string
  entries: FsEntry[]
}

export async function listDir(path?: string): Promise<FsListResponse> {
  const q = path ? `?path=${encodeURIComponent(path)}` : ''
  return api<FsListResponse>(`/api/v1/fs/list${q}`)
}

export async function getHomeDir(): Promise<string> {
  const res = await api<{ path: string }>('/api/v1/fs/home')
  return res.path
}

export async function makeDir(parent: string, name: string): Promise<string> {
  const res = await api<{ path: string }>('/api/v1/fs/mkdir', {
    method: 'POST',
    body: { parent, name },
  })
  return res.path
}

// readFile fetches the raw text of a single file. Returns null when
// the file doesn't exist (404) so callers can probe for optional
// task-runner manifests (package.json, Makefile, …) without throwing.
// Other errors propagate.
export async function readFile(path: string): Promise<string | null> {
  try {
    return await api<string>(
      `/api/v1/fs/read?path=${encodeURIComponent(path)}`,
    )
  } catch (e) {
    if (e instanceof APIError && e.status === 404) return null
    throw e
  }
}
