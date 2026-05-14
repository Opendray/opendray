import { formatDistanceToNow } from 'date-fns'
import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'
import { providerVisual, cwdTail } from '@/lib/providers'
import { providerIconKey } from '@/lib/providerIcons'
import { BrandAvatar } from '@/components/BrandAvatar'
import type { Session } from '@/lib/types'

interface SessionRowProps {
  session: Session
  active?: boolean
  onClick?: () => void
  onDelete?: () => void
  /** Resolved label for session.claude_account_id — undefined if not bound. */
  accountLabel?: string
}

function displayName(s: Session): string {
  if (s.name && s.name.length > 0) return s.name
  return cwdTail(s.cwd)
}

function relativeStarted(s: Session): string {
  const t = s.ended_at ?? s.started_at
  try {
    return formatDistanceToNow(new Date(t), { addSuffix: true })
  } catch {
    return '—'
  }
}

// Status-dot color matches StatePill's logic but stays minimal — the
// row uses a 2px dot inline with the subtitle, not a full pill.
function statusDot(s: Session): string {
  if (s.state === 'running') return 'bg-state-running'
  if (s.state === 'idle' || s.state === 'pending') return 'bg-state-idle'
  if (s.state === 'ended' && s.exit_code != null && s.exit_code !== 0) {
    return 'bg-state-failed'
  }
  return 'bg-muted-foreground/60'
}

export function SessionRow({
  session,
  active,
  onClick,
  onDelete,
  accountLabel,
}: SessionRowProps) {
  const { t } = useTranslation()
  const visual = providerVisual(session.provider_id)
  const isClaude = session.provider_id === 'claude'
  const acct = isClaude
    ? accountLabel ??
      (session.claude_account_id
        ? '…'
        : t('web.sessions.accountSwitcher.currentDefault'))
    : null

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onClick?.()
        }
      }}
      className={cn(
        'group relative w-full flex items-start gap-2.5 px-2.5 py-2.5 rounded-lg text-left transition-colors cursor-pointer',
        'border border-transparent',
        active
          ? 'bg-card border-border shadow-sm'
          : 'hover:bg-card/60 hover:border-border/60',
      )}
    >
      <BrandAvatar
        iconKey={providerIconKey(session.provider_id)}
        fallbackLetter={visual.letter}
        size={32}
        title={visual.name}
      />

      <div className="flex-1 min-w-0 flex flex-col gap-0.5 pr-4">
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-[13px] font-medium truncate flex-1 text-foreground leading-tight">
            {displayName(session)}
          </span>
          <span className="text-[10px] text-muted-foreground/60 shrink-0">
            {relativeStarted(session)}
          </span>
        </div>
        <div className="flex items-center gap-1.5 min-w-0 text-[11px] text-muted-foreground/80">
          <span
            className={cn(
              'size-1.5 rounded-full shrink-0',
              statusDot(session),
              session.state === 'running' && 'animate-pulse',
            )}
          />
          <span className="truncate font-mono">
            {visual.name} · {cwdTail(session.cwd)}
          </span>
          {acct && (
            <span
              className="ml-auto shrink-0 text-[10px] font-mono px-1.5 py-px rounded bg-card border border-border/60 text-muted-foreground/80"
              title={t('web.sessions.list.row.claudeAccountTitle', {
                label: acct,
              })}
            >
              @{acct}
            </span>
          )}
        </div>
      </div>

      {onDelete && (
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation()
            onDelete()
          }}
          aria-label={t('web.sessions.list.row.deleteAria')}
          title={
            session.state === 'ended' || session.state === 'stopped'
              ? t('web.sessions.list.row.titleRemoveHistory')
              : t('web.sessions.list.row.titleTerminate')
          }
          className={cn(
            'absolute top-2 right-2 size-5 rounded-sm flex items-center justify-center',
            'text-muted-foreground/50 hover:text-destructive hover:bg-destructive/10',
            'opacity-0 group-hover:opacity-100 focus-visible:opacity-100 transition-opacity',
          )}
        >
          <X className="size-3" />
        </button>
      )}
    </div>
  )
}
