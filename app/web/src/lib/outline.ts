// Outline / TOC extraction. Walks the note body line-by-line, picks
// out ATX headings (`#`/`##`/...), skips code-fence regions, and
// emits a flat array with depth + stable slug ids. Callers pair the
// slug with `<h1 id={...}>` rendered by ReactMarkdown to enable
// click-to-scroll navigation.

export interface OutlineHeading {
  level: number // 1..6
  text: string
  // slug is the URL-fragment-safe id used both as <h*>'s DOM id and
  // as the OutlineSidebar's React key. Deduped per-document via a
  // suffix when the same text appears more than once.
  slug: string
  // lineIndex of the heading in the source body (0-based). Useful
  // for future "jump in source mode" support.
  lineIndex: number
}

const HEADING_RE = /^(#{1,6})\s+(.+?)\s*#*\s*$/

export function extractOutline(body: string): OutlineHeading[] {
  const out: OutlineHeading[] = []
  const seen = new Map<string, number>()
  let inFence = false
  const lines = body.split('\n')
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    const trimmed = line.trim()
    // Toggle fence state on bare ``` / ~~~ lines (also catches ```lang).
    if (/^(```|~~~)/.test(trimmed)) {
      inFence = !inFence
      continue
    }
    if (inFence) continue
    // Frontmatter (--- ... ---) at the very top is not a heading even
    // though it doesn't start with #. Already filtered by HEADING_RE.
    const m = HEADING_RE.exec(line)
    if (!m) continue
    const level = m[1].length
    const text = m[2].trim()
    if (text === '') continue
    const base = slugify(text)
    let slug = base
    const n = seen.get(base) ?? 0
    if (n > 0) slug = `${base}-${n}`
    seen.set(base, n + 1)
    out.push({ level, text, slug, lineIndex: i })
  }
  return out
}

// slugify: lowercase, replace non-alnum runs with `-`, trim leading/
// trailing `-`. Matches the conventions used by GitHub markdown
// anchors closely enough for our internal use.
export function slugify(text: string): string {
  return (
    text
      .toLowerCase()
      .replace(/[^\p{Letter}\p{Number}]+/gu, '-')
      .replace(/^-+|-+$/g, '') || 'section'
  )
}
