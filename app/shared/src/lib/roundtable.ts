// Round Table API client (experimental) — a cross-vendor AI GROUP CHAT.
// Members are the seated providers (claude/codex/antigravity) plus the
// operator. The operator @mentions the members who should reply; each
// mentioned member reads the whole thread and answers in character.
// Wraps the /api/v1/round-tables/* endpoints via api<T>().
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

export type RoundTableStatus = 'active' | 'closed'

export type MessageRole = 'operator' | 'seat' | 'system'
export type MessageKind = 'message' | 'summary'

export interface Seat {
  provider: SeatProvider
  model?: string
  account_id?: string
}

export interface RoundTable {
  id: string
  topic: string
  cwd?: string
  seats: Seat[]
  status: RoundTableStatus
  resulting_session_id?: string
  origin: string
  integration_id?: string
  created_at: string
  updated_at: string
}

export interface Message {
  id: string
  round_table_id: string
  role: MessageRole
  seat_provider?: string
  seat_model?: string
  kind: MessageKind
  content: string
  mentions?: string[]
  created_at: string
}

export interface CreateRoundTableInput {
  // Optional — when omitted the chat auto-names itself from the first message.
  topic?: string
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

// GET one — the table plus its full chat thread.
export async function getRoundTable(
  id: string,
): Promise<{ round_table: RoundTable; messages: Message[] }> {
  return api<{ round_table: RoundTable; messages: Message[] }>(
    `/api/v1/round-tables/${id}`,
  )
}

// POST create — opens an active group chat.
export async function createRoundTable(
  input: CreateRoundTableInput,
): Promise<RoundTable> {
  return api<RoundTable>('/api/v1/round-tables', {
    method: 'POST',
    body: input,
  })
}

// POST a message. @mentioned members (@claude/@codex/@antigravity/@all)
// reply asynchronously — poll GET while replies land.
export async function postMessage(id: string, content: string): Promise<Message> {
  return api<Message>(`/api/v1/round-tables/${id}/messages`, {
    method: 'POST',
    body: { content },
  })
}

// POST summarize — a member condenses the chat into a plan (async).
export async function summarizeRoundTable(
  id: string,
  provider?: string,
): Promise<void> {
  await api<void>(`/api/v1/round-tables/${id}/summarize`, {
    method: 'POST',
    body: { provider: provider ?? '' },
  })
}

// POST close.
export async function closeRoundTable(id: string): Promise<void> {
  await api<void>(`/api/v1/round-tables/${id}/close`, { method: 'POST' })
}
