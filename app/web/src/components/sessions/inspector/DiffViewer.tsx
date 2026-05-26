import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Loader2, Copy, GitCompare } from 'lucide-react'
import { toast } from 'sonner'

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  getGitDiff,
  getGitShow,
  type DiffScope,
} from '@/lib/git'
import { cn } from '@/lib/utils'
import { copyText } from '@/lib/clipboard'

interface DiffViewerProps {
  open: boolean
  onOpenChange: (v: boolean) => void
  // Either { kind: 'file', ... } for a working-tree diff or
  // { kind: 'commit', ... } for `git show <hash>`. Modal renders both
  // through the same colored unified-diff view.
  target:
    | { kind: 'file'; cwd: string; file: string; scope?: DiffScope }
    | { kind: 'commit'; cwd: string; hash: string; subject?: string }
    | null
}

export function DiffViewer({ open, onOpenChange, target }: DiffViewerProps) {
  const { data, isLoading, error } = useQuery({
    queryKey: ['diff', target],
    queryFn: () => {
      if (!target) return ''
      if (target.kind === 'file') {
        return getGitDiff(target.cwd, target.scope ?? 'all', target.file)
      }
      return getGitShow(target.cwd, target.hash)
    },
    enabled: open && target != null,
    staleTime: 5_000,
  })

  const title = useMemo(() => {
    if (!target) return ''
    if (target.kind === 'file') return target.file
    return `${target.subject ?? 'commit'}`
  }, [target])

  const subtitle = useMemo(() => {
    if (!target) return ''
    if (target.kind === 'file') return target.cwd
    return target.hash
  }, [target])

  const copy = async () => {
    if (!data) return
    try {
      if (!(await copyText(data))) throw new Error('clipboard unavailable')
      toast.success('Diff copied')
    } catch (e) {
      toast.error('Copy failed', { description: (e as Error).message })
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[min(92vw,1100px)] w-[min(92vw,1100px)] h-[min(85vh,800px)] gap-2 flex flex-col">
        <DialogHeader className="shrink-0">
          <div className="flex items-start gap-2 min-w-0">
            <GitCompare className="size-4 mt-0.5 text-muted-foreground shrink-0" />
            <div className="flex flex-col min-w-0 flex-1">
              <DialogTitle className="truncate">{title}</DialogTitle>
              <span
                className="text-[11px] text-muted-foreground/70 font-mono truncate"
                title={subtitle}
              >
                {subtitle}
              </span>
            </div>
            {data && (
              <DiffStats text={data} />
            )}
          </div>
        </DialogHeader>

        <div
          className={cn(
            'rounded-md border border-border bg-card/30 overflow-hidden',
            'flex-1 min-h-0 flex',
          )}
        >
          {isLoading && (
            <div className="flex-1 flex items-center justify-center gap-2 text-[12px] text-muted-foreground">
              <Loader2 className="size-3.5 animate-spin" />
              Loading…
            </div>
          )}
          {error && (
            <div className="flex-1 flex items-center justify-center text-[12px] text-state-failed px-4 text-center">
              {(error as Error).message}
            </div>
          )}
          {!isLoading && !error && data === '' && (
            <div className="flex-1 flex items-center justify-center text-[12px] text-muted-foreground">
              No changes.
            </div>
          )}
          {!isLoading && !error && data && data.length > 0 && (
            <ScrollArea className="flex-1">
              <DiffBody text={data} />
            </ScrollArea>
          )}
        </div>

        <div className="flex items-center justify-end gap-2">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={copy}
            disabled={!data}
            className="text-[11px] gap-1.5"
          >
            <Copy className="size-3" />
            Copy
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => onOpenChange(false)}
            className="text-[11px]"
          >
            Close
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}

// DiffStats counts +/- lines (excluding the +++ / --- file headers)
// for the summary chip in the title bar.
function DiffStats({ text }: { text: string }) {
  const { add, del } = useMemo(() => {
    let add = 0
    let del = 0
    for (const line of text.split('\n')) {
      if (line.startsWith('+++') || line.startsWith('---')) continue
      if (line.startsWith('+')) add++
      else if (line.startsWith('-')) del++
    }
    return { add, del }
  }, [text])
  return (
    <div className="text-[10px] font-mono shrink-0 flex items-center gap-2">
      <span className="text-state-running">+{add}</span>
      <span className="text-state-failed">-{del}</span>
    </div>
  )
}

// DiffBody renders the unified diff with per-line color coding. Lines
// are split once and walked — keep the parser tiny and don't try to
// understand hunk semantics, just classify the prefix character.
function DiffBody({ text }: { text: string }) {
  const MAX_LINES = 8000
  const all = text.split('\n')
  const truncated = all.length > MAX_LINES
  const lines = truncated ? all.slice(0, MAX_LINES) : all

  return (
    <div className="font-mono text-[12px] leading-[1.55] py-2">
      {lines.map((line, i) => {
        let cls = 'text-foreground/85'
        if (line.startsWith('+++') || line.startsWith('---')) {
          cls = 'text-muted-foreground/70 font-medium'
        } else if (line.startsWith('@@')) {
          cls = 'text-sky-400/90 bg-sky-500/5'
        } else if (line.startsWith('diff --git') || line.startsWith('index ')) {
          cls = 'text-muted-foreground/60'
        } else if (line.startsWith('+')) {
          cls = 'text-state-running bg-state-running/5'
        } else if (line.startsWith('-')) {
          cls = 'text-state-failed bg-state-failed/5'
        } else if (
          line.startsWith('commit ') ||
          line.startsWith('Author:') ||
          line.startsWith('AuthorDate:') ||
          line.startsWith('Commit:') ||
          line.startsWith('CommitDate:') ||
          line.startsWith('Date:')
        ) {
          cls = 'text-amber-400/90'
        }
        return (
          <div key={i} className={cn('px-3 whitespace-pre', cls)}>
            {line || ' '}
          </div>
        )
      })}
      {truncated && (
        <div className="text-[11px] text-muted-foreground/70 px-3 pt-2 border-t border-border/40 mt-2">
          … {(all.length - MAX_LINES).toLocaleString()} more lines hidden
        </div>
      )}
    </div>
  )
}
