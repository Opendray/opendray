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
  // Description (markdown). Only populated by getGitPR (the single-PR
  // detail fetch); the list endpoint omits it to stay lean.
  body?: string
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

// getGitPR fetches a single PR including its body/description. Use this
// for the detail surface; listGitPRs leaves body empty.
export async function getGitPR(
  path: string,
  number: number,
): Promise<GitPullRequest> {
  const params = new URLSearchParams({ path })
  return api<GitPullRequest>(
    `/api/v1/git/prs/${number}?${params.toString()}`,
  )
}

// ── PR write ops ───────────────────────────────────────────────

export interface CreatePRRequest {
  dir: string
  title: string
  body?: string
  head: string // source branch (required)
  base?: string // target branch — server resolves default when omitted
  draft?: boolean
}

export interface MergePRRequest {
  dir: string
  number: number
  // GitHub merge methods. Gitea and GitLab adapters map these to
  // their native vocabularies; squash is the default everywhere.
  method?: 'squash' | 'merge' | 'rebase'
  commit_title?: string
  commit_message?: string
  delete_branch?: boolean
}

export async function createGitPR(
  req: CreatePRRequest,
): Promise<GitPullRequest> {
  return api<GitPullRequest>('/api/v1/git/prs', { method: 'POST', body: req })
}

export async function mergeGitPR(
  req: MergePRRequest,
): Promise<GitPullRequest> {
  return api<GitPullRequest>(`/api/v1/git/prs/${req.number}/merge`, {
    method: 'POST',
    body: req,
  })
}

// ── PR checks (CI) ─────────────────────────────────────────────

export interface CheckRun {
  name: string
  // GitHub Checks API vocabulary: queued | in_progress | completed
  status: string
  // Filled when status === 'completed'. success | failure | neutral |
  // cancelled | skipped | timed_out | action_required
  conclusion: string
  url: string
  updated_at: string
}

export async function getPRChecks(
  path: string,
  number: number,
): Promise<CheckRun[]> {
  const params = new URLSearchParams({ path })
  const res = await api<{ checks: CheckRun[] }>(
    `/api/v1/git/prs/${number}/checks?${params.toString()}`,
  )
  return res.checks ?? []
}

// ── PR detail tabs: commits / files / conversation ─────────────

export interface PRCommit {
  sha: string
  short_sha: string
  message: string // full message; UI shows the first line
  author: string
  date: string
  url: string
}

export interface PRFile {
  filename: string
  // added | modified | removed | renamed
  status: string
  additions: number
  deletions: number
  // Unified diff; absent when the host doesn't return it inline.
  patch?: string
}

export interface PRComment {
  author: string
  body: string
  created_at: string
  // Set only for review summaries: approved | changes_requested | commented
  state?: string
  url?: string
}

export async function getPRCommits(
  path: string,
  number: number,
): Promise<PRCommit[]> {
  const params = new URLSearchParams({ path })
  const res = await api<{ commits: PRCommit[] }>(
    `/api/v1/git/prs/${number}/commits?${params.toString()}`,
  )
  return res.commits ?? []
}

export async function getPRFiles(
  path: string,
  number: number,
): Promise<PRFile[]> {
  const params = new URLSearchParams({ path })
  const res = await api<{ files: PRFile[] }>(
    `/api/v1/git/prs/${number}/files?${params.toString()}`,
  )
  return res.files ?? []
}

export async function getPRComments(
  path: string,
  number: number,
): Promise<PRComment[]> {
  const params = new URLSearchParams({ path })
  const res = await api<{ comments: PRComment[] }>(
    `/api/v1/git/prs/${number}/comments?${params.toString()}`,
  )
  return res.comments ?? []
}

// ── Issues (read-only) ─────────────────────────────────────────

export interface GitLabel {
  name: string
  // Hex colour without the leading '#'. May be empty (GitLab issue
  // payloads carry only the label name).
  color: string
}

export interface GitIssue {
  number: number
  title: string
  state: 'open' | 'closed'
  author: string
  url: string
  // Description (markdown). Only populated by getGitIssue (the single-
  // issue detail fetch); the list endpoint omits it to stay lean.
  body?: string
  labels: GitLabel[]
  updated_at: string
}

export interface GitIssuesResponse {
  remote: GitRemote
  issues: GitIssue[]
  need_token?: boolean
  error?: string
}

export async function listGitIssues(
  path: string,
  state: 'open' | 'closed' | 'all' = 'open',
): Promise<GitIssuesResponse> {
  const params = new URLSearchParams({ path, state })
  return api<GitIssuesResponse>(`/api/v1/git/issues?${params.toString()}`)
}

// getGitIssue fetches a single issue including its body/description. Use
// this for the detail surface; listGitIssues leaves body empty.
export async function getGitIssue(
  path: string,
  number: number,
): Promise<GitIssue> {
  const params = new URLSearchParams({ path })
  return api<GitIssue>(`/api/v1/git/issues/${number}?${params.toString()}`)
}

// Issue comments reuse PRComment — the conversation shape is identical
// (issues never carry a review state, so it stays empty).
export async function getIssueComments(
  path: string,
  number: number,
): Promise<PRComment[]> {
  const params = new URLSearchParams({ path })
  const res = await api<{ comments: PRComment[] }>(
    `/api/v1/git/issues/${number}/comments?${params.toString()}`,
  )
  return res.comments ?? []
}
