import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
} from 'react'
import { Terminal as XTerm } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'

import { useAuth } from '@/stores/auth'
import { useTheme } from '@/stores/theme'
import { BinaryWS, wsURL } from '@/lib/ws'
import { resizeSession, uploadSessionFile } from '@/lib/sessions'
import { terminalBufferText } from './terminal-text'
import { AttachmentTray, type AttachmentItem } from './AttachmentTray'

interface TerminalProps {
  sessionId: string
}

export interface TerminalHandle {
  /**
   * Send a raw byte sequence to the PTY's stdin (e.g. ESC '\x1b',
   * Ctrl+C '\x03'). Kept on the handle so callers (header buttons,
   * inspector panels) can inject input without touching xterm.
   */
  sendInput: (data: string) => void
  /**
   * Multipart-upload `file` to the gateway and write the returned
   * server-side path into the PTY so the running CLI can attach it
   * as context. Used by the header "attach image" button.
   */
  uploadFile: (file: File) => Promise<void>
  /**
   * Return the full buffer (scrollback + viewport) as clean plain text,
   * trailing per-line padding and blank rows stripped. Feeds the
   * "Select & copy" dialog, which renders it as natively-selectable DOM
   * text so touch users can select a portion the canvas won't let them.
   */
  getBufferText: () => string
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
  const { t } = useTranslation()
  const [dragActive, setDragActive] = useState(false)
  const [pendingAttachments, setPendingAttachments] = useState<AttachmentItem[]>([])
  const rootRef = useRef<HTMLDivElement>(null)

  const sendInput = useCallback((data: string) => {
    const ws = wsRef.current
    if (!ws || !ws.isOpen()) return
    const enc = new TextEncoder().encode(data)
    ws.send(
      enc.buffer.slice(
        enc.byteOffset,
        enc.byteOffset + enc.byteLength,
      ) as ArrayBuffer,
    )
  }, [])

  // Upload an image and paste the resolved server path back into
  // the PTY so the CLI can attach it. Path-only — never the bytes —
  // because Claude / Codex / Antigravity all consume images via filename
  // references, not stdin streams.
  const uploadFile = useCallback(
    async (file: File): Promise<void> => {
      if (!file.type.startsWith('image/')) {
        toast.error(t('web.sessions.terminal.uploadInvalidTypeToast'), {
          description: file.type || file.name,
        })
        return
      }
      const toastId = `session-upload:${sessionId}`
      toast.loading(t('web.sessions.terminal.uploadingToast'), { id: toastId })
      try {
        const res = await uploadSessionFile(sessionId, file)
        setPendingAttachments((a) => [...a, { path: res.path, name: file.name }])
        toast.dismiss(toastId)
      } catch (err) {
        toast.error(t('web.sessions.terminal.uploadFailedToast'), {
          id: toastId,
          description: (err as Error).message,
        })
      }
    },
    [sessionId, t],
  )

  const insertAttachments = useCallback(() => {
    if (pendingAttachments.length === 0) return
    sendInput(pendingAttachments.map((a) => a.path).join(' ') + ' ')
    setPendingAttachments([])
  }, [pendingAttachments, sendInput])

  const removeAttachment = useCallback((index: number) => {
    setPendingAttachments((a) => a.filter((_, i) => i !== index))
  }, [])

  const clearAttachments = useCallback(() => setPendingAttachments([]), [])

  const getBufferText = useCallback(() => terminalBufferText(xtermRef.current), [])

