import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  MousePointerClick,
  Square,
  Eye,
  Send,
  Trash2,
  X,
  Monitor,
  Tablet,
  Smartphone,
  Wand2,
  LayoutTemplate,
  Workflow,
  Brain,
  Share2,
  FileText,
  Palette,
} from 'lucide-react'

import {
  listCanvas,
  getCanvas,
  deleteCanvas,
  submitFeedback,
  requestDesign,
  setCanvasFocus,
  getCanvasFocus,
  subscribeCanvas,
  type CanvasAnnotation,
  type CanvasKind,
} from '@/lib/canvas'
import { useAuth } from '@/stores/auth'
import { useTheme } from '@/stores/theme'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { DesignSystemSheet } from './DesignSystemSheet'

type DraftAnnotation = Omit<CanvasAnnotation, 'n'>
type Mode = 'preview' | 'pin' | 'region'
// The probe answers in DOCUMENT percentages (x/y, plus w/h for a region): the
// page itself is the only thing that knows how far it is scrolled, so it does
// the conversion. Absent when the probe timed out or the point hit nothing.
type ProbeResult = {
  selector: string
  html: string
  elements?: string[]
  x?: number
  y?: number
  w?: number
  h?: number
}
export type CanvasStageVariant = 'panel' | 'popout'

interface CanvasStageProps {
  sessionId: string
  cwd: string
  variant: CanvasStageVariant
  toolbarExtra?: React.ReactNode
}

type Viewport = 'full' | 'tablet' | 'mobile'
const VIEWPORT_WIDTH: Record<Viewport, number | null> = { full: null, tablet: 820, mobile: 390 }

// The kinds a canvas can be. The Canvas is a general visual surface: a screen
// mock, or a diagram the agent draws (flow / mind map / relationships / doc).
const KINDS: CanvasKind[] = ['ui', 'flow', 'mindmap', 'graph', 'doc']
function KindIcon({ kind, className }: { kind?: CanvasKind; className?: string }) {
  switch (kind) {
    case 'flow':
      return <Workflow className={className} />
    case 'mindmap':
      return <Brain className={className} />
    case 'graph':
      return <Share2 className={className} />
    case 'doc':
      return <FileText className={className} />
    default:
      return <LayoutTemplate className={className} />
  }
}

// Marks are drawn INSIDE the canvas document, so they stay glued to the content
// the operator picked. Drawing them in the parent instead — over the iframe —
// leaves them pinned to the frame: scroll the canvas and the pin sits on
// whatever slid under it. Mobile already does it this way; these class names and
// colours mirror app/mobile/lib/features/sessions/inspector/canvas_common.dart.
const MARK_CSS = `
.__od-mark{position:absolute;z-index:2147483000;pointer-events:none;box-sizing:border-box}
.__od-mark.rect{border:2px solid #f43f5e;background:rgba(244,63,94,.12);box-shadow:0 0 6px rgba(0,0,0,.45);outline:1px solid #fff;outline-offset:-3px}
.__od-badge{position:absolute;z-index:2147483001;pointer-events:none;
  width:22px;height:22px;margin:-11px 0 0 -11px;border-radius:50%;
  background:#f43f5e;color:#fff;font:bold 12px/22px sans-serif;text-align:center;
  box-shadow:0 1px 4px rgba(0,0,0,.5);border:2px solid #fff}
`

