import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog, DialogContent, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { updateRoundTable, type RoundTable } from '@/lib/roundtable'

// Live role/framing editor — reassign each member's persona and the shared
// framing directive on an active round table as the topic evolves. Mirrors the
// create dialog's persona field but for an existing chat; changes take effect
// on the next reply (the backend re-reads seats + framing each turn).
const PERSONA_PRESET_KEYS = [
  'security',
  'performance',
  'ux',
  'skeptic',
  'pragmatist',
] as const

interface Props {
  rt: RoundTable
  open: boolean
  onClose: () => void
}

export function RolesDialog({ rt, open, onClose }: Props) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [framing, setFraming] = useState(rt.framing ?? '')
  const [personas, setPersonas] = useState<Record<string, string>>(() =>
    Object.fromEntries(rt.seats.map((s) => [s.provider, s.persona ?? ''])),
  )

  const save = useMutation({
    mutationFn: () =>
      updateRoundTable(rt.id, {
        framing: framing.trim(),
        seats: rt.seats.map((s) => ({
          ...s,
          persona: personas[s.provider]?.trim() || undefined,
        })),
      }),
    onSuccess: () => {
      toast.success(t('web.roundTable.detail.rolesSaved'))
      qc.invalidateQueries({ queryKey: ['round-table', rt.id] })
      onClose()
    },
    onError: (e: Error) => toast.error(e.message),
  })

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (!o) onClose()
      }}
    >
      <DialogContent className="max-w-lg">
        <DialogTitle>{t('web.roundTable.detail.rolesTitle')}</DialogTitle>
        <p className="-mt-1 text-xs text-muted-foreground">
          {t('web.roundTable.detail.rolesHint')}
        </p>

        <div className="mt-2 flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="roles-framing">
              {t('web.roundTable.detail.rolesFraming')}
            </Label>
            <textarea
              id="roles-framing"
              value={framing}
              onChange={(e) => setFraming(e.target.value)}
              placeholder={t('web.roundTable.dialog.framingPlaceholder')}
              rows={3}
              className="w-full resize-none rounded-md border border-border bg-background px-2 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-ring"
            />
          </div>

          <div className="flex flex-col gap-3">
            {rt.seats.map((s) => (
              <div key={s.provider} className="flex flex-col gap-1">
                <Label className="capitalize">{s.provider}</Label>
                <textarea
                  value={personas[s.provider] ?? ''}
                  onChange={(e) =>
                    setPersonas((cur) => ({
                      ...cur,
                      [s.provider]: e.target.value,
                    }))
                  }
                  placeholder={t('web.roundTable.dialog.personaPlaceholder')}
                  rows={2}
                  className="w-full resize-none rounded-md border border-border bg-background px-2 py-1 text-xs focus:outline-none focus:ring-1 focus:ring-ring"
                />
                <div className="flex flex-wrap items-center gap-1">
                  {PERSONA_PRESET_KEYS.map((key) => {
                    const label = t(
                      `web.roundTable.dialog.personaPresets.${key}`,
                    )
                    return (
                      <button
                        key={key}
                        type="button"
                        onClick={() =>
                          setPersonas((cur) => ({ ...cur, [s.provider]: label }))
                        }
                        className="rounded-full border border-border bg-card px-1.5 py-0.5 text-[10px] text-muted-foreground transition-colors hover:text-foreground"
                      >
                        {label}
                      </button>
                    )
                  })}
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="mt-4 flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button
            size="sm"
            disabled={save.isPending}
            onClick={() => save.mutate()}
          >
            {t('web.roundTable.detail.rolesSave')}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
