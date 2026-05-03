import { useQuery } from '@tanstack/react-query'
import { Link2, Loader2 } from 'lucide-react'

import { notesBacklinks } from '@/lib/notes'
import { cn } from '@/lib/utils'

interface BacklinksPaneProps {
  path: string
  // onOpen lets the parent decide what happens when the user clicks
  // a backlinked note. Inline (Notes panel inside Inspector) opens a
  // NoteEditorDialog; in a future top-level /notes page it could
  // navigate.
  onOpen?: (path: string) => void
}

// BacklinksPane lists every note whose body contains a wiki-link to
// `path`. Backed by the backend's vault scan. Renders compact rows
// with a couple of context snippets each.
export function BacklinksPane({ path, onOpen }: BacklinksPaneProps) {
  const { data, isFetching, error } = useQuery({
    queryKey: ['notes-backlinks', path],
    queryFn: () => notesBacklinks(path),
    staleTime: 15_000,
  })

  const links = data ?? []

  return (
    <section className="flex flex-col gap-1.5">
      <div className="flex items-center gap-1.5 text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium px-1">
        <Link2 className="size-3" />
        Backlinks
        <span className="text-muted-foreground/50 normal-case tracking-normal">
          · {links.length}
        </span>
        {isFetching && <Loader2 className="size-3 animate-spin opacity-50" />}
      </div>
      {error && (
        <div className="text-[11px] text-state-failed px-1 py-1">
          {(error as Error).message}
        </div>
      )}
      {!isFetching && links.length === 0 && (
        <div className="text-[11px] text-muted-foreground/60 px-1 py-1">
          No notes link here yet.
        </div>
      )}
      <div className="flex flex-col gap-1">
        {links.map((b) => (
          <button
            key={b.path}
            type="button"
            onClick={() => onOpen?.(b.path)}
            className={cn(
              'group flex flex-col gap-0.5 px-2 py-1.5 text-left rounded-md',
              'hover:bg-card border border-transparent hover:border-border/60',
            )}
          >
            <div className="flex items-baseline gap-2 min-w-0">
              <span className="text-[12px] font-medium truncate">
                {b.title || pathBase(b.path)}
              </span>
              <span className="text-[10px] font-mono text-muted-foreground/60 truncate">
                {b.path}
              </span>
            </div>
            {b.lines.length > 0 && (
              <div className="text-[10.5px] text-muted-foreground/80 font-mono leading-snug">
                {b.lines.slice(0, 2).map((l, i) => (
                  <div key={i} className="truncate">
                    {l}
                  </div>
                ))}
              </div>
            )}
          </button>
        ))}
      </div>
    </section>
  )
}

function pathBase(p: string): string {
  const i = p.lastIndexOf('/')
  return (i >= 0 ? p.slice(i + 1) : p).replace(/\.md$/, '')
}