// Probe injected into the mock iframe so a pin captures the element selector and
// a region captures the framed DOM block + the components inside it — and so the
// committed marks can be drawn into the document itself.
const PROBE_SCRIPT = `
<script>
(function(){
  var style=document.createElement('style');
  style.textContent=${JSON.stringify(MARK_CSS)};
  (document.head||document.documentElement).appendChild(style);

  function docSize(){
    var de=document.documentElement, b=document.body;
    return {
      w: Math.max(de?de.scrollWidth:0, b?b.scrollWidth:0, 1),
      h: Math.max(de?de.scrollHeight:0, b?b.scrollHeight:0, 1)
    };
  }
  // Client -> document percentage, so a mark means "40% down the PAGE" rather
  // than "40% down the window". Without adding the scroll offset, a mark
  // recorded after scrolling points at the wrong content.
  function toDocPct(cx, cy){
    var d=docSize();
    return {
      x: ((cx + (window.scrollX||0)) / d.w) * 100,
      y: ((cy + (window.scrollY||0)) / d.h) * 100
    };
  }
  function cssPath(el){
    if(!(el instanceof Element)) return '';
    var path=[];
    while(el && el.nodeType===1 && path.length<5){
      var sel=el.nodeName.toLowerCase();
      if(el.id){ sel+='#'+el.id; path.unshift(sel); break; }
      var cls=(typeof el.className==='string'?el.className:'').trim();
      if(cls){ sel+='.'+cls.split(/\\s+/).slice(0,2).join('.'); }
      var p=el.parentNode;
      if(p && p.children){
        var same=Array.prototype.filter.call(p.children,function(c){return c.nodeName===el.nodeName;});
        if(same.length>1){ sel+=':nth-child('+(Array.prototype.indexOf.call(p.children,el)+1)+')'; }
      }
      path.unshift(sel);
      el=el.parentNode;
    }
    return path.join(' > ');
  }
  window.addEventListener('message',function(e){
    var d=e.data||{};
    if(d.type==='canvas:probe'){
      var el=document.elementFromPoint(d.x,d.y);
      var p=toDocPct(d.x,d.y);
      parent.postMessage({type:'canvas:probed',id:d.id,selector:el?cssPath(el):'',html:el?el.outerHTML:'',x:p.x,y:p.y},'*');
    } else if(d.type==='canvas:marks'){
      // Redraw the whole set: the parent owns the list, this just mirrors it.
      var old=document.querySelectorAll('.__od-mark, .__od-badge');
      for(var j=0;j<old.length;j++) old[j].remove();
      var ds=docSize();
      for(var k=0;k<(d.marks||[]).length;k++){
        var m=d.marks[k], mx=m.x/100*ds.w, my=m.y/100*ds.h;
        if(m.kind==='region'){
          var frame=document.createElement('div');
          frame.className='__od-mark rect';
          frame.style.left=mx+'px'; frame.style.top=my+'px';
          frame.style.width=(m.w/100*ds.w)+'px'; frame.style.height=(m.h/100*ds.h)+'px';
          document.body.appendChild(frame);
        }
        var badge=document.createElement('div');
        badge.className='__od-badge';
        badge.style.left=mx+'px'; badge.style.top=my+'px';
        badge.textContent=String(k+1);
        document.body.appendChild(badge);
      }
    } else if(d.type==='canvas:probeRect'){
      // Find the smallest element that fully contains the dragged rectangle —
      // "the block you framed" — then list the components inside it, so the
      // agent gets the real DOM instead of guessing from coordinates.
      var x=d.x, y=d.y, w=d.w, h=d.h;
      function fits(el){var r=el.getBoundingClientRect(); return r.left<=x+2 && r.top<=y+2 && r.right>=x+w-2 && r.bottom>=y+h-2;}
      var box=document.elementFromPoint(x+w/2, y+h/2);
      while(box && box.parentElement && box!==document.body && !fits(box)) box=box.parentElement;
      var kids=[];
      if(box){
        var all=box.querySelectorAll('*');
        for(var i=0;i<all.length && kids.length<10;i++){
          var c=all[i], r=c.getBoundingClientRect();
          if(r.width<12 || r.height<12) continue;
          if(r.right<x || r.left>x+w || r.bottom<y || r.top>y+h) continue;
          var cls=(typeof c.className==='string'?c.className:'').trim();
          var label=c.nodeName.toLowerCase()+(cls?'.'+cls.split(/\\s+/).slice(0,2).join('.'):'');
          var txt=(c.textContent||'').replace(/\\s+/g,' ').trim().slice(0,32);
          kids.push(label+(txt?' “'+txt+'”':''));
        }
      }
        var pa=toDocPct(x,y), pb=toDocPct(x+w,y+h);
      parent.postMessage({type:'canvas:probedRect',id:d.id,selector:box?cssPath(box):'',html:box?box.outerHTML.slice(0,1500):'',kids:kids,x:pa.x,y:pa.y,w:pb.x-pa.x,h:pb.y-pa.y},'*');
    }
  });
})();
<\/script>`

const COLOR_SCHEME_META = '<meta name="color-scheme" content="light dark">'
function injectProbe(html: string): string {
  if (html.includes('</body>')) return html.replace('</body>', PROBE_SCRIPT + '</body>')
  return html + PROBE_SCRIPT
}
function injectHead(html: string): string {
  if (html.includes('<head>')) return html.replace('<head>', '<head>' + COLOR_SCHEME_META)
  if (/<html[^>]*>/i.test(html)) return html.replace(/(<html[^>]*>)/i, `$1<head>${COLOR_SCHEME_META}</head>`)
  return COLOR_SCHEME_META + html
}

