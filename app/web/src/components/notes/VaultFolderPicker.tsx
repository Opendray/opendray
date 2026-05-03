import { useEffect, useMemo, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Folder, ChevronDown } from 'lucide-react'

import { cn } from '@/lib/utils'
import { listNotes, type Note } from '@/lib/notes'

interface VaultFolderPickerProps {
  // Current path (vault-relative, no trailing slash). Empty allowed —
  // means "default" / "unset" depending on the caller's semantics.
  value: string
  onChange: (next: string) => void
  placeholder?: string
  className?: string
  inputId?: string
}

// VaultFolderPicker is a combobox: free-text input on top + a dropdown
// of every directory present in the vault (derived from listNotes()
// — no extra round-trip). Filters live as the user types; clicking a
// suggestion fills the input. Free-form values are also accepted so
// the user can pin a path that doesn't exist yet (the parent will
// create it on first write).
export function VaultFolderPicker({
  value,
  onChange,
  placeholder,
  className,
  inputId,
}: VaultFolderPickerProps) {
  const { data: notes } = useQuery({
    queryKey: ['notes-list'],
    queryFn: () => listNotes(),
    staleTime: 30_000,
  })

  const allDirs = useMemo(() => {
    return extractDirs(notes ?? [])
  }, [notes])

  const [open, setOpen] = useState(false)
  const [activeIdx, setActiveIdx] = useState(0)
  const containerRef = useRef<HTMLDivElement>(null)

  const matches = useMemo(() => {
    const q = value.trim().toLowerCase()
    if (!q) return allDirs.slice(0, 50)
    // basename-prefix matches first, then any-substring matches.
    const prefixHits: string[] = []
    const subHits: string[] = []
    for (const d of allDirs) {
      const lower = d.toLowerCase()
      const base = (d.split('/').pop() ?? '').toLowerCase()
      if (base.startsWith(q) || lower.startsWith(q)) prefixHits.push(d)
      else if (lower.includes(q) || base.includes(q)) subHits.push(d)
    }
    return [...prefixHits, ...subHits].slice(0, 50)
  }, [value, allDirs])

  // Reset highlight when query / matches shift.
  useEffect(() => {
    setActiveIdx(0)
  }, [value, matches.length])

  // Close on outside click.
  useEffect(() => {
    if (!open) return
    const onDoc = (e: MouseEvent) => {
      if (
        containerRef.current &&
        !containerRef.current.contains(e.target as Node)
      ) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', onDoc)
    return () => document.removeEventListener('mousedown', onDoc)
  }, [open])

  const pick = (path: string) => {
    onChange(path)
    setOpen(false)
  }

  return (
    <div ref={containerRef} className={cn('relative', className)}>
      <div className="relative">
        <input
          id={inputId}
          value={value}
          onChange={(e) => {
            onChange(e.target.value)
            setOpen(true)
          }}
          onFocus={() => setOpen(true)}
          onKeyDown={(e) => {
            if (!open) {
              if (e.key === 'ArrowDown') {
                setOpen(true)
              }
              return
            }
            if (e.key === 'ArrowDown') {
              e.preventDefault()
              setActiveIdx((i) => Math.min(matches.length - 1, i + 1))
            } else if (e.key === 'ArrowUp') {
              e.preventDefault()
              setActiveIdx((i) => Math.max(0, i - 1))
            } else if (e.key === 'Enter') {
              if (matches[activeIdx]) {
                e.preventDefault()
                pick(matches[activeIdx])
              }
            } else if (e.key === 'Escape') {
              setOpen(false)
            } else if (e.key === 'Tab' && matches[activeIdx]) {
              // Tab completes without closing — gives Obsidian-style
              // navigate-into-subfolder feel.
              e.preventDefault()
              onChange(matches[activeIdx])
            }
          }}
          placeholder={placeholder}
          className={cn(
            'w-full h-8 pl-2 pr-8 text-[12px] font-mono rounded-md border border-border',
            'bg-input/40 text-foreground placeholder:text-muted-foreground/60',
            'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
          )}
          autoComplete="off"
          spellCheck={false}
        />
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="absolute right-1 top-1/2 -translate-y-1/2 p-1 text-muted-foreground/60 hover:text-foreground"
          tabIndex={-1}
          aria-label="Browse folders"
        >
          <ChevronDown className="size-3.5" />
        </button>
      </div>
      {open && (
        <div
          className={cn(
            'absolute z-50 left-0 right-0 mt-1 rounded-md border border-border bg-popover',
            'shadow-[0_8px_24px_rgba(0,0,0,0.32)]',
            'max-h-[260px] overflow-y-auto py-0.5',
          )}
          role="listbox"
        >
          <div className="px-2 py-1 text-[10px] uppercase tracking-wider text-muted-foreground/60 font-mono flex items-center justify-between">
            <span>
              {value
                ? `${matches.length} match${matches.length === 1 ? '' : 'es'}`
                : `${allDirs.length} folders in vault`}
            </span>
          </div>
          {matches.length === 0 ? (
            <div className="px-2 py-1.5 text-[11.5px] text-muted-foreground/70">
              No existing folder matches. Save anyway to use{' '}
              <code className="text-[11px]">{value}</code> (lazy-created on
              first write).
            </div>
          ) : (
            matches.map((d, i) => (
              <button
                key={d}
                type="button"
                onMouseDown={(e) => {
                  // mousedown fires before blur — keeps focus snug.
                  e.preventDefault()
                  pick(d)
                }}
                onMouseEnter={() => setActiveIdx(i)}
                className={cn(
                  'w-full text-left flex items-center gap-2 px-2 py-1 text-[12px] font-mono',
                  'hover:bg-card',
                  i === activeIdx && 'bg-card text-foreground',
                )}
              >
                <Folder className="size-3 shrink-0 text-muted-foreground/60" />
                <span className="truncate">{d}</span>
              </button>
            ))
          )}
        </div>
      )}
    </div>
  )
}

// extractDirs walks every note path and accumulates the unique
// directory paths (excluding the root). Sorted alphabetically with
// dotted/hidden directories filtered out.
function extractDirs(notes: Note[]): string[] {
  const set = new Set<string>()
  for (const n of notes) {
    const parts = n.path.split('/')
    parts.pop() // drop filename
    for (let i = 1; i <= parts.length; i++) {
      const segment = parts.slice(0, i).join('/')
      if (!segment) continue
      if (parts[i - 1].startsWith('.')) {
        // skip hidden segments
        break
      }
      set.add(segment)
    }
  }
  return [...set].sort((a, b) => a.localeCompare(b))
}
