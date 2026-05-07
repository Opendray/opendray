import { useCallback, useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { type BackupSummary, listBackups, MobileAPIError } from '../lib/api'

interface Props {
  serverURL: string
  token: string
  onBack: () => void
  onAuthExpired: () => void
}

type State =
  | { kind: 'loading' }
  | { kind: 'ready'; backups: BackupSummary[] }
  | { kind: 'error'; message: string }

// Backup history viewer (read-only on mobile). Backup targets,
// schedules, restore flow all live on desktop — mobile here is
// for "did the 03:00 schedule succeed last night?" glances.
export function BackupsScreen({ serverURL, token, onBack, onAuthExpired }: Props) {
  const [state, setState] = useState<State>({ kind: 'loading' })

  const refresh = useCallback(async () => {
    setState({ kind: 'loading' })
    try {
      const backups = await listBackups(serverURL, token)
      // Most recent first. Server returns reverse-chronological
      // already in practice, but make it explicit so a server
      // change doesn't silently regress UX.
      backups.sort((a, b) => Date.parse(b.started_at) - Date.parse(a.started_at))
      setState({ kind: 'ready', backups })
    } catch (err) {
      if (err instanceof MobileAPIError && err.status === 401) {
        onAuthExpired()
        return
      }
      setState({
        kind: 'error',
        message: err instanceof Error ? err.message : 'Failed to load backups',
      })
    }
  }, [serverURL, token, onAuthExpired])

  useEffect(() => {
    void refresh()
  }, [refresh])

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col">
      <header className="sticky top-0 z-10 bg-background/95 backdrop-blur border-b border-border px-3 py-2 flex items-center gap-2">
        <Button variant="ghost" size="sm" onClick={onBack}>
          ← Back
        </Button>
        <h1 className="flex-1 text-base font-semibold">Backups</h1>
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
        {state.kind === 'ready' && state.backups.length === 0 && (
          <div className="rounded-md border border-border bg-card p-6 text-center text-sm text-muted-foreground">
            No backups yet. Backup configuration lives on the desktop admin.
          </div>
        )}
        {state.kind === 'ready' && state.backups.length > 0 && (
          <ul className="space-y-2">
            {state.backups.map((b) => (
              <BackupCard key={b.id} backup={b} />
            ))}
          </ul>
        )}
      </main>
    </div>
  )
}

function BackupCard({ backup }: { backup: BackupSummary }) {
  return (
    <li className="rounded-md border border-border bg-card p-3 space-y-1">
      <div className="flex items-center justify-between gap-2">
        <span className="text-sm font-mono truncate">{backup.id}</span>
        <StatusBadge status={backup.status} />
      </div>
      <div className="text-xs text-muted-foreground space-y-0.5">
        <div>target: {backup.target_id}</div>
        <div>
          {humanSize(backup.bytes)}
          {backup.encrypted ? ' · encrypted' : ''}
        </div>
        <div>
          {backup.triggered_by} · {formatRelative(backup.started_at)}
          {backup.finished_at && ` · ${duration(backup.started_at, backup.finished_at)}`}
        </div>
      </div>
    </li>
  )
}

function StatusBadge({ status }: { status: string }) {
  const palette =
    status === 'succeeded'
      ? 'border-[var(--state-running)]/40 text-[var(--state-running)] bg-[var(--state-running)]/10'
      : status === 'running' || status === 'pending'
        ? 'border-[var(--state-idle)]/40 text-[var(--state-idle)] bg-[var(--state-idle)]/10'
        : status === 'failed'
          ? 'border-destructive/40 text-destructive bg-destructive/10'
          : 'border-muted-foreground/30 text-muted-foreground bg-muted/30'
  return (
    <span
      className={`text-[10px] uppercase tracking-wide rounded px-1.5 py-0.5 border ${palette}`}
    >
      {status}
    </span>
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

function duration(startIso: string, endIso: string): string {
  const ms = Date.parse(endIso) - Date.parse(startIso)
  if (!Number.isFinite(ms) || ms < 0) return ''
  const seconds = Math.round(ms / 1000)
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes}m`
  return `${Math.round(minutes / 60)}h`
}

function humanSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`
  return `${(bytes / 1024 / 1024 / 1024).toFixed(1)} GB`
}
