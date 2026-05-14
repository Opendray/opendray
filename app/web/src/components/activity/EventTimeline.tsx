import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  ChevronDown,
  ChevronRight,
  Loader2,
  RefreshCw,
  Activity as ActivityIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Code } from '@/components/ui/code'
import { listAudit, type AuditEntry, type AuditQuery } from '@/lib/audit'
import { cn } from '@/lib/utils'

// ── kind helpers ─────────────────────────────────────────────────
type Kind = 'session' | 'channel' | 'integration' | 'admin' | 'other'

function kindOf(action: string): Kind {
  if (action.startsWith('session.')) return 'session'
  if (action.startsWith('channel.')) return 'channel'
  if (action.startsWith('integration.')) return 'integration'
  if (action.startsWith('admin.')) return 'admin'
  return 'other'
}

const KIND_BADGE: Record<Kind, 'success' | 'accent' | 'warning' | 'muted' | 'outline'> =
  {
    session: 'success',
    channel: 'accent',
    integration: 'warning',
    admin: 'muted',
    other: 'outline',
  }

const KIND_DOT: Record<Kind, string> = {
  session: 'bg-state-running',
  channel: 'bg-accent',
  integration: 'bg-state-idle',
  admin: 'bg-muted-foreground',
  other: 'bg-muted-foreground/40',
}

function summarize(metadata: unknown): string {
  if (metadata == null) return ''
  if (typeof metadata !== 'object') return String(metadata)
  const obj = metadata as Record<string, unknown>
  const parts: string[] = []
  for (const k of [
    'name',
    'author',
    'text',
    'exit_code',
    'state',
    'from',
    'to',
    'topic',
    'route_prefix',
    'idle_for_ms',
  ]) {
    if (k in obj && obj[k] !== '' && obj[k] != null) {
      parts.push(`${k}=${formatValue(obj[k])}`)
    }
  }
  return parts.join(' · ')
}

function formatValue(v: unknown): string {
  if (typeof v === 'string') {
    if (v.length > 60) return JSON.stringify(v.slice(0, 57) + '…')
    return v
  }
  if (typeof v === 'number') return v.toString()
  if (typeof v === 'boolean') return v.toString()
  return JSON.stringify(v)
}

// ── component ────────────────────────────────────────────────────
export interface EventTimelineProps {
  /** Scope to a single object; omit for global view. */
  subject?: { kind: 'session' | 'channel' | 'integration' | 'admin'; id: string }
  /** Action filter — exact ("session.idle") or prefix ("session.*"). */
  action?: string
  /** Free-text search applied client-side over loaded entries. */
  search?: string
  /** Polling interval for the latest page, in ms. 0 disables. */
  refetchInterval?: number
  /** Page size, default 50, max 500. */
  pageSize?: number
  /** Compact mode tightens spacing and hides the "filtered" caption. */
  dense?: boolean
  /** Optional hint shown in the empty state. */
  emptyHint?: string
  className?: string
}

