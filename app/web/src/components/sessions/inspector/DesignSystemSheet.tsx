import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Palette, X, Sun, Moon, ScanSearch, LayoutDashboard } from 'lucide-react'

import {
  CANVAS_DESIGN_TOKENS,
  CANVAS_THEMED_TOKENS,
  CANVAS_PALETTES,
  cssVarForToken,
  getDesignSystem,
  setDesignSystem,
  runDesignTask,
  type CanvasPalette,
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
// Two things make it usable by someone who is not a designer:
//   • a row of complete starting palettes — choosing colours from memory is the
//     part people can't do, so a new project starts from a checked light+dark
//     pair and tweaks, instead of facing thirteen empty fields;
//   • colour fields carry a real colour picker, so nobody has to type a hex.
// The accurate route for an EXISTING project is still to ask the agent to read
// the real theme and write it through the canvas_design MCP tool.

const COLOR_TOKENS = new Set<string>(CANVAS_THEMED_TOKENS.filter((k) => k !== 'shadow'))
const THEMED = new Set<string>(CANVAS_THEMED_TOKENS)

interface DesignSystemSheetProps {
  sessionId: string
  cwd: string
  onClose: () => void
}

export function DesignSystemSheet({ sessionId, cwd, onClose }: DesignSystemSheetProps) {
  const { t } = useTranslation()
  const [tokens, setTokens] = useState<Record<string, string>>({})
  const [dark, setDark] = useState<Record<string, string>>({})
  const [notes, setNotes] = useState('')
  const [theme, setTheme] = useState<'light' | 'dark'>('light')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [tasking, setTasking] = useState(false)

  useEffect(() => {
    let cancelled = false
    void getDesignSystem(cwd)
      .then((d) => {
        if (cancelled) return
        setTokens(d.tokens)
        setDark(d.tokens_dark)
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

  function applyPalette(p: CanvasPalette) {
    setTokens(p.tokens)
    setDark(p.tokensDark)
  }

  // Hand a one-click job to the agent: read the project's real theme and record
  // it, or draw the system as a canvas. The prompt lives in the gateway.
  async function task(kind: 'extract' | 'showcase') {
    setTasking(true)
    try {
      await runDesignTask(sessionId, cwd, kind)
      toast.success(t('web.sessions.inspector.canvas.designTaskSent'))
      onClose()
    } catch (e) {
      toast.error(t('web.sessions.inspector.canvas.designTaskFailed'), {
        description: (e as Error).message,
      })
    } finally {
      setTasking(false)
    }
  }

  const editing = theme === 'light' ? tokens : dark
  const setEditing = theme === 'light' ? setTokens : setDark
  // Dark inherits from light where it's blank, which is exactly what the
  // gateway does — so show that value greyed rather than an empty box.
  const inherited = (key: string) => (theme === 'dark' ? tokens[key] ?? '' : '')

  async function save() {
    setSaving(true)
    try {
      const saved = await setDesignSystem(cwd, { tokens, tokensDark: dark, notes })
      setTokens(saved.tokens)
      setDark(saved.tokens_dark)
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
      <div className="flex max-h-[88vh] w-full max-w-lg flex-col rounded-md border border-border bg-background shadow-lg">
        <div className="flex items-center gap-2 border-b border-border px-4 py-3">
          <Palette className="size-4 text-primary" />
          <div className="min-w-0 flex-1">
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
              {/* Hand the job to the agent. Reading the project's real theme
                  beats any hand-entry, and seeing the system beats reading it. */}
              <div className="mb-3 flex flex-wrap gap-1.5">
                <button
                  type="button"
                  onClick={() => void task('extract')}
                  disabled={tasking}
                  className="flex items-center gap-1.5 rounded-sm border border-primary/50 px-2 py-1 text-[11px] text-foreground hover:border-primary disabled:opacity-50"
                  title={t('web.sessions.inspector.canvas.extractHint')}
                >
                  <ScanSearch className="size-3" />
                  {t('web.sessions.inspector.canvas.extractBtn')}
                </button>
                <button
                  type="button"
                  onClick={() => void task('showcase')}
                  disabled={tasking}
                  className="flex items-center gap-1.5 rounded-sm border border-border px-2 py-1 text-[11px] text-muted-foreground hover:text-foreground disabled:opacity-50"
                  title={t('web.sessions.inspector.canvas.showcaseHint')}
                >
                  <LayoutDashboard className="size-3" />
                  {t('web.sessions.inspector.canvas.showcaseBtn')}
                </button>
              </div>

              {/* Starting palettes — the answer to "I don't know what colours
                  to pick". One click fills both themes with a checked pair. */}
              <div className="mb-1 text-[11px] text-muted-foreground">
                {t('web.sessions.inspector.canvas.paletteLabel')}
              </div>
              <div className="mb-3 flex flex-wrap gap-1.5">
                {CANVAS_PALETTES.map((p) => (
                  <button
                    key={p.id}
                    type="button"
                    onClick={() => applyPalette(p)}
                    title={t(`web.sessions.inspector.canvas.palette_${p.id}`)}
                    className={cn(
                      'flex items-center gap-1.5 rounded-sm border px-2 py-1 text-[11px] hover:border-primary/50',
                      tokens.primary === p.tokens.primary
                        ? 'border-primary/60 bg-card text-foreground'
                        : 'border-border text-muted-foreground',
                    )}
                  >
                    <span
                      className="size-3 rounded-full"
                      style={{ background: p.accent }}
                    />
                    {t(`web.sessions.inspector.canvas.palette_${p.id}`)}
                  </button>
                ))}
              </div>

              {/* Theme switch — colour tokens hold two sets. */}
              <div className="mb-2 flex items-center gap-1">
                <ThemeTab
                  active={theme === 'light'}
                  onClick={() => setTheme('light')}
                  icon={<Sun className="size-3" />}
                  label={t('web.sessions.inspector.canvas.themeLight')}
                />
                <ThemeTab
                  active={theme === 'dark'}
                  onClick={() => setTheme('dark')}
                  icon={<Moon className="size-3" />}
                  label={t('web.sessions.inspector.canvas.themeDark')}
                />
                <span className="ml-1 text-[10px] text-muted-foreground">
                  {theme === 'dark'
                    ? t('web.sessions.inspector.canvas.themeDarkHint')
                    : t('web.sessions.inspector.canvas.themeLightHint')}
                </span>
              </div>

              <div className="grid grid-cols-[104px_1fr] items-center gap-x-2 gap-y-1.5">
                {CANVAS_DESIGN_TOKENS.filter(
                  (key) => theme === 'light' || THEMED.has(key),
                ).map((key) => (
                  <FieldRow
                    key={key}
                    label={t(`web.sessions.inspector.canvas.token_${key}`)}
                    cssVar={cssVarForToken(key)}
                    value={editing[key] ?? ''}
                    placeholder={inherited(key) || cssVarForToken(key)}
                    isColor={COLOR_TOKENS.has(key)}
                    onChange={(v) => setEditing((prev) => ({ ...prev, [key]: v }))}
                  />
                ))}
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

function ThemeTab({
  active,
  onClick,
  icon,
  label,
}: {
  active: boolean
  onClick: () => void
  icon: React.ReactNode
  label: string
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'flex items-center gap-1 rounded-sm border px-2 py-0.5 text-[11px]',
        active
          ? 'border-primary/50 bg-card text-foreground'
          : 'border-border text-muted-foreground hover:text-foreground',
      )}
    >
      {icon}
      {label}
    </button>
  )
}

/** Resolve ANY CSS colour — hex, rgb(), hsl(), oklch(), a named colour — to the
 *  #rrggbb the native picker needs. Projects write colours in whatever notation
 *  their theme uses (this one uses oklch), and the operator should still get a
 *  picker rather than a bare text box. Returns null only if it isn't a colour. */
function asHex(value: string): string | null {
  const v = value.trim()
  if (!v) return null
  if (/^#([0-9a-f]{3}|[0-9a-f]{6})$/i.test(v)) {
    return v.length === 4
      ? `#${v[1]}${v[1]}${v[2]}${v[2]}${v[3]}${v[3]}`
      : v.toLowerCase()
  }
  if (typeof document === 'undefined') return null
  const probe = document.createElement('span')
  probe.style.color = ''
  probe.style.color = v
  if (!probe.style.color) return null // the browser rejected it — not a colour
  probe.style.display = 'none'
  document.body.appendChild(probe)
  const resolved = getComputedStyle(probe).color
  probe.remove()
  const m = resolved.match(/^rgba?\(([^)]+)\)/)
  if (!m) return null
  const [r, g, b] = m[1].split(',').map((n) => Math.round(parseFloat(n)))
  if ([r, g, b].some((n) => Number.isNaN(n))) return null
  return `#${[r, g, b].map((n) => n.toString(16).padStart(2, '0')).join('')}`
}

function FieldRow({
  label,
  cssVar,
  value,
  placeholder,
  isColor,
  onChange,
}: {
  label: string
  cssVar: string
  value: string
  placeholder: string
  isColor: boolean
  onChange: (v: string) => void
}) {
  const hex = isColor ? asHex(value) ?? asHex(placeholder) : null
  return (
    <>
      <label className="truncate text-[11px] text-muted-foreground" title={cssVar}>
        {label}
      </label>
      <div className="flex items-center gap-1.5">
        {isColor && (
          // Always a real picker — clicking the swatch opens the OS colour
          // chooser, so nobody ever has to know a colour code. Empty fields
          // start from the inherited/placeholder colour.
          <input
            type="color"
            value={hex ?? '#888888'}
            onChange={(e) => onChange(e.target.value)}
            aria-label={label}
            title={label}
            className="size-5 shrink-0 cursor-pointer rounded-sm border border-border bg-transparent p-0"
          />
        )}
        <input
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          className="min-w-0 flex-1 rounded-sm border border-border bg-transparent px-1.5 py-0.5 font-mono text-[11px] focus:border-primary focus:outline-none"
        />
      </div>
    </>
  )
}
