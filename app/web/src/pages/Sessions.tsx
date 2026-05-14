import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Layers,
  Plus,
  Power,
  RotateCcw,
  Trash2,
  Loader2,
  PanelLeftClose,
  PanelLeftOpen,
  PanelRightClose,
  PanelRightOpen,
  Keyboard,
} from 'lucide-react'
import { toast } from 'sonner'
import { Trans, useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { useConfirmDialog } from '@/components/ConfirmDialog'
import { SessionList } from '@/components/sessions/SessionList'
import { SessionTabs } from '@/components/sessions/SessionTabs'
import {
  Terminal,
  type TerminalHandle,
} from '@/components/sessions/Terminal'
import { TerminalToolbar } from '@/components/sessions/TerminalToolbar'
import { EndedSessionView } from '@/components/sessions/EndedSessionView'
import { SpawnDialog } from '@/components/sessions/SpawnDialog'
import { StatePill } from '@/components/sessions/StatePill'
import { AccountSwitcher } from '@/components/sessions/AccountSwitcher'
import { InspectorPanel } from '@/components/sessions/InspectorPanel'
import { providerVisual, cwdTail } from '@/lib/providers'
import { providerIconKey } from '@/lib/providerIcons'
import { BrandAvatar } from '@/components/BrandAvatar'
import {
  listSessions,
  removeSession,
  startSession,
  stopSession,
} from '@/lib/sessions'
import { useSessionTabs } from '@/stores/sessionTabs'
import { useLayout } from '@/stores/layout'
import { cn } from '@/lib/utils'
import { isTerminalSessionState, type Session } from '@/lib/types'

export function SessionsPage() {
  const { t } = useTranslation()
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

  const remove = useMutation({
    mutationFn: removeSession,
    onSuccess: (_, id) => {
      qc.invalidateQueries({ queryKey: ['sessions'] })
      close(id)
      toast.success(t('web.sessions.page.removedToast'))
    },
    onError: (err: Error) =>
      toast.error(t('web.sessions.page.removeFailedToast'), {
        description: err.message,
      }),
  })

  const stop = useMutation({
    mutationFn: stopSession,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['sessions'] })
      toast.success(t('web.sessions.page.stoppedToast'))
    },
    onError: (err: Error) =>
      toast.error(t('web.sessions.page.stopFailedToast'), {
        description: err.message,
      }),
  })

  const start = useMutation({
    mutationFn: startSession,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['sessions'] })
      toast.success(t('web.sessions.page.restartedToast'))
    },
    onError: (err: Error) =>
      toast.error(t('web.sessions.page.restartFailedToast'), {
        description: err.message,
      }),
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
          handleCloseTab(currentId)
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
    // handleCloseTab depends on `sessions` (refetched every 4s) so the
    // listener rebinds at the same cadence — cheap and keeps the
    // closure's `sessions` view current.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tabs, currentId, setCurrent, sessions])

  const handleOpen = (s: Session) => {
    open({ id: s.id, name: s.name || s.provider_id })
  }

  const { confirm: confirmDialog, dialog: confirmDialogEl } = useConfirmDialog()

  // Tab ✕ = full destroy: terminate the CLI process if still running,
  // then drop the DB row. Confirms for live sessions so users don't
  // kill work by accident; ended/stopped rows go straight through.
  const handleCloseTab = async (id: string) => {
    const target = sessions?.find((s) => s.id === id)
    if (target && !isTerminalSessionState(target.state)) {
      const ok = await confirmDialog({
        title: t('web.sessions.page.confirmCloseTabTitle', {
          name: target.name || target.provider_id,
        }),
        description: t('web.sessions.page.confirmCloseTabDescription'),
        confirmLabel: t('web.sessions.page.confirmCloseTabConfirm'),
        destructive: true,
      })
      if (!ok) return
    }
    remove.mutate(id)
  }

  const listCollapsed = useLayout((s) => s.sessionListCollapsed)
  const toggleList = useLayout((s) => s.toggleSessionList)
  const toolbarOpen = useLayout((s) => s.terminalToolbarOpen)
  const toggleToolbar = useLayout((s) => s.toggleTerminalToolbar)
  const inspectorOpen = useLayout((s) => s.inspectorOpen)
  const toggleInspector = useLayout((s) => s.toggleInspector)

  const termRef = useRef<TerminalHandle>(null)

  // sessionListCollapsed is persisted in localStorage. The toggle
  // button lives inside WorkbenchHeader, which only renders when a
  // session is selected. If a user collapses the list and then
  // closes/ends every session, they end up locked out of the list
  // forever (EmptyWorkbench has no toggle). Override: always show
  // the list when there's nothing currently selected, so empty
  // state stays navigable.
  const showList = !listCollapsed || !currentId

  return (
    <div className="h-full flex">
      {showList && (
        <SessionList onSpawn={() => setSpawnOpen(true)} onOpen={handleOpen} />
      )}

      <div className="flex-1 flex flex-col min-w-0">
        <SessionTabs onCloseTab={handleCloseTab} />

        {!currentId ? (
          <EmptyWorkbench onSpawn={() => setSpawnOpen(true)} />
        ) : (
          <>
            <WorkbenchHeader
              session={currentSession}
              onStop={() => currentId && stop.mutate(currentId)}
              onStart={() => currentId && start.mutate(currentId)}
              onRemove={async () => {
                if (!currentId) return
                const ok = await confirmDialog({
                  title: currentSession?.name || currentSession?.provider_id
                    ? t('web.sessions.page.confirmRemoveTitle', {
                        name:
                          currentSession?.name ||
                          currentSession?.provider_id ||
                          '',
                      })
                    : t('web.sessions.page.confirmRemoveTitleFallback'),
                  description: t('web.sessions.page.confirmRemoveDescription'),
                  confirmLabel: t('web.sessions.page.confirmRemoveConfirm'),
                  destructive: true,
                })
                if (!ok) return
                remove.mutate(currentId)
              }}
              stopping={stop.isPending}
              starting={start.isPending}
              removing={remove.isPending}
              listCollapsed={listCollapsed}
              onToggleList={toggleList}
              toolbarOpen={toolbarOpen}
              onToggleToolbar={toggleToolbar}
              inspectorOpen={inspectorOpen}
              onToggleInspector={toggleInspector}
            />
            <div className="flex-1 min-h-0">
              {currentSession &&
              isTerminalSessionState(currentSession.state) ? (
                <EndedSessionView key={currentId} sessionId={currentId} />
              ) : (
                <Terminal
                  ref={termRef}
                  // pid in the key forces a remount when the underlying
                  // child process is replaced (e.g. account switch or
                  // restart) — the prior WS subscribed to a now-dead
                  // pump goroutine, so we must reconnect from scratch.
                  key={`${currentId}:${currentSession?.pid ?? 0}`}
                  sessionId={currentId}
                />
              )}
            </div>
            {toolbarOpen &&
              currentSession &&
              !isTerminalSessionState(currentSession.state) && (
              <TerminalToolbar
                onKey={(seq) => termRef.current?.sendInput(seq)}
              />
            )}
          </>
        )}
      </div>

      {inspectorOpen && currentSession && (
        <InspectorPanel session={currentSession} />
      )}

      <SpawnDialog
        open={spawnOpen}
        onOpenChange={setSpawnOpen}
        onSpawned={(s) => open({ id: s.id, name: s.name || s.provider_id })}
      />
      {confirmDialogEl}
    </div>
  )
}

