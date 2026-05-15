// ConflictsPanel — M-PC operator inbox for cross-layer
// contradictions surfaced by the daily detector. Each row shows
// the two conflicting items + the LLM's evidence + accept/dismiss
// buttons.

import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import {
  AlertTriangle,
  Check,
  Loader2,
  RefreshCw,
  Trash2,
  X,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  type MemoryConflict,
  decideMemoryConflict,
  detectMemoryConflicts,
  listMemoryConflicts,
} from '@/lib/memoryConflicts'
import { deleteMemory, getMemory } from '@/lib/memory'

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

  // Pending fact-delete the dialog is currently confirming. We
  // hold the conflict + which side ("A" / "B") so the dialog can
  // render the picked-side fact prominently AND the other-side
  // fact alongside it (the operator's actual decision is "which
  // of these two is wrong" — they need to see both).
  const [pendingDelete, setPendingDelete] = useState<{
    conflict: MemoryConflict
    side: 'A' | 'B'
  } | null>(null)

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
      setPendingDelete(null)
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
              onRequestDeleteFact={(side) =>
                setPendingDelete({ conflict: c, side })
              }
              onJumpTab={onJumpTab}
            />
          ))}
        </div>
      </div>

      <DeleteFactConfirmDialog
        pending={pendingDelete}
        busy={deleteFactAndAccept.isPending}
        onCancel={() => setPendingDelete(null)}
        onConfirm={(factId) => {
          if (pendingDelete) {
            deleteFactAndAccept.mutate({
              conflictId: pendingDelete.conflict.id,
              factId,
            })
          }
        }}
      />
    </div>
  )
}

// ─── Delete-fact confirmation dialog ───────────────────────────
// Loads the two conflicting fact rows so the operator sees the
// full text of both sides before pulling the trigger. The chosen
// side is highlighted; the other is shown muted for reference —
// the actual decision is "which of these two claims is wrong",
// which needs both visible at once.

interface DeleteFactConfirmDialogProps {
  pending: { conflict: MemoryConflict; side: 'A' | 'B' } | null
  busy: boolean
  onCancel: () => void
  onConfirm: (factId: string) => void
}

