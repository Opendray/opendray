import { useEffect, useMemo, useRef, useState } from 'react'
import {
  Download,
  Eraser,
  Pause,
  Play,
  Search as SearchIcon,
  Wifi,
  WifiOff,
} from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { downloadLogs, streamLogs, type LogRecord } from '@/lib/logs'
import { cn } from '@/lib/utils'

const MAX_BUFFERED = 5000 // hard cap on records held in browser memory

const LEVEL_STYLES: Record<LogRecord['level'], string> = {
  DEBUG: 'text-muted-foreground/70',
  INFO: 'text-foreground/80',
  WARN: 'text-amber-300',
  ERROR: 'text-rose-400',
}

const LEVEL_BADGE: Record<LogRecord['level'], string> = {
  DEBUG: 'bg-muted/30 text-muted-foreground border-muted/40',
  INFO: 'bg-blue-500/10 text-blue-300 border-blue-500/30',
  WARN: 'bg-amber-500/15 text-amber-300 border-amber-500/30',
  ERROR: 'bg-rose-500/15 text-rose-300 border-rose-500/30',
}

// LogViewer is the live-tail console embedded in Settings → Logging.
// On mount it opens a WS to /admin/logs/stream which replays the
// in-process ring buffer once and then pushes every new record. The
// header lets the operator pause auto-scroll, filter with substring
// search, clear the local view, and download the full ring as text.
export function LogViewer() {
  const [records, setRecords] = useState<LogRecord[]>([])
  const [filter, setFilter] = useState('')
  const [paused, setPaused] = useState(false)
  const [connected, setConnected] = useState(false)
  const scrollRef = useRef<HTMLDivElement | null>(null)

  // Keep refs to paused/scroll behavior so the WS handler closure
  // (registered once) can read the latest values without re-binding.
  const pausedRef = useRef(paused)
  pausedRef.current = paused

  useEffect(() => {
    setConnected(true)
    const stop = streamLogs(
      (rec) => {
        setRecords((cur) => {
          const next = [...cur, rec]
          // Trim from the head if we exceed the local cap; the
          // server's ring is the source of truth so old records can
          // always be re-fetched on reconnect.
          if (next.length > MAX_BUFFERED) {
            return next.slice(next.length - MAX_BUFFERED)
          }
          return next
        })
      },
      () => setConnected(false),
    )
    return stop
  }, [])

  // Auto-scroll when new records arrive, unless paused or the
  // operator has manually scrolled away from the bottom.
  useEffect(() => {
    if (paused) return
    const el = scrollRef.current
    if (!el) return
    const nearBottom =
      el.scrollHeight - el.scrollTop - el.clientHeight < 80
    if (nearBottom) {
      el.scrollTop = el.scrollHeight
    }
  }, [records, paused])

  const filtered = useMemo(() => {
    const q = filter.trim().toLowerCase()
    if (!q) return records
    return records.filter((r) => r.text.toLowerCase().includes(q))
  }, [records, filter])

  const stats = useMemo(() => {
    const counts = { DEBUG: 0, INFO: 0, WARN: 0, ERROR: 0 }
    for (const r of records) counts[r.level]++
    return counts
  }, [records])

  return (
    <div className="rounded-lg border border-border bg-card/30 overflow-hidden">
      {/* Toolbar */}
      <div className="flex items-center gap-2 px-3 py-2 border-b border-border bg-background/50">
        <div className="relative">
          <SearchIcon className="absolute left-2 top-1/2 -translate-y-1/2 size-3 text-muted-foreground/60" />
          <Input
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            placeholder="Filter…"
            className="h-7 pl-7 text-xs w-44"
          />
        </div>
        <div className="flex items-center gap-1.5 text-[10px] font-mono text-muted-foreground">
          <span title="Debug count">{stats.DEBUG} D</span>
          <span title="Info count">{stats.INFO} I</span>
          <span className="text-amber-300/80" title="Warn count">
            {stats.WARN} W
          </span>
          <span className="text-rose-400/80" title="Error count">
            {stats.ERROR} E
          </span>
        </div>
        <div className="ml-auto flex items-center gap-1">
          <span
            className={cn(
              'flex items-center gap-1 text-[10px] mr-2',
              connected ? 'text-emerald-400' : 'text-muted-foreground',
            )}
            title={connected ? 'Streaming' : 'Disconnected'}
          >
            {connected ? (
              <Wifi className="size-3" />
            ) : (
              <WifiOff className="size-3" />
            )}
            {connected ? 'live' : 'offline'}
          </span>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="h-7 px-2 text-[11px]"
            onClick={() => setPaused((v) => !v)}
            title={paused ? 'Resume auto-scroll' : 'Pause auto-scroll'}
          >
            {paused ? (
              <Play className="size-3" />
            ) : (
              <Pause className="size-3" />
            )}
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="h-7 px-2 text-[11px]"
            onClick={() => setRecords([])}
            title="Clear local view (server ring untouched)"
          >
            <Eraser className="size-3" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="h-7 px-2 text-[11px]"
            onClick={() => downloadLogs()}
            title="Download full ring as .log file"
          >
            <Download className="size-3" />
          </Button>
        </div>
      </div>

      {/* Body */}
      <div
        ref={scrollRef}
        className="font-mono text-[11px] leading-relaxed p-3 h-80 overflow-y-auto bg-background/30"
      >
        {filtered.length === 0 && (
          <p className="text-muted-foreground/50 italic">
            {records.length === 0
              ? 'Waiting for log records…'
              : `No records match "${filter}"`}
          </p>
        )}
        {filtered.map((r, i) => (
          <Line key={`${r.time}-${i}`} rec={r} />
        ))}
      </div>
    </div>
  )
}

function Line({ rec }: { rec: LogRecord }) {
  const time = formatTime(rec.time)
  return (
    <div
      className={cn(
        'flex items-start gap-2 py-0.5',
        LEVEL_STYLES[rec.level],
      )}
    >
      <span className="text-muted-foreground/50 shrink-0">{time}</span>
      <span
        className={cn(
          'shrink-0 px-1 rounded border text-[9px] uppercase font-bold tracking-wider',
          LEVEL_BADGE[rec.level],
        )}
      >
        {rec.level}
      </span>
      <span className="break-all whitespace-pre-wrap min-w-0">
        <span className="font-medium">{rec.message}</span>
        {rec.attrs &&
          Object.keys(rec.attrs).length > 0 &&
          renderAttrs(rec.attrs)}
      </span>
    </div>
  )
}

function renderAttrs(attrs: Record<string, unknown>) {
  return Object.entries(attrs).map(([k, v]) => (
    <span key={k} className="opacity-70 ml-2">
      <span className="text-muted-foreground/60">{k}=</span>
      <span>{String(v)}</span>
    </span>
  ))
}

function formatTime(iso: string): string {
  // ISO → HH:MM:SS.mmm. Empty string when invalid.
  const d = new Date(iso)
  if (isNaN(d.getTime())) return ''
  const pad = (n: number, w = 2) => n.toString().padStart(w, '0')
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}.${pad(d.getMilliseconds(), 3)}`
}
