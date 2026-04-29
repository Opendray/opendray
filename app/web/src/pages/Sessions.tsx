import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Layers, Plus, Power, Loader2 } from 'lucide-react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { SessionList } from '@/components/sessions/SessionList'
import { SessionTabs } from '@/components/sessions/SessionTabs'
import { Terminal } from '@/components/sessions/Terminal'
import { SpawnDialog } from '@/components/sessions/SpawnDialog'
import { StatePill } from '@/components/sessions/StatePill'
import { listSessions, terminateSession } from '@/lib/sessions'
import { useSessionTabs } from '@/stores/sessionTabs'
import type { Session } from '@/lib/types'

export function SessionsPage() {
  const [spawnOpen, setSpawnOpen] = useState(false)
  const tabs = useSessionTabs((s) => s.tabs)
  const currentId = useSessionTabs((s) => s.currentId)
  const open = useSessionTabs((s) => s.open)
  const close = useSessionTabs((s) => s.close)
  const setCurrent = useSessionTabs((s) => s.setCurrent)

  const qc = useQueryClient()
  const { data: sessions } = useQuery({
    queryKey: ['sessions'],
    queryFn: listSessions,
    refetchInterval: 4_000,
  })

  const currentSession = sessions?.find((s) => s.id === currentId)

  const terminate = useMutation({
    mutationFn: terminateSession,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['sessions'] })
      toast.success('Session terminated')
    },
    onError: (err: Error) =>
      toast.error('Terminate failed', { description: err.message }),
  })

  // Reconcile: if a tab's session is gone from server, drop it.
  useEffect(() => {
    if (!sessions) return
    const live = new Set(sessions.map((s) => s.id))
    for (const t of tabs) {
      if (!live.has(t.id)) {
        close(t.id)
      }
    }
  }, [sessions, tabs, close])

  // Keyboard shortcuts: ⌘N spawn, ⌘W close current, ⌘1..9 switch.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const meta = e.metaKey || e.ctrlKey
      if (!meta) return
      if (e.key === 'n' || e.key === 'N') {
        e.preventDefault()
        setSpawnOpen(true)
        return
      }
      if (e.key === 'w' || e.key === 'W') {
        if (currentId) {
          e.preventDefault()
          close(currentId)
        }
        return
      }
      if (/^[1-9]$/.test(e.key)) {
        const idx = parseInt(e.key, 10) - 1
        const tab = tabs[idx]
        if (tab) {
          e.preventDefault()
          setCurrent(tab.id)
        }
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [tabs, currentId, close, setCurrent])

  const handleOpen = (s: Session) => {
    open({ id: s.id, name: s.name || s.provider_id })
  }

  return (
    <div className="h-full flex">
      <SessionList onSpawn={() => setSpawnOpen(true)} onOpen={handleOpen} />

      <div className="flex-1 flex flex-col min-w-0">
        <SessionTabs />

        {!currentId ? (
          <EmptyWorkbench onSpawn={() => setSpawnOpen(true)} />
        ) : (
          <>
            <WorkbenchHeader
              session={currentSession}
              onTerminate={() => currentId && terminate.mutate(currentId)}
              terminating={terminate.isPending}
            />
            <div className="flex-1 min-h-0">
              <Terminal key={currentId} sessionId={currentId} />
            </div>
          </>
        )}
      </div>

      <SpawnDialog
        open={spawnOpen}
        onOpenChange={setSpawnOpen}
        onSpawned={(s) => open({ id: s.id, name: s.name || s.provider_id })}
      />
    </div>
  )
}

function EmptyWorkbench({ onSpawn }: { onSpawn: () => void }) {
  return (
    <div className="flex-1 flex flex-col items-center justify-center gap-4 text-center p-6">
      <Layers className="size-10 text-muted-foreground/40" strokeWidth={1.5} />
      <div className="space-y-1">
        <h2 className="text-[14px] font-semibold">No session open</h2>
        <p className="text-[12px] text-muted-foreground max-w-[320px]">
          Pick a session from the list, or spawn a new one. Keyboard:{' '}
          <kbd>⌘N</kbd> new, <kbd>⌘W</kbd> close,{' '}
          <kbd>⌘1</kbd>–<kbd>⌘9</kbd> switch.
        </p>
      </div>
      <Button onClick={onSpawn} variant="accent" size="sm">
        <Plus className="size-3.5" /> Spawn session
      </Button>
    </div>
  )
}

function WorkbenchHeader({
  session,
  onTerminate,
  terminating,
}: {
  session?: Session
  onTerminate: () => void
  terminating: boolean
}) {
  if (!session) {
    return (
      <div className="h-9 border-b border-border flex items-center px-3 text-[12px] text-muted-foreground">
        <Loader2 className="size-3 animate-spin" />
        <span className="ml-2">Loading session…</span>
      </div>
    )
  }
  return (
    <div className="h-9 border-b border-border flex items-center px-3 gap-3">
      <span className="text-[12px] font-medium truncate flex-1">
        {session.name || session.provider_id}
        <span className="text-muted-foreground/70 font-mono ml-2 text-[11px]">
          {session.cwd}
        </span>
      </span>
      <StatePill state={session.state} exitCode={session.exit_code} />
      {session.pid != null && (
        <span className="text-[10px] text-muted-foreground font-mono">
          pid {session.pid}
        </span>
      )}
      {session.state !== 'ended' && (
        <Button
          variant="ghost"
          size="sm"
          onClick={onTerminate}
          disabled={terminating}
          className="text-[11px] gap-1 text-muted-foreground hover:text-destructive"
        >
          <Power className="size-3" />
          {terminating ? 'Terminating…' : 'Terminate'}
        </Button>
      )}
    </div>
  )
}
