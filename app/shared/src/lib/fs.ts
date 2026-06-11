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

/**
 * Build a same-origin URL that, when navigated to, streams the
 * referenced file as a download (server sets `Content-Disposition:
 * attachment`). Auth rides in the query string — browsers can't add
 * Authorization headers to anchor navigations, and the gateway's
 * combined middleware accepts a `?token=` fallback (same path the
 * Terminal WS uses).
 */
export function fsDownloadURL(path: string, token: string): string {
  return (
    `/api/v1/fs/download?path=${encodeURIComponent(path)}` +
    `&token=${encodeURIComponent(token)}`
  )
}

/**
 * Same as `fsDownloadURL`, but for a directory subtree — the gateway
 * streams a zip archive built on the fly. Hidden entries and symlinks
 * are skipped to match the file-tree's listing behaviour.
 */
export function fsZipURL(path: string, token: string): string {
  return (
    `/api/v1/fs/zip?path=${encodeURIComponent(path)}` +
    `&token=${encodeURIComponent(token)}`
  )
}

/**
 * Trigger a browser download for the given URL via an off-DOM `<a
 * download>`. Used by the Files inspector so the click doesn't have
 * to be on an anchor (we want the hover-icon affordance, not an
 * always-visible link).
 */
export function triggerDownload(url: string, suggestedName?: string): void {
  const a = document.createElement('a')
  a.href = url
  if (suggestedName) a.download = suggestedName
  a.rel = 'noopener'
  a.style.display = 'none'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}
