import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Loader2,
  Camera,
  RotateCcw,
  Trash2,
  GitCompare,
  AlertTriangle,
  FileWarning,
  CheckCircle2,
} from 'lucide-react'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import {
  listCheckpoints,
  captureCheckpoint,
  readCheckpointDiff,
  restoreCheckpoint,
  deleteCheckpoint,
  type Checkpoint,
} from '@/lib/checkpoints'
import type { Session } from '@/lib/types'

interface CheckpointsPanelProps {
  session: Session
}

// CheckpointsPanel surfaces a session's context checkpoints (uncommitted
// git diff + untracked files + input history). It can capture one on demand
// and, per checkpoint, view the stored diff or restore it back onto the cwd
// under the gateway's strict guards (HEAD match, clean tracked tree, dry-run
// — a guard failure comes back as a 409 the panel shows verbatim).
export function CheckpointsPanel({ session }: CheckpointsPanelProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const sessionId = session.id
  const key = ['checkpoints', sessionId]

  const list = useQuery({
    queryKey: key,
    queryFn: () => listCheckpoints(sessionId),
    refetchInterval: 15_000,
  })

  const capture = useMutation({
    mutationFn: () => captureCheckpoint(sessionId),
    onSuccess: (cp) => {
      qc.invalidateQueries({ queryKey: key })
      const clean =
        cp.is_git && cp.diff_bytes === 0 && cp.untracked_files === 0
      toast.success(t('web.sessions.inspector.checkpoints.captured'), {
        description: !cp.is_git
          ? t('web.sessions.inspector.checkpoints.capturedNonGit')
          : clean
            ? t('web.sessions.inspector.checkpoints.capturedClean')
            : t('web.sessions.inspector.checkpoints.capturedGit', {
                diff: cp.diff_bytes,
                files: cp.untracked_files,
              }),
      })
    },
    onError: (e: Error) => toast.error(e.message),
  })

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <p className="text-[11px] text-muted-foreground/80 leading-snug pr-2">
          {t('web.sessions.inspector.checkpoints.blurb')}
        </p>
        <Button
          size="sm"
          variant="secondary"
          className="h-7 shrink-0 gap-1.5 text-[12px]"
          disabled={capture.isPending}
          onClick={() => capture.mutate()}
        >
          {capture.isPending ? (
            <Loader2 className="size-3 animate-spin" />
          ) : (
            <Camera className="size-3" />
          )}
          {t('web.sessions.inspector.checkpoints.capture')}
        </Button>
      </div>

      {list.isLoading ? (
        <div className="flex items-center gap-2 text-[12px] text-muted-foreground py-3">
          <Loader2 className="size-3 animate-spin" />
          {t('web.sessions.inspector.checkpoints.loading')}
        </div>
      ) : list.error ? (
        <div className="text-[12px] text-state-failed py-3">
          {(list.error as Error).message}
        </div>
      ) : (list.data?.length ?? 0) === 0 ? (
        <div className="text-[12px] text-muted-foreground/70 py-6 text-center">
          {t('web.sessions.inspector.checkpoints.empty')}
        </div>
      ) : (
        <ul className="flex flex-col gap-2">
          {list.data!.map((cp) => (
            <CheckpointCard key={cp.id} cp={cp} invalidate={key} />
          ))}
        </ul>
      )}
    </div>
  )
}

