import { api } from './api'
import { readFile } from './fs'
import { BinaryWS, wsURL } from './ws'

// Canvas (beta): a live HTML preview any cloud agent pushes to via the
// canvas_render MCP tool. Scoped by cwd (project working dir), exactly like the
// session Inspector tabs. The operator annotates the preview and that feedback
// is seeded back into the session as a prompt.

// What a canvas IS. The Canvas is a general visual surface, not only screen
// mocks: an agent also draws flowcharts, mind maps and relationship diagrams.
export type CanvasKind = 'ui' | 'flow' | 'mindmap' | 'graph' | 'doc'

export interface CanvasSummary {
  id: string
  cwd: string
  slug: string
  title: string
  kind: CanvasKind
  version: number
  created_at: string
  updated_at: string
}

export interface CanvasArtifact extends CanvasSummary {
  html: string
}

// CanvasAnnotation is one operator mark. Coordinates are percentages of the
// iframe viewport (0–100) so they survive a resize. `kind` is 'pin' (a point,
// usually on a specific element) or 'region' (a dragged rectangle).
export interface CanvasAnnotation {
  n: number
  kind: 'pin' | 'region'
  note: string
  selector?: string
  html?: string
  // For a mock region: the components found inside the framed rectangle
  // (label + snippet), so the agent knows the real elements, not just coords.
  elements?: string[]
  x?: number
  y?: number
  w?: number
  h?: number
}

export interface CanvasFeedback {
  session_id: string
  message?: string
  annotations: CanvasAnnotation[]
}

// listCanvas returns the artifact summaries (no html) for a project.
export async function listCanvas(cwd: string): Promise<CanvasSummary[]> {
  const res = await api<{ artifacts: CanvasSummary[] }>(
    `/api/v1/canvas?cwd=${encodeURIComponent(cwd)}`,
  )
  return res.artifacts ?? []
}

// getCanvas returns one artifact with its full html.
export async function getCanvas(id: string): Promise<CanvasArtifact> {
  return api<CanvasArtifact>(`/api/v1/canvas/${encodeURIComponent(id)}`)
}

// renderCanvas pushes/replaces an artifact (used by the Files "Preview" action;
// agents use the canvas_render MCP tool which hits the same endpoint).
export async function renderCanvas(input: {
  cwd: string
  slug?: string
  title?: string
  kind?: CanvasKind
  html: string
}): Promise<CanvasArtifact> {
  return api<CanvasArtifact>('/api/v1/canvas', { method: 'POST', body: input })
}

// isPreviewableHtml reports whether a path is an HTML file the Files panel can
// render on the Canvas.
export function isPreviewableHtml(path: string): boolean {
  return /\.(html?|htm)$/i.test(path)
}

// previewFileToCanvas reads an HTML file from the project and pushes it to the
// Canvas as a stable per-file artifact (slug "file:<path>"), so the operator
// can preview + annotate a file the agent wrote without the agent re-rendering.
export async function previewFileToCanvas(cwd: string, path: string): Promise<void> {
  const html = await readFile(path)
  if (html == null) throw new Error('file not found or unreadable')
  const name = path.split('/').pop() || path
  await renderCanvas({ cwd, slug: `file:${path}`, title: name, html })
}

// A target mock the agent should update in place (its slug/title), so an
// ambiguous request iterates on the selected canvas instead of making a new one.
export interface CanvasTarget {
  slug?: string
  title?: string
}

// requestDesign is opendray's deterministic Canvas entry point: the operator's
// ask is seeded into the live session as a prompt that names canvas_render, so
// the agent renders to the Canvas with full session context. When a target mock
// is passed, the prompt tells the agent to update THAT slug in place.
export async function requestDesign(
  sessionId: string,
  cwd: string,
  prompt: string,
  target?: CanvasTarget,
  kind?: CanvasKind,
): Promise<void> {
  await api('/api/v1/canvas/request', {
    method: 'POST',
    body: { session_id: sessionId, cwd, prompt, slug: target?.slug, title: target?.title, kind },
  })
}

// The project's focused canvas — what the operator means by "this canvas" in
// plain session conversation.
export interface CanvasFocus {
  slug: string
  title?: string
  kind?: CanvasKind
}

// setCanvasFocus records which canvas the operator is working on. notify=true
// (an explicit switch, not a panel mount) also seeds a one-line focus note into
// the session, so the agent follows along in ordinary conversation.
export async function setCanvasFocus(input: {
  cwd: string
  slug: string
  sessionId?: string
  notify?: boolean
}): Promise<CanvasFocus> {
  return api<CanvasFocus>('/api/v1/canvas/focus', {
    method: 'POST',
    body: { cwd: input.cwd, slug: input.slug, session_id: input.sessionId, notify: input.notify },
  })
}