// EventTimeline displays audit_log entries newest-first, grouped by
// day, with on-demand pagination and live polling for new entries.
//
// Two reads share state inside the component:
//  - the "head" query (cursor=undefined) is React-Queried and polled
//    on refetchInterval; new ids are merged into the local list.
//  - older pages are fetched imperatively via Load More and appended.
export function EventTimeline({
  subject,
  action,
  search = '',
  refetchInterval = 5_000,
  pageSize = 50,
  dense = false,
  emptyHint,
  className,
}: EventTimelineProps) {
  const { t } = useTranslation()
  const baseQuery: AuditQuery = useMemo(
    () => ({
      subject_kind: subject?.kind,
      subject_id: subject?.id,
      action: action || undefined,
      limit: pageSize,
    }),
    [subject?.kind, subject?.id, action, pageSize],
  )

  const headKey = useMemo(
    () => ['audit-head', baseQuery] as const,
    [baseQuery],
  )

  // The head query gives us "what's new" plus the initial page on mount.
  const head = useQuery({
    queryKey: headKey,
    queryFn: () => listAudit(baseQuery),
    refetchInterval: refetchInterval > 0 ? refetchInterval : false,
    staleTime: 1_000,
  })

  // Local accumulator so loaded-older pages persist across head refetches.
  // Keyed by id so dedupe is O(1).
  const [byId, setById] = useState<Record<number, AuditEntry>>({})
  const [nextCursor, setNextCursor] = useState<string | null>(null)
  const [loadingMore, setLoadingMore] = useState(false)
  const headSeenRef = useRef(false)

  // Merge the head page result into the accumulator. Use the returned
  // next_cursor as the "load more" cursor only on the very first
  // successful response — subsequent head refetches don't extend
  // pagination, they only graft in new ids.
  useEffect(() => {
    if (!head.data) return
    setById((prev) => {
      const out = { ...prev }
      for (const e of head.data!.entries) out[e.id] = e
      return out
    })
    if (!headSeenRef.current) {
      headSeenRef.current = true
      setNextCursor(head.data.next_cursor)
    }
  }, [head.data])

  // Reset accumulator when filters change (the queryKey change
  // triggers head refetch; we also wipe accumulated older pages).
  useEffect(() => {
    setById({})
    setNextCursor(null)
    headSeenRef.current = false
  }, [baseQuery])

  const entries = useMemo(() => {
    const arr = Object.values(byId).sort((a, b) => b.id - a.id)
    if (!search.trim()) return arr
    const q = search.trim().toLowerCase()
    return arr.filter(
      (e) =>
        e.action.toLowerCase().includes(q) ||
        (e.subject_id && e.subject_id.toLowerCase().includes(q)) ||
        JSON.stringify(e.metadata).toLowerCase().includes(q),
    )
  }, [byId, search])

  const groups = useMemo(
    () =>
      groupByDay(entries, {
        today: t('web.activity.events.today'),
        yesterday: t('web.activity.events.yesterday'),
      }),
    [entries, t],
  )

  const onLoadMore = useCallback(async () => {
    if (!nextCursor || loadingMore) return
    setLoadingMore(true)
    try {
      const page = await listAudit({ ...baseQuery, cursor: nextCursor })
      setById((prev) => {
        const out = { ...prev }
        for (const e of page.entries) out[e.id] = e
        return out
      })
      setNextCursor(page.next_cursor)
    } catch {
      // toast is handled by api.ts; nothing else to do
    } finally {
      setLoadingMore(false)
    }
  }, [baseQuery, nextCursor, loadingMore])

  const isLoading = head.isLoading && entries.length === 0

  return (
    <div className={cn('flex flex-col min-h-0', className)}>
      {isLoading ? (
        <div className="flex items-center justify-center py-8 gap-2 text-[12px] text-muted-foreground">
          <Loader2 className="size-3.5 animate-spin" />
          {t('web.activity.events.loading')}
        </div>
      ) : entries.length === 0 ? (
        <Empty hint={emptyHint} hasFilter={!!search} />
      ) : (
        <ul className="flex flex-col">
          {groups.map((g) => (
            <DayGroup key={g.label} label={g.label} count={g.entries.length}>
              {g.entries.map((e) => (
                <EventRow key={e.id} entry={e} dense={dense} />
              ))}
            </DayGroup>
          ))}
        </ul>
      )}

      {nextCursor && (
        <div className="px-3 py-2 border-t border-border flex items-center justify-center">
          <Button
            variant="ghost"
            size="sm"
            onClick={onLoadMore}
            disabled={loadingMore}
            className="text-muted-foreground hover:text-foreground"
          >
            {loadingMore ? (
              <Loader2 className="size-3.5 animate-spin" />
            ) : (
              <RefreshCw className="size-3.5" />
            )}
            {t('web.activity.events.loadOlder')}
          </Button>
        </div>
      )}
    </div>
  )
}

