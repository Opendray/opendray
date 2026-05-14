import React, { useEffect, useMemo, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2, Eye, Pencil, Check, AlertCircle, Hash } from 'lucide-react'
import { toast } from 'sonner'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'
import { listNotes, readNote, writeNote } from '@/lib/notes'
import { detectWikiLinkContext, getCaretCoords } from '@/lib/caret'
import { slugify } from '@/lib/outline'

import { BacklinksPane } from './BacklinksPane'
import { WikiLinkSuggestions } from './WikiLinkSuggestions'
import type { WikiLinkContext } from './WikiLinkSuggestions'

interface NoteEditorProps {
  path: string
  // initialMode picks the default tab. "preview" suits read-mostly
  // surfaces (project docs); "source" suits scratchpads.
  initialMode?: 'source' | 'preview'
  // minHeight controls the textarea/preview min height in inline mode
  // (where the editor grows naturally inside a scrolling parent).
  // Ignored when fillParent is true.
  minHeight?: number
  // fillParent makes the editor expand to fill the available height of
  // its parent and scroll internally. Use this in dialogs / fixed-
  // height shells; leave false for inline embedding inside a scroll
  // area that already handles overflow.
  fillParent?: boolean
  // showStatus toggles the per-instance save indicator. Dialogs may
  // prefer a header-level indicator instead.
  showStatus?: boolean
  // placeholder lets the parent customise the empty-textarea hint.
  placeholder?: string
  // showBacklinks renders a backlinks pane below the editor body.
  // Default false so the inline scratchpad stays minimal — the dialog
  // turns it on so opened project docs show their incoming links.
  showBacklinks?: boolean
  // onOpenLink fires when the user clicks a `[[wiki-link]]` in preview
  // mode. Caller decides what to do (open dialog, navigate, ...).
  onOpenLink?: (path: string) => void
  // onBodyChange streams every keystroke to the parent (used by the
  // /notes page to drive a live outline sidebar without waiting for
  // the debounced save).
  onBodyChange?: (body: string) => void
  // previewScrollRef receives the scrollable preview container so the
  // parent can wire features like scroll-tied outline highlighting.
  previewScrollRef?: React.MutableRefObject<HTMLDivElement | null>
}

const SAVE_DEBOUNCE_MS = 800

