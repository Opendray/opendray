import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  Search,
  Loader2,
  CaseSensitive,
  FileCode,
  ChevronDown,
  ChevronRight,
  AlertCircle,
} from 'lucide-react'

import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { searchCwd, type SearchCase, type SearchMatch } from '@/lib/search'
import { cn } from '@/lib/utils'

import { FileViewerDialog } from './FileViewerDialog'

interface SearchPanelProps {
  cwd: string
}

// Quick-action chips that fire common code-marker searches in one
// click. Patterns use \b word boundaries so "TODOOOOO" doesn't match.
const PRESETS: { label: string; pattern: string; case: SearchCase }[] = [
  { label: 'TODO', pattern: '\\bTODO\\b', case: 'sensitive' },
  { label: 'FIXME', pattern: '\\bFIXME\\b', case: 'sensitive' },
  { label: 'XXX', pattern: '\\bXXX\\b', case: 'sensitive' },
  { label: 'HACK', pattern: '\\bHACK\\b', case: 'sensitive' },
]

// SearchPanel — ripgrep-backed full-text search across the session's
// cwd. Replaces the old "Logs" tab which was redundant with the live
// terminal. Files tab handles tree browsing; this handles content.
export function SearchPanel({ cwd }: SearchPanelProps) {
  const [query, setQuery] = useState('')
  const [debounced, setDebounced] = useState('')
  const [caseMode, setCaseMode] = useState<SearchCase>('smart')
  const [include, setInclude] = useState('')
  const [viewing, setViewing] = useState<string | null>(null)

  // 250ms debounce — enough that typing isn't laggy but rg isn't fired
  // on every keystroke. rg is fast but pointless query thrash.
  useEffect(() => {
    const t = setTimeout(() => setDebounced(query.trim()), 250)
    return () => clearTimeout(t)
  }, [query])

  const enabled = debounced.length >= 2
  const { data, isFetching, error } = useQuery({
    queryKey: ['search', cwd, debounced, caseMode, include],
    queryFn: () =>
      searchCwd({ path: cwd, q: debounced, case: caseMode, include }),
    enabled,
    // Searches are cheap; keep results around briefly to avoid
    // re-fetching when toggling case mode and back.
    staleTime: 10_000,
  })

  // Group matches by file for the {file → [matches]} display.
  const grouped = useMemo(() => {
    if (!data) return []
    const map = new Map<string, SearchMatch[]>()
    for (const m of data.matches) {
      const list = map.get(m.path) ?? []
      list.push(m)
      map.set(m.path, list)
    }
    return [...map.entries()].map(([path, matches]) => ({ path, matches }))
  }, [data])

  return (
    <>
      <div className="flex flex-col gap-2.5">
        <div className="flex items-center gap-1">
          <div className="relative flex-1">
            <Search className="absolute left-2 top-1/2 -translate-y-1/2 size-3 text-muted-foreground/60" />
            <Input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search code…"
              className="pl-7 h-7 text-[12px]"
              autoFocus
            />
          </div>
          <Button
            variant="ghost"
            size="icon"
            onClick={() =>
              setCaseMode(
                caseMode === 'sensitive'
                  ? 'smart'
                  : caseMode === 'smart'
                    ? 'insensitive'
                    : 'sensitive',
              )
            }
            className={cn(
              'size-7 shrink-0',
              caseMode === 'sensitive' && 'text-foreground',
              caseMode === 'insensitive' && 'text-muted-foreground/40',
            )}
            title={`Case: ${caseMode}  (click to cycle)`}
          >
            <CaseSensitive className="size-3.5" />
          </Button>
        </div>

        <Input
          value={include}
          onChange={(e) => setInclude(e.target.value)}
          placeholder="Filter files (e.g. *.ts !test/*)"
          className="h-7 text-[11px] font-mono"
        />

        <div className="flex flex-wrap gap-1">
          {PRESETS.map((p) => (
            <button
              key={p.label}
              type="button"
              onClick={() => {
                setQuery(p.pattern)
                setCaseMode(p.case)
              }}
              className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-card border border-border/60 hover:border-border hover:text-foreground text-muted-foreground/80 transition-colors"
              title={`Search for ${p.label} markers`}
            >
              {p.label}
            </button>
          ))}
        </div>

        {!enabled && (
          <div className="text-[11px] text-muted-foreground/60 px-1 py-2">
            Type at least 2 characters to search.
          </div>
        )}

        {enabled && isFetching && (
          <div className="flex items-center gap-2 text-[11px] text-muted-foreground py-1 px-1">
            <Loader2 className="size-3 animate-spin" />
            Searching…
          </div>
        )}

        {error && (
          <div className="flex items-start gap-2 text-[11px] text-state-failed bg-state-failed/10 border border-state-failed/30 rounded-md px-2 py-1.5">
            <AlertCircle className="size-3 mt-0.5 shrink-0" />
            <span>{(error as Error).message}</span>
          </div>
        )}

        {data && !isFetching && (
          <ResultsHeader
            total={data.matches.length}
            files={grouped.length}
            elapsed={data.elapsed}
            truncated={data.truncated}
          />
        )}

        {data && grouped.length === 0 && !isFetching && (
          <div className="text-[11px] text-muted-foreground/60 px-1 py-2">
            No matches.
          </div>
        )}

        <div className="flex flex-col gap-1">
          {grouped.map((g) => (
            <FileGroup
              key={g.path}
              path={g.path}
              matches={g.matches}
              onOpen={() => setViewing(`${cwd}/${g.path}`)}
            />
          ))}
        </div>
      </div>

      <FileViewerDialog
        path={viewing}
        open={viewing != null}
        onOpenChange={(v) => !v && setViewing(null)}
      />
    </>
  )
}

