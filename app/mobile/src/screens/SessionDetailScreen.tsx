import { useCallback, useEffect, useRef, useState } from 'react'
import { Terminal as XTerm } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

import { Button } from '@/components/ui/button'
import { BinaryWS } from '@/lib/ws'
import { mobileWSURL } from '../lib/ws'
import { mobileFetch, type SessionSummary } from '../lib/api'

interface Props {
  serverURL: string
  token: string
  sessionId: string
  // Lookup of the session metadata we already loaded in the list. If
  // the user navigates here without a fresh list (e.g. deep link in
  // the future) we'd need to fetch /sessions/:id; for B6 the list
  // is always populated first, so we just pass the summary in.
  session: SessionSummary | undefined
  onBack: () => void
  // Note: a future polish task will plumb 401 detection from
  // BinaryWS / mobileFetch into a callback similar to SessionsScreen's
  // onAuthExpired. Today the user simply taps Back, the sessions list
  // refetches, and that 401 path bounces them to LoginScreen.
}

// B6 — Session detail with embedded terminal. Read-most-of-the-time:
// the user watches the agent's output and occasionally taps a key
// or types a short command. Layout is mobile-first:
//
//   ┌─────────────────────────────┐
//   │ ← {session-name}      ⓘ      │  AppBar
//   ├─────────────────────────────┤
//   │                              │
//   │   xterm.js viewport          │  flex-1
//   │                              │
//   ├─────────────────────────────┤
//   │  Esc Tab Ctrl ↑↓←→ ⏎          │  Touch-keyboard row
//   ├─────────────────────────────┤
//   │ [____ input ____]   [Send]   │  Input bar
//   └─────────────────────────────┘
//
// Pinch-zoom, long-press copy, and resize-to-PTY are deferred to
// follow-up polish.
export function SessionDetailScreen({
  serverURL,
  token,
  sessionId,
  session,
  onBack,
}: Props) {
  const containerRef = useRef<HTMLDivElement>(null)
  const xtermRef = useRef<XTerm | null>(null)
  const fitRef = useRef<FitAddon | null>(null)
  const wsRef = useRef<BinaryWS | null>(null)
  const [connected, setConnected] = useState(false)
  const [input, setInput] = useState('')

  // Mount xterm once + connect WS.
  useEffect(() => {
    if (!containerRef.current) return

    const term = new XTerm({
      fontFamily:
        'JetBrains Mono Variable, JetBrains Mono, ui-monospace, Menlo, Consolas, monospace',
      fontSize: 13,
      lineHeight: 1.2,
      cursorBlink: true,
      convertEol: true,
      theme: buildTheme(),
      // Disable selection scrolling on mobile — pinch / native scroll
      // gestures are handled by the WebView, not xterm.
      scrollback: 5000,
    })
    const fit = new FitAddon()
    term.loadAddon(fit)

    term.open(containerRef.current)
    fit.fit()
    xtermRef.current = term
    fitRef.current = fit

    const url = mobileWSURL(
      serverURL,
      `/api/v1/sessions/${sessionId}/stream`,
      token,
    )
    const ws = new BinaryWS(url, {
      onOpen: () => setConnected(true),
      onClose: () => setConnected(false),
      onMessage: (data) => {
        // Server sends raw PTY bytes. xterm accepts Uint8Array directly.
        term.write(data)
      },
      onError: () => {
        // BinaryWS retries with backoff; we don't bail on first error.
        // TODO: detect a 401 close-code and call onAuthExpired.
      },
    })
    ws.start()
    wsRef.current = ws

    const onResize = () => {
      try {
        fit.fit()
      } catch {
        // Container not visible / 0-sized; ignore.
      }
    }
    window.addEventListener('resize', onResize)

    return () => {
      window.removeEventListener('resize', onResize)
      ws.close()
      term.dispose()
      xtermRef.current = null
      fitRef.current = null
      wsRef.current = null
    }
  }, [serverURL, token, sessionId])

  const sendBytes = useCallback((data: string) => {
    if (!wsRef.current) return
    wsRef.current.send(data)
  }, [])

  const sendInput = useCallback(async () => {
    const text = input
    if (!text) return
    setInput('')
    try {
      // Use REST /input endpoint rather than WS so we get the same
      // input-flow guarantees as web (server records the keystroke,
      // emits events, etc.). The terminal view will see the echo
      // arrive back through the stream.
      await mobileFetch(serverURL, `/api/v1/sessions/${sessionId}/input`, {
        method: 'POST',
        token,
        body: { text },
      })
    } catch (err) {
      // Surface as terminal output so the user sees what failed.
      xtermRef.current?.writeln(
        `\x1b[31m[input failed: ${err instanceof Error ? err.message : 'unknown'}]\x1b[0m`,
      )
    }
  }, [serverURL, token, sessionId, input])

  const sessionName = session?.name ?? sessionId

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col">
      <header className="sticky top-0 z-10 bg-background/95 backdrop-blur border-b border-border px-3 py-2 flex items-center gap-2">
        <Button variant="ghost" size="sm" onClick={onBack}>
          ← Back
        </Button>
        <div className="flex-1 min-w-0">
          <div className="text-sm font-medium truncate">{sessionName}</div>
          <div className="text-[10px] text-muted-foreground truncate">
            {session?.provider_id ?? '…'} · {connected ? 'connected' : 'connecting…'}
          </div>
        </div>
      </header>

      <main className="flex-1 flex flex-col">
        <div
          ref={containerRef}
          className="flex-1 bg-background overflow-hidden px-1 py-1"
          // Block iOS WebView's tap-to-zoom on the terminal area.
          style={{ touchAction: 'manipulation' }}
        />

        {/* Touch-keyboard row: keys WebKit doesn't deliver from
            soft keyboards on iOS (Esc, Ctrl, arrows, Tab). Each
            taps a raw byte sequence into the WS. */}
        <div className="flex gap-1 px-2 py-1.5 bg-card border-t border-border overflow-x-auto">
          <KeyButton onTap={() => sendBytes('\x1b')}>Esc</KeyButton>
          <KeyButton onTap={() => sendBytes('\t')}>Tab</KeyButton>
          <KeyButton onTap={() => sendBytes('\x03')}>Ctrl-C</KeyButton>
          <KeyButton onTap={() => sendBytes('\x04')}>Ctrl-D</KeyButton>
          <KeyButton onTap={() => sendBytes('\x1b[A')}>↑</KeyButton>
          <KeyButton onTap={() => sendBytes('\x1b[B')}>↓</KeyButton>
          <KeyButton onTap={() => sendBytes('\x1b[D')}>←</KeyButton>
          <KeyButton onTap={() => sendBytes('\x1b[C')}>→</KeyButton>
          <KeyButton onTap={() => sendBytes('\r')}>⏎</KeyButton>
        </div>

        <div className="flex gap-2 px-3 pb-3 pt-2 bg-card border-t border-border items-end">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Type a command…"
            rows={1}
            className="flex-1 resize-none rounded-md border border-border bg-input/40 px-3 py-2 text-[13px] text-foreground placeholder:text-muted-foreground/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring max-h-32"
            autoCapitalize="off"
            autoCorrect="off"
            spellCheck={false}
          />
          <Button
            variant="default"
            size="sm"
            onClick={() => {
              void sendInput()
            }}
            disabled={!input}
          >
            Send
          </Button>
        </div>
      </main>
    </div>
  )
}

