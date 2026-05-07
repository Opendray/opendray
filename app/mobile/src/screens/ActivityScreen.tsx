import { useCallback, useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { type AuditEntry, listAudit, MobileAPIError } from '../lib/api'

interface Props {
  serverURL: string
  token: string
  onAuthExpired: () => void
}

type State =
  | { kind: 'loading' }
  | { kind: 'ready'; entries: AuditEntry[] }
  | { kind: 'error'; message: string }

// Audit log viewer. Reads /api/v1/audit (default 50 most recent),
// renders one card per entry. Filtering / search is a polish task.
export function ActivityScreen({ serverURL, token, onAuthExpired }: Props) {
  const [state, setState] = useState<State>({ kind: 'loading' })

  const refresh = useCallback(async () => {
    setState({ kind: 'loading' })
    try {
      const entries = await listAudit(serverURL, token, { limit: 100 })
      setState({ kind: 'ready', entries })
    } catch (err) {
      if (err instanceof MobileAPIError && err.status === 401) {
        onAuthExpired()
        return
      }
      setState({
        kind: 'error',
        message: err instanceof Error ? err.message : 'Failed to load activity',
      })
    }
  }, [serverURL, token, onAuthExpired])

  useEffect(() => {
    void refresh()
  }, [refresh])

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col">
      <header className="sticky top-0 z-10 bg-background/95 backdrop-blur border-b border-border px-4 py-3 flex items-center justify-between">
        <h1 className="text-lg font-semibold leading-tight">Activity</h1>
        <Button variant="outline" size="sm" onClick={refresh}>
          Refresh
        </Button>
      </header>
      <main className="flex-1 px-4 py-3">
        {state.kind === 'loading' && (
          <div className="text-sm text-muted-foreground">Loading…</div>
        )}
        {state.kind === 'error' && (
          <div className="space-y-3">
            <div className="rounded-md border border-destructive/30 bg-destructive/10 text-destructive text-sm px-3 py-2">
              {state.message}
            </div>
            <Button variant="outline" size="sm" onClick={refresh}>
              Retry
            </Button>
          </div>
        )}
        {state.kind === 'ready' && state.entries.length === 0 && (
          <div className="rounded-md border border-border bg-card p-6 text-center text-sm text-muted-foreground">
            No recent activity.
          </div>
        )}
        {state.kind === 'ready' && state.entries.length > 0 && (
          <ul className="space-y-2">
            {state.entries.map((e) => (
              <EntryCard key={e.id} entry={e} />
            ))}
          </ul>
        )}
      </main>
    </div>
  )
}

function EntryCard({ entry }: { entry: AuditEntry }) {
  const subject = entry.subject_kind
    ? `${entry.subject_kind}${entry.subject_id ? `:${entry.subject_id}` : ''}`
    : null
  const actor = entry.actor_id
    ? `${entry.actor_kind}:${entry.actor_id}`
    : entry.actor_kind
  return (
    <li className="rounded-md border border-border bg-card p-3 text-sm space-y-1">
      <div className="flex items-baseline justify-between gap-2">
        <span className="font-mono text-[11px] text-accent truncate">{entry.action}</span>
        <span className="text-[10px] text-muted-foreground tabular-nums shrink-0">
          {formatRelative(entry.ts)}
        </span>
      </div>
      <div className="text-[11px] text-muted-foreground space-y-0.5">
        <div>by {actor}</div>
        {subject && <div className="truncate">on {subject}</div>}
      </div>
    </li>
  )
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
