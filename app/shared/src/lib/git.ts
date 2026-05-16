import { api } from './api'

export interface GitStatusFile {
  xy: string
  path: string
  old_path?: string
}

export interface GitStatus {
  is_repo: boolean
  branch?: string
  ahead: number
  behind: number
  upstream?: string
  files: GitStatusFile[]
}

export interface GitCommit {
  hash: string
  short_hash: string
  author: string
  when: string
  subject: string
}

export interface GitLog {
  is_repo: boolean
  commits: GitCommit[]
}

export async function getGitStatus(path: string): Promise<GitStatus> {
  return api<GitStatus>(
    `/api/v1/git/status?path=${encodeURIComponent(path)}`,
  )
}

export async function getGitLog(path: string, n = 20): Promise<GitLog> {
  return api<GitLog>(
    `/api/v1/git/log?path=${encodeURIComponent(path)}&n=${n}`,
  )
}

export type DiffScope = 'unstaged' | 'staged' | 'all'

// getGitDiff fetches a unified-diff text. `file` is repo-relative; pass
// undefined for a whole-tree diff. Returns '' when there are no
// changes — this is normal output, not an error.
export async function getGitDiff(
  path: string,
  scope: DiffScope = 'all',
  file?: string,
): Promise<string> {
  const params = new URLSearchParams({ path, scope })
  if (file) params.set('file', file)
  return api<string>(`/api/v1/git/diff?${params.toString()}`)
}

// getGitShow returns `git show <hash>` output: commit metadata + diff.
export async function getGitShow(path: string, hash: string): Promise<string> {
  const params = new URLSearchParams({ path, hash })
  return api<string>(`/api/v1/git/show?${params.toString()}`)
}

// ── Write ops (Phase 4) ────────────────────────────────────────

export interface GitBranchRef {
  name: string
  remote?: string
  is_remote: boolean
  is_current: boolean
  upstream?: string
}

export interface GitBranchList {
  branches: GitBranchRef[]
  current: string
}

export async function listGitBranches(path: string): Promise<GitBranchList> {
  return api<GitBranchList>(
    `/api/v1/git/write/branches?path=${encodeURIComponent(path)}`,
  )
}

export interface CreateBranchRequest {
  dir: string
  name: string
  from?: string
  switch?: boolean
}

export async function createGitBranch(
  req: CreateBranchRequest,
): Promise<void> {
  await api('/api/v1/git/write/branches', { method: 'POST', body: req })
}

export async function checkoutGitBranch(
  dir: string,
  name: string,
): Promise<void> {
  await api('/api/v1/git/write/checkout', {
    method: 'POST',
    body: { dir, name },
  })
}

export async function deleteGitBranch(
  path: string,
  name: string,
  force = false,
): Promise<void> {
  const params = new URLSearchParams({ path })
  if (force) params.set('force', 'true')
  await api(
    `/api/v1/git/write/branches/${encodeURIComponent(name)}?${params.toString()}`,
    { method: 'DELETE' },
  )
}

// Empty files[] stages all (`.`).
export async function stageGitFiles(
  dir: string,
  files: string[] = [],
): Promise<void> {
  await api('/api/v1/git/write/stage', {
    method: 'POST',
    body: { dir, files },
  })
}

export async function unstageGitFiles(
  dir: string,
  files: string[] = [],
): Promise<void> {
  await api('/api/v1/git/write/unstage', {
    method: 'POST',
    body: { dir, files },
  })
}

export async function commitGit(
  dir: string,
  message: string,
  allowEmpty = false,
): Promise<{ hash: string }> {
  return api<{ hash: string }>('/api/v1/git/write/commit', {
    method: 'POST',
    body: { dir, message, allow_empty: allowEmpty },
  })
}

export interface PushOptions {
  branch?: string
  force?: boolean
  set_upstream?: boolean
}

export async function pushGit(
  dir: string,
  opts: PushOptions = {},
): Promise<{ branch: string }> {
  return api<{ branch: string }>('/api/v1/git/write/push', {
    method: 'POST',
    body: { dir, ...opts },
  })
}
