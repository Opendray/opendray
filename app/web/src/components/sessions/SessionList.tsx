import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus, Loader2, X, CornerDownRight } from 'lucide-react'
import { toast } from 'sonner'
import { Trans, useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { listSessions, removeSession } from '@/lib/sessions'
import { listClaudeAccounts } from '@/lib/claudeAccounts'
import { isTerminalSessionState, type Session } from '@/lib/types'
import { useSessionTabs } from '@/stores/sessionTabs'
import { providerIconKey } from '@/lib/providerIcons'
import { BrandAvatar } from '@/components/BrandAvatar'
import { providerVisual, cwdTail } from '@/lib/providers'
import { cn } from '@/lib/utils'
import { SessionRow } from './SessionRow'

interface SessionListProps {
  onSpawn: () => void
  onOpen: (session: Session) => void
}

function order(a: Session, b: Session): number {
  // Live sessions first, then by started_at desc.
  const aLive = isTerminalSessionState(a.state) ? 1 : 0
  const bLive = isTerminalSessionState(b.state) ? 1 : 0
  if (aLive !== bLive) return aLive - bLive
  return new Date(b.started_at).getTime() - new Date(a.started_at).getTime()
}

// Group sessions by parent. A parent_session_id pointing at a row no
// longer in the list (deleted parent) gets promoted to top-level.
function buildTree(sessions: Session[]) {
  const ids = new Set(sessions.map((s) => s.id))
  const childrenOf = new Map<string, Session[]>()
  const tops: Session[] = []
  for (const s of sessions) {
    const parent = s.parent_session_id
    if (parent && ids.has(parent)) {
      const list = childrenOf.get(parent) ?? []
      list.push(s)
      childrenOf.set(parent, list)
    } else {
      tops.push(s)
    }
  }
  // Children oldest-first under each parent so the freshest task run
  // sits closest to the bottom — matches reading order.
  for (const list of childrenOf.values()) {
    list.sort(
      (a, b) =>
        new Date(a.started_at).getTime() - new Date(b.started_at).getTime(),
    )
  }
  return { tops, childrenOf }
}

export function SessionList({ onSpawn, onOpen }: SessionListProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const { data: sessions, isLoading } = useQuery({
    queryKey: ['sessions'],
    queryFn: listSessions,
    refetchInterval: 4_000,
  })
  const { data: claudeAccounts } = useQuery({
    queryKey: ['claude-accounts'],
    queryFn: listClaudeAccounts,
    staleTime: 30_000,
  })
  const accountById = new Map(
    (claudeAccounts ?? []).map((a) => [a.id, a.display_name || a.name]),
  )
  const labelFor = (s: Session): string | undefined => {
    if (s.provider_id !== 'claude') return undefined
    if (!s.claude_account_id)
      return t('web.sessions.accountSwitcher.currentDefault')
    return accountById.get(s.claude_account_id) ?? s.claude_account_id
  }

  const currentId = useSessionTabs((s) => s.currentId)
  const closeTab = useSessionTabs((s) => s.close)

  const all = sessions ?? []
  const { tops, childrenOf } = buildTree(all)
  const sortedTops = tops.slice().sort(order)
  const liveTops = sortedTops.filter((s) => !isTerminalSessionState(s.state))
  const endedTops = sortedTops.filter((s) => isTerminalSessionState(s.state))
  const liveCount = all.filter((s) => !isTerminalSessionState(s.state)).length
  const endedCount = all.length - liveCount

  const remove = useMutation({
    mutationFn: (s: Session) => removeSession(s.id),
    onMutate: (s) => closeTab(s.id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['sessions'] })
    },
    onError: (err: Error) =>
      toast.error(t('web.sessions.list.deleteFailedToast'), {
        description: err.message,
      }),
  })

  const handleDelete = (s: Session) => {
    if (!isTerminalSessionState(s.state)) {
      const childCount = childrenOf.get(s.id)?.length ?? 0
      const childNote =
        childCount > 0
          ? t(
              childCount === 1
                ? 'web.sessions.list.childPromoted'
                : 'web.sessions.list.childPromotedPlural',
              { count: childCount },
            )
          : ''
      if (
        !confirm(
          t('web.sessions.list.confirmTerminate', {
            name: s.name || s.provider_id,
          }) + childNote,
        )
      ) {
        return
      }
    }
    remove.mutate(s)
  }

  const renderBranch = (s: Session) => {
    const kids = childrenOf.get(s.id) ?? []
    return (
      <div key={s.id} className="flex flex-col gap-0.5">
        <SessionRow
          session={s}
          active={s.id === currentId}
          onClick={() => onOpen(s)}
          onDelete={() => handleDelete(s)}
          accountLabel={labelFor(s)}
        />
        {kids.map((c) => (
          <ChildRow
            key={c.id}
            session={c}
            active={c.id === currentId}
            onClick={() => onOpen(c)}
            onDelete={() => handleDelete(c)}
          />
        ))}
      </div>
    )
  }

  return (
    <aside className="w-72 shrink-0 border-r border-border flex flex-col bg-background">
      <div className="h-14 px-3 flex items-center justify-between border-b border-border">
        <div className="flex items-baseline gap-1.5">
          <span className="text-[11px] font-semibold tracking-[0.12em] uppercase text-muted-foreground">
            {t('web.sessions.list.title')}
          </span>
          <span className="text-[11px] text-muted-foreground/60 font-mono">
            · {all.length}
          </span>
        </div>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              onClick={onSpawn}
              aria-label={t('web.sessions.list.newAria')}
              className="size-6"
            >
              <Plus className="size-3.5" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            {t('web.sessions.list.newTooltip')}{' '}
            <kbd className="ml-1">⌘N</kbd>
          </TooltipContent>
        </Tooltip>
      </div>

      <ScrollArea className="flex-1">
        <div className="p-1.5 flex flex-col gap-0.5">
          {isLoading && (
            <div className="flex items-center gap-2 px-2 py-3 text-[12px] text-muted-foreground">
              <Loader2 className="size-3.5 animate-spin" />
              {t('web.sessions.list.loading')}
            </div>
          )}
          {!isLoading && all.length === 0 && (
            <div className="px-3 py-6 text-center text-[12px] text-muted-foreground">
              {t('web.sessions.list.emptyTitle')}
              <br />
              <Trans
                i18nKey="web.sessions.list.emptyHint"
                values={{ kbd: '⌘N' }}
                components={{ 0: <kbd /> }}
              />
            </div>
          )}

          {liveTops.map(renderBranch)}

          {endedTops.length > 0 && (
            <div className="px-2 py-1.5 mt-1 flex items-center justify-between gap-2">
              <span className="text-[10px] uppercase tracking-wider text-muted-foreground/60">
                {t('web.sessions.list.endedHeader', { count: endedCount })}
              </span>
              {endedCount > 1 && (
                <button
                  type="button"
                  onClick={() => {
                    if (
                      !confirm(
                        t('web.sessions.list.confirmClearAll', {
                          count: endedCount,
                        }),
                      )
                    )
                      return
                    // Remove children first to keep parent_session_id
                    // FK happy, then parents.
                    const endedAll = all.filter((s) =>
                      isTerminalSessionState(s.state),
                    )
                    const kids = endedAll.filter((s) => s.parent_session_id)
                    const parents = endedAll.filter((s) => !s.parent_session_id)
                    kids.forEach((s) => remove.mutate(s))
                    parents.forEach((s) => remove.mutate(s))
                  }}
                  className="text-[10px] text-muted-foreground/70 hover:text-destructive transition-colors"
                >
                  {t('web.sessions.list.clearAll')}
                </button>
              )}
            </div>
          )}
          {endedTops.map(renderBranch)}
        </div>
      </ScrollArea>

      <div className="px-3 py-2 border-t border-border text-[10px] text-muted-foreground/60 font-mono">
        {t('web.sessions.list.footer', { live: liveCount, ended: endedCount })}
      </div>
    </aside>
  )
}