// CanvasStage — the Mock canvas: an agent renders a self-contained HTML preview
// via canvas_render, the operator pins / region-selects elements on it, and
// those annotations (real selectors + framed DOM + notes) are seeded back into
// the session so the agent can iterate precisely.
export function CanvasStage({ sessionId, cwd, variant, toolbarExtra }: CanvasStageProps) {
  const { t } = useTranslation()
  const token = useAuth((s) => s.token)
  useTheme((s) => s.mode)
  const resolvedTheme = useTheme.getState().applied()
  const qc = useQueryClient()
  const popout = variant === 'popout'

  const [selectedId, setSelectedId] = useState<string | null>(null)
  // newMock: the operator explicitly picked "＋ New canvas" — requests create a
  // fresh canvas instead of updating the selected one. newKind is what that
  // fresh canvas should be (mock / flowchart / mind map / …).
  const [newMock, setNewMock] = useState(false)
  const [newKind, setNewKind] = useState<CanvasKind>('ui')
  const [mode, setMode] = useState<Mode>('preview')
  const [viewport, setViewport] = useState<Viewport>('full')
  const [annotations, setAnnotations] = useState<DraftAnnotation[]>([])
  const [message, setMessage] = useState('')
  const [sending, setSending] = useState(false)
  const [request, setRequest] = useState('')
  const [requesting, setRequesting] = useState(false)
  const [drag, setDrag] = useState<{ x: number; y: number; w: number; h: number } | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [designOpen, setDesignOpen] = useState(false)

  const iframeRef = useRef<HTMLIFrameElement>(null)
  const overlayRef = useRef<HTMLDivElement>(null)
  const dragStart = useRef<{ x: number; y: number } | null>(null)
  const probePending = useRef<Map<string, (r: ProbeResult) => void>>(new Map())

  const { data: list } = useQuery({ queryKey: ['canvas', cwd], queryFn: () => listCanvas(cwd), staleTime: 5_000 })
  const { data: artifact } = useQuery({
    queryKey: ['canvas-artifact', selectedId],
    queryFn: () => getCanvas(selectedId as string),
    enabled: !!selectedId,
  })
  useEffect(() => {
    if (!list || list.length === 0) {
      setSelectedId(null)
      return
    }
    if (!selectedId || !list.some((a) => a.id === selectedId)) setSelectedId(list[0].id)
  }, [list, selectedId])

  // Which canvas the agent is working on. Browsing the list does NOT change it:
  // picking a canvas is just a preview (free), while making it the workspace is
  // a deliberate act that costs one seeded note. Otherwise flicking through
  // canvases to decide would burn a turn per click.
  const [workspace, setWorkspace] = useState<string>('')
  const [committing, setCommitting] = useState(false)
  useEffect(() => {
    let cancelled = false
    void getCanvasFocus(cwd)
      .then((f) => {
        if (!cancelled) setWorkspace(f.slug || '')
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [cwd])

  // pickCanvas is preview-only — no request, no tokens.
  const pickCanvas = useCallback((id: string) => {
    setNewMock(false)
    setSelectedId(id)
    setAnnotations([])
  }, [])

  // setAsWorkspace is the deliberate "work on THIS one" action: it records the
  // focus and seeds the single note that makes plain terminal conversation
  // resolve here.
  const commitWorkspace = useCallback(
    async (slug: string) => {
      setCommitting(true)
      try {
        await setCanvasFocus({ cwd, slug, sessionId, notify: true })
        setWorkspace(slug)
        toast.success(t('web.sessions.inspector.canvas.workspaceSet'))
      } catch (e) {
        toast.error(t('web.sessions.inspector.canvas.workspaceFailed'), {
          description: (e as Error).message,
        })
      } finally {
        setCommitting(false)
      }
    },
    [cwd, sessionId, t],
  )

  const selRef = useRef({ slug: artifact?.slug, id: selectedId, newMock })
  selRef.current = { slug: artifact?.slug, id: selectedId, newMock }
  useEffect(() => {
    if (!token) return
    return subscribeCanvas(token, {
      onUpdated: (ev) => {
        if (ev.cwd !== cwd) return
        void qc.invalidateQueries({ queryKey: ['canvas', cwd] })
        // The agent just rendered while we were in "new mock" mode → that render
        // is the new canvas; jump to it and leave new-mock mode.
        if (selRef.current.newMock) {
          setNewMock(false)
          setSelectedId(ev.id)
        } else if (selRef.current.id && ev.slug === selRef.current.slug) {
          void qc.invalidateQueries({ queryKey: ['canvas-artifact', selRef.current.id] })
        }
      },
      // Another surface (the pop-out, or the phone) set the workspace — just
      // reflect it. Nothing is seeded from here; that side already did it.
      onFocus: (ev) => {
        if (ev.cwd !== cwd) return
        setWorkspace(ev.slug || '')
      },
    })
  }, [token, cwd, qc])

  // Collect the mock probe answers (point → element, or rect → container + the
  // components inside it).
  useEffect(() => {
    function onMsg(e: MessageEvent) {
      const d = e.data as {
        type?: string
        id?: string
        selector?: string
        html?: string
        kids?: string[]
        x?: number
        y?: number
        w?: number
        h?: number
      }
      if (d && (d.type === 'canvas:probed' || d.type === 'canvas:probedRect') && d.id) {
        const resolve = probePending.current.get(d.id)
        if (resolve) {
          probePending.current.delete(d.id)
          resolve({
            selector: d.selector || '',
            html: d.html || '',
            elements: d.kids,
            x: d.x,
            y: d.y,
            w: d.w,
            h: d.h,
          })
        }
      }
    }
    window.addEventListener('message', onMsg)
    return () => window.removeEventListener('message', onMsg)
  }, [])

  const srcDoc = useMemo(() => (artifact ? injectHead(injectProbe(artifact.html)) : ''), [artifact])
  const frameWidth = VIEWPORT_WIDTH[viewport]

  // Mirror the committed marks into the canvas document, which is what keeps
  // them glued to the content instead of to the frame. Driven from two places
  // because either can happen first: the list changing (effect), and the
  // document being replaced by a re-render, which takes the marks with it and
  // only has a probe to talk to once it has loaded (the iframe's onLoad).
  const paintMarks = useCallback(() => {
    iframeRef.current?.contentWindow?.postMessage(
      {
        type: 'canvas:marks',
        marks: annotations.map((a) => ({ kind: a.kind, x: a.x, y: a.y, w: a.w, h: a.h })),
      },
      '*',
    )
  }, [annotations])
  useEffect(() => {
    paintMarks()
  }, [paintMarks])

  const probe = useCallback((x: number, y: number) => {
    return new Promise<ProbeResult>((resolve) => {
      const win = iframeRef.current?.contentWindow
      if (!win) return resolve({ selector: '', html: '' })
      const id = Math.random().toString(36).slice(2)
      probePending.current.set(id, resolve)
      win.postMessage({ type: 'canvas:probe', id, x, y }, '*')
      setTimeout(() => {
        if (probePending.current.has(id)) {
          probePending.current.delete(id)
          resolve({ selector: '', html: '' })
        }
      }, 500)
    })
  }, [])

  // probeRect asks the mock iframe for the DOM block a dragged rectangle frames,
  // plus the components inside it — so a region annotation carries real elements.
  const probeRect = useCallback((x: number, y: number, w: number, h: number) => {
    return new Promise<ProbeResult>((resolve) => {
      const win = iframeRef.current?.contentWindow
      if (!win) return resolve({ selector: '', html: '' })
      const id = Math.random().toString(36).slice(2)
      probePending.current.set(id, resolve)
      win.postMessage({ type: 'canvas:probeRect', id, x, y, w, h }, '*')
      setTimeout(() => {
        if (probePending.current.has(id)) {
          probePending.current.delete(id)
          resolve({ selector: '', html: '' })
        }
      }, 500)
    })
  }, [])

  const overlayRect = () => overlayRef.current?.getBoundingClientRect()
  const annotating = mode !== 'preview'

  const onOverlayClick = useCallback(
    async (e: React.MouseEvent) => {
      if (mode !== 'pin') return
      const rect = overlayRect()
      if (!rect) return
      const px = e.clientX - rect.left
      const py = e.clientY - rect.top
      // px/py are where the click landed in the visible frame — the right thing
      // to probe with (elementFromPoint takes viewport coordinates) but the
      // wrong thing to STORE, because it says nothing about how far the canvas
      // is scrolled. The page answers in document percentages; fall back to the
      // frame-relative figure only when the probe couldn't answer at all.
      const { selector, html, x, y } = await probe(px, py)
      setAnnotations((prev) => [
        ...prev,
        {
          kind: 'pin',
          note: '',
          selector,
          html,
          x: x ?? (px / rect.width) * 100,
          y: y ?? (py / rect.height) * 100,
        },
      ])
    },
    [mode, probe],
  )
  const onOverlayPointerDown = useCallback(
    (e: React.PointerEvent) => {
      if (mode !== 'region') return
      const rect = overlayRect()
      if (!rect) return
      overlayRef.current?.setPointerCapture(e.pointerId)
      dragStart.current = { x: e.clientX - rect.left, y: e.clientY - rect.top }
      setDrag({ x: dragStart.current.x, y: dragStart.current.y, w: 0, h: 0 })
    },
    [mode],
  )
  const onOverlayPointerMove = useCallback((e: React.PointerEvent) => {
    if (!dragStart.current) return
    const rect = overlayRect()
    if (!rect) return
    const cx = e.clientX - rect.left
    const cy = e.clientY - rect.top
    setDrag({
      x: Math.min(dragStart.current.x, cx),
      y: Math.min(dragStart.current.y, cy),
      w: Math.abs(cx - dragStart.current.x),
      h: Math.abs(cy - dragStart.current.y),
    })
  }, [])
  const onOverlayPointerUp = useCallback(async () => {
    const rect = overlayRect()
    const d = drag
    dragStart.current = null
    setDrag(null)
    if (!rect || !d || d.w < 6 || d.h < 6) return
    const p = await probeRect(d.x, d.y, d.w, d.h)
    setAnnotations((prev) => [
      ...prev,
      {
        kind: 'region',
        note: '',
        selector: p.selector,
        html: p.html,
        elements: p.elements,
        // Document percentages from the page (see onOverlayClick); the
        // frame-relative figures are only a fallback for a failed probe.
        x: p.x ?? (d.x / rect.width) * 100,
        y: p.y ?? (d.y / rect.height) * 100,
        w: p.w ?? (d.w / rect.width) * 100,
        h: p.h ?? (d.h / rect.height) * 100,
      },
    ])
  }, [drag, probeRect])

  function setNote(idx: number, note: string) {
    setAnnotations((prev) => prev.map((a, i) => (i === idx ? { ...a, note } : a)))
  }
  function removeAnnotation(idx: number) {
    setAnnotations((prev) => prev.filter((_, i) => i !== idx))
  }

  async function send() {
    if (!artifact) return
    if (annotations.length === 0 && !message.trim()) {
      toast.error(t('web.sessions.inspector.canvas.nothingToSend'))
      return
    }
    setSending(true)
    try {
      await submitFeedback(artifact.id, {
        session_id: sessionId,
        message: message.trim() || undefined,
        annotations: annotations.map((a, i) => ({ ...a, n: i + 1 })),
      })
      toast.success(t('web.sessions.inspector.canvas.sent'))
      setAnnotations([])
      setMessage('')
      setMode('preview')
    } catch (e) {
      toast.error(t('web.sessions.inspector.canvas.sendFailed'), { description: (e as Error).message })
    } finally {
      setSending(false)
    }
  }

  const noMock = !list || list.length === 0
  const showMockPreview = !newMock && !!artifact

  // The mock the agent should update in place. Undefined when "＋ New mock" is
  // active → an ambiguous request creates a fresh canvas; otherwise it iterates
  // on THIS canvas in place.
  const targetMock = !newMock && artifact ? { slug: artifact.slug, title: artifact.title } : undefined

  const requestBar = (
    <form onSubmit={(e) => void sendRequest(e)} className="flex items-center gap-1 shrink-0">
      <input
        value={request}
        onChange={(e) => setRequest(e.target.value)}
        placeholder={t('web.sessions.inspector.canvas.requestPlaceholder')}
        className="flex-1 min-w-0 rounded-sm border border-border bg-transparent px-2 py-1 text-[12px] focus:outline-none focus:border-primary"
      />
      <Button size="sm" type="submit" disabled={requesting || !request.trim()}>
        <Wand2 className="size-3.5" />
        {t('web.sessions.inspector.canvas.requestSend')}
      </Button>
    </form>
  )

  async function sendRequest(e: React.FormEvent) {
    e.preventDefault()
    const p = request.trim()
    if (!p) return
    setRequesting(true)
    try {
      await requestDesign(sessionId, cwd, p, targetMock, newMock ? newKind : undefined)
      toast.success(t('web.sessions.inspector.canvas.requested'))
      setRequest('')
    } catch (err) {
      toast.error(t('web.sessions.inspector.canvas.requestFailed'), { description: (err as Error).message })
    } finally {
      setRequesting(false)
    }
  }

  async function deleteSelected() {
    if (!artifact) return
    if (!window.confirm(t('web.sessions.inspector.canvas.deleteConfirm', { title: artifact.title || artifact.slug }))) return
    setDeleting(true)
    try {
      await deleteCanvas(artifact.id)
      const remaining = (list ?? []).filter((a) => a.id !== artifact.id)
      setSelectedId(remaining[0]?.id ?? null)
      setAnnotations([])
      await qc.invalidateQueries({ queryKey: ['canvas', cwd] })
      toast.success(t('web.sessions.inspector.canvas.deleted'))
    } catch (err) {
      toast.error(t('web.sessions.inspector.canvas.deleteFailed'), { description: (err as Error).message })
    } finally {
      setDeleting(false)
    }
  }

  // Only the in-progress drag is drawn here. Committed marks live inside the
  // canvas document (see paintMarks) so they scroll with the content; a drag
  // rectangle is transient and frame-relative by nature — the page cannot
  // scroll underneath it while the pointer is down.
  const markers = (
    <>
      {drag && (
        <div
          className="absolute border-2 border-dashed border-rose-500 ring-1 ring-inset ring-white shadow-[0_0_0_9999px_rgba(0,0,0,0.35)]"
          style={{ left: drag.x, top: drag.y, width: drag.w, height: drag.h }}
        />
      )}
    </>
  )

  return (
    <div className={cn('flex flex-col gap-2 min-h-0', popout && 'h-full')}>
      <div className="flex items-center gap-1 shrink-0">
        <div className="flex-1">{requestBar}</div>
        {toolbarExtra}
      </div>

      {/* The project's design contract — what stops successive renders from
          drifting. Editing it is rare, so it lives behind one button. */}
      <div className="flex items-center gap-1 shrink-0">
        <button
          type="button"
          onClick={() => setDesignOpen(true)}
          className="flex items-center gap-1 px-2 py-0.5 rounded-sm text-[11px] border border-border text-muted-foreground hover:text-foreground"
          title={t('web.sessions.inspector.canvas.designBlurb')}
        >
          <Palette className="size-3" />
          {t('web.sessions.inspector.canvas.designTitle')}
        </button>
      </div>
      {designOpen && <DesignSystemSheet sessionId={sessionId} cwd={cwd} onClose={() => setDesignOpen(false)} />}

      {/* Canvas list. Picking one only PREVIEWS it — free. The dot marks the
          workspace: the canvas the agent works on and resolves "this canvas" to
          in ordinary conversation. "＋ New canvas" starts a fresh one. */}
      <div className="flex flex-wrap gap-1 shrink-0">
        {(list ?? []).map((a) => (
          <button
            key={a.id}
            type="button"
            onClick={() => pickCanvas(a.id)}
            className={cn('flex items-center gap-1 px-2 py-0.5 rounded-sm text-[11px] border', !newMock && a.id === selectedId ? 'bg-card border-primary/50 text-foreground' : 'border-border text-muted-foreground hover:text-foreground')}
            title={`${a.title || a.slug} · ${t(`web.sessions.inspector.canvas.kind_${a.kind || 'ui'}`)}`}
          >
            <KindIcon kind={a.kind} className="size-3" />
            {a.title || a.slug}
            {a.slug === workspace && (
              <span
                className="size-1.5 rounded-full bg-primary"
                title={t('web.sessions.inspector.canvas.workspaceBadge')}
              />
            )}
          </button>
        ))}
        <button
          type="button"
          onClick={() => { setNewMock(true); setAnnotations([]) }}
          className={cn('px-2 py-0.5 rounded-sm text-[11px] border border-dashed', newMock ? 'bg-card border-primary/50 text-foreground' : 'border-border text-muted-foreground hover:text-foreground')}
          title={t('web.sessions.inspector.canvas.newMockHint')}
        >
          + {t('web.sessions.inspector.canvas.newMock')}
        </button>
      </div>

      {/* Kind picker — only while starting a new canvas: what should the agent
          draw? A screen mock, or a flowchart / mind map / relationship diagram. */}
      {newMock && (
        <div className="flex flex-wrap items-center gap-1 shrink-0">
          <span className="text-[11px] text-muted-foreground mr-0.5">{t('web.sessions.inspector.canvas.kindLabel')}</span>
          {KINDS.map((k) => (
            <button
              key={k}
              type="button"
              onClick={() => setNewKind(k)}
              className={cn('flex items-center gap-1 px-2 py-0.5 rounded-sm text-[11px] border', newKind === k ? 'bg-card border-primary/50 text-foreground' : 'border-border text-muted-foreground hover:text-foreground')}
            >
              <KindIcon kind={k} className="size-3" />
              {t(`web.sessions.inspector.canvas.kind_${k}`)}
            </button>
          ))}
        </div>
      )}

      {/* Workspace line: which canvas the agent acts on. Selecting is free;
          committing costs exactly one seeded note, so it's an explicit act. */}
      <div className="flex items-center gap-1.5 shrink-0 text-[11px]">
        <span className="text-muted-foreground">{t('web.sessions.inspector.canvas.targetLabel')}</span>
        {newMock ? (
          <span className="font-medium text-foreground">
            {t('web.sessions.inspector.canvas.targetNew')} · {t(`web.sessions.inspector.canvas.kind_${newKind}`)}
          </span>
        ) : artifact ? (
          <span className="flex items-center gap-1 font-medium text-foreground truncate">
            <KindIcon kind={artifact.kind} className="size-3 shrink-0" />
            {artifact.title || artifact.slug} <span className="text-muted-foreground font-normal">v{artifact.version}</span>
          </span>
        ) : (
          <span className="text-muted-foreground">{t('web.sessions.inspector.canvas.targetNew')}</span>
        )}
        {!newMock && artifact && artifact.slug !== workspace && (
          <button
            type="button"
            onClick={() => void commitWorkspace(artifact.slug)}
            disabled={committing}
            className="ml-auto shrink-0 px-2 py-0.5 rounded-sm border border-primary/50 text-foreground hover:border-primary disabled:opacity-50"
            title={t('web.sessions.inspector.canvas.setWorkspaceHint')}
          >
            {t('web.sessions.inspector.canvas.setWorkspace')}
          </button>
        )}
        {!newMock && artifact && artifact.slug === workspace && (
          <span className="ml-auto shrink-0 text-primary">
            {t('web.sessions.inspector.canvas.isWorkspace')}
          </span>
        )}
        {!newMock && artifact && (
          <button
            type="button"
            onClick={() => void deleteSelected()}
            disabled={deleting}
            aria-label={t('web.sessions.inspector.canvas.deleteMock')}
            title={t('web.sessions.inspector.canvas.deleteMock')}
            className="flex items-center justify-center size-6 shrink-0 rounded-sm border border-border text-muted-foreground hover:text-state-failed hover:border-state-failed/50 disabled:opacity-50"
          >
            <Trash2 className="size-3.5" />
          </button>
        )}
      </div>
      {!newMock && artifact && (
        <p className="text-[10px] text-muted-foreground shrink-0 -mt-1">
          {artifact.slug === workspace
            ? t('web.sessions.inspector.canvas.focusHint')
            : t('web.sessions.inspector.canvas.previewOnlyHint')}
        </p>
      )}

      {/* Empty hint. */}
      {noMock && !newMock && (
        <div className="text-[12px] text-muted-foreground leading-relaxed px-1 py-2">
          {t('web.sessions.inspector.canvas.empty')}
        </div>
      )}

      {/* Annotate toolbar + viewport. */}
      {showMockPreview && (
        <div className="flex items-center gap-1 shrink-0">
          <ModeButton active={mode === 'preview'} onClick={() => setMode('preview')} icon={<Eye className="size-3.5" />} label={t('web.sessions.inspector.canvas.modePreview')} />
          <ModeButton active={mode === 'pin'} onClick={() => setMode('pin')} icon={<MousePointerClick className="size-3.5" />} label={t('web.sessions.inspector.canvas.modePin')} />
          <ModeButton active={mode === 'region'} onClick={() => setMode('region')} icon={<Square className="size-3.5" />} label={t('web.sessions.inspector.canvas.modeRegion')} />
          <div className="ml-auto flex items-center gap-0.5">
            <ViewportButton active={viewport === 'mobile'} onClick={() => setViewport('mobile')} icon={<Smartphone className="size-3.5" />} label={t('web.sessions.inspector.canvas.viewportMobile')} />
            <ViewportButton active={viewport === 'tablet'} onClick={() => setViewport('tablet')} icon={<Tablet className="size-3.5" />} label={t('web.sessions.inspector.canvas.viewportTablet')} />
            <ViewportButton active={viewport === 'full'} onClick={() => setViewport('full')} icon={<Monitor className="size-3.5" />} label={t('web.sessions.inspector.canvas.viewportFull')} />
          </div>
        </div>
      )}

      {/* New-mock placeholder: nothing to preview yet — the agent will render
          the fresh canvas here once you send a Design request. */}
      {newMock && (
        <div className={cn('flex items-center justify-center rounded-sm border border-dashed border-border bg-muted/20 text-center px-4', popout ? 'flex-1 min-h-0' : 'h-[420px]')}>
          <p className="text-[12px] text-muted-foreground leading-relaxed max-w-sm">
            {t('web.sessions.inspector.canvas.newMockPlaceholder')}
          </p>
        </div>
      )}

      {/* Mock preview (annotatable). */}
      {showMockPreview && (
        <div className={cn('relative w-full overflow-auto rounded-sm border border-border bg-muted/40', popout ? 'flex-1 min-h-0' : 'h-[420px]')}>
          <div className={cn('relative h-full mx-auto', frameWidth != null && 'ring-1 ring-border shadow-sm')} style={{ width: frameWidth != null ? `${frameWidth}px` : '100%' }}>
            <iframe ref={iframeRef} title="canvas-preview" srcDoc={srcDoc} onLoad={paintMarks} sandbox="allow-scripts" style={{ colorScheme: resolvedTheme }} className="w-full h-full block bg-background" />
            <div
              ref={overlayRef}
              onClick={onOverlayClick}
              onPointerDown={onOverlayPointerDown}
              onPointerMove={onOverlayPointerMove}
              onPointerUp={onOverlayPointerUp}
              className={cn('absolute inset-0', annotating ? 'cursor-crosshair' : 'pointer-events-none', mode === 'region' && 'touch-none')}
            >
              {markers}
            </div>
          </div>
        </div>
      )}

      {annotating && showMockPreview && (
        <p className="text-[11px] text-muted-foreground shrink-0">
          {mode === 'pin' ? t('web.sessions.inspector.canvas.hintPin') : t('web.sessions.inspector.canvas.hintRegion')}
        </p>
      )}

      {/* Feedback composer. */}
      {showMockPreview && (
        <div className={cn('flex flex-col gap-1.5 shrink-0', popout && 'max-h-[30vh] overflow-y-auto')}>
          {annotations.map((a, i) => (
            <div key={i} className="flex items-start gap-1.5">
              <span className="mt-1.5 size-4 shrink-0 rounded-full bg-primary text-primary-foreground text-[9px] font-bold flex items-center justify-center">{i + 1}</span>
              <div className="flex-1 min-w-0">
                {a.selector && (
                  <code className="block truncate text-[10px] text-muted-foreground" title={a.selector}>
                    {a.selector}
                  </code>
                )}
                <input
                  value={a.note}
                  onChange={(e) => setNote(i, e.target.value)}
                  placeholder={t('web.sessions.inspector.canvas.notePlaceholder')}
                  className="w-full bg-transparent border-b border-border text-[12px] py-0.5 focus:outline-none focus:border-primary"
                />
              </div>
              <button type="button" onClick={() => removeAnnotation(i)} aria-label={t('web.sessions.inspector.canvas.removeAnnotation')} className="mt-1 text-muted-foreground/60 hover:text-state-failed">
                <X className="size-3.5" />
              </button>
            </div>
          ))}
          <textarea
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            placeholder={t('web.sessions.inspector.canvas.messagePlaceholder')}
            rows={2}
            className="w-full resize-y rounded-sm border border-border bg-transparent px-2 py-1 text-[12px] focus:outline-none focus:border-primary"
          />
          <div className="flex items-center gap-2">
            <Button size="sm" onClick={() => void send()} disabled={sending}>
              <Send className="size-3.5" />
              {t('web.sessions.inspector.canvas.send')}
            </Button>
            {annotations.length > 0 && (
              <Button size="sm" variant="ghost" onClick={() => setAnnotations([])} disabled={sending}>
                <Trash2 className="size-3.5" />
                {t('web.sessions.inspector.canvas.clear')}
              </Button>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

function ModeButton({ active, onClick, icon, label }: { active: boolean; onClick: () => void; icon: React.ReactNode; label: string }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn('flex items-center gap-1 px-2 py-1 rounded-sm text-[11px] border', active ? 'bg-card border-primary/50 text-foreground' : 'border-border text-muted-foreground hover:text-foreground')}
    >
      {icon}
      {label}
    </button>
  )
}

function ViewportButton({ active, onClick, icon, label }: { active: boolean; onClick: () => void; icon: React.ReactNode; label: string }) {
  return (
    <button
      type="button"
      onClick={onClick}
      title={label}
      aria-label={label}
      aria-pressed={active}
      className={cn('flex items-center justify-center size-7 rounded-sm border', active ? 'bg-card border-primary/50 text-foreground' : 'border-border text-muted-foreground hover:text-foreground')}
    >
      {icon}
    </button>
  )
}
