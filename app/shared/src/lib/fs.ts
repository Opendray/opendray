import { api, APIError } from './api'
import { useAuth } from '@/stores/auth'

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
 * attachment`). The caller MUST pass a `root` — the server verifies
 * the resolved file path stays inside it, so downloads are confined
 * to the directory subtree the operator is browsing (typically the
 * session's `cwd`), not arbitrary system paths.
 *
 * Auth rides in the query string — browsers can't add Authorization
 * headers to anchor navigations, and the gateway's combined
 * middleware accepts a `?token=` fallback (same path the Terminal WS
 * uses).
 */
export function fsDownloadURL(
  path: string,
  root: string,
  token: string,
): string {
  return (
    `/api/v1/fs/download?path=${encodeURIComponent(path)}` +
    `&root=${encodeURIComponent(root)}` +
    `&token=${encodeURIComponent(token)}`
  )
}

/**
 * Same as `fsDownloadURL`, but for a directory subtree — the gateway
 * streams a zip archive built on the fly. Hidden entries and symlinks
 * are skipped to match the file-tree's listing behaviour. The `root`
 * confinement applies the same way: the resolved directory must live
 * inside the caller-supplied root.
 */
export function fsZipURL(
  path: string,
  root: string,
  token: string,
): string {
  return (
    `/api/v1/fs/zip?path=${encodeURIComponent(path)}` +
    `&root=${encodeURIComponent(root)}` +
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

export interface WalkedFile {
  relpath: string
  file: File
}

/**
 * Expand a drag-and-drop DataTransfer into a flat list of files, each
 * tagged with its path relative to the drop. Folder drops preserve
 * their subtree; loose files get `relpath = file.name`. Uses the
 * non-standard-but-universal webkitGetAsEntry API, falling back to a
 * flat `dt.files` list when no entries are exposed. Pure: no network,
 * no store.
 */
export async function walkDropEntries(dt: DataTransfer): Promise<WalkedFile[]> {
  const roots: FileSystemEntry[] = []
  for (const item of Array.from(dt.items)) {
    if (item.kind !== 'file') continue
    const entry = item.webkitGetAsEntry?.()
    if (entry) roots.push(entry)
  }
  if (roots.length === 0) {
    return Array.from(dt.files).map((file) => ({ relpath: file.name, file }))
  }
  const out: WalkedFile[] = []
  for (const entry of roots) await walkEntry(entry, '', out)
  return out
}

async function walkEntry(
  entry: FileSystemEntry,
  prefix: string,
  out: WalkedFile[],
): Promise<void> {
  if (entry.isFile) {
    const file = await new Promise<File>((resolve, reject) =>
      (entry as FileSystemFileEntry).file(resolve, reject),
    )
    out.push({ relpath: prefix + entry.name, file })
    return
  }
  const reader = (entry as FileSystemDirectoryEntry).createReader()
  const dirPrefix = `${prefix}${entry.name}/`
  // readEntries yields at most 100 per call — drain until empty.
  for (;;) {
    const batch = await new Promise<FileSystemEntry[]>((resolve, reject) =>
      reader.readEntries(resolve, reject),
    )
    if (batch.length === 0) break
    for (const child of batch) await walkEntry(child, dirPrefix, out)
  }
}

export interface UploadResult {
  path: string
  size: number
  renamed_from?: string
}

export interface UploadArgs {
  root: string
  dir: string
  relpath: string
  file: File
  signal?: AbortSignal
  onProgress?: (loaded: number, total: number) => void
}

/**
 * Upload a single file to /fs/upload. Metadata rides in the query
 * string (mirroring /fs/download); the raw file bytes are the body so
 * large files stream once. Uses XMLHttpRequest solely for
 * `upload.onprogress`; the bearer token is attached the same way api()
 * does, read synchronously from the auth store.
 */
export function uploadFile(args: UploadArgs): Promise<UploadResult> {
  const { root, dir, relpath, file, signal, onProgress } = args
  const url =
    `/api/v1/fs/upload?root=${encodeURIComponent(root)}` +
    `&dir=${encodeURIComponent(dir)}` +
    `&relpath=${encodeURIComponent(relpath)}`
  return new Promise<UploadResult>((resolve, reject) => {
    if (signal?.aborted) {
      reject(new DOMException('aborted', 'AbortError'))
      return
    }
    const xhr = new XMLHttpRequest()
    const onAbort = () => xhr.abort()
    const cleanup = () => signal?.removeEventListener('abort', onAbort)
    xhr.open('POST', url)
    const token = useAuth.getState().token
    if (token) xhr.setRequestHeader('Authorization', `Bearer ${token}`)
    xhr.responseType = 'json'
    if (onProgress) {
      xhr.upload.onprogress = (e) => {
        if (e.lengthComputable) onProgress(e.loaded, e.total)
      }
    }
    xhr.onload = () => {
      cleanup()
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve(xhr.response as UploadResult)
      } else {
        const body = xhr.response as { error?: string } | null
        reject(
          new APIError(
            xhr.status,
            body,
            body?.error ?? `upload failed (${xhr.status})`,
          ),
        )
      }
    }
    xhr.onerror = () => {
      cleanup()
      reject(new APIError(0, null, 'network error'))
    }
    if (signal) {
      signal.addEventListener('abort', onAbort, { once: true })
      xhr.onabort = () => {
        cleanup()
        reject(new DOMException('aborted', 'AbortError'))
      }
    }
    xhr.send(file)
  })
}
