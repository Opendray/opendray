import { useEffect, useRef, useState } from 'react'

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
const SHARED_BOX =
  'w-full px-3 py-2 font-mono text-[12px] leading-snug border ' +
  'whitespace-pre-wrap break-words'

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
}: HighlightedSourceProps) {
  const preRef = useRef<HTMLPreElement>(null)
  const [html, setHtml] = useState<string | null>(null)

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

  // The backdrop does not scroll on its own; it follows the textarea.
  const syncScroll = () => {
    const ta = textareaRef.current
    const pre = preRef.current
    if (!ta || !pre) return
    pre.scrollTop = ta.scrollTop
    pre.scrollLeft = ta.scrollLeft
  }

  const sizing = fillParent
    ? 'flex-1 min-h-0'
    : undefined
  const style = fillParent ? undefined : { minHeight: `${minHeight ?? 220}px` }

  return (
    <div className={cn('relative', sizing, className)} style={style}>
      <pre
        ref={preRef}
        aria-hidden="true"
        className={cn(
          SHARED_BOX,
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
        className={cn(
          SHARED_BOX,
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
