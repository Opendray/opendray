import { api } from './api'

// McpServer mirrors internal/mcp/mcp.go's Server. Env / Headers values
// can contain ${KEY} placeholders that the gateway substitutes from
// the secrets file at spawn time — the on-disk mcp.json keeps the
// placeholder so the file stays git-safe.
export interface McpServer {
  id: string
  name: string
  description?: string
  transport?: 'stdio' | 'sse' | 'http'
  command?: string
  args?: string[]
  env?: Record<string, string>
  url?: string
  headers?: Record<string, string>
  enabled: boolean
}

export interface McpSecretsState {
  // Absolute path the gateway persists secrets to. Shown to the user
  // so they know where the file lives (for backups, audits, etc.).
  path: string
  // True when the file exists on disk. False on first-run.
  present: boolean
  // True when the on-disk file is AES-GCM encrypted with a key from
  // the OS keychain. False = plaintext fallback (keychain unavailable
  // — gateway logs a warning on startup).
  encrypted: boolean
  // Sorted list of key names currently stored. Values are NEVER
  // returned over the wire — the only paths to view a secret are to
  // re-set it (overwrite) or to remove it.
  keys: string[]
}

export async function listMcps(): Promise<McpServer[]> {
  const res = await api<{ servers: McpServer[] }>('/api/v1/mcps')
  return res.servers ?? []
}

export async function getMcp(id: string): Promise<McpServer> {
  return api<McpServer>(`/api/v1/mcps/${id}`)
}

export async function createMcp(
  id: string,
  server: McpServer,
): Promise<McpServer> {
  return api<McpServer>('/api/v1/mcps', {
    method: 'POST',
    body: { id, server },
  })
}

export async function updateMcp(
  id: string,
  server: McpServer,
): Promise<McpServer> {
  return api<McpServer>(`/api/v1/mcps/${id}`, {
    method: 'PUT',
    body: { id, server },
  })
}

export async function deleteMcp(id: string): Promise<void> {
  await api(`/api/v1/mcps/${id}`, { method: 'DELETE' })
}

export async function getMcpSecrets(): Promise<McpSecretsState> {
  return api<McpSecretsState>('/api/v1/mcps/_secrets')
}

export async function setMcpSecret(
  key: string,
  value: string,
): Promise<McpSecretsState> {
  return api<McpSecretsState>(
    `/api/v1/mcps/_secrets/${encodeURIComponent(key)}`,
    { method: 'PUT', body: { value } },
  )
}

export async function deleteMcpSecret(key: string): Promise<void> {
  await api(`/api/v1/mcps/_secrets/${encodeURIComponent(key)}`, {
    method: 'DELETE',
  })
}

// McpCheck is one validation step; McpTestResult is the whole outcome
// of POST /mcps/{id}/test (mirrors internal/mcp/validate.go).
export interface McpCheck {
  name: string
  ok: boolean
  detail?: string
}

export interface McpTestResult {
  ok: boolean
  transport: string
  checks: McpCheck[]
  toolCount?: number
  tools?: string[]
  serverName?: string
  serverVersion?: string
  note?: string
  missingEnv?: string[]
  latencyMs?: number
}

// testMcp validates a server from the daemon: stdio → live MCP
// handshake (real tool count); sse/http → config-sanity + reachability.
export async function testMcp(id: string): Promise<McpTestResult> {
  return api<McpTestResult>(`/api/v1/mcps/${id}/test`, { method: 'POST' })
}

// McpTransport mirrors McpServer.transport. Exported so the Plugins
// editor can drive the transport selector that picks which template
// shape defaultMcpServer returns.
export type McpTransport = NonNullable<McpServer['transport']>

// defaultMcpServer returns a starter template for the New dialog.
// The shape depends on transport: stdio servers need command/args/env,
// while sse/http remote servers need url/headers — the backend rejects
// sse/http without a url (see internal/mcp/handler.go,
// prepareServerForWrite, added in #230).
export function defaultMcpServer(transport: McpTransport = 'stdio'): McpServer {
  const base: McpServer = {
    id: '',
    name: '',
    description: '',
    transport,
    enabled: true,
  }
  if (transport === 'sse' || transport === 'http') {
    return {
      ...base,
      url: 'https://example.com/mcp',
      headers: {},
    }
  }
  return {
    ...base,
    command: 'npx',
    args: ['-y', '@modelcontextprotocol/server-filesystem', '/path/to/expose'],
    env: {},
  }
}