  useImperativeHandle(
    ref,
    () => ({ sendInput, uploadFile, getBufferText }),
    [sendInput, uploadFile, getBufferText],
  )

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
      onMessage: (data) => {
        term.write(data)
      },
      onClose: () => {
        if (!alive) return
        term.writeln('')
        term.writeln('\x1b[33m[disconnected — reconnecting…]\x1b[0m')
      },
      onReconnect: () => {
        if (!alive) return
        term.writeln('\x1b[32m[reconnected]\x1b[0m')
      },
      onGiveUp: () => {
        if (!alive) return
        term.writeln(
          '\x1b[31m[connection lost — refresh the page to reconnect]\x1b[0m',
        )
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

    // Touch-scroll forwarding. A phone has no mouse wheel, and a
    // full-screen TUI that has grabbed the mouse (Claude Code / Codex /
    // Antigravity all enable mouse tracking, so modes.mouseTrackingMode is
    // not 'none') runs in the alternate screen — which has no xterm
    // scrollback to scroll, AND xterm doesn't translate a finger swipe
    // into the wheel events the app is waiting for. The conversation is
    // scrolled by the APP itself in response to wheel input, exactly
    // like a desktop mouse wheel. So translate a one-finger vertical
    // swipe into SGR (1006) wheel events sent straight to the PTY. For
    // a plain shell (mouseTrackingMode 'none') we leave the event alone
    // so xterm's own viewport scroll / native behaviour still works.
    const SWIPE_STEP = 18 // px of finger travel per wheel tick
    const touchHost = containerRef.current
    let touchActive = false
    let lastTouchY = 0
    let touchAccum = 0
    const sendWheel = (up: boolean) => {
      // SGR mouse: button 64 = wheel up, 65 = wheel down; press ('M').
      // Report the pointer at screen centre so the app treats it as a
      // scroll over its content region.
      const col = Math.max(1, Math.floor(term.cols / 2))
      const row = Math.max(1, Math.floor(term.rows / 2))
      const seq = `\x1b[<${up ? 64 : 65};${col};${row}M`
      const enc = new TextEncoder().encode(seq)
      ws.send(
        enc.buffer.slice(enc.byteOffset, enc.byteOffset + enc.byteLength) as ArrayBuffer,
      )
    }
    const onTouchStart = (e: TouchEvent) => {
      if (e.touches.length !== 1 || term.modes.mouseTrackingMode === 'none') {
        touchActive = false
        return
      }
      touchActive = true
      lastTouchY = e.touches[0].clientY
      touchAccum = 0
    }
    const onTouchMove = (e: TouchEvent) => {
      if (!touchActive || e.touches.length !== 1) return
      if (term.modes.mouseTrackingMode === 'none') return
      e.preventDefault() // stop the page/pane from scrolling instead
      const y = e.touches[0].clientY
      touchAccum += y - lastTouchY
      lastTouchY = y
      // Finger moving DOWN (positive delta) reveals earlier content →
      // wheel up; finger up → wheel down.
      while (Math.abs(touchAccum) >= SWIPE_STEP) {
        const up = touchAccum > 0
        sendWheel(up)
        touchAccum += up ? -SWIPE_STEP : SWIPE_STEP
      }
    }
    const onTouchEnd = () => {
      touchActive = false
    }
    touchHost?.addEventListener('touchstart', onTouchStart, { passive: true })
    touchHost?.addEventListener('touchmove', onTouchMove, { passive: false })
    touchHost?.addEventListener('touchend', onTouchEnd, { passive: true })
    touchHost?.addEventListener('touchcancel', onTouchEnd, { passive: true })

    // Coalesce resize bursts into one fit per animation frame. Calling
    // fit() synchronously inside the ResizeObserver callback mutates
    // the DOM mid-notification ("ResizeObserver loop completed with
    // undelivered notifications") and, at some widths, oscillates;
    // deferring to rAF lets the frame's layout settle first and folds
    // a window-drag storm of notifications into a single fit.
    let fitRaf = 0
    const scheduleFit = () => {
      if (!alive || fitRaf) return
      fitRaf = requestAnimationFrame(() => {
        fitRaf = 0
        if (!alive) return
        try {
          fit.fit()
        } catch {
          /* element not measured yet */
        }
      })
    }
    // Observe the in-flow root (not the absolute-positioned host).
    // WebKit's ResizeObserver fires late on absolute elements nested
    // inside `flex-1 min-h-0`, leaving the PTY a few cols too wide
    // after a sidebar/banner-driven layout shift — the user sees
    // input wrap past the visible right edge after 3-4 lines.
    const ro = new ResizeObserver(scheduleFit)
    if (rootRef.current) ro.observe(rootRef.current)

    // Safari/iOS don't surface keyboard show/hide or address-bar
    // collapse through window resize alone — the visualViewport API
    // is the only event that reliably fires. Without this, opening
    // the soft keyboard would clip xterm's bottom rows without ever
    // refitting, which reads as "chat goes below the browser window."
    const vv =
      typeof window !== 'undefined' ? window.visualViewport : null
    vv?.addEventListener('resize', scheduleFit)
    vv?.addEventListener('scroll', scheduleFit)

    // The synchronous fit() at open ran before the first post-mount
    // layout had settled (and before a webfont monospace face, on
    // deployments that ship one, has loaded) — both shift the measured
    // cell width, so re-fit once each has settled. A PTY left a few
    // columns too wide is exactly what reads as "long input doesn't
    // wrap, it runs off the right edge until I resize the window."
    scheduleFit()
    void document.fonts?.ready?.then(scheduleFit)

    return () => {
      alive = false
      if (fitRaf) cancelAnimationFrame(fitRaf)
      touchHost?.removeEventListener('touchstart', onTouchStart)
      touchHost?.removeEventListener('touchmove', onTouchMove)
      touchHost?.removeEventListener('touchend', onTouchEnd)
      touchHost?.removeEventListener('touchcancel', onTouchEnd)
      ro.disconnect()
      vv?.removeEventListener('resize', scheduleFit)
      vv?.removeEventListener('scroll', scheduleFit)
      ws.close()
      term.dispose()
      xtermRef.current = null
      fitRef.current = null
      wsRef.current = null
    }
  }, [sessionId, token])

  // Clipboard + drag-and-drop image attach. Browsers don't expose
  // clipboard images through xterm's default paste hook (which only
  // looks at text/plain), so we shadow paste at capture phase: when
  // the clipboard contains an image, we own the event and upload it;
  // otherwise we let xterm's normal text-paste flow proceed.
  useEffect(() => {
    const el = containerRef.current
    if (!el) return

    const onPaste = (e: ClipboardEvent) => {
      const dt = e.clipboardData
      if (!dt) return
      // DataTransferItemList may include both a text/plain fallback
      // (e.g. screenshot tools sometimes copy the filename too) and
      // an image/*; prefer the image entry. Files-only paste (e.g.
      // Finder → copy → ⌘V) ends up in dt.files.
      const items = Array.from(dt.items)
      const imageItem = items.find(
        (it) => it.kind === 'file' && it.type.startsWith('image/'),
      )
      const file = imageItem?.getAsFile() ?? null
      const filesImage =
        file ?? Array.from(dt.files).find((f) => f.type.startsWith('image/'))
      if (!filesImage) return // let xterm handle text paste
      e.preventDefault()
      e.stopPropagation()
      void uploadFile(filesImage)
    }

    const hasFiles = (dt: DataTransfer | null): boolean => {
      if (!dt) return false
      return Array.from(dt.types).includes('Files')
    }

    const onDragEnter = (e: DragEvent) => {
      if (!hasFiles(e.dataTransfer)) return
      e.preventDefault()
      setDragActive(true)
    }
    const onDragOver = (e: DragEvent) => {
      if (!hasFiles(e.dataTransfer)) return
      // preventDefault on dragover is required for drop to fire.
      e.preventDefault()
      if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy'
    }
    const onDragLeave = (e: DragEvent) => {
      // Fires when crossing into a child element too — only reset
      // when the cursor actually leaves the container box.
      if (e.target !== el) return
      setDragActive(false)
    }
    const onDrop = (e: DragEvent) => {
      if (!hasFiles(e.dataTransfer)) return
      e.preventDefault()
      setDragActive(false)
      const files = Array.from(e.dataTransfer?.files ?? [])
      const imageFile = files.find((f) => f.type.startsWith('image/'))
      if (!imageFile) {
        toast.error(t('web.sessions.terminal.uploadInvalidTypeToast'), {
          description: files[0]?.name,
        })
        return
      }
      void uploadFile(imageFile)
    }

    // Paste at capture phase: xterm's own listener lives on its
    // internal helper textarea and runs at the target. By taking
    // the event in capture we can decide before xterm whether the
    // payload is an image (we handle it) or text (we step aside).
    el.addEventListener('paste', onPaste, true)
    el.addEventListener('dragenter', onDragEnter)
    el.addEventListener('dragover', onDragOver)
    el.addEventListener('dragleave', onDragLeave)
    el.addEventListener('drop', onDrop)
    return () => {
      el.removeEventListener('paste', onPaste, true)
      el.removeEventListener('dragenter', onDragEnter)
      el.removeEventListener('dragover', onDragOver)
      el.removeEventListener('dragleave', onDragLeave)
      el.removeEventListener('drop', onDrop)
    }
  }, [uploadFile, t])

  // Refresh xterm theme when the site theme changes.
  useEffect(() => {
    const term = xtermRef.current
    if (!term) return
    term.options.theme = buildTheme(themeApplied)
    fitRef.current?.fit()
  }, [themeApplied])

  // While attachments are staged, Esc clears them instead of reaching the
  // CLI. Capture-phase window listener wins before xterm's own key handling.
  useEffect(() => {
    if (pendingAttachments.length === 0) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return
      // Only swallow Esc when focus is inside the terminal pane. Otherwise an
      // open dialog (Radix closes on a document bubble-phase Esc) would be
      // robbed of its Escape and the staged tray cleared unexpectedly.
      const root = rootRef.current
      if (root && !root.contains(e.target as Node)) return
      e.preventDefault()
      e.stopPropagation()
      setPendingAttachments([])
    }
    window.addEventListener('keydown', onKey, true)
    return () => window.removeEventListener('keydown', onKey, true)
  }, [pendingAttachments.length])

  return (
    <div ref={rootRef} className="h-full w-full bg-background relative overflow-hidden">
      {/* `contain: layout` + `overflow-hidden` isolate xterm's inner
          viewport from the surrounding flex/scroll layout: a line
          wider than the pane can't escape up to <main>'s scrollbar
          (which would re-measure the pane and re-run fit() — the
          jitter feedback loop). We previously used `absolute inset-0`
          for the same goal, but on WebKit ResizeObserver fires late
          on absolute-positioned elements inside `flex-1 min-h-0`,
          leaving fit() reading a stale size after layout shifts.
          Staying in-flow with containment gives us the same isolation
          and reliable RO timing across Safari/Chrome. */}
      <div
        ref={containerRef}
        className="h-full w-full p-3 overflow-hidden"
        style={{ contain: 'layout paint' }}
      />
      {dragActive && (
        <div className="pointer-events-none absolute inset-2 rounded-md border-2 border-dashed border-accent/70 bg-accent/10 flex items-center justify-center">
          <div className="text-[12px] font-mono text-accent">
            {t('web.sessions.terminal.dropToAttach')}
          </div>
        </div>
      )}
      <AttachmentTray
        items={pendingAttachments}
        onRemove={removeAttachment}
        onInsert={insertAttachments}
        onClear={clearAttachments}
      />
    </div>
  )
})
