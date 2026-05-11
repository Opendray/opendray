import { api } from './api'

export type GitHostKind = 'github' | 'gitea' | 'gitlab'

export interface GitHost {
  id: string
  kind: GitHostKind
  host: string
  name: string
  // The full token is never returned by the server after creation —
  // only this masked preview, e.g. "•••• AbC1".
  token_mask?: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface CreateGitHostRequest {
  kind: GitHostKind
  host: string
  name?: string
  token: string
}

export interface UpdateGitHostRequest {
  kind?: GitHostKind
  host?: string
  name?: string
  // Empty / omitted = keep existing token. Send a non-empty string to rotate.
  token?: string
  enabled?: boolean
}

export async function listGitHosts(): Promise<GitHost[]> {
  const res = await api<{ hosts: GitHost[] }>('/api/v1/git-hosts')
  return res.hosts ?? []
}

export async function createGitHost(
  req: CreateGitHostRequest,
): Promise<GitHost> {
  return api<GitHost>('/api/v1/git-hosts', { method: 'POST', body: req })
}

export async function updateGitHost(
  id: string,
  req: UpdateGitHostRequest,
): Promise<GitHost> {
  return api<GitHost>(`/api/v1/git-hosts/${id}`, { method: 'PUT', body: req })
}

export async function deleteGitHost(id: string): Promise<void> {
  await api(`/api/v1/git-hosts/${id}`, { method: 'DELETE' })
}

// ── Remote detection + PRs ─────────────────────────────────────

export interface GitRemote {
  url: string
  host: string
  owner: string
  repo: string
  kind?: GitHostKind
  has_token: boolean
  web_url?: string
}

export interface GitPullRequest {
  number: number
  title: string
  state: 'open' | 'closed' | 'merged'
  author: string
  head: string
  base: string
  url: string
  draft: boolean
  updated_at: string
}

export interface GitPullRequestsResponse {
  remote: GitRemote
  prs: GitPullRequest[]
  need_token?: boolean
  error?: string
}

export async function getGitRemote(path: string): Promise<GitRemote> {
  return api<GitRemote>(`/api/v1/git/remote?path=${encodeURIComponent(path)}`)
}

export async function listGitPRs(
  path: string,
  state: 'open' | 'closed' | 'all' = 'open',
): Promise<GitPullRequestsResponse> {
  const params = new URLSearchParams({ path, state })
  return api<GitPullRequestsResponse>(`/api/v1/git/prs?${params.toString()}`)
}
