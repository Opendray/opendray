import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Palette, X } from 'lucide-react'

import {
  CANVAS_DESIGN_TOKENS,
  cssVarForToken,
  getDesignSystem,
  setDesignSystem,
} from '@/lib/canvas'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'

// DesignSystemSheet — the project's canvas design contract, in one editor.
//
// It exists because an agent asked to "design a login page" re-invents colours,
// type and spacing every time, so successive canvases drift apart. Whatever is
// set here is put into every canvas request AND injected into each rendered
// canvas as CSS variables, which is why the agent is told to write
// var(--od-primary) rather than a hex value: change a token later and every
// canvas that used the variable restyles itself.
//
// The operator can fill it in by hand, but the accurate way is to ask the agent
// to read the project's real theme (tailwind config, CSS custom properties,
// existing components) and write it via the canvas_design MCP tool.

// Colour-ish tokens get a swatch; the rest are plain text.
const COLOR_TOKENS = new Set([
  'primary',
  'secondary',
  'background',
  'surface',
  'text',
  'muted',
  'border',
])

interface DesignSystemSheetProps {
  cwd: string
  onClose: () => void
}

export function DesignSystemSheet({ cwd, onClose }: DesignSystemSheetProps) {
  const { t } = useTranslation()
  const [tokens, setTokens] = useState<Record<string, string>>({})
  const [notes, setNotes] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    let cancelled = false
    void getDesignSystem(cwd)
      .then((d) => {
        if (cancelled) return
        setTokens(d.tokens)
        setNotes(d.notes)
      })
      .catch((e: unknown) => {
        if (!cancelled) toast.error((e as Error).message)
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [cwd])

  async function save() {
    setSaving(true)
    try {
      const saved = await setDesignSystem(cwd, { tokens, notes })
      setTokens(saved.tokens)
      setNotes(saved.notes)
      toast.success(t('web.sessions.inspector.canvas.designSaved'))
      onClose()
    } catch (e) {
      toast.error(t('web.sessions.inspector.canvas.designSaveFailed'), {
        description: (e as Error).message,
      })
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="flex max-h-[85vh] w-full max-w-lg flex-col rounded-md border border-border bg-background shadow-lg">
        <div className="flex items-center gap-2 border-b border-border px-4 py-3">
          <Palette className="size-4 text-primary" />
          <div className="flex-1 min-w-0">
            <div className="text-[13px] font-medium">
              {t('web.sessions.inspector.canvas.designTitle')}
            </div>
            <div className="text-[11px] text-muted-foreground">
              {t('web.sessions.inspector.canvas.designBlurb')}
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label={t('web.sessions.inspector.canvas.designClose')}
            className="text-muted-foreground hover:text-foreground"
          >
            <X className="size-4" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto px-4 py-3">
          {loading ? (
            <p className="text-[12px] text-muted-foreground">
              {t('web.sessions.inspector.canvas.designLoading')}
            </p>
          ) : (
            <>
              <div className="grid grid-cols-[110px_1fr] items-center gap-x-2 gap-y-1.5">
                {CANVAS_DESIGN_TOKENS.map((key) => {
                  const value = tokens[key] ?? ''
                  const isColor = COLOR_TOKENS.has(key)
                  return (
                    <FieldRow
                      key={key}
                      label={t(`web.sessions.inspector.canvas.token_${key}`)}
                      cssVar={cssVarForToken(key)}
                      value={value}
                      isColor={isColor}
                      onChange={(v) =>
                        setTokens((prev) => ({ ...prev, [key]: v }))
                      }
                    />
                  )
                })}
              </div>

              <div className="mt-4">
                <label className="mb-1 block text-[11px] text-muted-foreground">
                  {t('web.sessions.inspector.canvas.designNotesLabel')}
                </label>
                <textarea
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  rows={4}
                  placeholder={t('web.sessions.inspector.canvas.designNotesPlaceholder')}
                  className="w-full resize-y rounded-sm border border-border bg-transparent px-2 py-1 text-[12px] focus:border-primary focus:outline-none"
                />
              </div>

              <p className="mt-3 text-[11px] leading-relaxed text-muted-foreground">
                {t('web.sessions.inspector.canvas.designAgentHint')}
              </p>
            </>
          )}
        </div>

        <div className="flex items-center justify-end gap-2 border-t border-border px-4 py-3">
          <Button size="sm" variant="ghost" onClick={onClose} disabled={saving}>
            {t('web.sessions.inspector.canvas.designCancel')}
          </Button>
          <Button size="sm" onClick={() => void save()} disabled={saving || loading}>
            {t('web.sessions.inspector.canvas.designSave')}
          </Button>
        </div>
      </div>
    </div>
  )
}

function FieldRow({
  label,
  cssVar,
  value,
  isColor,
  onChange,
}: {
  label: string
  cssVar: string
  value: string
  isColor: boolean
  onChange: (v: string) => void
}) {
  return (
    <>
      <label className="truncate text-[11px] text-muted-foreground" title={cssVar}>
        {label}
      </label>
      <div className="flex items-center gap-1.5">
        {isColor && (
          <span
            className={cn(
              'size-4 shrink-0 rounded-sm border border-border',
              !value && 'bg-muted',
            )}
            style={value ? { background: value } : undefined}
          />
        )}
        <input
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={cssVar}
          className="min-w-0 flex-1 rounded-sm border border-border bg-transparent px-1.5 py-0.5 font-mono text-[11px] focus:border-primary focus:outline-none"
        />
      </div>
    </>
  )
}
