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
}

export async function getVersionInfo(): Promise<VersionInfo> {
  return api<VersionInfo>('/version')
}

export async function requestSelfUpdate(): Promise<SelfUpdateResponse> {
  return api<SelfUpdateResponse>('/version/update', { method: 'POST' })
}