// A project's canvas design system: the tokens every canvas must use plus the
// style rules tokens can't express. The gateway puts it in every canvas prompt
// AND injects the tokens into each canvas as CSS variables, so it is what stops
// successive renders from drifting apart.
export interface CanvasDesignSystem {
  cwd?: string
  tokens: Record<string, string>
  /** Dark-mode values for the colour tokens; missing keys fall back to light. */
  tokens_dark: Record<string, string>
  notes: string
  /**
   * Each token that IS a colour, resolved by the gateway to the #rrggbb a
   * swatch needs — projects write oklch/hsl/rgb and neither surface can read
   * those on its own. Tokens that aren't colours (radius, font, the shadow
   * triple) are absent, which is how a panel knows not to draw a swatch.
   */
  tokens_resolved?: Record<string, string>
  tokens_dark_resolved?: Record<string, string>
  /** Advice about the saved system — never a reason the save failed. */
  warnings?: CanvasDesignWarning[]
}

export interface CanvasDesignWarning {
  /** Stable identifier a panel can localise on; see WARNING_* below. */
  code: string
  /** English fallback, and what an agent reads back through canvas_design. */
  message: string
}

/** A system whose every colour resolves to a grey — usually a token mapped by
 *  name off a shadcn theme, where `--primary` is an ink and not a brand colour. */
export const WARNING_ACHROMATIC_PALETTE = 'achromatic_palette'

// The colour tokens that differ between themes. The rest (type, radius,
// spacing) are theme-independent, so the dark set doesn't carry them.
export const CANVAS_THEMED_TOKENS = [
  'primary',
  'secondary',
  'background',
  'surface',
  'text',
  'muted',
  'border',
  'shadow',
] as const

// Starting palettes. Picking colours from memory is the part non-designers
// can't do, so a new project starts from a complete, contrast-checked pair
// (light + dark) and tweaks from there rather than facing 13 empty fields.
// Values are plain hex because that is what people can read and edit.
export interface CanvasPalette {
  id: string
  accent: string
  tokens: Record<string, string>
  tokensDark: Record<string, string>
}

const SCALE = { baseSize: '14px', radius: '8px', spacing: '4px' }
const SANS = 'Inter, -apple-system, "Segoe UI", Roboto, "Helvetica Neue", sans-serif'

function palette(
  id: string,
  accent: string,
  accentDark: string,
  opts: { neutral?: 'slate' | 'zinc' | 'stone'; radius?: string } = {},
): CanvasPalette {
  const n = {
    slate: {
      bg: '#f8fafc', surface: '#ffffff', text: '#0f172a', muted: '#64748b', border: '#e2e8f0',
      dBg: '#0b1120', dSurface: '#151d2e', dText: '#e2e8f0', dMuted: '#94a3b8', dBorder: '#26324a',
    },
    zinc: {
      bg: '#fafafa', surface: '#ffffff', text: '#18181b', muted: '#71717a', border: '#e4e4e7',
      dBg: '#0c0c0e', dSurface: '#18181b', dText: '#e4e4e7', dMuted: '#a1a1aa', dBorder: '#2a2a2e',
    },
    stone: {
      bg: '#faf9f7', surface: '#ffffff', text: '#1c1917', muted: '#78716c', border: '#e7e5e4',
      dBg: '#0e0d0c', dSurface: '#1c1917', dText: '#e7e5e4', dMuted: '#a8a29e', dBorder: '#2d2926',
    },
  }[opts.neutral ?? 'slate']
  return {
    id,
    accent,
    tokens: {
      primary: accent,
      secondary: n.border,
      background: n.bg,
      surface: n.surface,
      text: n.text,
      muted: n.muted,
      border: n.border,
      font: SANS,
      headingFont: SANS,
      shadow: '0 1px 2px rgba(0,0,0,0.06)',
      ...SCALE,
      ...(opts.radius ? { radius: opts.radius } : {}),
    },
    tokensDark: {
      primary: accentDark,
      secondary: n.dBorder,
      background: n.dBg,
      surface: n.dSurface,
      text: n.dText,
      muted: n.dMuted,
      border: n.dBorder,
      shadow: '0 1px 2px rgba(0,0,0,0.4)',
    },
  }
}

export const CANVAS_PALETTES: CanvasPalette[] = [
  palette('indigo', '#4f46e5', '#818cf8'),
  palette('sky', '#0284c7', '#38bdf8'),
  palette('emerald', '#059669', '#34d399'),
  palette('amber', '#d97706', '#fbbf24', { neutral: 'stone' }),
  palette('rose', '#e11d48', '#fb7185', { neutral: 'stone' }),
  palette('violet', '#7c3aed', '#a78bfa', { neutral: 'zinc' }),
  palette('graphite', '#f97316', '#fb923c', { neutral: 'zinc', radius: '6px' }),
]

// withAccent swaps just the accent colour of a palette, so the operator can
// keep a checked neutral scale and still make it theirs.
export function withAccent(
  p: CanvasPalette,
  accent: string,
  accentDark: string,
): { tokens: Record<string, string>; tokensDark: Record<string, string> } {
  return {
    tokens: { ...p.tokens, primary: accent },
    tokensDark: { ...p.tokensDark, primary: accentDark },
  }
}