function DeleteFactConfirmDialog({
  pending,
  busy,
  onCancel,
  onConfirm,
}: DeleteFactConfirmDialogProps) {
  const { t } = useTranslation()

  const open = pending !== null
  const conflict = pending?.conflict ?? null
  const side = pending?.side ?? 'A'

  // Which ref is being deleted vs which is the "other side" the
  // operator should compare against. Computed once per render so
  // the queries below depend on stable strings.
  const targetRef =
    conflict == null
      ? null
      : side === 'A'
        ? conflict.ref_a
        : conflict.ref_b
  const otherRef =
    conflict == null
      ? null
      : side === 'A'
        ? conflict.ref_b
        : conflict.ref_a
  const otherLayer =
    conflict == null
      ? null
      : side === 'A'
        ? conflict.layer_b
        : conflict.layer_a

  const targetQuery = useQuery({
    queryKey: ['memory-by-id', targetRef],
    queryFn: () => getMemory(targetRef as string),
    enabled: open && !!targetRef,
  })
  // Only fetch the other side if it's a fact — plan/goal/journal
  // refs aren't memories.id and would 404. We fall back to "see
  // the {layer} tab" copy in those cases.
  const otherIsFact = otherLayer === 'fact'
  const otherQuery = useQuery({
    queryKey: ['memory-by-id', otherRef],
    queryFn: () => getMemory(otherRef as string),
    enabled: open && otherIsFact && !!otherRef,
  })

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) onCancel()
      }}
    >
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {t('web.conflicts.confirmDelete.title', { side })}
          </DialogTitle>
          <DialogDescription>
            {t('web.conflicts.confirmDelete.description')}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          <div className="bg-destructive/10 border-destructive/30 rounded-md border p-3">
            <div className="text-destructive mb-1 flex items-center gap-1 text-[10px] font-semibold">
              <Trash2 className="size-3" />
              {t('web.conflicts.confirmDelete.targetLabel', { side })}
              <span className="ml-1 font-mono text-muted-foreground">
                {targetRef}
              </span>
            </div>
            <p className="text-[12px] whitespace-pre-wrap">
              {targetQuery.isLoading
                ? t('web.conflicts.confirmDelete.loading')
                : targetQuery.isError
                  ? t('web.conflicts.confirmDelete.loadError')
                  : (targetQuery.data?.text ?? '')}
            </p>
          </div>

          <div className="bg-muted/30 border-border rounded-md border p-3">
            <div className="text-muted-foreground mb-1 flex items-center gap-1 text-[10px] font-semibold">
              {t('web.conflicts.confirmDelete.keepLabel', {
                side: side === 'A' ? 'B' : 'A',
              })}
              <span className="ml-1 font-mono">{otherRef}</span>
              {!otherIsFact && (
                <Badge variant="muted" className="ml-1 text-[9px]">
                  {otherLayer}
                </Badge>
              )}
            </div>
            <p className="text-muted-foreground text-[12px] whitespace-pre-wrap">
              {otherIsFact
                ? otherQuery.isLoading
                  ? t('web.conflicts.confirmDelete.loading')
                  : otherQuery.isError
                    ? t('web.conflicts.confirmDelete.loadError')
                    : (otherQuery.data?.text ?? '')
                : t('web.conflicts.confirmDelete.nonFactOther', {
                    layer: otherLayer ?? '',
                  })}
            </p>
          </div>

          {conflict && conflict.evidence && (
            <div className="text-muted-foreground text-[11px] italic">
              {t('web.conflicts.confirmDelete.evidenceLabel')} {conflict.evidence}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="ghost" onClick={onCancel} disabled={busy}>
            {t('web.conflicts.confirmDelete.cancel')}
          </Button>
          <Button
            variant="destructive"
            onClick={() => {
              if (targetRef) onConfirm(targetRef)
            }}
            disabled={busy || !targetRef || targetQuery.isLoading}
          >
            {busy ? (
              <Loader2 className="mr-1 size-3 animate-spin" />
            ) : (
              <Trash2 className="mr-1 size-3" />
            )}
            {t('web.conflicts.confirmDelete.confirm', { side })}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

interface ConflictCardProps {
  conflict: MemoryConflict
  onDecide: (action: 'accepted' | 'dismissed') => void
  /**
   * The card fires this when an operator clicks one of the delete
   * buttons. Confirmation + the actual mutation live in the parent
   * (ConflictsPanel) so the dialog can fetch fact text once instead
   * of every card pre-loading it.
   */
  onRequestDeleteFact: (side: 'A' | 'B') => void
  onJumpTab?: (tab: string) => void
  disabled: boolean
}

function ConflictCard({
  conflict,
  onDecide,
  onRequestDeleteFact,
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
  //   layer=fact   → "Delete A/B (mem_…)" → opens confirm dialog
  //                  showing full fact text + the other side
  //   layer=plan   → "Open plan editor" (jumps to the Plan tab)
  //   layer=goal   → "Open goal editor"
  //   layer=journal → no quick-action; operator dismisses or edits
  //
  // When both sides are facts (the common case for hard
  // contradictions like "DB is Postgres" vs "DB is SQLite"), we
  // emit TWO delete buttons — one per fact id — so the operator
  // can pick which side is wrong. Each button opens a confirm
  // dialog so they can verify the fact text before committing.
  // Plan/goal "Open editor" buttons dedupe on layer (one button
  // per unique tab target).
  type QuickAction =
    | { kind: 'delete-fact'; side: 'A' | 'B'; refId: string }
    | { kind: 'open-tab'; tab: string; label: string }
  const quick: QuickAction[] = []
  const seenTabs = new Set<string>()
  ;[
    { side: 'A' as const, layer: conflict.layer_a, ref: conflict.ref_a },
    { side: 'B' as const, layer: conflict.layer_b, ref: conflict.ref_b },
  ].forEach(({ side, layer, ref }) => {
    if (layer === 'fact') {
      quick.push({ kind: 'delete-fact', side, refId: ref })
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
                  onClick={() => onRequestDeleteFact(q.side)}
                  disabled={disabled}
                  className="h-6 text-[10px] font-mono"
                  title={q.refId}
                >
                  {t('web.conflicts.deleteFactSide', {
                    side: q.side,
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
