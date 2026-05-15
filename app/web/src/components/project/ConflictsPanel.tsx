// ConflictsPanel — M-PC operator inbox for cross-layer
// contradictions surfaced by the daily detector. Each row shows
// the two conflicting items + the LLM's evidence + accept/dismiss
// buttons.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import {
  AlertTriangle,
  Check,
  Loader2,
  RefreshCw,
  X,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  type MemoryConflict,
  decideMemoryConflict,
  detectMemoryConflicts,
  listMemoryConflicts,
} from '@/lib/memoryConflicts'
import { deleteMemory } from '@/lib/memory'

interface ConflictsPanelProps {
  cwd: string
  /**
   * Optional callback the parent can pass so the panel's quick-
   * actions can jump to another tab (e.g. "Open plan editor"
   * navigates to the Plan tab). Omit to disable the jump button.
   */
  onJumpTab?: (tab: string) => void
}

export function ConflictsPanel({ cwd, onJumpTab }: ConflictsPanelProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()

  const conflictsQuery = useQuery({
    queryKey: ['memory-conflicts', cwd, 'pending'],
    queryFn: () =>
      listMemoryConflicts({ cwd, status: 'pending', limit: 100 }),
    enabled: !!cwd,
  })

  const decide = useMutation({
    mutationFn: (input: {
      id: string
      action: 'accepted' | 'dismissed'
    }) => decideMemoryConflict(input.id, input.action),
    onSuccess: (_data, vars) => {
      toast.success(
        vars.action === 'accepted'
          ? t('web.conflicts.accepted')
          : t('web.conflicts.dismissed'),
      )
      qc.invalidateQueries({ queryKey: ['memory-conflicts', cwd] })
    },
    onError: (err: unknown) => {
      toast.error(`${err}`)
    },
  })

  const detect = useMutation({
    mutationFn: () => detectMemoryConflicts(cwd),
    onSuccess: (n) => {
      toast.success(t('web.conflicts.detected', { count: n }))
      qc.invalidateQueries({ queryKey: ['memory-conflicts', cwd] })
    },
    onError: (err: unknown) => {
      toast.error(`${err}`)
    },
  })

  // M-PD quick-action: "Delete this fact" — yanks the offending
  // memory row and auto-accepts the conflict so the inbox clears
  // in one click. Plan/goal sides delegate to a tab-jump instead
  // (we don't want one-click overwriting of operator-owned docs).
  const deleteFactAndAccept = useMutation({
    mutationFn: async (input: { conflictId: string; factId: string }) => {
      await deleteMemory(input.factId)
      await decideMemoryConflict(input.conflictId, 'accepted')
    },
    onSuccess: () => {
      toast.success(t('web.conflicts.deletedFact'))
      qc.invalidateQueries({ queryKey: ['memory-conflicts', cwd] })
      qc.invalidateQueries({ queryKey: ['memories'] })
    },
    onError: (err: unknown) => {
      toast.error(`${err}`)
    },
  })

  if (!cwd) {
    return (
      <div className="text-muted-foreground p-6 text-[12px]">
        {t('web.conflicts.pickCwd')}
      </div>
    )
  }

  const conflicts = conflictsQuery.data ?? []

  return (
    <div className="flex flex-1 flex-col">
      <div className="border-border flex items-center justify-between border-b px-4 py-3">
        <div>
          <h2 className="text-sm font-medium">{t('web.conflicts.title')}</h2>
          <p className="text-muted-foreground text-[11px]">
            {t('web.conflicts.subtitle')}
          </p>
        </div>
        <Button
          size="sm"
          variant="outline"
          onClick={() => detect.mutate()}
          disabled={detect.isPending}
        >
          {detect.isPending ? (
            <Loader2 className="mr-1 size-3 animate-spin" />
          ) : (
            <RefreshCw className="mr-1 size-3" />
          )}
          {t('web.conflicts.detectNow')}
        </Button>
      </div>

      <div className="flex-1 overflow-auto p-4">
        {conflictsQuery.isLoading && (
          <div className="text-muted-foreground flex items-center gap-2 text-[12px]">
            <Loader2 className="size-3 animate-spin" />
            {t('web.conflicts.loading')}
          </div>
        )}
        {!conflictsQuery.isLoading && conflicts.length === 0 && (
          <div className="text-muted-foreground rounded border border-dashed p-6 text-center text-[12px]">
            {t('web.conflicts.empty')}
          </div>
        )}
        <div className="space-y-3">
          {conflicts.map((c) => (
            <ConflictCard
              key={c.id}
              conflict={c}
              onDecide={(action) => decide.mutate({ id: c.id, action })}
              disabled={decide.isPending || deleteFactAndAccept.isPending}
              onDeleteFact={(factId) =>
                deleteFactAndAccept.mutate({ conflictId: c.id, factId })
              }
              onJumpTab={onJumpTab}
            />
          ))}
        </div>
      </div>
    </div>
  )
}

