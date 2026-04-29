export type SessionState = 'pending' | 'running' | 'idle' | 'ended'

export interface Session {
  id: string
  name?: string
  provider_id: string
  cwd: string
  args: string[]
  state: SessionState
  pid?: number
  started_at: string
  ended_at?: string
  exit_code?: number
}

export interface ProviderManifest {
  id: string
  displayName: string
  displayName_zh?: string
  description: string
  icon: string
  version: string
  kind: 'cli' | 'shell'
  executable: string
  capabilities: {
    supportsResume: boolean
    supportsStream: boolean
    supportsImages: boolean
    supportsMcp: boolean
  }
}

export interface Provider {
  manifest: ProviderManifest
  manifest_hash: string
  config: Record<string, unknown>
  enabled: boolean
}

export interface CreateSessionRequest {
  provider_id: string
  cwd: string
  name?: string
  args?: string[]
}
