import {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useRef,
} from 'react'
import { Terminal as XTerm } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'

import { useAuth } from '@/stores/auth'
import { useTheme } from '@/stores/theme'
import { BinaryWS, wsURL } from '@/lib/ws'
import { resizeSession } from '@/lib/sessions'

interface TerminalProps {
  sessionId: string
}

export interface TerminalHandle {
  /**
   * Send a raw byte sequence to the PTY's stdin (e.g. ESC '\x1b',
   * Ctrl+C '\x03', arrow up '\x1b[A'). Used by the on-screen
   * keyboard toolbar to forward keys browsers don't send naturally
   * on touch devices.
   */
  sendInput: (data: string) => void
}

function readVar(name: string): string {
  return getComputedStyle(document.documentElement)
    .getPropertyValue(name)
    .trim()
}

function buildTheme(applied: 'light' | 'dark') {
  // xterm.js needs concrete colors (no css var() / oklch in canvas).
  // Read computed values from the document so the terminal follows
  // the live theme tokens. `applied` is the dep that triggers the
  // useEffect refresh in the caller; the read happens during render.
  return {
    background: readVar('--background') || (applied === 'dark' ? '#13151b' : '#fafafa'),
    foreground: readVar('--foreground') || (applied === 'dark' ? '#f5f5f5' : '#1a1a1a'),
    cursor: readVar('--accent') || '#ff7b35',
    cursorAccent: readVar('--background') || '#13151b',
    selectionBackground: readVar('--accent') || '#ff7b35',
    selectionForeground: readVar('--accent-foreground') || '#1a1a1a',
    black: applied === 'dark' ? '#0a0a0c' : '#1a1a1a',
    red: '#e85b5b',
    green: '#4ad295',
    yellow: '#e8c050',
    blue: '#5b9eff',
    magenta: '#c084fc',
    cyan: '#5be8d4',
    white: applied === 'dark' ? '#e5e5e5' : '#fafafa',
    brightBlack: applied === 'dark' ? '#3a3a3a' : '#3a3a3a',
    brightRed: '#ff7373',
    brightGreen: '#5be0a8',
    brightYellow: '#ffd270',
    brightBlue: '#7eb4ff',
    brightMagenta: '#d8a0ff',
    brightCyan: '#7af0dc',
    brightWhite: '#ffffff',
  }
}

export const Terminal = forwardRef<TerminalHandle, TerminalProps>(function Terminal(
  { sessionId },
  ref,
) {
  const containerRef = useRef<HTMLDivElement>(null)
  const xtermRef = useRef<XTerm | null>(null)
  const fitRef = useRef<FitAddon | null>(null)
  const wsRef = useRef<BinaryWS | null>(null)
  const token = useAuth((s) => s.token)
  const themeApplied = useTheme((s) => s.applied())

  useImperativeHandle(ref, () => ({
    sendInput: (data: string) => {
      const ws = wsRef.current
      if (!ws || !ws.isOpen()) return
      const enc = new TextEncoder().encode(data)
      ws.send(
        enc.buffer.slice(
          enc.byteOffset,
          enc.byteOffset + enc.byteLength,
        ) as ArrayBuffer,
      )
    },
  }))

  // Mount xterm + WS once per session id.
  useEffect(() => {
    if (!containerRef.current || !token) return

    const term = new XTerm({
      fontFamily:
        '"JetBrains Mono Variable", "JetBrains Mono", ui-monospace, Menlo, Consolas, monospace',
      fontSize: 13,
      lineHeight: 1.25,
      letterSpacing: 0,
      cursorBlink: true,
      cursorStyle: 'bar',
      cursorWidth: 2,
      theme: buildTheme(themeApplied),
      scrollback: 8_000,
      allowProposedApi: true,
      convertEol: true,
    })
    const fit = new FitAddon()
    const links = new WebLinksAddon()
    term.loadAddon(fit)
    term.loadAddon(links)
    term.open(containerRef.current)
    fit.fit()
    xtermRef.current = term
    fitRef.current = fit

    // alive flips false on cleanup so any straggler resize/onOpen
    // callbacks scheduled before unmount don't fire `/resize` against
    // a session that's just transitioned to ended — server returns
    // 404 and browser logs it red in console even though we .catch().
    let alive = true

    const ws = new BinaryWS(wsURL(`/api/v1/sessions/${sessionId}/stream`, token), {
      onMessage: (data) => term.write(data),
      onClose: () => {
        if (!alive) return
        term.writeln('')
        term.writeln('\x1b[33m[disconnected — reconnecting…]\x1b[0m')
      },
      onOpen: () => {
        // After (re)connect, push current dimensions so server sizes the PTY.
        if (!alive) return
        const { cols, rows } = term
        if (cols && rows) {
          resizeSession(sessionId, cols, rows).catch(() => {})
        }
      },
    })
    wsRef.current = ws
    ws.start()

    term.onData((d) => {
      const enc = new TextEncoder().encode(d)
      ws.send(enc.buffer.slice(enc.byteOffset, enc.byteOffset + enc.byteLength) as ArrayBuffer)
    })
    term.onResize(({ cols, rows }) => {
      if (!alive) return
      resizeSession(sessionId, cols, rows).catch(() => {})
    })

    const ro = new ResizeObserver(() => {
      if (!alive) return
      try {
        fit.fit()
      } catch {
        /* element not measured yet */
      }
    })
    ro.observe(containerRef.current)

    return () => {
      alive = false
      ro.disconnect()
      ws.close()
      term.dispose()
      xtermRef.current = null
      fitRef.current = null
      wsRef.current = null
    }
  }, [sessionId, token])

  // Refresh xterm theme when the site theme changes.
  useEffect(() => {
    const term = xtermRef.current
    if (!term) return
    term.options.theme = buildTheme(themeApplied)
    fitRef.current?.fit()
  }, [themeApplied])

  return (
    <div className="h-full w-full bg-background">
      <div ref={containerRef} className="h-full w-full p-3" />
    </div>
  )
})
