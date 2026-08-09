import { useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ShieldAlert, ShieldCheck } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

// Rendering a vault document as HTML means running someone else's
// markup inside the admin app. "Someone else's" is not hypothetical:
// the Vault syncs from a git remote, so a document can arrive from a
// repo the operator merely pulled.
//
// Two rules follow, and both are load-bearing.
//
// 1. The document is NEVER served from opendray's origin. There is no
//    `GET /notes/raw?path=x.html` returning text/html — that would run
//    the document's scripts same-origin with the admin session, which
//    is account takeover via a pulled file. The body arrives as a
//    string through the existing JSON read endpoint and is handed to
//    srcDoc, which gives the frame an opaque origin. It cannot reach
//    window.parent, the app's cookies, or its localStorage.
//
// 2. Scripts are OFF by default. Most documentation HTML — exported
//    from Notion, Word, typedoc, asciidoc, Sphinx — is static markup
//    and renders identically without them. Documents that genuinely
//    need to run code (client-side mermaid, tabs, in-page search) get
//    an explicit per-document opt-in, so executing a pulled file is
//    always a decision someone made, never a default.
//
// `allow-popups` is granted in both modes. It does not permit script
// execution; without it a plain <a target="_blank"> silently does
// nothing, which reads as a broken document rather than a safe one.

interface HtmlPreviewProps {
  html: string
  /** Vault-relative path — scopes the per-document scripts opt-in. */
  path: string
  className?: string
}

const SCRIPTS_KEY_PREFIX = 'vault.html.scripts:'

// Opt-in is remembered per document and kept in localStorage rather
// than on the server: it is a judgement about one operator's trust in
// one file on one machine, not a property of the document.
function readOptIn(path: string): boolean {
  try {
    return localStorage.getItem(SCRIPTS_KEY_PREFIX + path) === '1'
  } catch {
    return false
  }
}

export function HtmlPreview({ html, path, className }: HtmlPreviewProps) {
  const { t } = useTranslation()
  const [scripts, setScripts] = useState(() => readOptIn(path))
  const frameRef = useRef<HTMLIFrameElement | null>(null)

  // Re-read when the viewer switches documents: the opt-in belongs to
  // the path, and a stale `true` would silently carry trust from one
  // file to the next.
  useEffect(() => {
    setScripts(readOptIn(path))
  }, [path])

  const toggle = () => {
    const next = !scripts
    setScripts(next)
    try {
      if (next) localStorage.setItem(SCRIPTS_KEY_PREFIX + path, '1')
      else localStorage.removeItem(SCRIPTS_KEY_PREFIX + path)
    } catch {
      // A blocked localStorage costs the memory of the choice, not the
      // choice itself.
    }
  }

  // Never `allow-same-origin`: combined with allow-scripts it hands the
  // document the app's origin and removes the sandbox entirely.
  const sandbox = scripts ? 'allow-scripts allow-popups' : 'allow-popups'

  // Keying the frame on the sandbox value forces a fresh element when
  // the mode changes. React patches the attribute in place otherwise,
  // and a live frame does not re-apply sandbox flags — the old policy
  // would linger until something else remounted it.
  const frameKey = `${path}:${sandbox}`

  const doc = useMemo(() => anchorBase(retargetLinks(html)), [html])

  return (
    <div className={cn('flex flex-col gap-1.5 min-h-0', className)}>
      <div className="flex items-center gap-2 shrink-0">
        {scripts ? (
          <ShieldAlert className="size-3.5 text-state-idle" />
        ) : (
          <ShieldCheck className="size-3.5 text-muted-foreground/70" />
        )}
        <span className="text-[11px] text-muted-foreground">
          {scripts
            ? t('web.noteEditor.html.scriptsOn')
            : t('web.noteEditor.html.scriptsOff')}
        </span>
        <Button
          type="button"
          size="sm"
          variant="outline"
          className="h-6 px-2 text-[11px] ml-auto"
          onClick={toggle}
        >
          {scripts
            ? t('web.noteEditor.html.disableScripts')
            : t('web.noteEditor.html.enableScripts')}
        </Button>
      </div>
      <iframe
        key={frameKey}
        ref={frameRef}
        title={path}
        srcDoc={doc}
        sandbox={sandbox}
        className="flex-1 min-h-0 w-full rounded-md border border-border bg-white"
      />
    </div>
  )
}

// Links that LEAVE the document open in a new tab: the frame has no
// address bar and no history UI, so navigating it away strands the
// reader on a page with no way back.
//
// Links to a fragment of THIS document must NOT be retargeted, and that
// distinction is why `<base target="_blank">` — the obvious one-line
// version of this — is wrong. <base> retargets every link in the
// document, including `href="#install"`. Verified in Chromium: clicking
// a table-of-contents entry opened a blank tab at `about:blank#install`
// instead of scrolling. Generated documentation is exactly where this
// bites — Sphinx, typedoc, asciidoc and Notion exports all ship a
// fragment TOC, and footnotes are fragment links too.
//
// Rewriting the anchors individually is more code than a <base>, but it
// is the only version that can tell the two cases apart.
// A srcdoc frame's document URL is `about:srcdoc`, but it inherits its
// BASE url from the parent page. Those disagreeing is what turns a
// `href="#install"` click into a real navigation: the link resolves
// against the parent, so the frame leaves `about:srcdoc` and loads
// `http://<gateway>/notes#install` — the opendray app renders inside
// the document viewer and the document is gone. Measured in both
// Chromium and WebKit; with the base pinned, the same click scrolls.
//
// Pinning it also stops a document's relative URLs from resolving
// against the gateway's origin, which is not somewhere a pulled
// document should be aiming requests.
//
// A document that declares its own <base href> is left alone: that is
// an explicit statement by whoever exported it, and the assets it
// points at keep working. Fragment links in such a document still
// navigate — the cost of respecting the declaration.
//
// MOBILE DOES NOT DO THIS, and must not: loadData gives the webview
// the document URL `about:blank`, which is also its base, so fragments
// already resolve to the same document. Pinning `about:srcdoc` there
// would introduce the very bug this removes.
function anchorBase(html: string): string {
  if (/<base\b[^>]*\bhref\s*=/i.test(html)) return html
  const base = '<base href="about:srcdoc">'
  const head = /<head[^>]*>/i
  // Prepend when there is no <head>: plenty of exported docs are bare
  // fragments, and the parser hoists a leading <base> into one.
  return head.test(html) ? html.replace(head, (m) => m + base) : base + html
}

const ANCHOR_TAG = /<a\b([^>]*)>/gi
const HREF_ATTR = /\bhref\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'>=`]+))/i
const TARGET_ATTR = /\btarget\s*=/i

function retargetLinks(html: string): string {
  return html.replace(ANCHOR_TAG, (tag, attrs: string) => {
    // An explicit target is the author's decision; don't second-guess it.
    if (TARGET_ATTR.test(attrs)) return tag
    const m = HREF_ATTR.exec(attrs)
    const href = (m?.[1] ?? m?.[2] ?? m?.[3] ?? '').trim()
    // No href is not a link. A leading '#' points inside this document.
    if (!href || href.startsWith('#')) return tag
    // Drop a self-closing slash so the injected attributes don't land
    // after it and break the tag.
    return `<a${attrs.replace(/\/\s*$/, '')} target="_blank" rel="noopener noreferrer">`
  })
}
