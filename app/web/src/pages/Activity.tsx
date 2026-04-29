import { useEffect, useMemo, useRef, useState } from 'react'
import { Pause, Play, Trash2, ArrowDownToLine, Activity as ActivityIcon, Loader2, ChevronDown, ChevronRight } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Code } from '@/components/ui/code'
import { useAuth } from '@/stores/auth'
import { wsURL } from '@/lib/ws'
import { cn } from '@/lib/utils'

interface BusEvent {
  topic: string
  ts: string
  data: unknown
  /** local arrival id, used for React keys */
  _id: number
}

const TOPIC_PATTERNS = [
  { value: 'session.*', label: 'session.*', kind: 'session' as const },
  { value: 'channel.*', label: 'channel.*', kind: 'channel' as const },
  {
    value: 'integration.*',
    label: 'integration.*',
    kind: 'integration' as const,
  },
  { value: 'admin.*', label: 'admin.*', kind: 'admin' as const },
]

type TopicKind = 'session' | 'channel' | 'integration' | 'admin'

function topicKind(topic: string): TopicKind | null {
  if (topic.startsWith('session.')) return 'session'
  if (topic.startsWith('channel.')) return 'channel'
  if (topic.startsWith('integration.')) return 'integration'
  if (topic.startsWith('admin.')) return 'admin'
  return null
}

const KIND_VARIANT: Record<TopicKind, 'success' | 'warning' | 'accent' | 'muted'> =
  {
    session: 'success',
    channel: 'accent',
    integration: 'warning',
    admin: 'muted',
  }

const MAX_EVENTS = 500

export function ActivityPage() {
  const token = useAuth((s) => s.token)
  const [enabled, setEnabled] = useState<Record<string, boolean>>({
    'session.*': true,
    'channel.*': true,
    'integration.*': true,
    'admin.*': true,
  })
  const [events, setEvents] = useState<BusEvent[]>([])
  const [paused, setPaused] = useState(false)
  const [autoScroll, setAutoScroll] = useState(true)
  const [status, setStatus] = useState<'idle' | 'connecting' | 'open' | 'closed'>(
    'idle',
  )
  const idRef = useRef(0)
  const wsRef = useRef<WebSocket | null>(null)
  const scrollerRef = useRef<HTMLDivElement | null>(null)

  const topicsCSV = useMemo(
    () =>
      TOPIC_PATTERNS.filter((p) => enabled[p.value])
        .map((p) => p.value)
        .join(','),
    [enabled],
  )

  // Manage one WS per topic-set; reconnect on topics change.
  useEffect(() => {
    if (!token || !topicsCSV) {
      wsRef.current?.close()
      wsRef.current = null
      setStatus('idle')
      return
    }

    setStatus('connecting')
    const url = wsURL(
      `/api/v1/integrations/_events?topics=${encodeURIComponent(topicsCSV)}`,
      token,
    )
    const ws = new WebSocket(url)
    wsRef.current = ws

    ws.onopen = () => setStatus('open')
    ws.onclose = () => setStatus('closed')
    ws.onerror = () => setStatus('closed')
    ws.onmessage = (ev) => {
      if (paused) return
      try {
        const parsed = JSON.parse(String(ev.data))
        const next: BusEvent = {
          topic: parsed.topic,
          ts: parsed.ts,
          data: parsed.data,
          _id: ++idRef.current,
        }
        setEvents((prev) => {
          const out = [...prev, next]
          if (out.length > MAX_EVENTS) out.splice(0, out.length - MAX_EVENTS)
          return out
        })
      } catch {
        /* ignore non-JSON frames (pings handled by browser) */
      }
    }

    return () => {
      ws.close()
      wsRef.current = null
    }
  }, [token, topicsCSV, paused])

  // Auto-scroll to bottom on new events.
  useEffect(() => {
    if (!autoScroll) return
    const el = scrollerRef.current
    if (!el) return
    el.scrollTop = el.scrollHeight
  }, [events, autoScroll])

  return (
    <div className="h-full flex flex-col bg-background">
      <header className="border-b border-border px-6 py-4 flex flex-wrap items-center gap-3">
        <div className="flex-1 min-w-[200px]">
          <h1 className="text-[16px] font-semibold tracking-tight flex items-center gap-2">
            Activity
            <Badge
              variant={
                status === 'open'
                  ? 'success'
                  : status === 'connecting'
                    ? 'warning'
                    : status === 'closed'
                      ? 'danger'
                      : 'muted'
              }
            >
              {status}
            </Badge>
          </h1>
          <p className="text-[12px] text-muted-foreground">
            Live event-bus stream. Filter by topic prefix; pause to inspect.
          </p>
        </div>
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setPaused((p) => !p)}
            title={paused ? 'Resume' : 'Pause'}
          >
            {paused ? (
              <>
                <Play className="size-3.5" />
                Resume
              </>
            ) : (
              <>
                <Pause className="size-3.5" />
                Pause
              </>
            )}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setAutoScroll((v) => !v)}
            title={autoScroll ? 'Auto-scroll on' : 'Auto-scroll off'}
            className={cn(autoScroll && 'text-foreground')}
          >
            <ArrowDownToLine className="size-3.5" />
            Follow
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setEvents([])}
            title="Clear"
            className="text-muted-foreground hover:text-destructive"
          >
            <Trash2 className="size-3.5" />
            Clear
          </Button>
        </div>
      </header>

      <div className="border-b border-border px-6 py-3 flex items-center gap-1.5 flex-wrap">
        <span className="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium mr-2">
          Topics
        </span>
        {TOPIC_PATTERNS.map((p) => {
          const on = enabled[p.value]
          return (
            <button
              key={p.value}
              type="button"
              onClick={() =>
                setEnabled((s) => ({ ...s, [p.value]: !s[p.value] }))
              }
              className={cn(
                'px-2 py-0.5 rounded-md border text-[11px] font-mono transition-colors',
                on
                  ? 'border-accent bg-accent/15 text-foreground'
                  : 'border-border bg-transparent text-muted-foreground hover:bg-card',
              )}
            >
              {p.label}
            </button>
          )
        })}
        <div className="flex-1" />
        <span className="text-[10px] text-muted-foreground/70 font-mono">
          {events.length} / {MAX_EVENTS}
        </span>
      </div>

      <ScrollArea className="flex-1">
        <div ref={scrollerRef} className="p-2 flex flex-col gap-1 max-h-full">
          {events.length === 0 && status === 'open' && (
            <div className="flex items-center justify-center py-16 gap-2 text-[12px] text-muted-foreground">
              <ActivityIcon className="size-4" />
              Waiting for events…
            </div>
          )}
          {status === 'connecting' && (
            <div className="flex items-center gap-2 text-[12px] text-muted-foreground px-3 py-2">
              <Loader2 className="size-3.5 animate-spin" />
              Connecting to events stream…
            </div>
          )}
          {events.map((ev) => (
            <EventRow key={ev._id} event={ev} />
          ))}
        </div>
      </ScrollArea>
    </div>
  )
}