function EmptyWorkbench({ onSpawn }: { onSpawn: () => void }) {
  const { t } = useTranslation()
  return (
    <div className="flex-1 flex flex-col items-center justify-center gap-4 text-center p-6">
      <Layers className="size-10 text-muted-foreground/40" strokeWidth={1.5} />
      <div className="space-y-1">
        <h2 className="text-[14px] font-semibold">
          {t('web.sessions.empty.title')}
        </h2>
        <p className="text-[12px] text-muted-foreground max-w-[320px]">
          <Trans
            i18nKey="web.sessions.empty.hint"
            values={{
              kbdN: '⌘N',
              kbdW: '⌘W',
              kbdRange: '⌘1–⌘9',
            }}
            components={{ 0: <kbd />, 1: <kbd />, 2: <kbd /> }}
          />
        </p>
      </div>
      <Button onClick={onSpawn} variant="accent" size="sm">
        <Plus className="size-3.5" /> {t('web.sessions.empty.spawn')}
      </Button>
    </div>
  )
}

function WorkbenchHeader({
  session,
  onStop,
  onStart,
  onRemove,
  stopping,
  starting,
  removing,
  listCollapsed,
  onToggleList,
  toolbarOpen,
  onToggleToolbar,
  inspectorOpen,
  onToggleInspector,
}: {
  session?: Session
  onStop: () => void
  onStart: () => void
  onRemove: () => void
  stopping: boolean
  starting: boolean
  removing: boolean
  listCollapsed: boolean
  onToggleList: () => void
  toolbarOpen: boolean
  onToggleToolbar: () => void
  inspectorOpen: boolean
  onToggleInspector: () => void
}) {
  const { t } = useTranslation()
  if (!session) {
    return (
      <div className="h-14 border-b border-border flex items-center px-3 text-[12px] text-muted-foreground">
        <Loader2 className="size-3 animate-spin" />
        <span className="ml-2">{t('web.sessions.header.loadingSession')}</span>
      </div>
    )
  }
  const visual = providerVisual(session.provider_id)
  const listLabel = listCollapsed
    ? t('web.sessions.header.showList')
    : t('web.sessions.header.hideList')
  const inspectorLabel = inspectorOpen
    ? t('web.sessions.header.hideInspector')
    : t('web.sessions.header.showInspector')
  const keyboardAria = toolbarOpen
    ? t('web.sessions.header.hideKeyboard')
    : t('web.sessions.header.showKeyboard')
  const keyboardTooltip = toolbarOpen
    ? t('web.sessions.header.hideKeyboard')
    : t('web.sessions.header.showKeyboardTooltip')
  return (
    <div className="h-14 border-b border-border flex items-center px-3 gap-3">
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            onClick={onToggleList}
            aria-label={listLabel}
            className="size-7 shrink-0"
          >
            {listCollapsed ? (
              <PanelLeftOpen className="size-3.5" />
            ) : (
              <PanelLeftClose className="size-3.5" />
            )}
          </Button>
        </TooltipTrigger>
        <TooltipContent>{listLabel}</TooltipContent>
      </Tooltip>
      <BrandAvatar
        iconKey={providerIconKey(session.provider_id)}
        fallbackLetter={visual.letter}
        size={36}
        title={visual.name}
      />
      <div className="flex-1 min-w-0 flex flex-col gap-0.5">
        <div className="text-[14px] font-semibold leading-tight truncate text-foreground">
          {session.name || cwdTail(session.cwd)}
        </div>
        <div className="text-[11px] text-muted-foreground/80 font-mono truncate">
          {visual.name} · {session.cwd}
          {session.pid != null && (
            <span className="ml-2 text-muted-foreground/60">
              {t('web.sessions.header.pid', { pid: session.pid })}
            </span>
          )}
        </div>
      </div>
      <StatePill state={session.state} exitCode={session.exit_code} />
      {session.provider_id === 'claude' &&
        !isTerminalSessionState(session.state) && (
          <AccountSwitcher session={session} />
        )}
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            onClick={onToggleToolbar}
            aria-label={keyboardAria}
            className={cn('size-7 shrink-0', toolbarOpen && 'text-foreground')}
          >
            <Keyboard className="size-3.5" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{keyboardTooltip}</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            onClick={onToggleInspector}
            aria-label={inspectorLabel}
            className={cn(
              'size-7 shrink-0',
              inspectorOpen && 'text-foreground',
            )}
          >
            {inspectorOpen ? (
              <PanelRightClose className="size-3.5" />
            ) : (
              <PanelRightOpen className="size-3.5" />
            )}
          </Button>
        </TooltipTrigger>
        <TooltipContent>{inspectorLabel}</TooltipContent>
      </Tooltip>
      {isTerminalSessionState(session.state) ? (
        <>
          <Button
            variant="ghost"
            size="sm"
            onClick={onStart}
            disabled={starting}
            className="text-[11px] gap-1 hover:text-foreground"
          >
            {starting ? (
              <Loader2 className="size-3 animate-spin" />
            ) : (
              <RotateCcw className="size-3" />
            )}
            {starting
              ? t('web.sessions.header.restarting')
              : t('web.sessions.header.restart')}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={onRemove}
            disabled={removing}
            className="text-[11px] gap-1 text-muted-foreground hover:text-destructive"
          >
            <Trash2 className="size-3" />
            {removing
              ? t('web.sessions.header.removing')
              : t('web.sessions.header.remove')}
          </Button>
        </>
      ) : (
        <>
          <Button
            variant="ghost"
            size="sm"
            onClick={onStop}
            disabled={stopping}
            className="text-[11px] gap-1 text-muted-foreground hover:text-destructive"
          >
            <Power className="size-3" />
            {stopping
              ? t('web.sessions.header.stopping')
              : t('web.sessions.header.stop')}
          </Button>
        </>
      )}
    </div>
  )
}
