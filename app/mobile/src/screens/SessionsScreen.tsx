import { useCallback, useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { listSessions, MobileAPIError, type SessionSummary } from '../lib/api'

interface Props {
  serverURL: string
  token: string
  username: string
  onLogout: () => void
  // Triggered when the request returns 401 — the token is no longer
  // accepted (revoked, rotated, or expired beyond local clock skew).
  // The host should clear auth and bounce back to LoginScreen.
  onAuthExpired: () => void
}

type FetchState =
  | { kind: 'loading' }
  | { kind: 'ready'; sessions: SessionSummary[] }
  | { kind: 'error'; message: string }

// B5 first real screen. Lists sessions via /api/v1/sessions and
// renders a card per session. Mobile-first layout: single column,
// touch-friendly tap targets, refresh button in the header (a real
// pull-to-refresh gesture lands later — needs @capacitor/keyboard
// or a third-party plugin).
//
// B6 will add per-session navigation (tap a card → terminal view).
// Today the cards are read-only.
export function SessionsScreen({
  serverURL,
  token,
  username,
  onLogout,
  onAuthExpired,
}: Props) {
  const [state, setState] = useState<FetchState>({ kind: 'loading' })

  const refresh = useCallback(async () => {
    setState({ kind: 'loading' })
    try {
      const sessions = await listSessions(serverURL, token)
      setState({ kind: 'ready', sessions })
    } catch (err) {
      if (err instanceof MobileAPIError && err.status === 401) {
        onAuthExpired()
        return
      }
      setState({
        kind: 'error',
        message: err instanceof Error ? err.message : 'Failed to load sessions',
      })
    }
  }, [serverURL, token, onAuthExpired])

  useEffect(() => {
    void refresh()
  }, [refresh])

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col">
      <header className="sticky top-0 z-10 bg-background/95 backdrop-blur border-b border-border px-4 py-3 flex items-center justify-between">
        <div className="flex flex-col">
          <h1 className="text-lg font-semibold leading-tight">Sessions</h1>
          <span className="text-xs text-muted-foreground">{username}</span>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={refresh}>
            Refresh
          </Button>
          <Button variant="outline" size="sm" onClick={onLogout}>
            Sign out
          </Button>
        </div>
      </header>
      <main className="flex-1 px-4 py-4">
        <SessionsBody state={state} onRetry={refresh} />
      </main>
    </div>
  )
}

interface BodyProps {
  state: FetchState
  onRetry: () => void
}

function SessionsBody({ state, onRetry }: BodyProps) {
  if (state.kind === 'loading') {
    return (
      <div className="text-sm text-muted-foreground">Loading sessions…</div>
    )
  }
  if (state.kind === 'error') {
    return (
      <div className="space-y-3">
        <div className="rounded-md border border-destructive/30 bg-destructive/10 text-destructive text-sm px-3 py-2">
          {state.message}
        </div>
        <Button variant="outline" onClick={onRetry}>
          Retry
        </Button>
      </div>
    )
  }
  if (state.sessions.length === 0) {
    return (
      <div className="rounded-md border border-border bg-card p-6 text-center text-sm text-muted-foreground">
        No sessions yet. Spawn one from the desktop admin to see it here.
      </div>
    )
  }
  return (
    <ul className="space-y-2">
      {state.sessions.map((s) => (
        <SessionCard key={s.id} session={s} />
      ))}
    </ul>
  )
}

function SessionCard({ session }: { session: SessionSummary }) {
  const label = session.name ?? session.id
  return (
    <li className="rounded-md border border-border bg-card text-card-foreground p-3 space-y-1">
      <div className="flex items-center justify-between gap-2">
        <span className="font-medium text-sm truncate">{label}</span>
        <StateBadge state={session.state} />
      </div>
      <div className="text-xs text-muted-foreground space-y-0.5">
        <div className="truncate">{session.provider_id}</div>
        <div className="truncate">{session.cwd}</div>
        <div>started {formatRelative(session.started_at)}</div>
      </div>
    </li>
  )
}

function StateBadge({ state }: { state: SessionSummary['state'] }) {
  const styles = STATE_STYLES[state]
  return (
    <span
      className={`text-[10px] uppercase tracking-wide rounded px-1.5 py-0.5 border ${styles}`}
    >
      {state}
    </span>
  )
}

const STATE_STYLES: Record<SessionSummary['state'], string> = {
  pending: 'border-muted-foreground/30 text-muted-foreground bg-muted/30',
  running: 'border-[var(--state-running)]/40 text-[var(--state-running)] bg-[var(--state-running)]/10',
  idle: 'border-[var(--state-idle)]/40 text-[var(--state-idle)] bg-[var(--state-idle)]/10',
  stopped: 'border-[var(--state-ended)]/40 text-[var(--state-ended)] bg-[var(--state-ended)]/10',
  ended: 'border-[var(--state-ended)]/40 text-[var(--state-ended)] bg-[var(--state-ended)]/10',
}

function formatRelative(iso: string): string {
  const ts = Date.parse(iso)
  if (Number.isNaN(ts)) return iso
  const seconds = Math.round((Date.now() - ts) / 1000)
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.round(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.round(hours / 24)
  return `${days}d ago`
}