function EventRow({ event }: { event: BusEvent }) {
  const [open, setOpen] = useState(false)
  const kind = topicKind(event.topic)
  const time = new Date(event.ts)
  const hh = time.toTimeString().slice(0, 8)
  const ms = time.getMilliseconds().toString().padStart(3, '0')

  return (
    <div className="border border-transparent hover:border-border/60 rounded-md transition-colors">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="w-full flex items-center gap-2 px-3 py-1.5 text-left"
      >
        {open ? (
          <ChevronDown className="size-3 text-muted-foreground/60 shrink-0" />
        ) : (
          <ChevronRight className="size-3 text-muted-foreground/60 shrink-0" />
        )}
        <span className="text-[10px] text-muted-foreground/70 font-mono shrink-0 w-[88px]">
          {hh}.{ms}
        </span>
        <Badge variant={kind ? KIND_VARIANT[kind] : 'muted'} className="shrink-0">
          {event.topic}
        </Badge>
        <span className="text-[11px] text-muted-foreground truncate font-mono flex-1">
          {summarize(event.data)}
        </span>
      </button>
      {open && (
        <div className="px-3 pb-3 pt-0">
          <Code>{JSON.stringify(event.data, null, 2)}</Code>
        </div>
      )}
    </div>
  )
}

function summarize(data: unknown): string {
  if (data == null) return ''
  if (typeof data !== 'object') return String(data)
  const obj = data as Record<string, unknown>
  const parts: string[] = []
  for (const k of [
    'session_id',
    'channel_id',
    'integration_id',
    'user',
    'name',
    'exit_code',
    'topic',
  ]) {
    if (k in obj && obj[k] !== '' && obj[k] != null) {
      parts.push(`${k}=${formatValue(obj[k])}`)
    }
  }
  return parts.join(' ')
}

function formatValue(v: unknown): string {
  if (typeof v === 'string') return v
  if (typeof v === 'number') return v.toString()
  if (typeof v === 'boolean') return v.toString()
  return JSON.stringify(v)
}