function CheckpointCard({
  cp,
  invalidate,
}: {
  cp: Checkpoint
  invalidate: unknown[]
}) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [showDiff, setShowDiff] = useState(false)
  const [confirmRestore, setConfirmRestore] = useState(false)

  const diff = useQuery({
    queryKey: ['checkpoint-diff', cp.id],
    queryFn: () => readCheckpointDiff(cp.id),
    enabled: showDiff && cp.diff_bytes > 0,
  })

  const restore = useMutation({
    mutationFn: () => restoreCheckpoint(cp.id),
    onSuccess: (res) => {
      setConfirmRestore(false)
      toast.success(t('web.sessions.inspector.checkpoints.restored'), {
        description: t('web.sessions.inspector.checkpoints.restoredDetail', {
          diff: res.diff_applied ? 1 : 0,
          files: res.untracked_restored,
          skipped: res.untracked_skipped?.length ?? 0,
        }),
      })
    },
    // Guard failures come back as 409 with git's explanation — show it.
    onError: (e: Error) => toast.error(e.message),
  })

  const remove = useMutation({
    mutationFn: () => deleteCheckpoint(cp.id),
    onSuccess: () => qc.invalidateQueries({ queryKey: invalidate }),
    onError: (e: Error) => toast.error(e.message),
  })

  const when = new Date(cp.created_at).toLocaleString()
  // A git checkpoint with no tracked diff and no untracked files captured
  // nothing restorable — the working tree was clean. Surfaced explicitly so
  // an empty diff doesn't read as "the feature didn't work".
  const isClean =
    cp.is_git && cp.diff_bytes === 0 && cp.untracked_files === 0

  return (
    <li className="rounded-md border border-border bg-card/30 px-2.5 py-2 flex flex-col gap-1.5">
      <div className="flex items-center gap-2">
        <span
          className={cn(
            'text-[10px] font-medium px-1.5 py-0.5 rounded uppercase tracking-wide',
            cp.trigger === 'manual'
              ? 'bg-primary/10 text-primary'
              : 'bg-amber-500/10 text-amber-500',
          )}
        >
          {t(`web.sessions.inspector.checkpoints.trigger.${cp.trigger}`)}
        </span>
        <span className="text-[11px] text-muted-foreground font-mono truncate">
          {when}
        </span>
        {cp.truncated && (
          <span
            title={t('web.sessions.inspector.checkpoints.truncatedHint')}
            className="ml-auto flex items-center gap-1 text-[10px] text-amber-500"
          >
            <FileWarning className="size-3" />
            {t('web.sessions.inspector.checkpoints.truncated')}
          </span>
        )}
      </div>

      <div className="text-[11px] text-muted-foreground/85 font-mono flex flex-wrap items-center gap-x-3 gap-y-0.5">
        {cp.is_git ? (
          isClean ? (
            <span className="flex items-center gap-1 text-muted-foreground/60">
              {cp.git_head ? cp.git_head.slice(0, 8) : '—'}
              <CheckCircle2 className="size-3 text-state-running/70" />
              {t('web.sessions.inspector.checkpoints.clean')}
            </span>
          ) : (
            <>
              <span>{cp.git_head ? cp.git_head.slice(0, 8) : '—'}</span>
              <span>Δ {cp.diff_bytes}B</span>
              <span>+{cp.untracked_files}f</span>
            </>
          )
        ) : (
          <span className="text-muted-foreground/60">
            {t('web.sessions.inspector.checkpoints.nonGit')}
          </span>
        )}
        <span>⌨ {cp.input_bytes}B</span>
      </div>

      {cp.note && (
        <p className="text-[11px] text-muted-foreground/70 italic break-words">
          {cp.note.split('\n')[0]}
        </p>
      )}

      <div className="flex items-center gap-1.5 pt-0.5">
        {cp.diff_bytes > 0 && (
          <Button
            size="sm"
            variant="ghost"
            className="h-6 px-2 gap-1 text-[11px]"
            onClick={() => setShowDiff((s) => !s)}
          >
            <GitCompare className="size-3" />
            {showDiff
              ? t('web.sessions.inspector.checkpoints.hideDiff')
              : t('web.sessions.inspector.checkpoints.viewDiff')}
          </Button>
        )}
        {cp.is_git && (
          <Button
            size="sm"
            variant="ghost"
            className="h-6 px-2 gap-1 text-[11px]"
            disabled={restore.isPending}
            onClick={() => setConfirmRestore((c) => !c)}
          >
            {restore.isPending ? (
              <Loader2 className="size-3 animate-spin" />
            ) : (
              <RotateCcw className="size-3" />
            )}
            {t('web.sessions.inspector.checkpoints.restore')}
          </Button>
        )}
        <Button
          size="sm"
          variant="ghost"
          className="h-6 px-2 gap-1 text-[11px] text-state-failed hover:text-state-failed ml-auto"
          disabled={remove.isPending}
          onClick={() => {
            if (window.confirm(t('web.sessions.inspector.checkpoints.deleteConfirm')))
              remove.mutate()
          }}
        >
          <Trash2 className="size-3" />
        </Button>
      </div>

      {confirmRestore && (
        <div className="rounded border border-amber-500/30 bg-amber-500/5 px-2 py-1.5 flex flex-col gap-1.5">
          <p className="text-[11px] text-amber-600 dark:text-amber-400 flex items-start gap-1.5">
            <AlertTriangle className="size-3 mt-0.5 shrink-0" />
            {t('web.sessions.inspector.checkpoints.restoreWarn')}
          </p>
          <div className="flex items-center gap-1.5">
            <Button
              size="sm"
              variant="secondary"
              className="h-6 px-2 text-[11px]"
              disabled={restore.isPending}
              onClick={() => restore.mutate()}
            >
              {t('web.sessions.inspector.checkpoints.restoreConfirm')}
            </Button>
            <Button
              size="sm"
              variant="ghost"
              className="h-6 px-2 text-[11px]"
              onClick={() => setConfirmRestore(false)}
            >
              {t('common.cancel')}
            </Button>
          </div>
        </div>
      )}

      {showDiff && cp.diff_bytes > 0 && (
        <div className="rounded border border-border bg-background/60 overflow-auto max-h-72">
          {diff.isLoading ? (
            <div className="flex items-center gap-2 text-[11px] text-muted-foreground p-2">
              <Loader2 className="size-3 animate-spin" />
              {t('web.sessions.inspector.checkpoints.loading')}
            </div>
          ) : diff.error ? (
            <div className="text-[11px] text-state-failed p-2">
              {(diff.error as Error).message}
            </div>
          ) : (
            <DiffText text={diff.data ?? ''} />
          )}
        </div>
      )}
    </li>
  )
}

// DiffText renders a unified diff with per-line color coding — a compact
// inline variant of DiffViewer's DiffBody, prefix-classified only.
function DiffText({ text }: { text: string }) {
  const MAX = 4000
  const all = text.split('\n')
  const lines = all.length > MAX ? all.slice(0, MAX) : all
  return (
    <div className="font-mono text-[11px] leading-[1.5] py-1">
      {lines.map((line, i) => {
        let clsName = 'text-foreground/85'
        if (line.startsWith('+++') || line.startsWith('---'))
          clsName = 'text-muted-foreground/70 font-medium'
        else if (line.startsWith('@@')) clsName = 'text-sky-400/90 bg-sky-500/5'
        else if (line.startsWith('diff --git') || line.startsWith('index '))
          clsName = 'text-muted-foreground/60'
        else if (line.startsWith('+'))
          clsName = 'text-state-running bg-state-running/5'
        else if (line.startsWith('-'))
          clsName = 'text-state-failed bg-state-failed/5'
        return (
          <div key={i} className={cn('px-2 whitespace-pre', clsName)}>
            {line || ' '}
          </div>
        )
      })}
      {all.length > MAX && (
        <div className="px-2 text-[10px] text-muted-foreground/60 py-1">
          … {all.length - MAX} more lines
        </div>
      )}
    </div>
  )
}
