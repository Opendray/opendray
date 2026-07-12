// Round Table API client (experimental) — cross-vendor multi-agent
// discussion. A deterministic chair drives claude/codex/antigravity seats
// through propose → critique → synthesize and produces a structured
// Verdict. Wraps the /api/v1/round-tables/* endpoints via api<T>().
import { api } from './api'

// Seat providers with a headless worker path (must match the backend's
// worker.AgentWorker.buildCommand switch). gemini/opencode/grok have no
// headless path yet, so they cannot take a seat.
export type SeatProvider = 'claude' | 'codex' | 'antigravity'

export const SEAT_PROVIDERS: readonly SeatProvider[] = [
  'claude',
  'codex',
  'antigravity',
]

// Vendor family behind each seat — the diversity is the whole point.
export const SEAT_VENDOR: Record<SeatProvider, string> = {
  claude: 'Anthropic',
  codex: 'OpenAI',
  antigravity: 'Google Gemini',
}

// Per-seat model hint. Blank = the CLI's own default. codex on a plain
// ChatGPT plan rejects most models — set a supported one (e.g. gpt-5.4-mini)
// or the seat fails with "model not supported".
export const SEAT_MODEL_PLACEHOLDER: Record<SeatProvider, string> = {
  claude: 'default (blank = CLI default)',
  codex: 'e.g. gpt-5.4-mini',
  antigravity: 'default (blank = CLI default)',
}

export type RoundTableStatus =
  | 'draft'
  | 'running'
  | 'awaiting_verdict'
  | 'failed'
  | 'closed'

export type Beat = 'propose' | 'critique' | 'synthesize'

export interface Seat {
  provider: SeatProvider
  model?: string
  account_id?: string
}

export interface SeatScore {
  provider: string
  blockers: number
  concerns: number
  confidence: number
}

export interface Verdict {
  recommended: string
  recommended_by: string
  alternatives: string[] | null
  tradeoffs: string[] | null
  open_questions: string[] | null
  task_breakdown: string[] | null
  ranking: SeatScore[] | null
}

export interface RoundTable {
  id: string
  topic: string
  cwd?: string
  seats: Seat[]
  status: RoundTableStatus
  verdict?: Verdict | null
  resulting_session_id?: string
  error?: string
  origin: string
  integration_id?: string
  created_at: string
  updated_at: string
}

export interface Turn {
  id: string
  round_table_id: string
  beat: Beat
  seat_provider?: string
  seat_model?: string
  role: 'seat' | 'chair' | 'system'
  content: string
  structured?: unknown
  created_at: string
}

export interface CreateRoundTableInput {
  topic: string
  cwd?: string
  seats: Seat[]
}

// GET list — unwraps the { round_tables } envelope, defaults to [].
export async function listRoundTables(cwd?: string): Promise<RoundTable[]> {
  const q = cwd ? `?cwd=${encodeURIComponent(cwd)}` : ''
  const res = await api<{ round_tables: RoundTable[] }>(
    `/api/v1/round-tables${q}`,
  )
  return res.round_tables ?? []
}

// GET one — returns the table plus its full discussion thread.
export async function getRoundTable(
  id: string,
): Promise<{ round_table: RoundTable; turns: Turn[] }> {
  return api<{ round_table: RoundTable; turns: Turn[] }>(
    `/api/v1/round-tables/${id}`,
  )
}

// POST create — opens a draft table.
export async function createRoundTable(
  input: CreateRoundTableInput,
): Promise<RoundTable> {
  return api<RoundTable>('/api/v1/round-tables', {
    method: 'POST',
    body: input,
  })
}

// POST start — kicks off the discussion (runs async; poll GET while running).
export async function startRoundTable(id: string): Promise<RoundTable> {
  return api<RoundTable>(`/api/v1/round-tables/${id}/start`, { method: 'POST' })
}

// POST close.
export async function closeRoundTable(id: string): Promise<void> {
  await api<void>(`/api/v1/round-tables/${id}/close`, { method: 'POST' })
}
