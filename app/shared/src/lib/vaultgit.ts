import { api } from './api'

export interface VaultStatusFile {
  xy: string
  path: string
}

export interface VaultGitState {
  rebase_in_progress?: boolean
  merge_in_progress?: boolean
  cherry_pick_in_progress?: boolean
  conflicted_files?: string[]
}

export interface VaultStatus {
  is_repo: boolean
  branch?: string
  upstream?: string
  ahead: number
  behind: number
  files: VaultStatusFile[]
  root: string
  state?: VaultGitState
}

export interface VaultCommit {
  hash: string
  short_hash: string
  author: string
  when: string
  subject: string
}

export interface VaultRemote {
  name: string
  url: string
}

export async function vaultStatus(): Promise<VaultStatus> {
  return api<VaultStatus>('/api/v1/vault/git/status')
}

export async function vaultInit(): Promise<{ output: string }> {
  return api<{ output: string }>('/api/v1/vault/git/init', { method: 'POST' })
}

export async function vaultCommit(opts: {
  message?: string
  files?: string[]
}): Promise<{ hash: string; message: string; output: string }> {
  return api('/api/v1/vault/git/commit', { method: 'POST', body: opts })
}

export async function vaultPull(): Promise<{ output: string }> {
  return api<{ output: string }>('/api/v1/vault/git/pull', { method: 'POST' })
}

export async function vaultPush(): Promise<{ output: string }> {
  return api<{ output: string }>('/api/v1/vault/git/push', { method: 'POST' })
}

export async function vaultLog(n = 20): Promise<VaultCommit[]> {
  const res = await api<{ commits: VaultCommit[] }>(
    `/api/v1/vault/git/log?n=${n}`,
  )
  return res.commits ?? []
}

export async function vaultGetRemotes(): Promise<VaultRemote[]> {
  const res = await api<{ remotes: VaultRemote[] }>('/api/v1/vault/git/remote')
  return res.remotes ?? []
}

export async function vaultSetRemote(name: string, url: string): Promise<void> {
  await api('/api/v1/vault/git/remote', {
    method: 'POST',
    body: { name, url },
  })
}

export interface VaultAuthInfo {
  has_remote: boolean
  remote_url?: string
  scheme?: 'ssh' | 'https' | 'http' | 'git' | string
  host?: string
  using_token?: boolean
  token_source?: string
  token_missing?: boolean
  helpful_hint?: string
}

export async function vaultAuthInfo(): Promise<VaultAuthInfo> {
  return api<VaultAuthInfo>('/api/v1/vault/git/auth')
}

// vaultAbort cancels an in-progress rebase / merge / cherry-pick.
// Pass kind to force a specific abort, or "auto" to detect.
export async function vaultAbort(
  kind: 'auto' | 'rebase' | 'merge' | 'cherry-pick' = 'auto',
): Promise<{ output: string; kind: string }> {
  return api('/api/v1/vault/git/abort', { method: 'POST', body: { kind } })
}

// vaultResetToRemote is destructive — wipes any local commits AND
// uncommitted changes by hard-resetting to the remote branch and
// `git clean -fd`-ing untracked junk. UI must confirm before calling.
export async function vaultResetToRemote(
  remoteBranch?: string,
): Promise<{ output: string; remote_branch: string }> {
  return api('/api/v1/vault/git/reset-to-remote', {
    method: 'POST',
    body: { remote_branch: remoteBranch ?? '' },
  })
}

// VaultSyncConfig mirrors the server's persistent auto-sync settings.
// Intervals are Go duration strings (e.g. "10m0s", "1h0m0s").
// All last_* timestamps are ISO 8601 strings or absent.
export interface VaultSyncConfig {
  enabled: boolean
  commit_interval: string
  push_enabled: boolean
  pull_enabled: boolean
  pull_interval: string
  commit_message?: string
  last_commit_at?: string
  last_commit_hash?: string
  last_push_at?: string
  last_pull_at?: string
  last_error?: string
  last_error_at?: string
}

// VaultSyncConfigUpdate carries only the fields the UI can change.
// Server-managed timestamps and last_error are read-only.
export interface VaultSyncConfigUpdate {
  enabled?: boolean
  commit_interval?: string
  push_enabled?: boolean
  pull_enabled?: boolean
  pull_interval?: string
  commit_message?: string
}

export async function vaultSyncConfig(): Promise<VaultSyncConfig> {
  return api<VaultSyncConfig>('/api/v1/vault/git/sync/config')
}

export async function setVaultSyncConfig(
  update: VaultSyncConfigUpdate,
): Promise<VaultSyncConfig> {
  return api<VaultSyncConfig>('/api/v1/vault/git/sync/config', {
    method: 'PUT',
    body: update,
  })
}

export async function vaultSyncRunNow(): Promise<{ status: string }> {
  return api<{ status: string }>('/api/v1/vault/git/sync/run', {
    method: 'POST',
  })
}
