import { useCallback, useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { type ChannelSummary, listChannels, MobileAPIError } from '../lib/api'

interface Props {
  serverURL: string
  token: string
  onBack: () => void
  onAuthExpired: () => void
}

type State =
  | { kind: 'loading' }
  | { kind: 'ready'; channels: ChannelSummary[] }
  | { kind: 'error'; message: string }

// Channels viewer (read-only on mobile). Lists every configured
// channel adapter with running / enabled / muted state. Editing
// channel config (bot tokens, webhooks, etc.) is desktop territory.
export function ChannelsScreen({ serverURL, token, onBack, onAuthExpired }: Props) {
  const [state, setState] = useState<State>({ kind: 'loading' })

  const refresh = useCallback(async () => {
    setState({ kind: 'loading' })
    try {
      const channels = await listChannels(serverURL, token)
      setState({ kind: 'ready', channels })
    } catch (err) {
      if (err instanceof MobileAPIError && err.status === 401) {
        onAuthExpired()
        return
      }
      setState({
        kind: 'error',
        message: err instanceof Error ? err.message : 'Failed to load channels',
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
        <h1 className="flex-1 text-base font-semibold">Channels</h1>
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
        {state.kind === 'ready' && state.channels.length === 0 && (
          <div className="rounded-md border border-border bg-card p-6 text-center text-sm text-muted-foreground">
            No channels configured. Configure adapters from the desktop admin.
          </div>
        )}
        {state.kind === 'ready' && state.channels.length > 0 && (
          <ul className="space-y-2">
            {state.channels.map((c) => (
              <li
                key={c.id}
                className="rounded-md border border-border bg-card p-3 space-y-1"
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="text-sm font-medium truncate">{c.id}</span>
                  <ChannelStateBadge channel={c} />
                </div>
                <div className="text-xs text-muted-foreground">{c.kind}</div>
                {c.capabilities && c.capabilities.length > 0 && (
                  <div className="flex gap-1 flex-wrap">
                    {c.capabilities.map((cap) => (
                      <span
                        key={cap}
                        className="text-[10px] uppercase tracking-wide rounded px-1.5 py-0.5 border border-border text-muted-foreground"
                      >
                        {cap}
                      </span>
                    ))}
                  </div>
                )}
              </li>
            ))}
          </ul>
        )}
      </main>
    </div>
  )
}

function ChannelStateBadge({ channel }: { channel: ChannelSummary }) {
  if (!channel.enabled) {
    return <Badge color="border-muted-foreground/30 text-muted-foreground bg-muted/30">disabled</Badge>
  }
  if (channel.muted) {
    return <Badge color="border-[var(--state-idle)]/40 text-[var(--state-idle)] bg-[var(--state-idle)]/10">muted</Badge>
  }
  if (channel.running) {
    return <Badge color="border-[var(--state-running)]/40 text-[var(--state-running)] bg-[var(--state-running)]/10">running</Badge>
  }
  return <Badge color="border-destructive/40 text-destructive bg-destructive/10">stopped</Badge>
}

function Badge({
  color,
  children,
}: {
  color: string
  children: React.ReactNode
}) {
  return (
    <span
      className={`text-[10px] uppercase tracking-wide rounded px-1.5 py-0.5 border ${color}`}
    >
      {children}
    </span>
  )
}
