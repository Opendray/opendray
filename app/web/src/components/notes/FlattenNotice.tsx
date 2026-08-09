import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { FolderTree, Loader2 } from 'lucide-react'
import { toast } from 'sonner'

import { flattenVault } from '@/lib/notes'
import type { FlattenResult } from '@/lib/notes'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

// A vault that still files every project under `projects/` can be
// converted, and the operator has no way to know that unless something
// says so where they are already looking.
//
// It is an offer, not a nag: dismissing it is remembered, and it never
// appears for a vault that is already flat or has nothing to move.
//
// Nothing moves without a preview. The dialog runs the migration as a
// dry run first and shows the actual list — including what it refuses
// to touch and why — because "convert my documents" is not a thing
// anyone should agree to from a one-line description.

const DISMISS_KEY = 'vault.flatten.dismissed'

export function FlattenNotice() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [dismissed, setDismissed] = useState(() => {
    try {
      return localStorage.getItem(DISMISS_KEY) === '1'
    } catch {
      return false
    }
  })
  const [plan, setPlan] = useState<FlattenResult | null>(null)

  const preview = useMutation({
    mutationFn: () => flattenVault(false),
    onSuccess: (res) => setPlan(res),
    onError: (e: Error) => toast.error(e.message),
  })

  const apply = useMutation({
    mutationFn: () => flattenVault(true),
    onSuccess: (res) => {
      setPlan(null)
      setDismissed(true)
      const moved = res.moves?.length ?? 0
      toast.success(t('web.notes.flatten.done', { count: moved }))
      // The tree, the listing and the layout badge are all now wrong.
      qc.invalidateQueries({ queryKey: ['notes-list'] })
      qc.invalidateQueries({ queryKey: ['notes-info'] })
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const dismiss = () => {
    setDismissed(true)
    try {
      localStorage.setItem(DISMISS_KEY, '1')
    } catch {
      // A blocked localStorage costs the memory of the choice only.
    }
  }

  if (dismissed) return null

  const moves = plan?.moves ?? []
  const skips = plan?.skips ?? []

  return (
    <>
      <div className="flex items-center gap-2 px-3 py-1.5 border-b border-border bg-muted/40 text-[11.5px]">
        <FolderTree className="size-3.5 text-muted-foreground shrink-0" />
        <span className="text-muted-foreground min-w-0 truncate">
          {t('web.notes.flatten.notice')}
        </span>
        <Button
          type="button"
          size="sm"
          variant="outline"
          className="h-6 px-2 text-[11px] ml-auto shrink-0"
          disabled={preview.isPending}
          onClick={() => preview.mutate()}
        >
          {preview.isPending && (
            <Loader2 className="size-3 animate-spin mr-1" />
          )}
          {t('web.notes.flatten.preview')}
        </Button>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          className="h-6 px-2 text-[11px] shrink-0"
          onClick={dismiss}
        >
          {t('web.notes.flatten.dismiss')}
        </Button>
      </div>

      <Dialog open={plan != null} onOpenChange={(v) => !v && setPlan(null)}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>{t('web.notes.flatten.title')}</DialogTitle>
            <DialogDescription>
              {t('web.notes.flatten.description')}
            </DialogDescription>
          </DialogHeader>

          <div className="max-h-[45vh] overflow-y-auto text-[11.5px] font-mono space-y-0.5">
            {moves.map((m) => (
              <div key={m.from} className="flex items-baseline gap-1.5">
                <span className="text-muted-foreground truncate">{m.from}</span>
                <span className="text-muted-foreground/50 shrink-0">→</span>
                <span className="truncate">{m.to}</span>
              </div>
            ))}
            {moves.length === 0 && (
              <p className="text-muted-foreground font-sans">
                {t('web.notes.flatten.nothingToMove')}
              </p>
            )}
          </div>

          {skips.length > 0 && (
            <div className="border-t border-border pt-2 space-y-0.5">
              <p className="text-[11px] text-muted-foreground font-sans">
                {t('web.notes.flatten.skipped', { count: skips.length })}
              </p>
              <div className="max-h-[18vh] overflow-y-auto text-[11px] font-mono">
                {skips.map((s) => (
                  <div key={s.path} className="text-muted-foreground truncate">
                    {s.path} — {s.reason}
                  </div>
                ))}
              </div>
            </div>
          )}

          <DialogFooter className="sm:justify-between gap-2">
            <p className="text-[11px] text-muted-foreground">
              {t('web.notes.flatten.restartHint')}
            </p>
            <div className="flex gap-2">
              <Button
                type="button"
                variant="outline"
                onClick={() => setPlan(null)}
              >
                {t('common.cancel')}
              </Button>
              <Button
                type="button"
                disabled={moves.length === 0 || apply.isPending}
                onClick={() => apply.mutate()}
              >
                {apply.isPending && (
                  <Loader2 className="size-3.5 animate-spin mr-1" />
                )}
                {t('web.notes.flatten.convert', { count: moves.length })}
              </Button>
            </div>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
