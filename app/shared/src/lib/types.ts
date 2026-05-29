// ── Sessions ────────────────────────────────────────────────

export type SessionState =
  | 'pending'
  | 'running'
  | 'idle'
  | 'stopped'
  | 'ended'

export const TERMINAL_SESSION_STATES: SessionState[] = ['stopped', 'ended']

export function isTerminalSessionState(s: SessionState): boolean {
  return s === 'stopped' || s === 'ended'
}

export interface Session {
  id: string
  name?: string
  provider_id: string
  cwd: string
  args: string[]
  state: SessionState
  pid?: number
  claude_account_id?: string
  claude_session_id?: string
  /** Set when this session was spawned on behalf of another (e.g. a Task). */
  parent_session_id?: string
  started_at: string
  ended_at?: string
  exit_code?: number
}

export interface CreateSessionRequest {
  provider_id: string
  cwd: string
  name?: string
  args?: string[]
  claude_account_id?: string
  parent_session_id?: string
}

// ── Claude accounts (OAuth-token-on-disk model, mirrors v1) ─

export interface ClaudeAccount {
  id: string
  name: string
  display_name: string
  config_dir: string
  token_path: string
  description: string
  enabled: boolean
  token_filled: boolean
  created_at: string
  updated_at: string
}

export interface CreateClaudeAccountRequest {
  name: string
  display_name?: string
  config_dir?: string
  token_path?: string
  description?: string
  enabled?: boolean
  token?: string
}

export interface UpdateClaudeAccountRequest {
  name?: string
  display_name?: string
  config_dir?: string
  token_path?: string
  description?: string
  enabled?: boolean
}

// ── Catalog (providers) ─────────────────────────────────────

export type ConfigFieldType =
  | 'string'
  | 'number'
  | 'boolean'
  | 'select'
  | 'secret'
  | 'args'

export interface ConfigField {
  key: string
  label: string
  label_zh?: string
  type: ConfigFieldType
  group?: string
  default?: unknown
  options?: string[]
  placeholder?: string
  description?: string
  description_zh?: string
  envVar?: string
  cliFlag?: string
  cliValue?: boolean
  dependsOn?: string
  dependsVal?: unknown
}

export interface ProviderManifest {
  id: string
  displayName: string
  displayName_zh?: string
  description: string
  description_zh?: string
  icon: string
  version: string
  kind: 'cli' | 'shell'
  executable: string
  npmPackage?: string
  modelFlag?: string
  knownModels?: string[]
  defaultArgs?: string[]
  capabilities: {
    supportsResume: boolean
    supportsStream: boolean
    supportsImages: boolean
    supportsMcp: boolean
  }
  configSchema?: ConfigField[]
}

// ProviderRuntime is the live, probed CLI state (not from the manifest):
// whether the binary is installed and its real `--version`, plus the
// latest npm version when an update-check has run.
export interface ProviderRuntime {
  installed: boolean
  installedVersion?: string
  path?: string
  latestVersion?: string
  updateAvailable: boolean
  checkedAt?: string
  // Non-terminal sessions currently using this provider's CLI. Used by
  // the Providers page to warn before upgrading a CLI that running
  // sessions are on it. 0 when the server didn't populate the counter.
  activeSessions: number
}

export interface Provider {
  manifest: ProviderManifest
  manifest_hash: string
  config: Record<string, unknown>
  enabled: boolean
  runtime?: ProviderRuntime
}

// ── Channels ────────────────────────────────────────────────

export interface Channel {
  id: string
  kind: string
  config: Record<string, unknown>
  enabled: boolean
  running: boolean
  capabilities: string[]
  muted: boolean
}

export interface CreateChannelRequest {
  kind: string
  config: Record<string, unknown>
  enabled: boolean
}

// ── Integrations ────────────────────────────────────────────

export type IntegrationHealth =
  | 'unknown'
  | 'healthy'
  | 'degraded'
  | 'unhealthy'

export interface Integration {
  id: string
  name: string
  base_url: string
  route_prefix: string
  scopes: string[]
  version?: string
  enabled: boolean
  health_status: IntegrationHealth
  health_payload?: Record<string, unknown>
  health_last_seen?: string
  created_at: string
  rotated_at?: string
  /** True for rows opendray manages itself (e.g. opendray-memory).
      Operators can't delete or rotate these from the UI. */
  is_system: boolean
}

export interface RegisterIntegrationRequest {
  name: string
  base_url: string
  route_prefix: string
  scopes?: string[]
  version?: string
}

export interface RegisterIntegrationResult {
  integration: Integration
  api_key: string
}

export const ALL_SCOPES = [
  'session:read',
  'session:create',
  'session:input',
  'channel:send',
  'channel:receive',
  'event:subscribe:session.*',
  'event:subscribe:channel.*',
  'event:subscribe:integration.*',
  'provider:read',
  'memory:read',
  'memory:write',
] as const

export type Scope = (typeof ALL_SCOPES)[number] | string
