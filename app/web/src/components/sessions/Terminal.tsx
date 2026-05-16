import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
} from 'react'
import { Terminal as XTerm } from '@xterm/xterm'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'

import { useAuth } from '@/stores/auth'
import { useTheme } from '@/stores/theme'
import { BinaryWS, wsURL } from '@/lib/ws'
import { uploadSessionFile } from '@/lib/sessions'

// Heuristic monospace-cell aspect ratio for JetBrains Mono /
// generic monospace. cellWidth ~= 0.6 × fontSize, cellHeight ~=
// fontSize × lineHeight. Used by computeFontSize to fit a fixed
// xterm grid into the container by scaling font alone.
const CELL_W_RATIO = 0.6
const LINE_HEIGHT = 1.25

// Bounds keep font size sane on extreme container sizes:
// minimum keeps text legible on a small split view; maximum keeps
// the TUI from looking absurdly chunky on an ultrawide monitor.
const MIN_FONT_SIZE = 9
const MAX_FONT_SIZE = 28

// computeFontSize returns the largest pixel font size such that a
// `cols × rows` xterm grid fits inside a container of
// `containerWidth × containerHeight`. Picks the limiting axis so
// the grid fills one dimension entirely without overflowing the
// other.
function computeFontSize(
  cols: number,
  rows: number,
  containerWidth: number,
  containerHeight: number,
): number {
  if (cols <= 0 || rows <= 0) return 13
  const cellWFromContainer = containerWidth / cols
  const cellHFromContainer = containerHeight / rows
  const fromWidth = cellWFromContainer / CELL_W_RATIO
  const fromHeight = cellHFromContainer / LINE_HEIGHT
  const raw = Math.floor(Math.min(fromWidth, fromHeight))
  return Math.max(MIN_FONT_SIZE, Math.min(MAX_FONT_SIZE, raw))
}

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
  // Grid dims locked to the server's PTY canvas (delivered via the
  // first pty_size control frame). Container ResizeObserver scales
  // fontSize against these to fill the browser viewport.
  const gridRef = useRef<{ cols: number; rows: number }>({
    cols: 100,
    rows: 32,
  })
  const wsRef = useRef<BinaryWS | null>(null)
  const token = useAuth((s) => s.token)
  const themeApplied = useTheme((s) => s.applied())
  const { t } = useTranslation()
  const [dragActive, setDragActive] = useState(false)

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

  useImperativeHandle(ref, () => ({ sendInput, uploadFile }), [
    sendInput,
    uploadFile,
  ])

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
    const links = new WebLinksAddon()
    term.loadAddon(links)
    term.open(containerRef.current)
    xtermRef.current = term
    // Initial grid: matches the in-ref default; the server's first
    // pty_size frame overrides as soon as the WS connects.
    term.resize(gridRef.current.cols, gridRef.current.rows)

    // alive flips false on cleanup so any straggler ResizeObserver
    // / onOpen callbacks scheduled before unmount don't act on a
    // disposed terminal.
    let alive = true

    // applyFontSize fits the locked grid into the current
    // container by recomputing pixel font size. Pure-local: never
    // touches the PTY. The server's pty_size frame already locked
    // the grid; this just decides how big to render each cell.
    const applyFontSize = () => {
      if (!alive || !containerRef.current) return
      const { cols, rows } = gridRef.current
      const w = containerRef.current.clientWidth
      const h = containerRef.current.clientHeight
      if (w <= 0 || h <= 0) return
      const size = computeFontSize(cols, rows, w, h)
      if (term.options.fontSize !== size) {
        term.options.fontSize = size
      }
      // term.resize is idempotent when dims haven't changed; the
      // call ensures xterm reflows after font/size changes so the
      // grid actually paints at the new cell dimensions.
      term.resize(cols, rows)
    }

    // `?client=web` tags this subscriber so the server's
    // Manager.Resize gate can suppress this client's resize
    // requests while a mobile client is attached. With auto-
    // fontSize the web grid is always pinned to the server's PTY
    // size; the gate is now a safety net rather than a
    // user-visible mechanism.
    const ws = new BinaryWS(
      wsURL(`/api/v1/sessions/${sessionId}/stream?client=web`, token),
      {
        onMessage: (data) => term.write(data),
        onText: (text) => {
          // Currently the only control frame is pty_size. Quietly
          // ignore anything else — future server-pushed control
          // payloads can land alongside.
          try {
            const msg = JSON.parse(text) as {
              type?: string
              cols?: number
              rows?: number
            }
            if (
              msg.type === 'pty_size' &&
              typeof msg.cols === 'number' &&
              typeof msg.rows === 'number' &&
              msg.cols > 0 &&
              msg.rows > 0
            ) {
              gridRef.current = { cols: msg.cols, rows: msg.rows }
              term.resize(msg.cols, msg.rows)
              applyFontSize()
            }
          } catch {
            /* malformed control frame — ignore */
          }
        },
        onClose: () => {
          if (!alive) return
          term.writeln('')
          term.writeln('\x1b[33m[disconnected — reconnecting…]\x1b[0m')
        },
      },
    )
    wsRef.current = ws
    ws.start()

    term.onData((d) => {
      const enc = new TextEncoder().encode(d)
      ws.send(
        enc.buffer.slice(
          enc.byteOffset,
          enc.byteOffset + enc.byteLength,
        ) as ArrayBuffer,
      )
    })

    const ro = new ResizeObserver(applyFontSize)
    ro.observe(containerRef.current)
    applyFontSize() // initial fit before first pty_size lands

    return () => {
      alive = false
      ro.disconnect()
      ws.close()
      term.dispose()
      xtermRef.current = null
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

  // Refresh xterm theme when the site theme changes. No fit call
  // here — the grid is server-pinned and font sizing is handled by
  // the container ResizeObserver inside the mount effect.
  useEffect(() => {
    const term = xtermRef.current
    if (!term) return
    term.options.theme = buildTheme(themeApplied)
  }, [themeApplied])

  return (
    <div className="h-full w-full bg-background relative">
      <div ref={containerRef} className="h-full w-full p-3" />
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
