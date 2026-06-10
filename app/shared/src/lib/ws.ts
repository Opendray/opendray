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
  /**
   * Fires once per disconnect *cycle* — the first time the connection
   * drops, and not again until a successful reconnect resets the state.
   * Lets callers surface a "reconnecting" line without stacking it on
   * every bounce when an idle proxy drops the socket every few seconds.
   */
  onClose?: () => void
  /**
   * Fires once when the socket comes back up after a prior onClose.
   * Lets callers clear/replace the "reconnecting" line they wrote.
   */
  onReconnect?: () => void
  /**
   * Fires when reconnect attempts have been exhausted and the socket
   * is permanently dead. Callers should surface a CTA (refresh / reopen)
   * rather than the still-trying "reconnecting" line.
   */
  onGiveUp?: () => void
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
  private readonly maxBackoff = 15_000
  // High enough that an idle-proxy bouncing the socket every few
  // seconds doesn't hit the permanent-give-up path during normal
  // use; capped so a server that's gone for good still eventually
  // surfaces the give-up state instead of polling forever.
  private readonly maxRetries = 30
  private retries = 0
  private timer: ReturnType<typeof setTimeout> | null = null
  // True between the first onClose of a disconnect cycle and the
  // next successful onopen. Prevents stacking "[disconnected]" lines
  // when the proxy closes the socket every backoff window.
  private disconnectAnnounced = false

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
      const wasDisconnected = this.disconnectAnnounced
      this.disconnectAnnounced = false
      this.cb.onOpen?.()
      if (wasDisconnected) this.cb.onReconnect?.()
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
      // Announce the disconnect ONCE per cycle — not on every retry.
      // A flaky proxy that drops the socket every backoff window used
      // to write a fresh "[disconnected]" line each bounce, which
      // stacked on screen and obscured what Claude was actually saying.
      if (!this.closed && !this.disconnectAnnounced) {
        this.disconnectAnnounced = true
        this.cb.onClose?.()
      }
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
        this.cb.onGiveUp?.()
        return
      }
      const wait = this.backoff
      this.backoff = Math.min(this.maxBackoff, this.backoff * 2)
      this.timer = setTimeout(() => this.connect(), wait)
    }
  }
}