interface ChildRowProps {
  session: Session
  active?: boolean
  onClick: () => void
  onDelete: () => void
}

// ChildRow is the compact, indented row for sessions whose parent_session_id
// points at a row in the list — typically Tasks-spawned shells. Keeps the
// project's grouping visible without taking the full vertical real estate
// of a top-level row.
function ChildRow({ session, active, onClick, onDelete }: ChildRowProps) {
  const { t } = useTranslation()
  const visual = providerVisual(session.provider_id)
  const dot =
    session.state === 'running'
      ? 'bg-state-running'
      : session.state === 'idle' || session.state === 'pending'
        ? 'bg-state-idle'
        : session.exit_code != null && session.exit_code !== 0
          ? 'bg-state-failed'
          : 'bg-muted-foreground/60'
  const deleteTitle =
    session.state === 'ended' || session.state === 'stopped'
      ? t('web.sessions.list.row.titleRemove')
      : t('web.sessions.list.row.titleTerminate')
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onClick()
        }
      }}
      className={cn(
        'group relative flex items-center gap-1.5 pl-3 pr-1.5 py-1 rounded-md cursor-pointer transition-colors',
        'border border-transparent ml-3',
        active
          ? 'bg-card border-border'
          : 'hover:bg-card/60 hover:border-border/60',
      )}
    >
      <CornerDownRight className="size-3 text-muted-foreground/40 shrink-0" />
      <BrandAvatar
        iconKey={providerIconKey(session.provider_id)}
        fallbackLetter={visual.letter}
        size={16}
        title={visual.name}
        className="rounded-sm"
      />
      <span
        className={cn(
          'size-1.5 rounded-full shrink-0',
          dot,
          session.state === 'running' && 'animate-pulse',
        )}
      />
      <span className="text-[11.5px] truncate flex-1 text-foreground/90">
        {session.name || cwdTail(session.cwd)}
      </span>
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation()
          onDelete()
        }}
        aria-label={t('web.sessions.list.row.deleteAria')}
        title={deleteTitle}
        className={cn(
          'size-4 rounded-sm flex items-center justify-center shrink-0',
          'text-muted-foreground/40 hover:text-destructive hover:bg-destructive/10',
          'opacity-0 group-hover:opacity-100 focus-visible:opacity-100 transition-opacity',
        )}
      >
        <X className="size-2.5" />
      </button>
    </div>
  )
}
