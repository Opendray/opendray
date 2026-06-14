import { api } from './api'

export interface VersionInfo {
  current: string
  commit?: string
  latest?: string
  updateAvailable: boolean
  notesUrl?: string
  // selfUpdate: this host can apply a one-click background upgrade (Linux
  // with the privileged self-update units installed). When false the UI
  // falls back to showing the manual `opendray update` command.
  selfUpdate: boolean
  // pending: an upgrade request is already queued.
  pending: boolean
  // checkError: set when the release feed couldn't be reached (offline /
  // rate-limited); `latest` is then absent and only `current` is known.
  checkError?: string
}

export interface SelfUpdateResponse {
  queued?: boolean
  from?: string
  to?: string
  note?: string
  error?: string
  pending?: boolean
  // needsForce + liveSessions accompany the 409 the gateway returns when
  // live sessions would be interrupted by the restart. The body arrives
  // via the thrown APIError (non-2xx), so callers read these off
  // APIError.body to offer an "upgrade anyway" retry.
  needsForce?: boolean
  liveSessions?: number
}

export async function getVersionInfo(): Promise<VersionInfo> {
  return api<VersionInfo>('/api/v1/version')
}

export async function requestSelfUpdate(
  force = false,
): Promise<SelfUpdateResponse> {
  const q = force ? '?force=true' : ''
  return api<SelfUpdateResponse>(`/api/v1/version/update${q}`, { method: 'POST' })
}
