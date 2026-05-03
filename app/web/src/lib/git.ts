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
