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

interface ConflictsPanelProps {
  cwd: string
}

export function ConflictsPanel({ cwd }: ConflictsPanelProps) {
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
              disabled={decide.isPending}
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
  disabled: boolean
}

function ConflictCard({ conflict, onDecide, disabled }: ConflictCardProps) {
  const { t } = useTranslation()
  const severityTone =
    conflict.severity === 'high'
      ? 'danger'
      : conflict.severity === 'medium'
        ? 'warn'
        : 'muted'
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
    </div>
  )
}

function shortRef(ref: string): string {
  if (ref.length <= 12) return ref
  return ref.slice(0, 8) + '…'
}
