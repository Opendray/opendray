import { useCallback, useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import {
  type ProviderSummary,
  listProviders,
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
  | { kind: 'ready'; providers: ProviderSummary[] }
  | { kind: 'error'; message: string }

// CLI providers viewer (read-only on mobile). Configuration forms
// (api keys, model selection) live on desktop.
export function ProvidersScreen({
  serverURL,
  token,
  onBack,
  onAuthExpired,
}: Props) {
  const [state, setState] = useState<State>({ kind: 'loading' })

  const refresh = useCallback(async () => {
    setState({ kind: 'loading' })
    try {
      const providers = await listProviders(serverURL, token)
      setState({ kind: 'ready', providers })
    } catch (err) {
      if (err instanceof MobileAPIError && err.status === 401) {
        onAuthExpired()
        return
      }
      setState({
        kind: 'error',
        message: err instanceof Error ? err.message : 'Failed to load providers',
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
        <h1 className="flex-1 text-base font-semibold">CLI Providers</h1>
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
        {state.kind === 'ready' && state.providers.length === 0 && (
          <div className="rounded-md border border-border bg-card p-6 text-center text-sm text-muted-foreground">
            No providers configured.
          </div>
        )}
        {state.kind === 'ready' && state.providers.length > 0 && (
          <ul className="space-y-2">
            {state.providers.map((p) => (
              <li
                key={p.id}
                className="rounded-md border border-border bg-card p-3 space-y-1"
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="text-sm font-medium truncate">{p.name}</span>
                  <span
                    className={`text-[10px] uppercase tracking-wide rounded px-1.5 py-0.5 border ${
                      p.enabled
                        ? 'border-[var(--state-running)]/40 text-[var(--state-running)] bg-[var(--state-running)]/10'
                        : 'border-muted-foreground/30 text-muted-foreground bg-muted/30'
                    }`}
                  >
                    {p.enabled ? 'enabled' : 'disabled'}
                  </span>
                </div>
                <div className="text-xs text-muted-foreground space-y-0.5">
                  <div className="font-mono truncate">{p.id}</div>
                  <div className="font-mono text-[10px] truncate text-muted-foreground/70">
                    manifest {p.manifest_hash.slice(0, 12)}…
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
