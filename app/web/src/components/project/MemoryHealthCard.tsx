// MemoryHealthCard — M-PA dashboard panel summarising the per-cwd
// state of both memory subsystems. Renders inside ProjectScreen's
// Health tab and a fetched on demand (no polling — the dashboard
// is informational, not real-time).

import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import {
  Activity,
  AlertCircle,
  BookOpen,
  Brain,
  CheckCircle2,
  Database,
  Inbox,
  Loader2,
  TrendingUp,
} from 'lucide-react'

import { getMemoryHealth } from '@/lib/memoryHealth'

interface MemoryHealthCardProps {
  cwd: string
}

export function MemoryHealthCard({ cwd }: MemoryHealthCardProps) {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: ['memory-health', cwd],
    queryFn: () => getMemoryHealth(cwd),
    enabled: !!cwd,
    staleTime: 60_000,
  })

  if (!cwd) {
    return (
      <div className="text-muted-foreground p-6 text-[12px]">
        {t('web.memoryHealth.pickCwd')}
      </div>
    )
  }
  if (query.isLoading) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 p-6 text-[12px]">
        <Loader2 className="size-3 animate-spin" />
        {t('web.memoryHealth.loading')}
      </div>
    )
  }
  if (query.isError || !query.data) {
    return (
      <div className="text-destructive flex items-center gap-2 p-6 text-[12px]">
        <AlertCircle className="size-3" />
        {t('web.memoryHealth.errorLoading')}
      </div>
    )
  }

  const snap = query.data
  const planAge = formatRelative(snap.plan_last_updated_at, t)
  const goalAge = formatRelative(snap.goal_last_updated_at, t)

  return (
    <div className="space-y-4 p-4">
      <header>
        <h2 className="text-sm font-medium">
          {t('web.memoryHealth.title', { days: snap.lookback_days })}
        </h2>
        <p className="text-muted-foreground mt-0.5 text-[11px]">
          {t('web.memoryHealth.subtitle')}
        </p>
      </header>

      <div className="grid grid-cols-2 gap-3 md:grid-cols-3">
        <Metric
          icon={<Database className="size-3.5" />}
          label={t('web.memoryHealth.newFacts')}
          value={snap.new_facts_count}
          hint={t('web.memoryHealth.newFactsHint', {
            total: snap.total_facts_count,
          })}
        />
        <Metric
          icon={<Activity className="size-3.5" />}
          label={t('web.memoryHealth.captureFires')}
          value={snap.capture_fires}
          hint={t('web.memoryHealth.captureFiresHint', {
            stored: snap.capture_facts_stored,
            deduped: snap.capture_facts_deduped,
          })}
          tone={snap.capture_failed_fires > 0 ? 'warn' : 'ok'}
        />
        <Metric
          icon={<BookOpen className="size-3.5" />}
          label={t('web.memoryHealth.newJournal')}
          value={snap.new_journal_count}
          hint={t('web.memoryHealth.newJournalHint', {
            total: snap.total_journal_count,
          })}
        />
        <Metric
          icon={<Brain className="size-3.5" />}
          label={t('web.memoryHealth.planAge')}
          value={planAge}
          hint={
            snap.plan_drift_proposals > 0
              ? t('web.memoryHealth.planAgeHint', {
                  count: snap.plan_drift_proposals,
                })
              : t('web.memoryHealth.planAgeHintNone')
          }
        />
        <Metric
          icon={<Brain className="size-3.5" />}
          label={t('web.memoryHealth.goalAge')}
          value={goalAge}
        />
        <Metric
          icon={<Inbox className="size-3.5" />}
          label={t('web.memoryHealth.pending')}
          value={snap.pending_proposals}
          hint={
            snap.pending_proposals > 0 && snap.oldest_pending_days > 0
              ? t('web.memoryHealth.pendingHint', {
                  days: snap.oldest_pending_days,
                })
              : undefined
          }
          tone={snap.oldest_pending_days >= 7 ? 'warn' : 'ok'}
        />
      </div>

      {snap.top_hit_fact_text && (
        <div className="bg-muted/20 rounded border p-3 text-[12px]">
          <div className="text-muted-foreground mb-1 flex items-center gap-1 text-[10px]">
            <TrendingUp className="size-3" />
            {t('web.memoryHealth.topHit', { hits: snap.top_hit_fact_hits })}
          </div>
          <div className="font-mono break-words">{snap.top_hit_fact_text}</div>
        </div>
      )}

      {snap.zero_hit_facts_count > 0 && (
        <div className="text-muted-foreground flex items-center gap-2 text-[11px]">
          <CheckCircle2 className="size-3" />
          {t('web.memoryHealth.zeroHit', {
            count: snap.zero_hit_facts_count,
          })}
        </div>
      )}
    </div>
  )
}

interface MetricProps {
  icon: React.ReactNode
  label: string
  value: number | string
  hint?: string
  tone?: 'ok' | 'warn'
}

function Metric({ icon, label, value, hint, tone = 'ok' }: MetricProps) {
  return (
    <div className="bg-card/50 rounded-md border p-3">
      <div className="text-muted-foreground flex items-center gap-1 text-[10px]">
        {icon}
        {label}
      </div>
      <div
        className={`mt-1 text-xl font-semibold ${
          tone === 'warn' ? 'text-amber-500' : ''
        }`}
      >
        {value}
      </div>
      {hint && (
        <div className="text-muted-foreground mt-1 text-[10px]">{hint}</div>
      )}
    </div>
  )
}

function formatRelative(
  iso: string | undefined,
  t: ReturnType<typeof useTranslation>['t'],
): string {
  if (!iso) return t('web.memoryHealth.never')
  const then = new Date(iso).getTime()
  if (Number.isNaN(then)) return '—'
  const days = Math.floor((Date.now() - then) / 86_400_000)
  if (days <= 0) return t('web.memoryHealth.today')
  if (days === 1) return t('web.memoryHealth.daysAgo', { count: 1 })
  return t('web.memoryHealth.daysAgo', { count: days })
}
