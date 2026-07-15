import { useEffect, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { FolderOpen, Loader2, Play, Plus, Sparkles, Trash2 } from 'lucide-react'

import { Dialog, DialogContent, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { ProviderIcon } from '@/components/ProviderIcon'
import { FileBrowserDialog } from '@/components/sessions/FileBrowserDialog'
import { RunStepDialog } from './RunStepDialog'
import {
  draftPlan,
  setPlan,
  updateRoundTable,
  type PlanStep,
  type RoundTable,
} from '@/lib/roundtable'

// Role-based execution plan editor — draft an ordered, member-assigned plan
// from the discussion, tweak it, then run each step (spawns a real session in
// the shared project cwd so the specialists collaborate through the working
// tree). Operator-driven: you advance step by step.
interface Props {
  rt: RoundTable
  open: boolean
  onClose: () => void
}

export function PlanDialog({ rt, open, onClose }: Props) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [steps, setSteps] = useState<PlanStep[]>(rt.plan ?? [])
  // Which step's run-options dialog is open (account + bypass), or null.
  const [runIndex, setRunIndex] = useState<number | null>(null)
  // Local cwd input when the table has no project bound yet (draft-then-bind).
  const [cwd, setCwd] = useState(rt.cwd ?? '')
  const [browserOpen, setBrowserOpen] = useState(false)

  // Keep local edits in sync when the drafted plan arrives via polling.
  useEffect(() => {
    setSteps(rt.plan ?? [])
  }, [rt.plan])
  useEffect(() => {
    setCwd(rt.cwd ?? '')
  }, [rt.cwd])

  const seatProviders = rt.seats.map((s) => s.provider)
  const hasCwd = !!rt.cwd
  const dirty = JSON.stringify(steps) !== JSON.stringify(rt.plan ?? [])

  const draft = useMutation({
    mutationFn: () => draftPlan(rt.id),
    onSuccess: () => {
      toast.success(t('web.roundTable.plan.drafting'))
      qc.invalidateQueries({ queryKey: ['round-table', rt.id] })
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const save = useMutation({
    mutationFn: () => setPlan(rt.id, steps),
    onSuccess: () => {
      toast.success(t('web.roundTable.plan.saved'))
      qc.invalidateQueries({ queryKey: ['round-table', rt.id] })
    },
    onError: (e: Error) => toast.error(e.message),
  })

  // Bind the shared project working dir after the fact — a table drafted with
  // no cwd otherwise has no runnable steps.
  const bind = useMutation({
    mutationFn: () => updateRoundTable(rt.id, { cwd: cwd.trim() }),
    onSuccess: () => {
      toast.success(t('web.roundTable.plan.projectBound'))
      qc.invalidateQueries({ queryKey: ['round-table', rt.id] })
    },
    onError: (e: Error) => toast.error(e.message),
  })

  // Running a step first persists edits (so the seed matches the screen), then
  // opens the run-options dialog (account + bypass) for that step.
  const openRun = async (index: number) => {
    if (dirty) {
      try {
        await setPlan(rt.id, steps)
        qc.invalidateQueries({ queryKey: ['round-table', rt.id] })
      } catch (e) {
        toast.error((e as Error).message)
        return
      }
    }
    setRunIndex(index)
  }

  const updateStep = (i: number, patch: Partial<PlanStep>) =>
    setSteps((cur) => cur.map((s, j) => (j === i ? { ...s, ...patch } : s)))
  const removeStep = (i: number) =>
    setSteps((cur) => cur.filter((_, j) => j !== i))
  const addStep = () =>
    setSteps((cur) => [
      ...cur,
      { assignee: seatProviders[0] ?? '', task: '', status: 'pending' },
    ])

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (!o) onClose()
      }}
    >
      <DialogContent className="max-w-xl">
        <DialogTitle>{t('web.roundTable.plan.title')}</DialogTitle>
        <p className="-mt-1 text-xs text-muted-foreground">
          {t('web.roundTable.plan.hint')}
        </p>

        <div className="mt-2 flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={draft.isPending || messagesEmpty(rt)}
            onClick={() => draft.mutate()}
          >
            {draft.isPending ? (
              <Loader2 className="size-3.5 animate-spin" />
            ) : (
              <Sparkles className="size-3.5" />
            )}
            {t('web.roundTable.plan.draft')}
          </Button>
          <Button variant="ghost" size="sm" onClick={addStep}>
            <Plus className="size-3.5" />
            {t('web.roundTable.plan.addStep')}
          </Button>
        </div>

        {!hasCwd && (
          <div className="mt-2 flex flex-col gap-1.5 rounded-md border border-state-failed/40 bg-state-failed/5 p-2">
            <p className="text-[11px] text-state-failed">
              {t('web.roundTable.plan.needProject')}
            </p>
            <div className="flex gap-1.5">
              <Input
                value={cwd}
                onChange={(e) => setCwd(e.target.value)}
                placeholder={t('web.roundTable.dialog.cwdPlaceholder')}
                className="flex-1 font-mono text-xs"
              />
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => setBrowserOpen(true)}
                className="shrink-0 gap-1"
              >
                <FolderOpen className="size-3.5" />
                {t('web.roundTable.dialog.browse')}
              </Button>
              <Button
                size="sm"
                className="shrink-0"
                disabled={!cwd.trim() || bind.isPending}
                onClick={() => bind.mutate()}
              >
                {bind.isPending ? (
                  <Loader2 className="size-3.5 animate-spin" />
                ) : null}
                {t('web.roundTable.plan.bindProject')}
              </Button>
            </div>
          </div>
        )}

        <div className="mt-3 flex max-h-[52vh] flex-col gap-2 overflow-y-auto">
          {steps.length === 0 && (
            <p className="py-6 text-center text-xs text-muted-foreground">
              {t('web.roundTable.plan.empty')}
            </p>
          )}
          {steps.map((step, i) => (
            <div
              key={i}
              className="flex flex-col gap-1.5 rounded-md border border-border p-2"
            >
              <div className="flex items-center gap-2">
                <span className="text-xs font-medium text-muted-foreground">
                  {i + 1}
                </span>
                <Select
                  value={step.assignee}
                  onValueChange={(v) => updateStep(i, { assignee: v })}
                >
                  <SelectTrigger className="h-7 w-40 text-xs">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {seatProviders.map((p) => (
                      <SelectItem key={p} value={p}>
                        <span className="flex items-center gap-1.5">
                          <ProviderIcon providerId={p} size={14} />
                          {p}
                        </span>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <StatusPill status={step.status} />
                <div className="ml-auto flex items-center gap-1">
                  {step.session_id && (
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-7"
                      onClick={() => {
                        onClose()
                        navigate({
                          to: '/sessions',
                          search: { open: step.session_id },
                        })
                      }}
                    >
                      {t('web.roundTable.plan.openSession')}
                    </Button>
                  )}
                  <Button
                    variant="accent"
                    size="sm"
                    className="h-7"
                    disabled={!hasCwd || !step.task.trim()}
                    onClick={() => openRun(i)}
                  >
                    <Play className="size-3" />
                    {step.status === 'done'
                      ? t('web.roundTable.plan.rerun')
                      : t('web.roundTable.plan.run')}
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="size-7"
                    onClick={() => removeStep(i)}
                  >
                    <Trash2 className="size-3.5" />
                  </Button>
                </div>
              </div>
              <textarea
                value={step.task}
                onChange={(e) => updateStep(i, { task: e.target.value })}
                placeholder={t('web.roundTable.plan.taskPlaceholder')}
                rows={2}
                className="w-full resize-none rounded-md border border-border bg-background px-2 py-1 text-xs focus:outline-none focus:ring-1 focus:ring-ring"
              />
            </div>
          ))}
        </div>

        <div className="mt-4 flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button
            size="sm"
            disabled={!dirty || save.isPending}
            onClick={() => save.mutate()}
          >
            {t('web.roundTable.plan.save')}
          </Button>
        </div>

        <FileBrowserDialog
          open={browserOpen}
          onOpenChange={setBrowserOpen}
          initialPath={cwd.trim() || undefined}
          onSelect={(p) => setCwd(p)}
        />

        {runIndex !== null && steps[runIndex] && (
          <RunStepDialog
            rt={rt}
            index={runIndex}
            step={steps[runIndex]}
            open={runIndex !== null}
            onClose={() => setRunIndex(null)}
            onLaunched={(sessionId) => {
              setRunIndex(null)
              onClose()
              navigate({ to: '/sessions', search: { open: sessionId } })
            }}
          />
        )}
      </DialogContent>
    </Dialog>
  )
}

function messagesEmpty(rt: RoundTable): boolean {
  // Draft needs a discussion to summarize; the topic is a cheap proxy for
  // "something was said" (auto-derived from the first message).
  return !rt.topic
}

function StatusPill({ status }: { status: PlanStep['status'] }) {
  const { t } = useTranslation()
  const cls =
    status === 'done'
      ? 'border-emerald-500/40 text-emerald-500'
      : status === 'running'
        ? 'border-sky-500/40 text-sky-500'
        : 'border-border text-muted-foreground'
  return (
    <span className={`rounded-full border px-1.5 py-0.5 text-[10px] ${cls}`}>
      {t(`web.roundTable.plan.${status}`)}
    </span>
  )
}