// The tokens the Canvas documents, in the order they're presented. Each becomes
// a CSS variable (primary → var(--od-primary)).
export const CANVAS_DESIGN_TOKENS = [
  'primary',
  'secondary',
  'background',
  'surface',
  'text',
  'muted',
  'border',
  'font',
  'headingFont',
  'baseSize',
  'radius',
  'spacing',
  'shadow',
] as const

// cssVarForToken mirrors the gateway's naming so the UI can show the variable
// the agent is told to use.
export function cssVarForToken(key: string): string {
  return `--od-${key.replace(/[A-Z]/g, (c) => `-${c.toLowerCase()}`)}`
}

function normalizeDesignSystem(d: CanvasDesignSystem): CanvasDesignSystem {
  return {
    tokens: d.tokens ?? {},
    tokens_dark: d.tokens_dark ?? {},
    notes: d.notes ?? '',
    tokens_resolved: d.tokens_resolved ?? {},
    tokens_dark_resolved: d.tokens_dark_resolved ?? {},
    warnings: d.warnings ?? [],
  }
}

export async function getDesignSystem(cwd: string): Promise<CanvasDesignSystem> {
  return normalizeDesignSystem(
    await api<CanvasDesignSystem>(`/api/v1/canvas/design?cwd=${encodeURIComponent(cwd)}`),
  )
}

export async function setDesignSystem(
  cwd: string,
  input: {
    tokens: Record<string, string>
    tokensDark: Record<string, string>
    notes: string
  },
): Promise<CanvasDesignSystem> {
  const d = await api<CanvasDesignSystem>('/api/v1/canvas/design', {
    method: 'POST',
    body: {
      cwd,
      tokens: input.tokens,
      tokens_dark: input.tokensDark,
      notes: input.notes,
      // The panel's colour picker can only emit hex, so ask the gateway to
      // write a picked colour back in whatever notation that token already
      // used — otherwise editing one swatch of an oklch theme converts that
      // token to hex and the palette ends up written two ways.
      preserve_notation: true,
    },
  })
  return normalizeDesignSystem(d)
}

/** One-click design-system jobs handed to the agent from the panel. */
export type CanvasDesignTask = 'extract' | 'showcase'

// runDesignTask asks the agent either to read the project's REAL theme and
// record it as the design system, or to draw the system as a canvas. The prompt
// lives in the gateway so every surface sends the same wording.
export async function runDesignTask(
  sessionId: string,
  cwd: string,
  task: CanvasDesignTask,
): Promise<void> {
  await api('/api/v1/canvas/design/task', {
    method: 'POST',
    body: { session_id: sessionId, cwd, task },
  })
}

// getCanvasFocus reads the project's focused canvas (empty slug when none).
export async function getCanvasFocus(cwd: string): Promise<CanvasFocus> {
  return api<CanvasFocus>(`/api/v1/canvas/focus?cwd=${encodeURIComponent(cwd)}`)
}

// deleteCanvas removes a mock artifact.
export async function deleteCanvas(id: string): Promise<void> {
  await api(`/api/v1/canvas/${encodeURIComponent(id)}`, { method: 'DELETE' })
}

// submitFeedback seeds the operator's annotations into the target session.
export async function submitFeedback(
  id: string,
  feedback: CanvasFeedback,
): Promise<void> {
  await api(`/api/v1/canvas/${encodeURIComponent(id)}/feedback`, {
    method: 'POST',
    body: feedback,
  })
}

export interface CanvasUpdatedEvent {
  id: string
  cwd: string
  slug: string
  title: string
  kind: CanvasKind
  version: number
}

export interface CanvasFocusEvent {
  cwd: string
  slug: string
}

// subscribeCanvas opens the integration eventbus WS and dispatches canvas
// events to the given handler. Returns an unsubscribe function. Reuses the
// terminal's reconnecting BinaryWS; events arrive as JSON text frames which
// BinaryWS surfaces as bytes, so we decode + parse here.
export function subscribeCanvas(
  token: string,
  handlers: {
    onUpdated?: (ev: CanvasUpdatedEvent) => void
    onFocus?: (ev: CanvasFocusEvent) => void
  },
): () => void {
  const ws = new BinaryWS(
    wsURL('/api/v1/integrations/_events?topics=canvas.updated,canvas.focus_changed', token),
    {
      onMessage: (bytes) => {
        try {
          const frame = JSON.parse(new TextDecoder().decode(bytes)) as {
            topic?: string
            data?: unknown
          }
          if (frame.topic === 'canvas.updated' && frame.data) {
            handlers.onUpdated?.(frame.data as CanvasUpdatedEvent)
          } else if (frame.topic === 'canvas.focus_changed' && frame.data) {
            handlers.onFocus?.(frame.data as CanvasFocusEvent)
          }
        } catch {
          // Ignore non-JSON frames (pings, malformed) — the socket stays up.
        }
      },
    },
  )
  ws.start()
  return () => ws.close()
}
