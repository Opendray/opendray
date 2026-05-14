import { useEffect, useState } from 'react'
import { ListTree } from 'lucide-react'
import { Trans, useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'
import type { OutlineHeading } from '@/lib/outline'

interface OutlineSidebarProps {
  headings: OutlineHeading[]
  // onJump receives the heading slug — caller scrolls the editor /
  // preview pane to the matching `<h*>` element by id.
  onJump: (slug: string) => void
  // editorScrollEl is watched (when provided) to highlight the
  // currently-visible heading. Pass the scroll container that holds
  // the rendered preview; outline finds nearest h* above the top.
  editorScrollEl?: HTMLElement | null
}

// OutlineSidebar renders the document's heading hierarchy as a
// nested-indent list. Active heading (closest above the scroll top)
// is highlighted live so the user knows where they are in long docs.
export function OutlineSidebar({
  headings,
  onJump,
  editorScrollEl,
}: OutlineSidebarProps) {
  const [activeSlug, setActiveSlug] = useState<string | null>(null)

  // Track scroll position and update the active heading. Re-bind when
  // the scroll container changes (e.g. the editor is remounted on a
  // new note).
  useEffect(() => {
    if (!editorScrollEl) {
      setActiveSlug(null)
      return
    }
    const compute = () => {
      const top = editorScrollEl.getBoundingClientRect().top
      let chosen: string | null = null
      for (const h of headings) {
        const el = editorScrollEl.querySelector<HTMLElement>(
          `[data-outline-id="${cssEscape(h.slug)}"]`,
        )
        if (!el) continue
        // A heading counts as "visible at top" once its top edge has
        // scrolled past 24px below the container top.
        if (el.getBoundingClientRect().top - top <= 24) {
          chosen = h.slug
        } else {
          break
        }
      }
      setActiveSlug(chosen)
    }
    compute()
    editorScrollEl.addEventListener('scroll', compute, { passive: true })
    const ro = new ResizeObserver(compute)
    ro.observe(editorScrollEl)
    return () => {
      editorScrollEl.removeEventListener('scroll', compute)
      ro.disconnect()
    }
  }, [editorScrollEl, headings])

  if (headings.length === 0) {
    return (
      <div className="px-3 py-3 text-[11px] text-muted-foreground/60">
        <Trans
          i18nKey="web.notes.outline.empty"
          components={{ 1: <code className="text-[10px]" /> }}
        />
      </div>
    )
  }

  // Normalise depth so the leftmost level in the doc maps to 0 and
  // each subsequent level adds 12px indent. Saves space when an
  // author uses h2 as the top — without normalisation, h2 would
  // already be indented one level for no reason.
  const minLevel = Math.min(...headings.map((h) => h.level))

  return (
    <div className="flex flex-col font-mono text-[11.5px] py-1">
      {headings.map((h) => {
        const indent = (h.level - minLevel) * 12
        const isActive = h.slug === activeSlug
        return (
          <button
            key={h.slug}
            type="button"
            onClick={() => onJump(h.slug)}
            style={{ paddingLeft: `${indent + 8}px` }}
            className={cn(
              'flex items-baseline gap-2 py-0.5 pr-2 text-left rounded-sm transition-colors',
              'hover:bg-card',
              isActive
                ? 'text-foreground bg-card border-l-2 border-state-running'
                : 'text-muted-foreground/85',
              h.level === 1 && 'font-semibold',
            )}
            title={h.text}
          >
            <span className="truncate">{h.text}</span>
          </button>
        )
      })}
    </div>
  )
}

// CSS.escape polyfill for identifiers we control — used for
// data-outline-id attribute selectors.
function cssEscape(s: string): string {
  if (typeof CSS !== 'undefined' && typeof CSS.escape === 'function') {
    return CSS.escape(s)
  }
  return s.replace(/[^a-zA-Z0-9_-]/g, (c) => `\\${c}`)
}

export function OutlineHeader({ count }: { count: number }) {
  const { t } = useTranslation()
  return (
    <div className="flex items-center gap-1.5 px-3 py-2 border-b border-border text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium">
      <ListTree className="size-3" />
      {t('web.notes.outline.label')}
      <span className="text-muted-foreground/50 normal-case tracking-normal">
        · {count}
      </span>
    </div>
  )
}
