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
  /** Account/org the remote points at — what selects the credential. */
  remote_owner?: string
  using_token?: boolean
  token_source?: string
  /** Owner the resolved credential is scoped to; "" = host-wide. */
  token_owner?: string
  /** That credential's display name, so several on one host differ. */
  token_name?: string
  /** The remote's owner has no credential of its own; the host-wide
   * one is being used. Legitimate, but also the shape of the mistake
   * where a token silently authenticates as the wrong identity. */
  token_is_fallback?: boolean
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

/** What a reset would destroy that exists nowhere else. */
export interface VaultResetLoss {
  unpushed_commits: number
  untracked_files: number
  modified_files: number
  /** A few affected paths, so the warning names real files. */
  sample?: string[]
}

export interface VaultResetResponse {
  output: string
  remote_branch: string
  /** Branch holding the unpushed commits, if any were parked. */
  rescue_ref?: string
  /** Stash message holding the working tree, if any was parked. */
  rescue_stash?: string
}

// vaultResetToRemote hard-resets the vault onto its remote branch and
// `git clean -fd`s the rest.
//
// Call it WITHOUT confirm first. When anything would be lost the server
// answers 409 with a `loss` breakdown; show those numbers and call
// again with confirm: true. This two-step exists because the old
// one-step version, behind a confirmation that named no quantity,
// destroyed 354 unpushed files on a vault whose remote was empty — the
// remote was empty precisely because every push had been failing.
//
// Confirming is still safe: the server parks the unpushed commits on a
// branch and stashes the working tree before resetting.
export async function vaultResetToRemote(
  remoteBranch?: string,
  confirm = false,
): Promise<VaultResetResponse> {
  return api('/api/v1/vault/git/reset-to-remote', {
    method: 'POST',
    body: { remote_branch: remoteBranch ?? '', confirm },
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
