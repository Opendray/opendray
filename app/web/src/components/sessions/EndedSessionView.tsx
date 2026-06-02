import { useEffect, useRef } from 'react'
import { Terminal as XTerm } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { useTranslation } from 'react-i18next'

import { api } from '@/lib/api'
import { useTheme } from '@/stores/theme'

interface EndedSessionViewProps {
  sessionId: string
}

function readVar(name: string): string {
  return getComputedStyle(document.documentElement)
    .getPropertyValue(name)
    .trim()
}

function buildTheme(applied: 'light' | 'dark') {
  return {
    background: readVar('--background') || (applied === 'dark' ? '#13151b' : '#fafafa'),
    foreground: readVar('--foreground') || (applied === 'dark' ? '#f5f5f5' : '#1a1a1a'),
    cursor: 'transparent',
    cursorAccent: 'transparent',
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

/**
 * Read-only terminal showing the ring-buffer snapshot of an ended
 * session. No WebSocket — fixes the reconnect loop that happens when
 * the workbench tries to stream a process that has already exited.
 */
export function EndedSessionView({ sessionId }: EndedSessionViewProps) {
  const { t } = useTranslation()
  const containerRef = useRef<HTMLDivElement>(null)
  const xtermRef = useRef<XTerm | null>(null)
  const fitRef = useRef<FitAddon | null>(null)
  const themeApplied = useTheme((s) => s.applied())

  useEffect(() => {
    if (!containerRef.current) return

    const term = new XTerm({
      fontFamily:
        '"JetBrains Mono Variable", "JetBrains Mono", ui-monospace, Menlo, Consolas, monospace',
      fontSize: 13,
      lineHeight: 1.25,
      cursorBlink: false,
      cursorStyle: 'underline',
      disableStdin: true,
      theme: buildTheme(themeApplied),
      scrollback: 8_000,
      allowProposedApi: true,
      convertEol: true,
    })
    const fit = new FitAddon()
    term.loadAddon(fit)
    term.open(containerRef.current)
    fit.fit()
    xtermRef.current = term
    fitRef.current = fit

    let cancelled = false
    api(`/api/v1/sessions/${sessionId}/buffer`, { raw: true })
      .then(async (res) => {
        if (cancelled) return
        if (res instanceof Response) {
          const buf = await res.arrayBuffer()
          term.write(new Uint8Array(buf))
        }
      })
      .catch(() => {
        if (!cancelled) {
          term.writeln(`\x1b[31m${t('web.sessions.ended.bufferUnavailable')}\x1b[0m`)
        }
      })
      .finally(() => {
        if (!cancelled) {
          term.writeln('')
          term.writeln(`\x1b[33m${t('web.sessions.ended.readOnlyBanner')}\x1b[0m`)
        }
      })

    // rAF-coalesced fit — see Terminal.tsx for the full rationale
    // (mid-notification DOM mutation + scrollbar-toggle oscillation).
    let fitRaf = 0
    const scheduleFit = () => {
      if (cancelled || fitRaf) return
      fitRaf = requestAnimationFrame(() => {
        fitRaf = 0
        if (cancelled) return
        try {
          fit.fit()
        } catch {
          /* not measured yet */
        }
      })
    }
    const ro = new ResizeObserver(scheduleFit)
    ro.observe(containerRef.current)
    scheduleFit()
    void document.fonts?.ready?.then(scheduleFit)

    return () => {
      cancelled = true
      if (fitRaf) cancelAnimationFrame(fitRaf)
      ro.disconnect()
      term.dispose()
      xtermRef.current = null
      fitRef.current = null
    }
  }, [sessionId, themeApplied, t])

  return (
    <div className="h-full w-full bg-background relative overflow-hidden">
      {/* Clipped + absolutely sized so replay content can't escape to
          the scrollable <main> and start a fit()/scrollbar feedback
          loop — same isolation as the live Terminal. */}
      <div ref={containerRef} className="absolute inset-0 p-2 overflow-hidden" />
    </div>
  )
}
