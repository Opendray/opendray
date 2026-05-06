import { useAuth } from '@/stores/auth'

// LogRecord mirrors the Go applog.Record JSON shape.
export interface LogRecord {
  time: string
  level: 'DEBUG' | 'INFO' | 'WARN' | 'ERROR'
  message: string
  attrs?: Record<string, unknown>
  text: string
}

// streamLogs opens a WebSocket against /admin/logs/stream and
// invokes onRecord for every record (replay + live). Returns a
// cleanup function the caller must invoke on unmount.
//
// The bearer token is appended as a query param because browser
// WebSockets can't carry custom headers.
export function streamLogs(
  onRecord: (rec: LogRecord) => void,
  onClose?: (ev: CloseEvent) => void,
): () => void {
  const token = useAuth.getState().token ?? ''
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const url = `${proto}//${window.location.host}/api/v1/admin/logs/stream?token=${encodeURIComponent(token)}`
  const ws = new WebSocket(url)
  ws.onmessage = (ev) => {
    try {
      const rec = JSON.parse(ev.data) as LogRecord
      onRecord(rec)
    } catch {
      // skip malformed
    }
  }
  ws.onclose = (ev) => onClose?.(ev)
  ws.onerror = () => {
    // close will follow
  }
  return () => {
    try {
      ws.close()
    } catch {
      // ignore
    }
  }
}

// downloadLogs triggers a browser download of the entire ring
// buffer as a plain-text .log file.
export function downloadLogs(): void {
  const token = useAuth.getState().token ?? ''
  // Authorize via fetch so we get the bearer header onto the request,
  // then turn the response into an object URL the browser saves.
  fetch('/api/v1/admin/logs/download', {
    headers: { Authorization: `Bearer ${token}` },
  })
    .then(async (res) => {
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const blob = await res.blob()
      const cd = res.headers.get('Content-Disposition') ?? ''
      const m = cd.match(/filename="([^"]+)"/)
      const name = m?.[1] ?? `opendray-${Date.now()}.log`
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = name
      document.body.appendChild(a)
      a.click()
      a.remove()
      URL.revokeObjectURL(url)
    })
    .catch((err) => {
      console.error('download logs failed:', err)
    })
}
