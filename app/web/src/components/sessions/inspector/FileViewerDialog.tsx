import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Loader2, Copy, FileText } from 'lucide-react'
import { toast } from 'sonner'

import {
  detectLanguage,
  highlightCode,
  splitHighlightedLines,
} from '@/lib/highlight'

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { readFile } from '@/lib/fs'
import { cn } from '@/lib/utils'

interface FileViewerDialogProps {
  path: string | null
  open: boolean
  onOpenChange: (v: boolean) => void
}

// FileViewerDialog opens the file at `path` via /api/v1/fs/read and
// renders it in a wide modal with line numbers. Binary files (contain
// NUL bytes in the first 1 KiB) are detected and skipped — we show
// metadata only rather than dumping garbled text into the viewer.
export function FileViewerDialog({
  path,
  open,
  onOpenChange,
}: FileViewerDialogProps) {
  const { data, isLoading, error } = useQuery({
    queryKey: ['file-content', path],
    queryFn: () => readFile(path!),
    enabled: open && !!path,
    staleTime: 5_000,
  })

  const meta = useMemo(() => {
    if (data == null) return null
    // NUL byte in the first KiB → almost certainly binary. Avoids
    // dumping garbled text into the viewer for executables / images.
    const isBinary = data.slice(0, 1024).indexOf('\x00') >= 0
    const lines = isBinary ? 0 : data.split('\n').length
    const bytes = new Blob([data]).size
    return { isBinary, lines, bytes }
  }, [data])

  const filename = path ? path.split('/').pop() ?? path : ''

  const copy = async () => {
    if (!data) return
    try {
      await navigator.clipboard.writeText(data)
      toast.success('Copied to clipboard')
    } catch (e) {
      toast.error('Copy failed', { description: (e as Error).message })
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        // Wider than the default max-w-md — code needs room. Height
        // capped to viewport so long files scroll inside the modal
        // instead of pushing it past the bottom of the screen.
        className="max-w-[min(92vw,1100px)] w-[min(92vw,1100px)] h-[min(85vh,800px)] gap-2 flex flex-col"
      >
        <DialogHeader className="shrink-0">
          <div className="flex items-start gap-2 min-w-0">
            <FileText className="size-4 mt-0.5 text-muted-foreground shrink-0" />
            <div className="flex flex-col min-w-0 flex-1">
              <DialogTitle className="truncate">{filename}</DialogTitle>
              <span
                className="text-[11px] text-muted-foreground/70 font-mono truncate"
                title={path ?? ''}
              >
                {path}
              </span>
            </div>
            {meta && !meta.isBinary && (
              <div className="text-[10px] text-muted-foreground/70 font-mono shrink-0 flex items-center gap-2">
                <span>{meta.lines.toLocaleString()} lines</span>
                <span>·</span>
                <span>{formatBytes(meta.bytes)}</span>
              </div>
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
          {!isLoading && !error && data != null && meta?.isBinary && (
            <div className="flex-1 flex flex-col items-center justify-center gap-2 text-[12px] text-muted-foreground p-6 text-center">
              <FileText className="size-6 opacity-40" strokeWidth={1.5} />
              <div>Binary file — preview not shown.</div>
              <div className="text-[11px] opacity-70 font-mono">
                {formatBytes(meta.bytes)}
              </div>
            </div>
          )}
          {!isLoading && !error && data != null && meta && !meta.isBinary && (
            <ScrollArea className="flex-1">
              <CodeView text={data} path={path} />
            </ScrollArea>
          )}
        </div>

        <div className="flex items-center justify-end gap-2">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={copy}
            disabled={!data || meta?.isBinary}
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

function CodeView({ text, path }: { text: string; path: string | null }) {
  // Cap displayed lines so a 256 KiB minified file doesn't render
  // hundreds of thousands of DOM nodes. Past the limit we tell the
  // user how many lines were trimmed.
  const MAX_LINES = 5000
  const allLines = useMemo(() => text.split('\n'), [text])
  const truncated = allLines.length > MAX_LINES
  const visibleText = useMemo(
    () => (truncated ? allLines.slice(0, MAX_LINES).join('\n') : text),
    [text, allLines, truncated],
  )
  const lnWidth = String(truncated ? MAX_LINES : allLines.length).length
  const lang = useMemo(() => detectLanguage(path), [path])

  // Lazy-loaded hljs lives off the entry chunk; resolve once per
  // file and split into per-line HTML so the gutter layout stays
  // intact. Plain text still renders via the same code path with
  // pre-escaped HTML.
  const [htmlLines, setHtmlLines] = useState<string[] | null>(null)
  useEffect(() => {
    let cancelled = false
    setHtmlLines(null)
    highlightCode(visibleText, lang)
      .then((html) => {
        if (cancelled) return
        setHtmlLines(splitHighlightedLines(html))
      })
      .catch(() => {
        if (cancelled) return
        setHtmlLines(null)
      })
    return () => {
      cancelled = true
    }
  }, [visibleText, lang])

  // While hljs loads, render plain text so the dialog isn't blank.
  const lines = htmlLines ?? allLines.slice(0, MAX_LINES)
  const isHTML = htmlLines != null

  return (
    <div className="font-mono text-[12px] leading-[1.55] py-2">
      {lines.map((line, i) => (
        <div key={i} className="flex">
          <span
            className="select-none text-right pr-3 pl-3 text-muted-foreground/40 shrink-0"
            style={{ minWidth: `${lnWidth + 2}ch` }}
          >
            {i + 1}
          </span>
          {isHTML ? (
            <span
              className="whitespace-pre flex-1 text-foreground/90 hljs"
              dangerouslySetInnerHTML={{ __html: line || ' ' }}
            />
          ) : (
            <span className="whitespace-pre flex-1 text-foreground/90">
              {line || ' '}
            </span>
          )}
        </div>
      ))}
      {truncated && (
        <div className="text-[11px] text-muted-foreground/70 px-3 pt-2 border-t border-border/40 mt-2">
          … {(allLines.length - MAX_LINES).toLocaleString()} more lines hidden
        </div>
      )}
    </div>
  )
}

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KiB`
  return `${(n / (1024 * 1024)).toFixed(2)} MiB`
}