interface ConflictCardProps {
  conflict: MemoryConflict
  onDecide: (action: 'accepted' | 'dismissed') => void
  onDeleteFact: (factId: string) => void
  onJumpTab?: (tab: string) => void
  disabled: boolean
}

function ConflictCard({
  conflict,
  onDecide,
  onDeleteFact,
  onJumpTab,
  disabled,
}: ConflictCardProps) {
  const { t } = useTranslation()
  const severityTone =
    conflict.severity === 'high'
      ? 'danger'
      : conflict.severity === 'medium'
        ? 'warn'
        : 'muted'

  // Quick-action targets — one button per conflicting side:
  //   layer=fact   → "Delete fact A/B (mem_…)" (yanks the row +
  //                  auto-accepts the conflict)
  //   layer=plan   → "Open plan editor" (jumps to the Plan tab)
  //   layer=goal   → "Open goal editor"
  //   layer=journal → no quick-action; operator can dismiss
  //
  // When both sides are facts (the common case for hard
  // contradictions like "DB is Postgres" vs "DB is SQLite"), we
  // emit TWO delete buttons — one per fact id — so the operator
  // can pick which side is wrong without leaving the panel. Same
  // when both sides are plan/goal we'd dedupe the editor button.
  type QuickAction =
    | { kind: 'delete-fact'; sideLabel: string; refId: string }
    | { kind: 'open-tab'; tab: string; label: string }
  const quick: QuickAction[] = []
  const seenTabs = new Set<string>()
  ;[
    { side: 'A', layer: conflict.layer_a, ref: conflict.ref_a },
    { side: 'B', layer: conflict.layer_b, ref: conflict.ref_b },
  ].forEach(({ side, layer, ref }) => {
    if (layer === 'fact') {
      quick.push({ kind: 'delete-fact', sideLabel: side, refId: ref })
    } else if (layer === 'plan' || layer === 'goal') {
      if (seenTabs.has(layer)) return
      seenTabs.add(layer)
      quick.push({
        kind: 'open-tab',
        tab: layer,
        label: t(`web.conflicts.openLayer.${layer}`),
      })
    }
  })

  return (
    <div className="bg-card/50 rounded-md border p-3">
      <div className="mb-2 flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <AlertTriangle
            className={`size-3.5 ${
              conflict.severity === 'high' ? 'text-destructive' : ''
            }`}
          />
          <Badge variant={severityTone === 'danger' ? 'danger' : 'muted'}>
            {t(`web.conflicts.severity.${conflict.severity}`)}
          </Badge>
          <span className="text-[10px] font-mono text-muted-foreground">
            {conflict.layer_a}:{shortRef(conflict.ref_a)} ⟷{' '}
            {conflict.layer_b}:{shortRef(conflict.ref_b)}
          </span>
        </div>
        <div className="flex items-center gap-1">
          <Button
            size="sm"
            variant="outline"
            onClick={() => onDecide('accepted')}
            disabled={disabled}
            className="h-7 text-[11px]"
          >
            <Check className="mr-1 size-3" />
            {t('web.conflicts.accept')}
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => onDecide('dismissed')}
            disabled={disabled}
            className="h-7 text-[11px]"
          >
            <X className="mr-1 size-3" />
            {t('web.conflicts.dismiss')}
          </Button>
        </div>
      </div>
      <p className="text-[12px] whitespace-pre-wrap">{conflict.evidence}</p>
      {quick.length > 0 && (
        <div className="border-border/50 mt-2 flex items-center gap-2 border-t pt-2 text-[11px]">
          <span className="text-muted-foreground">
            {t('web.conflicts.quickActions')}
          </span>
          {quick.map((q, i) => {
            if (q.kind === 'delete-fact') {
              return (
                <Button
                  key={`del-${q.refId}`}
                  size="sm"
                  variant="outline"
                  onClick={() => onDeleteFact(q.refId)}
                  disabled={disabled}
                  className="h-6 text-[10px] font-mono"
                  title={q.refId}
                >
                  {t('web.conflicts.deleteFactSide', {
                    side: q.sideLabel,
                    ref: shortRef(q.refId),
                  })}
                </Button>
              )
            }
            return (
              <Button
                key={`tab-${q.tab}-${i}`}
                size="sm"
                variant="ghost"
                onClick={() => onJumpTab?.(q.tab)}
                disabled={!onJumpTab}
                className="h-6 text-[10px]"
              >
                {q.label}
              </Button>
            )
          })}
        </div>
      )}
    </div>
  )
}

function shortRef(ref: string): string {
  if (ref.length <= 12) return ref
  return ref.slice(0, 8) + '…'
}
