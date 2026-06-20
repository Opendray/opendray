// Wire types for the opendray API. Kept hand-maintained against
// docs/integration-guide.md — when the gateway exposes an OpenAPI
// spec these should be generated instead.

export type Iso8601 = string;

// ── Integrations ────────────────────────────────────────────────────

export type IntegrationScope = string; // free-form, see docs/integration-guide.md

export interface Integration {
  id: string;
  name: string;
  base_url?: string;
  route_prefix: string;
  scopes: IntegrationScope[];
  version: string;
  enabled: boolean;
  created_at: Iso8601;
  last_health_check?: Iso8601;
  health_status?: "ok" | "degraded" | "down" | "unknown";
}

export interface IntegrationRegistration {
  name: string;
  base_url?: string;
  route_prefix: string;
  scopes: IntegrationScope[];
  version: string;
}

export interface IntegrationCreated extends Integration {
  /** Plaintext API key. Returned exactly once. */
  api_key: string;
}

export interface IntegrationUpdate {
  base_url?: string;
  scopes?: IntegrationScope[];
  version?: string;
  enabled?: boolean;
}

// ── Sessions ────────────────────────────────────────────────────────

export type SessionState =
  | "starting"
  | "running"
  | "idle"
  | "ended"
  | "errored";

export interface Session {
  id: string;
  provider: string; // "claude" | "codex" | "antigravity" | "shell" | ...
  state: SessionState;
  cwd?: string;
  cols?: number;
  rows?: number;
  created_at: Iso8601;
  last_activity_at?: Iso8601;
  ended_at?: Iso8601;
  exit_code?: number;
}

export interface SessionCreateRequest {
  provider: string;
  cwd?: string;
  cols?: number;
  rows?: number;
  env?: Record<string, string>;
}

export interface SessionInputRequest {
  /** Raw bytes to write to the PTY's stdin. */
  data: string;
}

export interface SessionResizeRequest {
  cols: number;
  rows: number;
}

export interface SessionBuffer {
  /** Ring-buffer replay of recent terminal output. */
  data: string;
  truncated: boolean;
}

// ── Providers / channels ────────────────────────────────────────────

export interface Provider {
  id: string;
  name: string;
  command: string;
  config?: Record<string, unknown>;
  available: boolean;
}

export interface Channel {
  id: string;
  kind: string; // "telegram" | "slack" | "discord" | ...
  label: string;
  enabled: boolean;
}

// ── Event WS frames ─────────────────────────────────────────────────

export interface EventFrame<T = unknown> {
  topic: string;
  ts: Iso8601;
  data: T;
}

export interface SessionOutputData {
  session_id: string;
  data: string;
}

export interface SessionEndedData {
  session_id: string;
  exit_code?: number;
  reason?: string;
}

export interface SessionIdleData {
  session_id: string;
  idle_for_ms: number;
}
