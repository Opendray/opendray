import { useEffect, useMemo, useRef, useState } from 'react'

import { highlightCode } from '@/lib/highlight'
import { cn } from '@/lib/utils'

// A markdown source view that is coloured while staying editable.
//
// The file viewer has always highlighted what it shows, so raw markdown
// in the Vault — the one place people actually read and write it — was
// the only flat grey surface left. Fixing that in a textarea means the
// classic backdrop trick: a <pre> rendering highlighted HTML, and the
// real <textarea> on top of it with transparent text and a visible
// caret.
//
// The trick has exactly one failure mode, and it is unforgiving: if the
// two layers disagree about ANY metric — font, size, line-height,
// padding, letter-spacing, wrapping — the caret drifts further from the
// glyphs the further down you type. Both layers therefore take their
// box and type styling from the same SHARED constant below. Change it
// in one place or not at all.
// Line numbers force one more agreement: WRAPPING.
//
// A wrapped logical line occupies several visual rows, so a gutter
// counting 1..N drifts the moment any line is longer than the box —
// and in an HTML document, most are. Every code editor answers this
// the same way, by not wrapping. So the gutter and `whitespace-pre`
// travel together; without numbers the view keeps wrapping, which is
// what prose wants.
const SHARED_BOX =
  'w-full py-2 font-mono text-[12px] leading-snug border'
const WRAP_BOX = 'whitespace-pre-wrap break-words'
const NOWRAP_BOX = 'whitespace-pre overflow-auto'

// Width of the gutter, in ch, for a given line count. Both layers pad
// left by exactly this much — a disagreement here is the caret drift
// this file exists to avoid.
function gutterCh(lines: number): number {
  return String(Math.max(lines, 1)).length + 2
}

interface HighlightedSourceProps {
  value: string
  onChange: (next: string) => void
  /** highlight.js language for the backdrop. Follows the document's
   * extension — highlighting an HTML doc as markdown produced a wall
   * of undifferentiated text. */
  language?: string
  /** Forwarded to the textarea so callers keep caret/selection access. */
  textareaRef: React.RefObject<HTMLTextAreaElement | null>
  placeholder?: string
  fillParent?: boolean
  minHeight?: number
  className?: string
  onKeyUp?: React.KeyboardEventHandler<HTMLTextAreaElement>
  onClick?: React.MouseEventHandler<HTMLTextAreaElement>
  onBlur?: React.FocusEventHandler<HTMLTextAreaElement>
  /** Show a line-number gutter. Implies no wrapping — see SHARED_BOX. */
  showLineNumbers?: boolean
}

export function HighlightedSource({
  value,
  onChange,
  language = 'markdown',
  textareaRef,
  placeholder,
  fillParent,
  minHeight,
  className,
  onKeyUp,
  onClick,
  onBlur,
  showLineNumbers = false,
}: HighlightedSourceProps) {
  const preRef = useRef<HTMLPreElement>(null)
  const gutterRef = useRef<HTMLDivElement>(null)
  const [html, setHtml] = useState<string | null>(null)

  const lineCount = useMemo(() => value.split('\n').length, [value])
  const gutterWidth = showLineNumbers ? gutterCh(lineCount) : 0
  const padLeft = showLineNumbers ? `${gutterWidth + 1}ch` : undefined

  // hljs loads lazily, and highlighting a long note on every keystroke
  // would make typing feel heavy. Debounce, and render plain text until
  // the first result lands — a brief moment of grey beats a stutter.
  useEffect(() => {
    let cancelled = false
    const timer = setTimeout(() => {
      highlightCode(value, language)
        .then((out) => {
          if (!cancelled) setHtml(out)
        })
        .catch(() => {
          if (!cancelled) setHtml(null)
        })
    }, 120)
    return () => {
      cancelled = true
      clearTimeout(timer)
    }
  }, [value, language])

  // Neither the backdrop nor the gutter scrolls on its own; both
  // follow the textarea. The gutter tracks vertical only — it must
  // stay put while the code scrolls sideways.
  const syncScroll = () => {
    const ta = textareaRef.current
    if (!ta) return
    if (preRef.current) {
      preRef.current.scrollTop = ta.scrollTop
      preRef.current.scrollLeft = ta.scrollLeft
    }
    if (gutterRef.current) {
      gutterRef.current.scrollTop = ta.scrollTop
    }
  }

  const sizing = fillParent
    ? 'flex-1 min-h-0'
    : undefined
  const style = fillParent ? undefined : { minHeight: `${minHeight ?? 220}px` }

  return (
    <div className={cn('relative', sizing, className)} style={style}>
      {showLineNumbers && (
        <div
          ref={gutterRef}
          aria-hidden="true"
          className={cn(
            'pointer-events-none absolute left-0 top-0 bottom-0 z-10 overflow-hidden',
            'py-2 pr-2 font-mono text-[12px] leading-snug text-right',
            'select-none rounded-l-md border-y border-l border-border',
            'bg-input/60 text-muted-foreground/50',
          )}
          style={{ width: `${gutterWidth}ch` }}
        >
          {Array.from({ length: lineCount }, (_, i) => (
            <div key={i}>{i + 1}</div>
          ))}
        </div>
      )}
      <pre
        ref={preRef}
        aria-hidden="true"
        style={{ paddingLeft: padLeft }}
        className={cn(
          SHARED_BOX,
          showLineNumbers ? NOWRAP_BOX : WRAP_BOX,
          !showLineNumbers && 'px-3',
          showLineNumbers && 'pr-3',
          'hljs pointer-events-none absolute inset-0 m-0 overflow-hidden',
          'rounded-md border-border bg-input/40 text-foreground',
        )}
      >
        {/* A trailing newline keeps the last line's height stable while
            typing at the very end of the note. */}
        {html !== null ? (
          <code dangerouslySetInnerHTML={{ __html: html + '\n' }} />
        ) : (
          <code>{value + '\n'}</code>
        )}
      </pre>
      <textarea
        ref={textareaRef}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onScroll={syncScroll}
        onKeyUp={onKeyUp}
        onClick={onClick}
        onBlur={onBlur}
        placeholder={placeholder}
        spellCheck={false}
        style={{ paddingLeft: padLeft }}
        className={cn(
          SHARED_BOX,
          showLineNumbers ? NOWRAP_BOX : WRAP_BOX,
          !showLineNumbers && 'px-3',
          showLineNumbers && 'pr-3',
          'absolute inset-0 h-full resize-none rounded-md border-border',
          // The glyphs come from the backdrop; only the caret and the
          // selection highlight belong to the textarea.
          'bg-transparent text-transparent caret-foreground',
          'selection:bg-accent/30 selection:text-transparent',
          'placeholder:text-muted-foreground/60',
          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        )}
      />
    </div>
  )
}
