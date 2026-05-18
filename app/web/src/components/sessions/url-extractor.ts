/**
 * URL extraction helpers for the session terminal.
 *
 * Pulled out of DetectedURLs.tsx because the regexes intentionally
 * match ANSI control bytes (eslint's `no-control-regex` only trips
 * on `.tsx` here), and to keep DetectedURLs.tsx a clean component
 * module for `react-refresh/only-export-components`.
 */

/**
 * URL syntax characters per RFC 3986 — the body of an http(s) URL
 * is any of these. Used by the state-machine walker below to decide
 * whether a `\n` mid-URL is a CLI-inserted soft-wrap (continue) or
 * a real terminator (stop).
 */
const URL_BODY_CHAR = /[A-Za-z0-9!$&'()*+,\-./:;=?@_~%#[\]]/

/**
 * Strip ANSI CSI escape sequences from text so URL extraction isn't
 * confused by colour resets or cursor moves embedded in the stream.
 */
// eslint-disable-next-line no-control-regex -- \x1b is the literal ESC byte
const ANSI_CSI_REGEX = /\x1b\[[0-9;?]*[a-zA-Z]/g

export function stripANSI(text: string): string {
  return text.replace(ANSI_CSI_REGEX, '')
}

/**
 * Extract http(s) URLs from text. Handles the case AI CLIs (notably
 * claude-code) actively hard-wrap long OAuth URLs across multiple
 * terminal lines by inserting literal `\n` chars every N columns:
 *
 *     https://claude.com/cai/oauth/authorize?code=true&client_
 *     id=9d1c250a-...&state=oJ-...
 *
 * A naive `\bhttps?://[^\s]+` regex stops at the first `\n` and
 * captures only the first soft-wrapped segment — exactly the bug
 * the v2.0.3 badge ran into. (Pre-processing the text to strip
 * `\n`-between-URL-chars almost works, but it also destroys word
 * boundaries between prose lines and gluing "open\nhttps://…" into
 * one word makes the `\b` anchor fail entirely.)
 *
 * Approach instead: anchor on `https?://`, then walk char by char,
 * consuming URL-body chars (and skipping ONE intermediate `\r`/`\n`
 * if it's followed by another URL-body char). Stop on:
 *   - Real horizontal whitespace (` `, `\t`)
 *   - Quote / angle bracket (`"`, `'`, `<`, `>`)
 *   - Two or more consecutive newlines (paragraph break)
 *   - Single newline followed by non-URL char (prose continuation)
 *   - Other ASCII control characters
 *
 * Embedded `\n`s are stripped from the captured URL before return.
 */
const URL_START_REGEX = /https?:\/\//g

// Minimum length of the "current line" for a single `\n` to be treated
// as a CLI soft-wrap. Lines shorter than this are assumed to be real
// content breaks (prose, multi-paragraph output, …). 40 cols is well
// below where AI CLIs actually wrap (~55-80) but well above any
// realistic "<intro phrase>\n<url>" prose pattern.
const SOFT_WRAP_MIN_LINE_LEN = 40

export function extractURLs(text: string): string[] {
  const results = new Set<string>()
  // Reset lastIndex on a fresh regex object so the walker doesn't
  // accidentally pick up state from a sibling exec elsewhere.
  URL_START_REGEX.lastIndex = 0
  let match: RegExpExecArray | null
  while ((match = URL_START_REGEX.exec(text)) !== null) {
    const start = match.index
    let i = start + match[0].length // skip past `https://` so we don't immediately re-match
    while (i < text.length) {
      const ch = text[i]
      const code = ch.charCodeAt(0)

      // Hard terminators — these never appear inside URLs.
      if (
        ch === ' ' ||
        ch === '\t' ||
        ch === '<' ||
        ch === '>' ||
        ch === '"' ||
        ch === "'"
      ) {
        break
      }

      // Newlines: allow up to one INTERNAL newline as a CLI soft-wrap;
      // two consecutive newlines = paragraph break and the URL is done.
      if (ch === '\n' || ch === '\r') {
        let j = i + 1
        let nlCount = ch === '\n' ? 1 : 0
        // Sweep through the CR/LF run so `\r\n` counts as ONE newline.
        while (j < text.length && (text[j] === '\n' || text[j] === '\r')) {
          if (text[j] === '\n') nlCount++
          j++
        }
        if (nlCount >= 2) break // paragraph break terminates
        if (j >= text.length) break // trailing newline at end of buffer
        if (!URL_BODY_CHAR.test(text[j])) break // followed by non-URL prose

        // Heuristic — distinguish a CLI soft-wrap (continue) from a real
        // paragraph break that happens to be followed by URL-body chars
        // (stop). claude-code / codex / gemini wrap long URLs at the
        // terminal column width (~55-80); prose lines like "see\nhttps://x"
        // are usually < 40 chars before the break. If the CURRENT line is
        // shorter than the wrap-width floor, treat the `\n` as real and
        // stop the URL there.
        const prevNlIdx = text.lastIndexOf('\n', i - 1)
        const currentLineStart = prevNlIdx === -1 ? 0 : prevNlIdx + 1
        const currentLineLen = i - currentLineStart
        if (currentLineLen < SOFT_WRAP_MIN_LINE_LEN) break

        // Single newline with URL-body chars on both sides + a "long"
        // current line: CLI soft-wrap. Skip the newline, continue.
        i = j
        continue
      }

      // Other control chars terminate.
      if (code < 0x20) break

      i++
    }

    const raw = text.slice(start, i).replace(/[\r\n]+/g, '')
    const cleaned = trimTrailingPunctuation(raw)
    if (cleaned.length > 0) {
      results.add(cleaned)
    }

    // Advance the start regex past the URL we just consumed so we
    // don't try to start a new match inside it (where a URL-encoded
    // `https%3A%2F%2F` lives, for instance).
    URL_START_REGEX.lastIndex = i
  }
  return Array.from(results)
}

/**
 * Drop sentence punctuation that almost never ends a URL. Handles
 * the common "see https://example.com." → `https://example.com`
 * case, plus stray `,` `;` `:` `!` `?` `'`. For `)` and `]` we only
 * trim when there's no matching open bracket inside the URL — that
 * preserves things like `https://example.com/foo(bar)`.
 */
function trimTrailingPunctuation(url: string): string {
  let end = url.length
  while (end > 0) {
    const ch = url[end - 1]
    if (
      ch === '.' ||
      ch === ',' ||
      ch === ';' ||
      ch === ':' ||
      ch === '!' ||
      ch === '?' ||
      ch === '\'' ||
      (ch === ')' && !url.slice(0, end - 1).includes('(')) ||
      (ch === ']' && !url.slice(0, end - 1).includes('['))
    ) {
      end -= 1
    } else {
      break
    }
  }
  return url.slice(0, end)
}
