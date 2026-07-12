// Shared create/edit dialog for custom tasks. Lives here (rather than
// inside the Plugins page) so the session Task Runner can open it
// inline — pre-scoped to the current project's cwd — instead of
// bouncing the operator to the Plugins page to type the path by hand.
import { useEffect, useState, type FormEvent } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Textarea } from '@/components/ui/textarea'
import {
  createCustomTask,
  updateCustomTask,
  type CustomTask,
} from '@/lib/customTasks'

interface CustomTaskDialogProps {
  open: boolean
  onOpenChange: (v: boolean) => void
  mode: 'create' | 'edit'
  task?: CustomTask
  // Pre-fills the cwd field for a new task so operators creating a
  // task from within a project don't have to retype its path.
  initialCwd?: string
}

export function CustomTaskDialog({
  open,
  onOpenChange,
  mode,
  task,
  initialCwd,
}: CustomTaskDialogProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [name, setName] = useState(task?.name ?? '')
  const [command, setCommand] = useState(task?.command ?? '')
  const [description, setDescription] = useState(task?.description ?? '')
  const [cwd, setCwd] = useState(task?.cwd ?? initialCwd ?? '')

  useEffect(() => {
    setName(task?.name ?? '')
    setCommand(task?.command ?? '')
    setDescription(task?.description ?? '')
    setCwd(task?.cwd ?? initialCwd ?? '')
  }, [task?.id, initialCwd])

  const create = useMutation({
    mutationFn: () =>
      createCustomTask({ name, command, description, cwd: cwd || undefined }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['custom-tasks-all'] })
      qc.invalidateQueries({ queryKey: ['custom-tasks'] })
      toast.success(t('web.plugins.customTasks.dialog.addedToast'))
      onOpenChange(false)
    },
    onError: (err: Error) =>
      toast.error(t('web.plugins.customTasks.dialog.addFailedToast'), {
        description: err.message,
      }),
  })

  const update = useMutation({
    mutationFn: () =>
      updateCustomTask(task!.id, { name, command, description, cwd }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['custom-tasks-all'] })
      qc.invalidateQueries({ queryKey: ['custom-tasks'] })
      toast.success(t('web.plugins.customTasks.dialog.updatedToast'))
      onOpenChange(false)
    },
    onError: (err: Error) =>
      toast.error(t('web.plugins.customTasks.dialog.updateFailedToast'), {
        description: err.message,
      }),
  })

  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (mode === 'create') create.mutate()
    else update.mutate()
  }

  const busy = create.isPending || update.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {mode === 'create'
              ? t('web.plugins.customTasks.dialog.addTitle')
              : t('web.plugins.customTasks.dialog.editTitle', { name: task?.name })}
          </DialogTitle>
          <DialogDescription>
            {t('web.plugins.customTasks.dialog.description')}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="flex flex-col gap-3 mt-2">
          <div className="space-y-1.5">
            <Label htmlFor="task-name">
              {t('web.plugins.customTasks.dialog.nameLabel')}
            </Label>
            <Input
              id="task-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t('web.plugins.customTasks.dialog.namePlaceholder')}
              required
              autoFocus
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="task-cmd">
              {t('web.plugins.customTasks.dialog.commandLabel')}
            </Label>
            <Textarea
              id="task-cmd"
              value={command}
              onChange={(e) => setCommand(e.target.value)}
              placeholder={t('web.plugins.customTasks.dialog.commandPlaceholder')}
              rows={2}
              required
              className="font-mono text-[12px]"
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="task-desc">
              {t('web.plugins.customTasks.dialog.descLabel')}
            </Label>
            <Input
              id="task-desc"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder={t('web.plugins.customTasks.dialog.descPlaceholder')}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="task-cwd">
              {t('web.plugins.customTasks.dialog.cwdLabel')}
            </Label>
            <Input
              id="task-cwd"
              value={cwd}
              onChange={(e) => setCwd(e.target.value)}
              placeholder={t('web.plugins.customTasks.dialog.cwdPlaceholder')}
              className="font-mono text-[12px]"
            />
            <p className="text-[10.5px] text-muted-foreground/80">
              {t('web.plugins.customTasks.dialog.cwdHint')}
            </p>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => onOpenChange(false)}
              disabled={busy}
            >
              {t('web.plugins.common.cancel')}
            </Button>
            <Button type="submit" variant="accent" size="sm" disabled={busy}>
              {busy && <Loader2 className="size-3.5 animate-spin" />}
              {mode === 'create'
                ? t('web.plugins.common.add')
                : t('web.plugins.common.save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