function ResultsHeader({
  total,
  files,
  elapsed,
  truncated,
}: {
  total: number
  files: number
  elapsed?: string
  truncated?: boolean
}) {
  return (
    <div className="text-[10px] text-muted-foreground/70 px-1 font-mono flex items-center gap-2">
      <span>
        {total} match{total === 1 ? '' : 'es'} · {files} file
        {files === 1 ? '' : 's'}
      </span>
      {elapsed && <span>· {elapsed}</span>}
      {truncated && (
        <span className="text-state-idle">· capped, refine to see more</span>
      )}
    </div>
  )
}

function FileGroup({
  path,
  matches,
  onOpen,
}: {
  path: string
  matches: SearchMatch[]
  onOpen: () => void
}) {
  const [open, setOpen] = useState(true)
  return (
    <div className="rounded-md border border-border/60 bg-card/40 overflow-hidden">
      <div className="flex items-center gap-1.5 px-2 py-1.5 border-b border-border/40">
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="text-muted-foreground/60 hover:text-foreground"
          aria-label={open ? 'Collapse' : 'Expand'}
        >
          {open ? (
            <ChevronDown className="size-3" />
          ) : (
            <ChevronRight className="size-3" />
          )}
        </button>
        <FileCode className="size-3 text-muted-foreground/60 shrink-0" />
        <button
          type="button"
          onClick={onOpen}
          className="flex-1 text-left text-[11.5px] font-mono truncate hover:text-foreground"
          title={`Open ${path}`}
        >
          {path}
        </button>
        <span className="text-[10px] text-muted-foreground/60 font-mono shrink-0">
          {matches.length}
        </span>
      </div>
      {open && (
        <div className="flex flex-col">
          {matches.map((m, i) => (
            <button
              key={`${m.line}:${i}`}
              type="button"
              onClick={onOpen}
              className="flex items-start gap-2 px-2 py-1 text-left hover:bg-card border-l-2 border-transparent hover:border-foreground/20"
              title="Open file"
            >
              <span className="text-[10px] text-muted-foreground/50 font-mono shrink-0 w-8 text-right pt-px tabular-nums">
                {m.line}
              </span>
              <span className="text-[11px] font-mono break-all leading-snug">
                {renderHighlighted(m.text, m.submatches)}
              </span>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

// renderHighlighted slices the line at submatch boundaries so the
// matched spans render with a yellow tint while context stays muted.
function renderHighlighted(
  text: string,
  submatches?: { start: number; end: number }[],
) {
  if (!submatches || submatches.length === 0) {
    return <span className="text-foreground/85">{text}</span>
  }
  const parts: React.ReactNode[] = []
  let cursor = 0
  for (let i = 0; i < submatches.length; i++) {
    const sm = submatches[i]
    if (sm.start > cursor) {
      parts.push(
        <span key={`pre-${i}`} className="text-muted-foreground/80">
          {text.slice(cursor, sm.start)}
        </span>,
      )
    }
    parts.push(
      <span
        key={`hit-${i}`}
        className="text-amber-300 bg-amber-500/15 rounded-sm"
      >
        {text.slice(sm.start, sm.end)}
      </span>,
    )
    cursor = sm.end
  }
  if (cursor < text.length) {
    parts.push(
      <span key="post" className="text-muted-foreground/80">
        {text.slice(cursor)}
      </span>,
    )
  }
  return <>{parts}</>
}
