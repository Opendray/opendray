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
import { Copy } from 'lucide-react'

import { Button } from 'shared-ui/primitives/button'
import { useAuth } from '@/stores/auth'
import { useTheme } from '@/stores/theme'
import { BinaryWS, wsURL } from '@/lib/ws'
import { resizeSession, uploadSessionFile } from '@/lib/sessions'
import { copyText } from '@/lib/clipboard'
import { DetectedURLs } from './DetectedURLs'
import { extractURLs, stripANSI } from './url-extractor'

// Keep the last N URLs we've seen in this session. 50 is enough for
// any realistic OAuth-heavy session (each CLI prints 1-2 auth URLs),
// and bounds the dialog scroll length on a long-lived session that
// keeps printing links (e.g. one with a notes vault git push hint).
const MAX_DETECTED_URLS = 50

// Sliding window the URL scanner keeps in-memory across PTY frames.
// PTY frames are byte-arbitrary boundaries that can land mid-URL, so
// we prepend this tail to each new chunk before regex-matching.
// 4 KB is much larger than even claude-code's full OAuth URL (~600
// chars), so any single URL fits entirely inside the window with
// room to spare.
const URL_SCAN_TAIL_BYTES = 4096

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
   * Copy the current selection, or the whole buffer when nothing is
   * selected, to the clipboard. Used by the header "Copy output"
   * button — the always-available path for touch devices that can't
   * tap-select the canvas.
   */
  copyAll: () => void
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
  // URLs spotted in this session's PTY output. Surfaced by the
  // floating <DetectedURLs /> badge so the operator can tap one to
  // open in a browser instead of fighting with line-wrapped text in
  // the terminal — particularly useful for OAuth flows on mobile.
  const [detectedURLs, setDetectedURLs] = useState<string[]>([])
  const rootRef = useRef<HTMLDivElement>(null)
  // Position (root-relative px) of the contextual copy pill, anchored
  // at where the selection ended. null = no selection, pill hidden.
  // `below` flips the pill under the point when selecting near the top
  // edge so it never clips off-screen.
  const [pill, setPill] = useState<{ x: number; y: number; below: boolean } | null>(
    null,
  )

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
  // because Claude / Codex / Gemini all consume images via filename
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
        sendInput(res.path)
        toast.success(t('web.sessions.terminal.uploadedToast'), {
          id: toastId,
          description: res.path,
        })
      } catch (err) {
        toast.error(t('web.sessions.terminal.uploadFailedToast'), {
          id: toastId,
          description: (err as Error).message,
        })
      }
    },
    [sessionId, sendInput, t],
  )

  const writeClip = useCallback(
    async (text: string) => {
      if (await copyText(text)) {
        toast.success(t('web.sessions.terminal.copiedToast'))
      } else {
        toast.error(t('web.sessions.terminal.copyFailedToast'))
      }
    },
    [t],
  )

  // Copy the current selection — triggered by the contextual pill that
  // only appears while text is selected.
  const copySelection = useCallback(() => {
    const text = xtermRef.current?.getSelection() ?? ''
    if (text.trim()) void writeClip(text)
  }, [writeClip])

  // Copy the selection if present, otherwise the whole buffer. Exposed
  // on the handle for the session header's "Copy output" button — the
  // reliable path on iOS, where tap-selecting xterm's canvas doesn't
  // work and the async Clipboard API is unavailable over plain-http LAN
  // (copyText falls back to execCommand).
  const copyAll = useCallback(() => {
    const term = xtermRef.current
    if (!term) return
    let text = term.getSelection()
    if (!text) {
      term.selectAll()
      text = term.getSelection()
      term.clearSelection()
    }
    if (!text.trim()) {
      toast(t('web.sessions.terminal.copyEmptyToast'))
      return
    }
    void writeClip(text)
  }, [writeClip, t])

  useImperativeHandle(ref, () => ({ sendInput, uploadFile, copyAll }), [
    sendInput,
    uploadFile,
    copyAll,
  ])

  // Mount xterm + WS once per session id.
  useEffect(() => {
    if (!containerRef.current || !token) return

    // URL scanner state. Lives inside the effect so it resets when
    // the session changes (different session ID → fresh URL list).
    const textDecoder = new TextDecoder('utf-8', { fatal: false })
    let urlScanTail = ''

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

    // Hide the copy pill the moment a selection is cleared (click
    // elsewhere, typing, scroll-reset). Showing/positioning it happens
    // on pointerup — see the pill-anchor effect. term.dispose() (cleanup
    // below) tears this listener down with it.
    term.onSelectionChange(() => {
      if (!term.hasSelection()) setPill(null)
    })

    // alive flips false on cleanup so any straggler resize/onOpen
    // callbacks scheduled before unmount don't fire `/resize` against
    // a session that's just transitioned to ended — server returns
    // 404 and browser logs it red in console even though we .catch().
    let alive = true

    const ws = new BinaryWS(wsURL(`/api/v1/sessions/${sessionId}/stream`, token), {
      onMessage: (data) => {
        term.write(data)
        // Scan the same bytes we just gave xterm. Strip ANSI so colour
        // resets in the middle of a URL don't truncate it; combine
        // with the carry-over tail so URLs spanning the boundary
        // between WS frames still match.
        try {
          const chunk = textDecoder.decode(data, { stream: true })
          const combined = urlScanTail + stripANSI(chunk)
          const found = extractURLs(combined)
          if (found.length > 0) {
            setDetectedURLs((prev) => {
              const seen = new Set(prev)
              const next = [...prev]
              for (const u of found) {
                if (!seen.has(u)) {
                  seen.add(u)
                  next.push(u)
                }
              }
              // Cap retention — newest at the tail; oldest get
              // dropped first when we overflow.
              return next.length > MAX_DETECTED_URLS
                ? next.slice(next.length - MAX_DETECTED_URLS)
                : next
            })
          }
          urlScanTail =
            combined.length > URL_SCAN_TAIL_BYTES
              ? combined.slice(combined.length - URL_SCAN_TAIL_BYTES)
              : combined
        } catch {
          // URL extraction is best-effort. If decode / regex throws
          // (malformed UTF-8, weird ANSI), drop this chunk's scan
          // and keep the terminal stream flowing.
        }
      },
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
    const ro = new ResizeObserver(scheduleFit)
    ro.observe(containerRef.current)

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
      ro.disconnect()
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

  // Anchor the copy pill where the selection ended. pointerup covers
  // both mouse-drag and touch; we defer one frame so xterm finalizes
  // the selection first, then place the pill at the pointer (clamped
  // inside the box, flipped below the point near the top edge).
  useEffect(() => {
    const root = rootRef.current
    if (!root) return
    const onPointerUp = (e: PointerEvent) => {
      requestAnimationFrame(() => {
        const term = xtermRef.current
        if (!term || !term.hasSelection()) {
          setPill(null)
          return
        }
        const rect = root.getBoundingClientRect()
        const rawY = e.clientY - rect.top
        setPill({
          x: Math.min(Math.max(e.clientX - rect.left, 48), rect.width - 48),
          y: rawY,
          below: rawY < 44,
        })
      })
    }
    root.addEventListener('pointerup', onPointerUp)
    return () => root.removeEventListener('pointerup', onPointerUp)
  }, [])

  return (
    <div ref={rootRef} className="h-full w-full bg-background relative overflow-hidden">
      {/* The xterm host is absolutely positioned + clipped so its size
          is driven SOLELY by this (flex/viewport-sized) pane and never
          by its own rendered content. Without the clip, a line wider
          than the pane escapes up to the scrollable <main>, toggling
          its scrollbar, which re-measures the pane and re-runs fit() —
          a feedback loop that reads as the terminal "jittering" as you
          type a long line. Clipping breaks the loop at the source and
          forces xterm to wrap at the real visible width. */}
      <div ref={containerRef} className="absolute inset-0 p-3 overflow-hidden" />
      <DetectedURLs urls={detectedURLs} />
      {pill && (
        <Button
          type="button"
          variant="secondary"
          size="sm"
          // preventDefault on mousedown so clicking the pill doesn't
          // blur/clear the xterm selection before copySelection reads it.
          onMouseDown={(e) => e.preventDefault()}
          onClick={copySelection}
          title={t('web.sessions.terminal.copySelectionTooltip')}
          style={{
            left: pill.x,
            top: pill.y,
            transform: `translate(-50%, ${pill.below ? '8px' : 'calc(-100% - 8px)'})`,
          }}
          className="absolute z-20 h-7 gap-1.5 px-2.5 text-[11px] border border-border shadow-md"
        >
          <Copy className="size-3.5" />
          {t('web.sessions.terminal.copySelection')}
        </Button>
      )}
      {dragActive && (
        <div className="pointer-events-none absolute inset-2 rounded-md border-2 border-dashed border-accent/70 bg-accent/10 flex items-center justify-center">
          <div className="text-[12px] font-mono text-accent">
            {t('web.sessions.terminal.dropToAttach')}
          </div>
        </div>
      )}
    </div>
  )
})
