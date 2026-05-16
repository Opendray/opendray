/**
 * Build a WebSocket URL with the bearer token in the ?token= query.
 * Browsers cannot set Authorization headers on the WS handshake, so the
 * token rides in the query — opendray's combined middleware accepts it
 * via bearerFromRequest fallback.
 */
export function wsURL(path: string, token: string): string {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const sep = path.includes('?') ? '&' : '?'
  return `${proto}//${window.location.host}${path}${sep}token=${encodeURIComponent(token)}`
}

export interface BinaryWSCallbacks {
  onMessage?: (data: Uint8Array) => void
  onOpen?: () => void
  onClose?: () => void
  onError?: (err: Event) => void
}

/**
 * BinaryWS is a thin wrapper around WebSocket that:
 *   - sends the bearer in the URL query
 *   - reconnects with exponential backoff up to maxBackoffMs
 *   - notifies listeners only when not in a deliberate close()
 *
 * Used by Terminal for the /sessions/{id}/stream endpoint.
 */
export class BinaryWS {
  private url: string
  private cb: BinaryWSCallbacks
  private ws: WebSocket | null = null
  private closed = false
  private backoff = 500
  private readonly maxBackoff = 8_000
  private readonly maxRetries = 6
  private retries = 0
  private timer: ReturnType<typeof setTimeout> | null = null

  constructor(url: string, cb: BinaryWSCallbacks = {}) {
    this.url = url
    this.cb = cb
  }

  start() {
    if (this.closed) return
    this.connect()
  }

  send(data: string | ArrayBuffer) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(data)
    }
  }

  close() {
    this.closed = true
    if (this.timer) {
      clearTimeout(this.timer)
      this.timer = null
    }
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  isOpen(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }

  private connect() {
    const ws = new WebSocket(this.url)
    ws.binaryType = 'arraybuffer'
    this.ws = ws

    ws.onopen = () => {
      this.backoff = 500
      this.retries = 0
      this.cb.onOpen?.()
    }
    ws.onmessage = (ev) => {
      if (ev.data instanceof ArrayBuffer) {
        this.cb.onMessage?.(new Uint8Array(ev.data))
      } else if (typeof ev.data === 'string') {
        this.cb.onMessage?.(new TextEncoder().encode(ev.data))
      }
    }
    ws.onerror = (ev) => {
      this.cb.onError?.(ev)
    }
    ws.onclose = (ev) => {
      this.cb.onClose?.()
      if (this.closed) return
      // Stop retrying for normal / explicit server-side close. The
      // server uses 1000 (normal) or 1001 (going away); also halt
      // after maxRetries attempts so a permanently-broken endpoint
      // doesn't reconnect forever.
      if (ev.code === 1000 || ev.code === 1001) {
        this.closed = true
        return
      }
      this.retries++
      if (this.retries >= this.maxRetries) {
        this.closed = true
        return
      }
      const wait = this.backoff
      this.backoff = Math.min(this.maxBackoff, this.backoff * 2)
      this.timer = setTimeout(() => this.connect(), wait)
    }
  }
}
