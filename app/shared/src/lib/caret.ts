// Compute the pixel coordinates of the caret inside a <textarea>.
// Standard "mirror div" technique: build a hidden div with the same
// text-affecting CSS as the textarea, copy the pre-caret content into
// it, then measure the bounding box of a sentinel span at the caret.
//
// Returns viewport-relative (page-fixed) coordinates so callers can
// position a `position: fixed` element next to the caret without
// worrying about the textarea's own scroll/transform context.

const COPIED_PROPS = [
  'boxSizing',
  'width',
  'height',
  'overflowX',
  'overflowY',
  'borderTopWidth',
  'borderRightWidth',
  'borderBottomWidth',
  'borderLeftWidth',
  'paddingTop',
  'paddingRight',
  'paddingBottom',
  'paddingLeft',
  'fontStyle',
  'fontVariant',
  'fontWeight',
  'fontStretch',
  'fontSize',
  'fontSizeAdjust',
  'lineHeight',
  'fontFamily',
  'textAlign',
  'textTransform',
  'textIndent',
  'letterSpacing',
  'wordSpacing',
  'tabSize',
] as const

export interface CaretCoords {
  top: number // viewport
  left: number // viewport
  height: number
}

export function getCaretCoords(
  el: HTMLTextAreaElement,
  position: number,
): CaretCoords {
  const div = document.createElement('div')
  document.body.appendChild(div)

  const style = window.getComputedStyle(el)
  for (const p of COPIED_PROPS) {
    ;(div.style as any)[p] = style.getPropertyValue(camelToKebab(p))
  }
  div.style.position = 'absolute'
  div.style.visibility = 'hidden'
  div.style.whiteSpace = 'pre-wrap'
  div.style.wordWrap = 'break-word'
  div.style.overflow = 'hidden'
  div.style.top = '0'
  div.style.left = '-9999px'

  const before = el.value.substring(0, position)
  div.textContent = before

  const span = document.createElement('span')
  // The trailing char makes sure the span has bbox even at end of text.
  span.textContent = el.value.substring(position) || '.'
  div.appendChild(span)

  const elRect = el.getBoundingClientRect()
  const divRect = div.getBoundingClientRect()
  const spanRect = span.getBoundingClientRect()

  // Translate from the mirror div's coordinate space back to the
  // textarea's, then from the textarea's into viewport-fixed coords.
  const top = elRect.top + (spanRect.top - divRect.top) - el.scrollTop
  const left = elRect.left + (spanRect.left - divRect.left) - el.scrollLeft
  const lineHeight =
    parseInt(style.lineHeight) ||
    Math.round(parseInt(style.fontSize) * 1.2)

  document.body.removeChild(div)
  return { top, left, height: lineHeight }
}

function camelToKebab(s: string): string {
  return s.replace(/[A-Z]/g, (m) => '-' + m.toLowerCase())
}

// detectWikiLinkContext inspects the text right before the caret and
// returns the active `[[...` query if the caret is inside an unclosed
// wiki-link. Returns null otherwise. Aborts if the bracket span
// contains `]` or a newline (treats those as closes).
export function detectWikiLinkContext(
  text: string,
  caretPos: number,
): { query: string; openIdx: number } | null {
  // Walk backwards from caret looking for `[[` without a closing `]]`
  // in between. Hard cap the search distance — a runaway buffer
  // shouldn't burn cycles scanning hundreds of KB.
  const SEARCH_LIMIT = 256
  const start = Math.max(0, caretPos - SEARCH_LIMIT)
  for (let i = caretPos - 1; i >= start; i--) {
    const ch = text[i]
    if (ch === '\n' || ch === ']') return null
    if (ch === '[' && text[i - 1] === '[') {
      const query = text.slice(i + 1, caretPos)
      // [[`]` would have aborted above, so query is guaranteed clean.
      return { query, openIdx: i - 1 }
    }
  }
  return null
}