function KeyButton({
  children,
  onTap,
}: {
  children: React.ReactNode
  onTap: () => void
}) {
  return (
    <button
      type="button"
      onClick={onTap}
      className="shrink-0 rounded-md border border-border bg-background hover:bg-accent/10 active:bg-accent/20 px-2.5 py-1 text-xs text-foreground tabular-nums select-none"
    >
      {children}
    </button>
  )
}

function readVar(name: string): string {
  return getComputedStyle(document.documentElement)
    .getPropertyValue(name)
    .trim()
}

function buildTheme() {
  // Mirror the palette mapping from web's Terminal.tsx — keeps the
  // terminal visually consistent across surfaces.
  return {
    background: readVar('--background') || '#13151b',
    foreground: readVar('--foreground') || '#f5f5f5',
    cursor: readVar('--accent') || '#ff7b35',
    cursorAccent: readVar('--background') || '#13151b',
    selectionBackground: readVar('--accent') || '#ff7b35',
    selectionForeground: readVar('--accent-foreground') || '#1a1a1a',
    black: '#0a0a0c',
    red: '#e85b5b',
    green: '#4ad295',
    yellow: '#e8c050',
    blue: '#5b9eff',
    magenta: '#c084fc',
    cyan: '#5be8d4',
    white: '#e5e5e5',
    brightBlack: '#3a3a3a',
    brightRed: '#ff7373',
    brightGreen: '#5be0a8',
    brightYellow: '#ffd270',
    brightBlue: '#7eb4ff',
    brightMagenta: '#d8a0ff',
    brightCyan: '#7af0dc',
    brightWhite: '#ffffff',
  }
}
