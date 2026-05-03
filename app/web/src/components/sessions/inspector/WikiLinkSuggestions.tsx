import { useEffect, useMemo, useRef, useState } from 'react'
import { FileText, FilePlus2 } from 'lucide-react'

import { cn } from '@/lib/utils'
import type { Note } from '@/lib/notes'

// What the textarea owner detects and feeds to us. When `query` is
// null the suggestion popup stays hidden.
export interface WikiLinkContext {
  // The substring between `[[` and the caret. Empty string = `[[|`
  // (cursor right after the opening), pop the full list.
  query: string
  // Absolute caret coords in the page (px). The popup positions
  // itself just below.
  caret: { top: number; left: number; height: number }
}

interface WikiLinkSuggestionsProps {
  context: WikiLinkContext | null
  notes: Note[]
  // Path of the note currently being edited; we hide it from the
  // candidate list (a note linking to itself is rarely the intent).
  excludePath?: string
  // onSelect receives either an existing note or a special "create"
  // sentinel (path is the synthesised target). The textarea owner
  // splices the `[[Title]]` text into the buffer.
  onSelect: (sel: { display: string; path: string; create?: boolean }) => void
  onDismiss: () => void
}

const MAX_RESULTS = 8

// WikiLinkSuggestions is the floating popup shown while the user is
// typing inside `[[...`. Pure controlled component — the parent owns
// query / caret state and decides when to mount us. We just render
// the candidate list and route keyboard events.
export function WikiLinkSuggestions({
  context,
  notes,
  excludePath,
  onSelect,
  onDismiss,
}: WikiLinkSuggestionsProps) {
  const [activeIdx, setActiveIdx] = useState(0)
  const containerRef = useRef<HTMLDivElement>(null)

  // Filter + score candidates. Basename matches rank above path
  // matches; title (frontmatter / H1) matches in between.
  const candidates = useMemo(() => {
    if (!context) return []
    const q = context.query.trim().toLowerCase()
    const out: Array<{ note: Note; score: number; display: string }> = []
    for (const n of notes) {
      if (n.path === excludePath) continue
      const base = (n.path.split('/').pop() ?? '').replace(/\.md$/i, '')
      const display = n.title || base
      let score = -1
      const lp = n.path.toLowerCase()
      const lb = base.toLowerCase()
      const lt = (n.title || '').toLowerCase()
      if (q === '') {
        score = 0
      } else if (lb === q || lt === q) score = 100
      else if (lb.startsWith(q)) score = 80
      else if (lt.startsWith(q)) score = 70
      else if (lb.includes(q)) score = 50
      else if (lt.includes(q)) score = 40
      else if (lp.includes(q)) score = 20
      if (score >= 0) out.push({ note: n, score, display })
    }
    out.sort((a, b) => {
      if (a.score !== b.score) return b.score - a.score
      // Tie-break: most recently modified first.
      return (
        new Date(b.note.modified).getTime() -
        new Date(a.note.modified).getTime()
      )
    })
    return out.slice(0, MAX_RESULTS)
  }, [context, notes, excludePath])

  // Reset highlight whenever the query changes.
  useEffect(() => {
    setActiveIdx(0)
  }, [context?.query])

  // When the user has typed something but nothing matched, offer a
  // "create new note" entry. Path defaults to library/<query>.md so
  // it goes somewhere predictable.
  const showCreate =
    context !== null &&
    context.query.trim().length > 0 &&
    !candidates.some(
      (c) =>
        c.display.toLowerCase() === context.query.trim().toLowerCase() ||
        c.note.path.replace(/\.md$/i, '').toLowerCase() ===
          context.query.trim().toLowerCase(),
    )

  const totalRows = candidates.length + (showCreate ? 1 : 0)

  // Keyboard handler: arrows + enter / tab. Bound at document level
  // so we catch keys regardless of where focus is (textarea keeps
  // focus the whole time — the popup never receives keyboard).
  useEffect(() => {
    if (!context) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        setActiveIdx((i) => (i + 1) % Math.max(1, totalRows))
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        setActiveIdx((i) => (i - 1 + Math.max(1, totalRows)) % Math.max(1, totalRows))
      } else if (e.key === 'Enter' || e.key === 'Tab') {
        if (totalRows === 0) return
        e.preventDefault()
        const sel = pickAt(activeIdx)
        if (sel) onSelect(sel)
      } else if (e.key === 'Escape') {
        e.preventDefault()
        onDismiss()
      }
    }
    document.addEventListener('keydown', onKey, true)
    return () => document.removeEventListener('keydown', onKey, true)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [context, totalRows, activeIdx, candidates, showCreate])

  // Scroll active row into view when navigating with arrows.
  useEffect(() => {
    const el = containerRef.current?.querySelector(
      `[data-idx="${activeIdx}"]`,
    ) as HTMLElement | null
    el?.scrollIntoView({ block: 'nearest' })
  }, [activeIdx])

  if (!context) return null

  function pickAt(idx: number) {
    if (idx < candidates.length) {
      const c = candidates[idx]
      return { display: c.display, path: c.note.path, create: false }
    }
    if (showCreate && idx === candidates.length) {
      const q = context!.query.trim()
      const path = q.includes('/') ? withMd(q) : `library/${q}.md`
      return { display: q, path, create: true }
    }
    return null
  }

  // Position: below the caret. If that would clip past the viewport
  // bottom, flip above. Horizontal stays anchored to caret left.
  const popupHeight = 240 // approximate; CSS caps via max-h.
  const wouldOverflow =
    context.caret.top + context.caret.height + popupHeight >
    window.innerHeight
  const top = wouldOverflow
    ? context.caret.top - popupHeight - 4
    : context.caret.top + context.caret.height + 2
  const left = Math.min(
    context.caret.left,
    window.innerWidth - 320 - 8,
  )

  return (
    <div
      ref={containerRef}
      style={{
        position: 'fixed',
        top: `${Math.max(8, top)}px`,
        left: `${Math.max(8, left)}px`,
        width: '320px',
        maxHeight: `${popupHeight}px`,
      }}
      className={cn(
        'z-50 overflow-y-auto rounded-md border border-border bg-popover',
        'shadow-[0_8px_24px_rgba(0,0,0,0.32)] py-0.5',
      )}
      role="listbox"
    >
      <div className="px-2 py-1 text-[10px] uppercase tracking-wider text-muted-foreground/60 font-mono">
        {context.query
          ? `Wiki-link · ${candidates.length} match${
              candidates.length === 1 ? '' : 'es'
            }`
          : 'Wiki-link · type to filter'}
      </div>
      {candidates.length === 0 && !showCreate && (
        <div className="px-2 py-1.5 text-[11px] text-muted-foreground/60">
          No notes match.
        </div>
      )}
      {candidates.map((c, i) => (
        <button
          key={c.note.path}
          type="button"
          data-idx={i}
          onMouseDown={(e) => {
            // mousedown fires before textarea blur — keeps the caret in place
            e.preventDefault()
            onSelect({ display: c.display, path: c.note.path })
          }}
          onMouseEnter={() => setActiveIdx(i)}
          className={cn(
            'w-full text-left flex items-center gap-2 px-2 py-1 text-[12px]',
            'hover:bg-card',
            i === activeIdx && 'bg-card text-foreground',
          )}
        >
          <FileText className="size-3 shrink-0 text-muted-foreground/60" />
          <div className="flex flex-col min-w-0 flex-1">
            <span className="truncate">{c.display}</span>
            <span className="text-[10px] text-muted-foreground/60 font-mono truncate">
              {c.note.path}
            </span>
          </div>
        </button>
      ))}
      {showCreate && (
        <button
          type="button"
          data-idx={candidates.length}
          onMouseDown={(e) => {
            e.preventDefault()
            const sel = pickAt(candidates.length)
            if (sel) onSelect(sel)
          }}
          onMouseEnter={() => setActiveIdx(candidates.length)}
          className={cn(
            'w-full text-left flex items-center gap-2 px-2 py-1 text-[12px] border-t border-border/40',
            'hover:bg-card',
            activeIdx === candidates.length && 'bg-card text-foreground',
          )}
        >
          <FilePlus2 className="size-3 shrink-0 text-state-running" />
          <div className="flex flex-col min-w-0 flex-1">
            <span className="truncate">
              Create "{context.query.trim()}"
            </span>
            <span className="text-[10px] text-muted-foreground/60 font-mono truncate">
              {context.query.trim().includes('/')
                ? withMd(context.query.trim())
                : `library/${context.query.trim()}.md`}
            </span>
          </div>
        </button>
      )}
    </div>
  )
}

function withMd(p: string): string {
  return p.toLowerCase().endsWith('.md') ? p : p + '.md'
}