// ── presentation pieces ─────────────────────────────────────────
function DayGroup({
  label,
  count,
  children,
}: {
  label: string
  count: number
  children: React.ReactNode
}) {
  return (
    <li>
      <div className="sticky top-0 z-10 bg-background/95 backdrop-blur px-3 py-1 border-b border-border flex items-center gap-2">
        <span className="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium">
          {label}
        </span>
        <span className="text-[10px] text-muted-foreground/40 font-mono tabular-nums">
          {count}
        </span>
      </div>
      <ul>{children}</ul>
    </li>
  )
}

function EventRow({ entry, dense }: { entry: AuditEntry; dense: boolean }) {
  const [open, setOpen] = useState(false)
  const k = kindOf(entry.action)
  const ts = new Date(entry.ts)
  const time = isNaN(ts.getTime())
    ? '--:--:--'
    : ts.toTimeString().slice(0, 8)

  return (
    <li className="flex items-stretch hover:bg-card/60 transition-colors group">
      <span
        className={cn(
          'w-[2px] shrink-0 opacity-50 group-hover:opacity-100',
          KIND_DOT[k],
        )}
      />
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className={cn(
          'flex-1 min-w-0 text-left flex items-start gap-2',
          dense ? 'px-2 py-1' : 'px-3 py-1.5',
        )}
      >
        {open ? (
          <ChevronDown className="size-3 mt-0.5 text-muted-foreground/60 shrink-0" />
        ) : (
          <ChevronRight className="size-3 mt-0.5 text-muted-foreground/60 shrink-0" />
        )}
        <span className="text-[10px] text-muted-foreground/70 font-mono shrink-0 w-[60px] tabular-nums pt-0.5">
          {time}
        </span>
        <div className="flex-1 min-w-0 flex flex-col gap-0.5">
          <div className="flex items-center gap-2 flex-wrap">
            <Badge
              variant={KIND_BADGE[k]}
              className="shrink-0 max-w-[200px]"
            >
              <span className="truncate">{entry.action}</span>
            </Badge>
            {entry.subject_id && (
              <span className="text-[10px] text-muted-foreground/60 font-mono truncate">
                {entry.subject_kind}={entry.subject_id.slice(0, 12)}
              </span>
            )}
          </div>
          <span className="text-[11px] text-muted-foreground truncate font-mono">
            {summarize(entry.metadata)}
          </span>
          {open && (
            <div className="mt-1">
              <Code>{JSON.stringify(entry.metadata ?? null, null, 2)}</Code>
            </div>
          )}
        </div>
      </button>
    </li>
  )
}

function Empty({
  hint,
  hasFilter,
}: {
  hint: string | undefined
  hasFilter: boolean
}) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-col items-center justify-center py-12 gap-2 text-[12px] text-muted-foreground">
      <ActivityIcon className="size-4 text-muted-foreground/50" />
      <span>
        {hasFilter
          ? t('web.activity.events.emptyFiltered')
          : t('web.activity.events.empty')}
      </span>
      {!hasFilter && hint && (
        <span className="text-[11px] text-muted-foreground/60 max-w-md text-center">
          {hint}
        </span>
      )}
    </div>
  )
}

// ── helpers ─────────────────────────────────────────────────────
interface DayGroupData {
  label: string
  entries: AuditEntry[]
}

function groupByDay(
  entries: AuditEntry[],
  labels: { today: string; yesterday: string },
): DayGroupData[] {
  const today = startOfDay(new Date())
  const yesterday = new Date(today.getTime() - 86_400_000)
  const out: DayGroupData[] = []
  let current: DayGroupData | null = null
  for (const e of entries) {
    const d = startOfDay(new Date(e.ts))
    const label =
      d.getTime() === today.getTime()
        ? labels.today
        : d.getTime() === yesterday.getTime()
          ? labels.yesterday
          : d.toLocaleDateString(undefined, {
              month: 'short',
              day: 'numeric',
              year:
                d.getFullYear() === today.getFullYear() ? undefined : 'numeric',
            })
    if (!current || current.label !== label) {
      current = { label, entries: [] }
      out.push(current)
    }
    current.entries.push(e)
  }
  return out
}

function startOfDay(d: Date): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate())
}