// NoteEditor is the reusable note source/preview editor with debounced
// auto-save. Used both inline in the Notes panel (personal scratchpad)
// and inside a dialog (project docs). All variants speak to the same
// /api/v1/notes/{read,write} endpoints.
export function NoteEditor({
  path,
  initialMode = 'source',
  minHeight = 300,
  fillParent = false,
  showStatus = true,
  placeholder,
  showBacklinks = false,
  onOpenLink,
  onBodyChange,
  previewScrollRef,
}: NoteEditorProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ['note', path],
    queryFn: () => readNote(path),
    staleTime: 5_000,
  })

  const [body, setBody] = useState('')
  const [lastSaved, setLastSaved] = useState('')
  const [mode, setMode] = useState<'source' | 'preview'>(initialMode)
  const [error, setError] = useState<string | null>(null)
  const dirty = body !== lastSaved
  const saveTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Wiki-link autocomplete state. Active when the caret is inside an
  // open `[[...` span; stays in sync via re-detection on every input
  // and selection change. Null when not active so the popup unmounts.
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const [linkCtx, setLinkCtx] = useState<
    (WikiLinkContext & { openIdx: number }) | null
  >(null)

  useEffect(() => {
    if (data === undefined) return
    const raw = data?.body ?? ''
    setBody(raw)
    setLastSaved(raw)
    setError(null)
  }, [path, data])

  // Stream body changes to the parent (used for outline + future
  // word-count / live-render features). Fires after every state
  // commit, so debounced enough that callers don't get hammered.
  useEffect(() => {
    onBodyChange?.(body)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [body])

  const save = useMutation({
    mutationFn: (next: string) => writeNote(path, next),
    onSuccess: (_n, next) => {
      setLastSaved(next)
      setError(null)
      qc.invalidateQueries({ queryKey: ['note', path] })
      // Bump the parent listing too — caller may be showing this file
      // in a list with mtime/size that now changed.
      qc.invalidateQueries({ queryKey: ['notes-list'] })
    },
    onError: (err: Error) => {
      setError(err.message)
      toast.error(t('web.noteEditor.saveFailedToast'), { description: err.message })
    },
  })

  useEffect(() => {
    if (!dirty) return
    if (saveTimer.current) clearTimeout(saveTimer.current)
    saveTimer.current = setTimeout(() => {
      save.mutate(body)
    }, SAVE_DEBOUNCE_MS)
    return () => {
      if (saveTimer.current) clearTimeout(saveTimer.current)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [body, dirty])

  // Flush on unmount so tab switches / dialog close don't lose edits.
  useEffect(() => {
    return () => {
      if (saveTimer.current) clearTimeout(saveTimer.current)
      if (body !== lastSaved) {
        void writeNote(path, body).catch(() => {})
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [path])

  const status = useMemo<{
    label: string
    tone: 'idle' | 'saving' | 'saved' | 'error'
  }>(() => {
    if (error) return { label: t('web.noteEditor.status.saveFailed'), tone: 'error' }
    if (save.isPending) return { label: t('web.noteEditor.status.saving'), tone: 'saving' }
    if (dirty) return { label: t('web.noteEditor.status.unsaved'), tone: 'idle' }
    if (data == null) return { label: t('web.noteEditor.status.newNote'), tone: 'idle' }
    return { label: t('web.noteEditor.status.saved'), tone: 'saved' }
  }, [error, save.isPending, dirty, data, t])

  // Tag chips and wiki-link resolution rely on the global note list
  // for basename → path mapping. Cached aggressively (30s) since list
  // refetches happen via other queries on writes anyway.
  const { data: allNotes } = useQuery({
    queryKey: ['notes-list-all'],
    queryFn: () => listNotes(),
    staleTime: 30_000,
  })

  const tags = useMemo(() => extractTags(body), [body])

  const currentDir = useMemo(() => {
    const i = path.lastIndexOf('/')
    return i >= 0 ? path.slice(0, i) : ''
  }, [path])

  const resolveLink = useMemo(() => {
    return (target: string): string => {
      const cleaned = target.replace(/^\/+/, '').replace(/\.md$/i, '')
      if (cleaned.includes('/')) return cleaned + '.md'
      // basename match (case-insensitive) across the whole vault
      const t = cleaned.toLowerCase()
      for (const n of allNotes ?? []) {
        const base = (n.path.split('/').pop() ?? '')
          .replace(/\.md$/i, '')
          .toLowerCase()
        if (base === t) return n.path
      }
      // not found — tentative path next to current note. Clicking will
      // create a fresh note at this location (lazy create on write).
      return currentDir ? `${currentDir}/${cleaned}.md` : `${cleaned}.md`
    }
  }, [allNotes, currentDir])

  // Compose the markdown component map with closure over the link
  // resolver so wiki-links render as clickable buttons. Memoised so
  // ReactMarkdown doesn't re-instantiate components every render.
  const mdComponents = useMemo(
    () => buildMdComponents(resolveLink, onOpenLink),
    [resolveLink, onOpenLink],
  )

  // Recompute the wiki-link suggestion context from the textarea's
  // current selection. Called on every input/keyup/click so the popup
  // appears/disappears as the caret moves in or out of `[[...`.
  const recomputeLinkCtx = (el: HTMLTextAreaElement) => {
    const pos = el.selectionStart ?? 0
    const found = detectWikiLinkContext(el.value, pos)
    if (!found) {
      setLinkCtx(null)
      return
    }
    const caret = getCaretCoords(el, pos)
    setLinkCtx({ query: found.query, openIdx: found.openIdx, caret })
  }

  // completeWikiLink replaces the in-progress `[[query` span with the
  // chosen note's display name and closes the brackets, then puts the
  // caret right after `]]`. For "create new" picks we lazy-create the
  // note via writeNote (no body) so future autocompletes find it.
  const completeWikiLink = (sel: {
    display: string
    path: string
    create?: boolean
  }) => {
    const el = textareaRef.current
    if (!el || !linkCtx) return
    const start = linkCtx.openIdx // points at first `[`
    const end = el.selectionStart ?? start
    // Insert `[[Display]]` (alias-less form). Users can edit the
    // alias afterwards if they want a different visible label.
    const inserted = `[[${sel.display}]]`
    const next = el.value.slice(0, start) + inserted + el.value.slice(end)
    setBody(next)
    setLinkCtx(null)
    // Focus + caret restore after React renders the new value.
    requestAnimationFrame(() => {
      const node = textareaRef.current
      if (!node) return
      node.focus()
      const caret = start + inserted.length
      node.setSelectionRange(caret, caret)
    })
    if (sel.create) {
      // Lazy-create with empty body so the note exists in the index.
      // Failure is non-fatal — it'll be created on first save anyway.
      void writeNote(sel.path, `# ${sel.display}\n\n`).then(() => {
        qc.invalidateQueries({ queryKey: ['notes-list'] })
        qc.invalidateQueries({ queryKey: ['notes-list-all'] })
      })
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-[12px] text-muted-foreground py-3">
        <Loader2 className="size-3 animate-spin" />
        {t('web.noteEditor.loading')}
      </div>
    )
  }

  // Layout: in fillParent mode the editor is a flex column whose
  // active pane gets `flex-1 min-h-0 overflow-auto`, so long content
  // scrolls inside the dialog without pushing it off-screen. In inline
  // mode the panes use a min-height and grow into their (scrolling)
  // parent — same behaviour as before.
  return (
    <div
      className={cn(
        'flex flex-col gap-2',
        fillParent && 'flex-1 min-h-0',
      )}
    >
      <div className="flex items-center gap-1 text-[11px] shrink-0">
        <button
          type="button"
          onClick={() => setMode('source')}
          className={cn(
            'inline-flex items-center gap-1 px-2 py-1 rounded-md transition-colors',
            mode === 'source'
              ? 'bg-card border border-border text-foreground'
              : 'text-muted-foreground hover:text-foreground',
          )}
        >
          <Pencil className="size-3" />
          {t('web.noteEditor.source')}
        </button>
        <button
          type="button"
          onClick={() => setMode('preview')}
          className={cn(
            'inline-flex items-center gap-1 px-2 py-1 rounded-md transition-colors',
            mode === 'preview'
              ? 'bg-card border border-border text-foreground'
              : 'text-muted-foreground hover:text-foreground',
          )}
        >
          <Eye className="size-3" />
          {t('web.noteEditor.preview')}
        </button>
        <div className="flex-1" />
        {showStatus && <SaveStatus status={status} />}
      </div>

      {tags.length > 0 && (
        <div className="flex flex-wrap gap-1 shrink-0">
          {tags.map((tag) => (
            <span
              key={tag}
              className="inline-flex items-center gap-0.5 text-[10px] font-mono px-1.5 py-px rounded bg-card border border-border/60 text-muted-foreground/80"
              title={t('web.noteEditor.tagTitle', { tag })}
            >
              <Hash className="size-2.5" />
              {tag}
            </span>
          ))}
        </div>
      )}

      {mode === 'source' ? (
        <>
          <textarea
            ref={textareaRef}
            value={body}
            onChange={(e) => {
              setBody(e.target.value)
              recomputeLinkCtx(e.target)
            }}
            onKeyUp={(e) => recomputeLinkCtx(e.currentTarget)}
            onClick={(e) => recomputeLinkCtx(e.currentTarget)}
            onBlur={() => {
              // Defer slightly so a click on the popup (mousedown
              // handler) fires before we tear it down.
              setTimeout(() => setLinkCtx(null), 80)
            }}
            placeholder={placeholder}
            style={fillParent ? undefined : { minHeight: `${minHeight}px` }}
            className={cn(
              'w-full font-mono text-[12px] leading-snug rounded-md',
              'border border-border bg-input/40 px-3 py-2 text-foreground',
              'placeholder:text-muted-foreground/60',
              'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
              fillParent ? 'flex-1 min-h-0 resize-none' : 'resize-y',
            )}
            spellCheck={false}
          />
          <WikiLinkSuggestions
            context={linkCtx}
            notes={allNotes ?? []}
            excludePath={path}
            onSelect={(sel) => completeWikiLink(sel)}
            onDismiss={() => setLinkCtx(null)}
          />
        </>
      ) : (
        <div
          ref={(el) => {
            if (previewScrollRef) previewScrollRef.current = el
          }}
          style={fillParent ? undefined : { minHeight: `${minHeight}px` }}
          className={cn(
            'rounded-md border border-border bg-card/30 px-3 py-2',
            'text-[12px] leading-relaxed prose-md',
            fillParent && 'flex-1 min-h-0 overflow-auto',
          )}
        >
          {body.trim().length === 0 ? (
            <div className="text-muted-foreground/60">
              {t('web.noteEditor.emptyNote')}
            </div>
          ) : (
            <ReactMarkdown remarkPlugins={[remarkGfm]} components={mdComponents}>
              {preprocessWikiLinks(body)}
            </ReactMarkdown>
          )}
        </div>
      )}

      {showBacklinks && (
        <div
          className={cn(
            'pt-2 border-t border-border/40',
            // In fillParent (dialog / full-page editor) the backlinks
            // list can run to dozens of rows. Cap it at ~40% of the
            // container and scroll internally so it never pushes the
            // editor pane (or the dialog itself) off-screen.
            fillParent
              ? 'min-h-0 max-h-[40%] overflow-y-auto'
              : 'shrink-0',
          )}
        >
          <BacklinksPane path={path} onOpen={onOpenLink} />
        </div>
      )}
    </div>
  )
}

function SaveStatus({
  status,
}: {
  status: {
    label: string
    tone: 'idle' | 'saving' | 'saved' | 'error'
  }
}) {
  const Icon =
    status.tone === 'saving'
      ? Loader2
      : status.tone === 'error'
        ? AlertCircle
        : Check
  const cls =
    status.tone === 'saved'
      ? 'text-state-running'
      : status.tone === 'error'
        ? 'text-state-failed'
        : 'text-muted-foreground/70'
  return (
    <span className={cn('inline-flex items-center gap-1', cls)}>
      <Icon className={cn('size-3', status.tone === 'saving' && 'animate-spin')} />
      {status.label}
    </span>
  )
}

// WIKI_LINK_TOKEN is the placeholder we substitute `[[...]]` with so
// react-markdown sees it as a special inline string we can recognise
// inside text nodes. Choosing a sentinel that never collides with
// real markdown content while staying safe through GFM's text passes.
const WIKI_LINK_RE = /\[\[([^\]|]+)(?:\|([^\]]+))?\]\]/g

// preprocessWikiLinks rewrites `[[Target]]` / `[[Target|Alias]]` into
// a sentinel form `⁣[[Target|Alias]]⁣` that survives GFM's
// inline parsing intact. The text node renderer then splits on the
// sentinels and emits clickable buttons.
function preprocessWikiLinks(body: string): string {
  return body.replace(
    WIKI_LINK_RE,
    (_m, target, alias) =>
      `⁣[[${target}${alias ? `|${alias}` : ''}]]⁣`,
  )
}

// splitWikiLinks turns the sentinel-wrapped string back into a mix of
// plain text and button elements. Called by the markdown text-node
// renderer so links work inside paragraphs, list items, table cells,
// etc. — wherever GFM puts text.
function splitWikiLinks(
  s: string,
  resolve: (target: string) => string,
  onClick?: (path: string) => void,
): React.ReactNode {
  if (s.indexOf('⁣') < 0) return s
  const parts: React.ReactNode[] = []
  const rx = /⁣\[\[([^\]|]+)(?:\|([^\]]+))?\]\]⁣/g
  let last = 0
  let m: RegExpExecArray | null
  let i = 0
  while ((m = rx.exec(s)) !== null) {
    if (m.index > last) parts.push(s.slice(last, m.index))
    const target = m[1]
    const alias = m[2] ?? target
    const resolved = resolve(target)
    parts.push(
      <button
        key={`wl-${i++}`}
        type="button"
        onClick={(e) => {
          e.preventDefault()
          onClick?.(resolved)
        }}
        className="text-state-running hover:underline font-mono"
        title={`→ ${resolved}`}
      >
        {alias}
      </button>,
    )
    last = m.index + m[0].length
  }
  if (last < s.length) parts.push(s.slice(last))
  return <>{parts}</>
}

// extractTags pulls #tag mentions and frontmatter `tags:` arrays from
// the body. Mirrors the backend tag scanner so client and server agree
// on what counts as a tag.
function extractTags(body: string): string[] {
  const tags = new Set<string>()
  // Frontmatter
  if (body.startsWith('---')) {
    const end = body.slice(3).indexOf('---')
    if (end >= 0) {
      const header = body.slice(3, 3 + end)
      const inline = header.match(/tags:\s*\[(.*?)\]/)
      if (inline) {
        for (const t of inline[1].split(',')) {
          const cleaned = t.trim().replace(/^['"]|['"]$/g, '')
          if (cleaned) tags.add(cleaned)
        }
      }
      const blockMatch = header.match(/tags:\s*\n((?:\s*-\s*.+\n?)+)/)
      if (blockMatch) {
        for (const line of blockMatch[1].split('\n')) {
          const t = line
            .replace(/^\s*-\s*/, '')
            .trim()
            .replace(/^['"]|['"]$/g, '')
          if (t) tags.add(t)
        }
      }
    }
  }
  // Body — strip code blocks first.
  const stripped = body
    .replace(/```[\s\S]*?```/g, ' ')
    .replace(/`[^`]*`/g, ' ')
  const tagRe = /(?:^|[^A-Za-z0-9_-])#([A-Za-z][A-Za-z0-9_/-]{0,40})/g
  let m: RegExpExecArray | null
  while ((m = tagRe.exec(stripped)) !== null) {
    tags.add(m[1].replace(/\/$/, ''))
  }
  return [...tags]
}

// buildMdComponents assembles the react-markdown component map with
// the wiki-link resolver baked in. Anything that can host text
// (paragraph, list item, headings, table cells, blockquote, …) routes
// through wrapText so [[wiki]] works everywhere.
function buildMdComponents(
  resolve: (target: string) => string,
  onClick?: (path: string) => void,
) {
  const wrapText = (children: React.ReactNode): React.ReactNode => {
    if (typeof children === 'string') {
      return splitWikiLinks(children, resolve, onClick)
    }
    if (Array.isArray(children)) {
      return children.map((c, i) =>
        typeof c === 'string' ? (
          <React.Fragment key={i}>
            {splitWikiLinks(c, resolve, onClick)}
          </React.Fragment>
        ) : (
          c
        ),
      )
    }
    return children
  }
  // Heading components emit a `data-outline-id` based on the
  // slugified text so OutlineSidebar can scroll to them via
  // querySelector. Dedup is the outline extractor's job — the same
  // slug rule (slugify) is used in both places.
  return {
    h1: ({ children, ...props }: any) => (
      <h1
        data-outline-id={slugify(textOf(children))}
        className="text-[15px] font-semibold mt-3 mb-2 scroll-mt-2"
        {...props}
      >
        {wrapText(children)}
      </h1>
    ),
    h2: ({ children, ...props }: any) => (
      <h2
        data-outline-id={slugify(textOf(children))}
        className="text-[13px] font-semibold mt-3 mb-1.5 scroll-mt-2"
        {...props}
      >
        {wrapText(children)}
      </h2>
    ),
    h3: ({ children, ...props }: any) => (
      <h3
        data-outline-id={slugify(textOf(children))}
        className="text-[12.5px] font-semibold mt-2 mb-1 scroll-mt-2"
        {...props}
      >
        {wrapText(children)}
      </h3>
    ),
    h4: ({ children, ...props }: any) => (
      <h4
        data-outline-id={slugify(textOf(children))}
        className="text-[12px] font-semibold mt-2 mb-1 scroll-mt-2"
        {...props}
      >
        {wrapText(children)}
      </h4>
    ),
    h5: ({ children, ...props }: any) => (
      <h5
        data-outline-id={slugify(textOf(children))}
        className="text-[11.5px] font-semibold mt-2 mb-1 scroll-mt-2"
        {...props}
      >
        {wrapText(children)}
      </h5>
    ),
    h6: ({ children, ...props }: any) => (
      <h6
        data-outline-id={slugify(textOf(children))}
        className="text-[11.5px] font-semibold mt-2 mb-1 text-muted-foreground/80 scroll-mt-2"
        {...props}
      >
        {wrapText(children)}
      </h6>
    ),
    p: ({ children, ...props }: any) => (
      <p className="my-1.5 text-foreground/90" {...props}>
        {wrapText(children)}
      </p>
    ),
    ul: (props: any) => (
      <ul className="list-disc pl-4 my-1.5 space-y-0.5" {...props} />
    ),
    ol: (props: any) => (
      <ol className="list-decimal pl-4 my-1.5 space-y-0.5" {...props} />
    ),
    li: ({ children, ...props }: any) => (
      <li className="my-0.5" {...props}>
        {wrapText(children)}
      </li>
    ),
    code: ({ inline, className, children, ...rest }: any) => {
      if (inline) {
        return (
          <code
            className="px-1 py-px rounded bg-muted/60 text-[11.5px] font-mono"
            {...rest}
          >
            {children}
          </code>
        )
      }
      return (
        <pre className="bg-card/40 border border-border/60 rounded-md p-2 my-2 overflow-auto">
          <code
            className={cn('hljs font-mono text-[11.5px]', className)}
            {...rest}
          >
            {children}
          </code>
        </pre>
      )
    },
    blockquote: ({ children, ...props }: any) => (
      <blockquote
        className="border-l-2 border-border pl-2 my-2 text-muted-foreground/90 italic"
        {...props}
      >
        {wrapText(children)}
      </blockquote>
    ),
    a: (props: any) => (
      <a
        className="text-state-running hover:underline"
        target="_blank"
        rel="noopener noreferrer"
        {...props}
      />
    ),
    table: (props: any) => (
      <table
        className="border-collapse my-2 text-[11.5px] [&_th]:border [&_td]:border [&_th]:border-border [&_td]:border-border [&_th]:px-2 [&_td]:px-2 [&_th]:py-1 [&_td]:py-1"
        {...props}
      />
    ),
    td: ({ children, ...props }: any) => (
      <td {...props}>{wrapText(children)}</td>
    ),
    hr: () => <hr className="border-border/60 my-3" />,
  }
}

// textOf flattens react-markdown's heading children into a plain
// string. Used solely to compute slugs for outline anchors.
function textOf(children: React.ReactNode): string {
  if (children == null) return ''
  if (typeof children === 'string') return children
  if (typeof children === 'number') return String(children)
  if (Array.isArray(children)) return children.map(textOf).join('')
  if (
    typeof children === 'object' &&
    'props' in (children as any) &&
    (children as any).props
  ) {
    return textOf((children as any).props.children)
  }
  return ''
}
