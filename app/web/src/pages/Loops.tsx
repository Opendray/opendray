import { useState, type FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Plus,
  Loader2,
  Pause,
  Play,
  Square,
  ListTree,
  Repeat,
  Target,
  RefreshCw,
} from 'lucide-react'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectTrigger,
  SelectContent,
  SelectItem,
  SelectValue,
} from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  listLoops,
  createLoop,
  pauseLoop,
  resumeLoop,
  stopLoop,
  listLoopRuns,
} from '@/lib/loops'
import {
  isTerminalLoopStatus,
  type CreateLoopRequest,
  type Loop,
  type LoopKind,
  type LoopStatus,
} from '@/lib/types'

const STATUS_CLASS: Record<LoopStatus, string> = {
  pending: 'bg-muted text-muted-foreground',
  running: 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400',
  paused: 'bg-amber-500/15 text-amber-600 dark:text-amber-400',
  done: 'bg-sky-500/15 text-sky-600 dark:text-sky-400',
  stopped: 'bg-muted text-muted-foreground',
  failed: 'bg-destructive/15 text-destructive',
  escalated: 'bg-orange-500/15 text-orange-600 dark:text-orange-400',
}

export function LoopsPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [createOpen, setCreateOpen] = useState(false)
  const [runsFor, setRunsFor] = useState<Loop | null>(null)

  const { data: loops, isLoading } = useQuery({
    queryKey: ['loops'],
    queryFn: listLoops,
    // No per-loop WS in Phase 1 — poll for live status/iteration updates.
    refetchInterval: 4_000,
  })

  const invalidate = () => qc.invalidateQueries({ queryKey: ['loops'] })

  const pause = useMutation({
    mutationFn: pauseLoop,
    onSuccess: invalidate,
    onError: (e: Error) => toast.error(e.message),
  })
  const resume = useMutation({
    mutationFn: resumeLoop,
    onSuccess: invalidate,
    onError: (e: Error) => toast.error(e.message),
  })
  const stop = useMutation({
    mutationFn: stopLoop,
    onSuccess: invalidate,
    onError: (e: Error) => toast.error(e.message),
  })

  return (
    <div className="flex h-full flex-col bg-background">
      <header className="flex items-center justify-between border-b border-border px-6 py-4">
        <div>
          <h1 className="text-lg font-semibold">{t('web.loops.title')}</h1>
          <p className="text-sm text-muted-foreground">
            {t('web.loops.subtitle')}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="icon" onClick={invalidate}>
            <RefreshCw />
          </Button>
          <Button onClick={() => setCreateOpen(true)}>
            <Plus />
            {t('web.loops.create')}
          </Button>
        </div>
      </header>

      <ScrollArea className="flex-1">
        <div className="mx-auto flex max-w-3xl flex-col gap-3 p-6">
          {isLoading && (
            <div className="flex justify-center py-12 text-muted-foreground">
              <Loader2 className="animate-spin" />
            </div>
          )}
          {!isLoading && (loops?.length ?? 0) === 0 && (
            <div className="rounded-lg border border-dashed border-border py-16 text-center text-sm text-muted-foreground">
              {t('web.loops.empty')}
            </div>
          )}
          {loops?.map((loop) => (
            <LoopCard
              key={loop.id}
              loop={loop}
              onPause={() => pause.mutate(loop.id)}
              onResume={() => resume.mutate(loop.id)}
              onStop={() => stop.mutate(loop.id)}
              onDetails={() => setRunsFor(loop)}
            />
          ))}
        </div>
      </ScrollArea>

      <CreateLoopDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onCreated={() => {
          invalidate()
          setCreateOpen(false)
        }}
      />
      <RunsDialog loop={runsFor} onClose={() => setRunsFor(null)} />
    </div>
  )
}

