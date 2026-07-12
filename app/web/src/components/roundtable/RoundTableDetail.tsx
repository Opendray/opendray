import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Loader2, Play, Gavel, MessagesSquare } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import {
  closeRoundTable,
  getRoundTable,
  startRoundTable,
  type Beat,
  type RoundTableStatus,
  type Turn,
} from '@/lib/roundtable'

// Structured shapes the seats emit (mirror the Go structs).
interface ProposalStruct {
  summary: string
  plan: string
  tasks: string[] | null
  tradeoffs: string[] | null
  confidence: number
}
interface CritiqueStruct {
  critiques:
    | { target_provider: string; severity: string; point: string }[]
    | null
}

const STATUS_VARIANT: Record<
  RoundTableStatus,
  'muted' | 'accent' | 'success' | 'danger' | 'outline'
> = {
  draft: 'muted',
  running: 'accent',
  awaiting_verdict: 'success',
  failed: 'danger',
  closed: 'outline',
}

const BEAT_ORDER: Beat[] = ['propose', 'critique', 'synthesize']

export function RoundTableDetail({ id }: { id: string }) {
  const { t } = useTranslation()
  const qc = useQueryClient()

  const query = useQuery({
    queryKey: ['round-table', id],
    queryFn: () => getRoundTable(id),
    // Poll while the discussion is in flight; stop once it settles.
    refetchInterval: (q) => {
      const s = q.state.data?.round_table.status
      return s === 'running' || s === 'draft' ? 3000 : false
    },
  })

  const start = useMutation({
    mutationFn: () => startRoundTable(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['round-table', id] })
      qc.invalidateQueries({ queryKey: ['round-tables'] })
      toast.success(t('web.roundTable.detail.started'))
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const close = useMutation({
    mutationFn: () => closeRoundTable(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['round-table', id] })
      qc.invalidateQueries({ queryKey: ['round-tables'] })
    },
    onError: (e: Error) => toast.error(e.message),
  })

  if (query.isLoading) {
    return (
      <div className="flex items-center gap-2 p-6 text-sm text-muted-foreground">
        <Loader2 className="size-4 animate-spin" />
        {t('web.roundTable.detail.loading')}
      </div>
    )
  }
  if (query.isError || !query.data) {
    return (
      <div className="p-6 text-sm text-state-failed">
        {t('web.roundTable.detail.loadFailed')}
      </div>
    )
  }

  const { round_table: rt, turns } = query.data
  const running = rt.status === 'running'

  return (
    <div className="flex flex-col gap-5">
      {/* Header */}
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <h2 className="text-sm font-medium truncate">{rt.topic}</h2>
            <Badge variant={STATUS_VARIANT[rt.status]}>
              {t(`web.roundTable.status.${rt.status}`)}
            </Badge>
          </div>
          <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
            {rt.seats.map((s) => (
              <Badge key={s.provider} variant="muted" className="capitalize">
                {s.provider}
              </Badge>
            ))}
            {rt.cwd && (
              <span className="text-[11px] text-muted-foreground font-mono truncate">
                {rt.cwd}
              </span>
            )}
          </div>
        </div>
        <div className="flex shrink-0 gap-2">
          {rt.status === 'draft' && (
            <Button
              size="sm"
              disabled={start.isPending}
              onClick={() => start.mutate()}
            >
              {start.isPending ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <Play className="size-3.5" />
              )}
              {t('web.roundTable.detail.start')}
            </Button>
          )}
          {rt.status !== 'closed' && rt.status !== 'draft' && (
            <Button
              variant="outline"
              size="sm"
              disabled={close.isPending}
              onClick={() => close.mutate()}
            >
              {t('web.roundTable.detail.close')}
            </Button>
          )}
        </div>
      </div>

      {running && (
        <div className="flex items-center gap-2 rounded-md border border-accent/30 bg-accent/10 px-3 py-2 text-xs text-foreground">
          <Loader2 className="size-3.5 animate-spin text-accent" />
          {t('web.roundTable.detail.running')}
        </div>
      )}
      {rt.status === 'failed' && rt.error && (
        <div className="rounded-md border border-state-failed/30 bg-state-failed/10 px-3 py-2 text-xs text-state-failed">
          {rt.error}
        </div>
      )}

      {/* Verdict */}
      {rt.verdict && (
        <div className="rounded-lg border border-border bg-card/40 p-4">
          <div className="mb-2 flex items-center gap-2">
            <Gavel className="size-4 text-accent" />
            <h3 className="text-[13px] font-medium">
              {t('web.roundTable.verdict.title')}
            </h3>
            <Badge variant="accent" className="capitalize">
              {t('web.roundTable.verdict.recommendedBy', {
                provider: rt.verdict.recommended_by,
              })}
            </Badge>
          </div>
          <p className="whitespace-pre-wrap text-[13px] leading-relaxed">
            {rt.verdict.recommended}
          </p>

          <VerdictList
            title={t('web.roundTable.verdict.tasks')}
            items={rt.verdict.task_breakdown}
          />
          <VerdictList
            title={t('web.roundTable.verdict.tradeoffs')}
            items={rt.verdict.tradeoffs}
          />
          <VerdictList
            title={t('web.roundTable.verdict.openQuestions')}
            items={rt.verdict.open_questions}
          />
          <VerdictList
            title={t('web.roundTable.verdict.alternatives')}
            items={rt.verdict.alternatives}
          />

          {rt.verdict.ranking && rt.verdict.ranking.length > 0 && (
            <div className="mt-3">
              <div className="mb-1 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                {t('web.roundTable.verdict.ranking')}
              </div>
              <div className="flex flex-col gap-1">
                {rt.verdict.ranking.map((r, i) => (
                  <div
                    key={r.provider}
                    className="flex items-center gap-2 text-[12px]"
                  >
                    <span className="w-4 tabular-nums text-muted-foreground">
                      {i + 1}
                    </span>
                    <span className="w-24 capitalize">{r.provider}</span>
                    <span className="text-muted-foreground">
                      {t('web.roundTable.verdict.scoreLine', {
                        blockers: r.blockers,
                        concerns: r.concerns,
                        confidence: r.confidence.toFixed(2),
                      })}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Discussion thread */}
      <div>
        <div className="mb-2 flex items-center gap-2">
          <MessagesSquare className="size-4 text-muted-foreground" />
          <h3 className="text-[13px] font-medium">
            {t('web.roundTable.thread.title')}
          </h3>
        </div>
        {turns.length === 0 ? (
          <p className="text-xs text-muted-foreground">
            {t('web.roundTable.thread.empty')}
          </p>
        ) : (
          <div className="flex flex-col gap-4">
            {BEAT_ORDER.filter((b) => turns.some((tn) => tn.beat === b)).map(
              (beat) => (
                <div key={beat}>
                  <div className="mb-1.5 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                    {t(`web.roundTable.beat.${beat}`)}
                  </div>
                  <div className="flex flex-col gap-2">
                    {turns
                      .filter((tn) => tn.beat === beat)
                      .map((tn) => (
                        <TurnCard key={tn.id} turn={tn} />
                      ))}
                  </div>
                </div>
              ),
            )}
          </div>
        )}
      </div>
    </div>
  )
}

function VerdictList({
  title,
  items,
}: {
  title: string
  items: string[] | null | undefined
}) {
  if (!items || items.length === 0) return null
  return (
    <div className="mt-3">
      <div className="mb-1 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
        {title}
      </div>
      <ul className="list-disc pl-4 text-[12px] leading-relaxed">
        {items.map((it, i) => (
          <li key={i}>{it}</li>
        ))}
      </ul>
    </div>
  )
}

function TurnCard({ turn }: { turn: Turn }) {
  const { t } = useTranslation()
  const isSystem = turn.role === 'system'
  const isChair = turn.role === 'chair'

  const proposal =
    turn.beat === 'propose' && turn.role === 'seat'
      ? (turn.structured as ProposalStruct | undefined)
      : undefined
  const critique =
    turn.beat === 'critique' && turn.role === 'seat'
      ? (turn.structured as CritiqueStruct | undefined)
      : undefined

  return (
    <div
      className={cn(
        'rounded-md border px-3 py-2',
        isSystem
          ? 'border-state-failed/30 bg-state-failed/5'
          : isChair
            ? 'border-accent/30 bg-accent/5'
            : 'border-border bg-card/40',
      )}
    >
      <div className="mb-1 flex items-center gap-2">
        <span className="text-[12px] font-medium capitalize">
          {turn.seat_provider || (isChair ? t('web.roundTable.chair') : 'system')}
        </span>
        {proposal && (
          <Badge variant="muted">
            {t('web.roundTable.confidence', {
              value: proposal.confidence.toFixed(2),
            })}
          </Badge>
        )}
      </div>

      {proposal ? (
        <div className="flex flex-col gap-1.5">
          <p className="whitespace-pre-wrap text-[12px] leading-relaxed">
            {proposal.plan || proposal.summary}
          </p>
          {proposal.tasks && proposal.tasks.length > 0 && (
            <ul className="list-disc pl-4 text-[11px] text-muted-foreground">
              {proposal.tasks.map((task, i) => (
                <li key={i}>{task}</li>
              ))}
            </ul>
          )}
        </div>
      ) : critique ? (
        <div className="flex flex-col gap-1">
          {(critique.critiques ?? []).length === 0 ? (
            <span className="text-[12px] text-muted-foreground">
              {t('web.roundTable.thread.noCritiques')}
            </span>
          ) : (
            (critique.critiques ?? []).map((c, i) => (
              <div key={i} className="flex items-start gap-1.5 text-[12px]">
                <Badge
                  variant={
                    c.severity === 'blocker'
                      ? 'danger'
                      : c.severity === 'concern'
                        ? 'warning'
                        : 'muted'
                  }
                >
                  {c.severity}
                </Badge>
                <span className="text-muted-foreground">→ {c.target_provider}</span>
                <span className="flex-1">{c.point}</span>
              </div>
            ))
          )}
        </div>
      ) : (
        <p className="whitespace-pre-wrap text-[12px] leading-relaxed text-muted-foreground">
          {turn.content}
        </p>
      )}
    </div>
  )
}
