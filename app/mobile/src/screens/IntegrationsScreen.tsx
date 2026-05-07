import { useCallback, useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import {
  type IntegrationSummary,
  listIntegrations,
  MobileAPIError,
} from '../lib/api'

interface Props {
  serverURL: string
  token: string
  onBack: () => void
  onAuthExpired: () => void
}

type State =
  | { kind: 'loading' }
  | { kind: 'ready'; integrations: IntegrationSummary[] }
  | { kind: 'error'; message: string }

// Integrations viewer (read-only on mobile). Registration / key
// rotation / scope edits happen on desktop where a 16-character API
// key is easier to copy.
export function IntegrationsScreen({
  serverURL,
  token,
  onBack,
  onAuthExpired,
}: Props) {
  const [state, setState] = useState<State>({ kind: 'loading' })

  const refresh = useCallback(async () => {
    setState({ kind: 'loading' })
    try {
      const integrations = await listIntegrations(serverURL, token)
      setState({ kind: 'ready', integrations })
    } catch (err) {
      if (err instanceof MobileAPIError && err.status === 401) {
        onAuthExpired()
        return
      }
      setState({
        kind: 'error',
        message:
          err instanceof Error ? err.message : 'Failed to load integrations',
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
        <h1 className="flex-1 text-base font-semibold">Integrations</h1>
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
        {state.kind === 'ready' && state.integrations.length === 0 && (
          <div className="rounded-md border border-border bg-card p-6 text-center text-sm text-muted-foreground">
            No integrations registered. Register from the desktop admin.
          </div>
        )}
        {state.kind === 'ready' && state.integrations.length > 0 && (
          <ul className="space-y-2">
            {state.integrations.map((i) => (
              <li
                key={i.id}
                className="rounded-md border border-border bg-card p-3 space-y-1"
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="text-sm font-medium truncate">{i.name}</span>
                  <HealthBadge status={i.health_status} enabled={i.enabled} />
                </div>
                <div className="text-xs text-muted-foreground space-y-0.5">
                  <div className="truncate">{i.base_url}</div>
                  <div>prefix: <span className="font-mono text-foreground">{i.route_prefix}</span></div>
                  {i.version && <div>version: {i.version}</div>}
                  <div>
                    {i.scopes.length} scope{i.scopes.length === 1 ? '' : 's'}
                  </div>
                </div>
              </li>
            ))}
          </ul>
        )}
      </main>
    </div>
  )
}

function HealthBadge({
  status,
  enabled,
}: {
  status: string
  enabled: boolean
}) {
  if (!enabled) {
    return (
      <span className="text-[10px] uppercase tracking-wide rounded px-1.5 py-0.5 border border-muted-foreground/30 text-muted-foreground bg-muted/30">
        disabled
      </span>
    )
  }
  const palette =
    status === 'healthy'
      ? 'border-[var(--state-running)]/40 text-[var(--state-running)] bg-[var(--state-running)]/10'
      : status === 'degraded'
        ? 'border-[var(--state-idle)]/40 text-[var(--state-idle)] bg-[var(--state-idle)]/10'
        : 'border-destructive/40 text-destructive bg-destructive/10'
  return (
    <span
      className={`text-[10px] uppercase tracking-wide rounded px-1.5 py-0.5 border ${palette}`}
    >
      {status}
    </span>
  )
}