function LoopCard({
  loop,
  onPause,
  onResume,
  onStop,
  onDetails,
}: {
  loop: Loop
  onPause: () => void
  onResume: () => void
  onStop: () => void
  onDetails: () => void
}) {
  const { t } = useTranslation()
  const terminal = isTerminalLoopStatus(loop.status)
  const KindIcon = loop.kind === 'goal' ? Target : Repeat
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <KindIcon className="size-4 shrink-0 text-muted-foreground" />
            <span className="text-sm font-medium">
              {t(`web.loops.kind.${loop.kind}`)}
            </span>
            <span
              className={`rounded px-1.5 py-0.5 text-[11px] font-medium ${STATUS_CLASS[loop.status]}`}
            >
              {t(`web.loops.status.${loop.status}`)}
            </span>
            {loop.origin === 'integration' && (
              <span className="rounded bg-muted px-1.5 py-0.5 text-[11px] text-muted-foreground">
                {t('web.loops.origin.integration')}
              </span>
            )}
          </div>
          <p className="mt-1 truncate font-mono text-xs text-muted-foreground">
            {loop.session_id}
          </p>
          {loop.goal && (
            <p className="mt-1 line-clamp-2 text-sm">{loop.goal}</p>
          )}
          {loop.last_reason && (
            <p className="mt-1 line-clamp-2 text-xs text-muted-foreground">
              {loop.last_verdict ? `${loop.last_verdict} — ` : ''}
              {loop.last_reason}
            </p>
          )}
        </div>
        <div className="shrink-0 text-right text-xs text-muted-foreground">
          {t('web.loops.iterationOf', {
            n: loop.iteration,
            max: loop.max_iterations,
          })}
        </div>
      </div>
      <div className="mt-3 flex items-center gap-2">
        {loop.status === 'running' && (
          <Button variant="outline" size="sm" onClick={onPause}>
            <Pause />
            {t('web.loops.action.pause')}
          </Button>
        )}
        {loop.status === 'paused' && (
          <Button variant="outline" size="sm" onClick={onResume}>
            <Play />
            {t('web.loops.action.resume')}
          </Button>
        )}
        {!terminal && (
          <Button variant="outline" size="sm" onClick={onStop}>
            <Square />
            {t('web.loops.action.stop')}
          </Button>
        )}
        <Button variant="ghost" size="sm" onClick={onDetails}>
          <ListTree />
          {t('web.loops.action.details')}
        </Button>
      </div>
    </div>
  )
}

const DEFAULTS = {
  kind: 'goal' as LoopKind,
  intervalSeconds: 60,
  maxIterations: 20,
  durationMinutes: 60,
  failureCap: 3,
}

function CreateLoopDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean
  onOpenChange: (v: boolean) => void
  onCreated: () => void
}) {
  const { t } = useTranslation()
  const [sessionId, setSessionId] = useState('')
  const [kind, setKind] = useState<LoopKind>(DEFAULTS.kind)
  const [prompt, setPrompt] = useState('')
  const [goal, setGoal] = useState('')
  const [intervalSeconds, setIntervalSeconds] = useState(DEFAULTS.intervalSeconds)
  const [maxIterations, setMaxIterations] = useState(DEFAULTS.maxIterations)
  const [durationMinutes, setDurationMinutes] = useState(DEFAULTS.durationMinutes)
  const [failureCap, setFailureCap] = useState(DEFAULTS.failureCap)

  const create = useMutation({
    mutationFn: createLoop,
    onSuccess: () => {
      toast.success(t('web.loops.createdToast'))
      reset()
      onCreated()
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const reset = () => {
    setSessionId('')
    setKind(DEFAULTS.kind)
    setPrompt('')
    setGoal('')
    setIntervalSeconds(DEFAULTS.intervalSeconds)
    setMaxIterations(DEFAULTS.maxIterations)
    setDurationMinutes(DEFAULTS.durationMinutes)
    setFailureCap(DEFAULTS.failureCap)
  }

  const submit = (e: FormEvent) => {
    e.preventDefault()
    const deadline = new Date(
      Date.now() + Math.max(1, durationMinutes) * 60_000,
    ).toISOString()
    const req: CreateLoopRequest = {
      session_id: sessionId.trim(),
      kind,
      prompt: prompt.trim(),
      max_iterations: maxIterations,
      deadline_at: deadline,
      failure_cap: failureCap,
    }
    if (kind === 'goal') {
      req.goal = goal.trim()
    } else {
      req.interval_seconds = intervalSeconds
    }
    create.mutate(req)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('web.loops.create')}</DialogTitle>
          <DialogDescription>{t('web.loops.createHint')}</DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="loop-session">{t('web.loops.field.session')}</Label>
            <Input
              id="loop-session"
              value={sessionId}
              onChange={(e) => setSessionId(e.target.value)}
              placeholder={t('web.loops.field.sessionPlaceholder')}
              required
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>{t('web.loops.field.kind')}</Label>
            <Select value={kind} onValueChange={(v) => setKind(v as LoopKind)}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="goal">
                  {t('web.loops.kind.goal')} — {t('web.loops.kindHint.goal')}
                </SelectItem>
                <SelectItem value="interval">
                  {t('web.loops.kind.interval')} —{' '}
                  {t('web.loops.kindHint.interval')}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          {kind === 'goal' && (
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="loop-goal">{t('web.loops.field.goal')}</Label>
              <Textarea
                id="loop-goal"
                value={goal}
                onChange={(e) => setGoal(e.target.value)}
                placeholder={t('web.loops.field.goalPlaceholder')}
                rows={2}
                required
              />
            </div>
          )}

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="loop-prompt">
              {kind === 'goal'
                ? t('web.loops.field.seedPrompt')
                : t('web.loops.field.prompt')}
            </Label>
            <Textarea
              id="loop-prompt"
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder={t('web.loops.field.promptPlaceholder')}
              rows={2}
              required
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            {kind === 'interval' && (
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="loop-interval">
                  {t('web.loops.field.intervalSeconds')}
                </Label>
                <Input
                  id="loop-interval"
                  type="number"
                  min={30}
                  value={intervalSeconds}
                  onChange={(e) => setIntervalSeconds(Number(e.target.value))}
                />
              </div>
            )}
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="loop-maxiter">
                {t('web.loops.field.maxIterations')}
              </Label>
              <Input
                id="loop-maxiter"
                type="number"
                min={1}
                value={maxIterations}
                onChange={(e) => setMaxIterations(Number(e.target.value))}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="loop-duration">
                {t('web.loops.field.durationMinutes')}
              </Label>
              <Input
                id="loop-duration"
                type="number"
                min={1}
                value={durationMinutes}
                onChange={(e) => setDurationMinutes(Number(e.target.value))}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="loop-failcap">
                {t('web.loops.field.failureCap')}
              </Label>
              <Input
                id="loop-failcap"
                type="number"
                min={1}
                value={failureCap}
                onChange={(e) => setFailureCap(Number(e.target.value))}
              />
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              onClick={() => onOpenChange(false)}
            >
              {t('web.loops.action.cancel')}
            </Button>
            <Button type="submit" disabled={create.isPending}>
              {create.isPending && <Loader2 className="animate-spin" />}
              {t('web.loops.action.create')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function RunsDialog({
  loop,
  onClose,
}: {
  loop: Loop | null
  onClose: () => void
}) {
  const { t } = useTranslation()
  const { data: runs, isLoading } = useQuery({
    queryKey: ['loops', loop?.id, 'runs'],
    queryFn: () => listLoopRuns(loop!.id),
    enabled: !!loop,
    refetchInterval: 4_000,
  })

  return (
    <Dialog open={!!loop} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{t('web.loops.runs.title')}</DialogTitle>
          <DialogDescription className="font-mono text-xs">
            {loop?.id}
          </DialogDescription>
        </DialogHeader>
        <ScrollArea className="max-h-[60vh]">
          {isLoading && (
            <div className="flex justify-center py-8 text-muted-foreground">
              <Loader2 className="animate-spin" />
            </div>
          )}
          {!isLoading && (runs?.length ?? 0) === 0 && (
            <p className="py-8 text-center text-sm text-muted-foreground">
              {t('web.loops.runs.empty')}
            </p>
          )}
          <div className="flex flex-col gap-2">
            {runs?.map((run) => (
              <div
                key={run.id}
                className="rounded-md border border-border p-2.5 text-sm"
              >
                <div className="flex items-center justify-between">
                  <span className="font-medium">
                    {t('web.loops.runs.iteration', { n: run.iteration })}
                  </span>
                  {run.verdict && (
                    <span className="text-xs text-muted-foreground">
                      {run.verdict}
                    </span>
                  )}
                </div>
                {run.reason && (
                  <p className="mt-1 text-xs text-muted-foreground">
                    {run.reason}
                  </p>
                )}
              </div>
            ))}
          </div>
        </ScrollArea>
      </DialogContent>
    </Dialog>
  )
}
