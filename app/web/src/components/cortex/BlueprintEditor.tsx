// BlueprintEditor — edit a project's doc blueprint (Cortex Phase 3):
// add/remove/rename sections, switch maintainer modes, reorder, and ask
// the AI to propose a section set tailored to the project type. Changes
// only persist on Apply.

import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import {
  ArrowDown,
  ArrowUp,
  Loader2,
  Plus,
  Sparkles,
  Trash2,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  type BlueprintSection,
  type MaintainerMode,
  listBlueprintSections,
} from '@/lib/projectDocs'
import { applyBlueprint, proposeBlueprint } from '@/lib/cortex'

const SLUG_RE = /^[a-z][a-z0-9_]{1,47}$/

interface BlueprintEditorProps {
  cwd: string
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Called after Apply so the host refetches sections + docs. */
  onApplied: () => void
}

export function BlueprintEditor({ cwd, open, onOpenChange, onApplied }: BlueprintEditorProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [draft, setDraft] = useState<BlueprintSection[]>([])
  const [proposalNote, setProposalNote] = useState('')

  const sectionsQuery = useQuery({
    queryKey: ['blueprint', cwd],
    queryFn: () => listBlueprintSections(cwd),
    enabled: open && !!cwd,
  })

  // Re-seed the draft whenever the dialog (re)opens with fresh data.
  useEffect(() => {
    if (open && sectionsQuery.data) {
      setDraft(sectionsQuery.data.map((s) => ({ ...s })))
      setProposalNote('')
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, sectionsQuery.data])

  const propose = useMutation({
    mutationFn: () => proposeBlueprint(cwd),
    onSuccess: (p) => {
      setDraft(p.sections.map((s) => ({ ...s, cwd })))
      setProposalNote(
        t('web.cortex.blueprint.proposalNote', {
          type: p.project_type,
          reason: p.reason,
        }),
      )
    },
    onError: (e: Error) =>
      toast.error(t('web.cortex.blueprint.proposeFailed'), { description: e.message }),
  })

  const apply = useMutation({
    mutationFn: () => applyBlueprint(cwd, normalize(draft)),
    onSuccess: () => {
      toast.success(t('web.cortex.blueprint.appliedToast'))
      qc.invalidateQueries({ queryKey: ['blueprint', cwd] })
      onApplied()
      onOpenChange(false)
    },
    onError: (e: Error) =>
      toast.error(t('web.cortex.blueprint.applyFailed'), { description: e.message }),
  })

  const update = (i: number, patch: Partial<BlueprintSection>) =>
    setDraft((d) => d.map((s, j) => (j === i ? { ...s, ...patch } : s)))

  const move = (i: number, dir: -1 | 1) =>
    setDraft((d) => {
      const j = i + dir
      if (j < 0 || j >= d.length) return d
      const next = [...d]
      ;[next[i], next[j]] = [next[j], next[i]]
      return next.map((s, k) => ({ ...s, position: k }))
    })

  const addSection = () =>
    setDraft((d) => [
      ...d,
      {
        cwd,
        slug: '',
        title: '',
        description: '',
        position: d.length,
        maintainer_mode: 'ai' as MaintainerMode,
        prompt_hint: '',
        pinned: false,
        inject: true,
      },
    ])

  const invalid = draft.some(
    (s) =>
      !SLUG_RE.test(s.slug) || s.slug.startsWith('kb_') || !s.title.trim(),
  )

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-auto sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>{t('web.cortex.blueprint.title')}</DialogTitle>
          <DialogDescription>{t('web.cortex.blueprint.description')}</DialogDescription>
        </DialogHeader>

        <div className="flex items-center justify-between gap-2">
          <Button
            size="sm"
            variant="outline"
            disabled={propose.isPending}
            onClick={() => propose.mutate()}
            title={t('web.cortex.blueprint.proposeHint')}
          >
            {propose.isPending ? (
              <Loader2 className="mr-1 h-3 w-3 animate-spin" />
            ) : (
              <Sparkles className="mr-1 h-3 w-3" />
            )}
            {t('web.cortex.blueprint.propose')}
          </Button>
          <Button size="sm" variant="ghost" onClick={addSection}>
            <Plus className="mr-1 h-3 w-3" />
            {t('web.cortex.blueprint.addSection')}
          </Button>
        </div>

        {proposalNote && (
          <div className="rounded-md border border-blue-500/30 bg-blue-500/10 p-2.5 text-xs text-blue-300">
            {proposalNote}
          </div>
        )}

        {sectionsQuery.isLoading ? (
          <Loader2 className="mx-auto my-6 h-4 w-4 animate-spin" />
        ) : (
          <div className="space-y-2">
            {draft.map((s, i) => (
              <div key={`${s.slug}-${i}`} className="bg-card space-y-2 rounded-md border p-2.5">
                <div className="flex items-center gap-2">
                  <Input
                    value={s.slug}
                    disabled={s.pinned}
                    onChange={(e) => update(i, { slug: e.target.value })}
                    placeholder={t('web.cortex.blueprint.slugPlaceholder')}
                    className="h-7 w-40 font-mono text-xs"
                  />
                  <Input
                    value={s.title}
                    onChange={(e) => update(i, { title: e.target.value })}
                    placeholder={t('web.cortex.blueprint.titlePlaceholder')}
                    className="h-7 flex-1 text-xs"
                  />
                  <Select
                    value={s.maintainer_mode}
                    onValueChange={(v) => update(i, { maintainer_mode: v as MaintainerMode })}
                  >
                    <SelectTrigger className="h-7 w-28 text-xs">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="ai">{t('web.cortex.blueprint.mode.ai')}</SelectItem>
                      <SelectItem value="human">{t('web.cortex.blueprint.mode.human')}</SelectItem>
                      <SelectItem value="scanner">{t('web.cortex.blueprint.mode.scanner')}</SelectItem>
                    </SelectContent>
                  </Select>
                  <label className="text-muted-foreground flex items-center gap-1 text-[11px]">
                    <input
                      type="checkbox"
                      checked={s.inject}
                      onChange={(e) => update(i, { inject: e.target.checked })}
                    />
                    {t('web.cortex.blueprint.inject')}
                  </label>
                  <div className="flex items-center">
                    <Button size="sm" variant="ghost" className="h-6 w-6 p-0" onClick={() => move(i, -1)}>
                      <ArrowUp className="h-3 w-3" />
                    </Button>
                    <Button size="sm" variant="ghost" className="h-6 w-6 p-0" onClick={() => move(i, 1)}>
                      <ArrowDown className="h-3 w-3" />
                    </Button>
                    {s.pinned ? (
                      <Badge variant="muted" className="ml-1 text-[9px]">
                        {t('web.cortex.blueprint.reserved')}
                      </Badge>
                    ) : (
                      <Button
                        size="sm"
                        variant="ghost"
                        className="text-destructive h-6 w-6 p-0"
                        onClick={() => setDraft((d) => d.filter((_, j) => j !== i))}
                      >
                        <Trash2 className="h-3 w-3" />
                      </Button>
                    )}
                  </div>
                </div>
                <Input
                  value={s.prompt_hint ?? ''}
                  onChange={(e) => update(i, { prompt_hint: e.target.value })}
                  placeholder={t('web.cortex.blueprint.hintPlaceholder')}
                  className="h-7 text-xs"
                />
              </div>
            ))}
          </div>
        )}

        <p className="text-muted-foreground text-[11px]">
          {t('web.cortex.blueprint.deleteNote')}
        </p>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('web.cortex.blueprint.cancel')}
          </Button>
          <Button disabled={invalid || apply.isPending || draft.length === 0} onClick={() => apply.mutate()}>
            {apply.isPending && <Loader2 className="mr-1 h-3 w-3 animate-spin" />}
            {t('web.cortex.blueprint.apply')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function normalize(sections: BlueprintSection[]): BlueprintSection[] {
  return sections.map((s, i) => ({ ...s, position: i, slug: s.slug.trim() }))
}
