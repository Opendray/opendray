/**
 * URL extraction helpers for the session terminal.
 *
 * Pulled out of DetectedURLs.tsx because:
 *   - the regexes intentionally match ANSI control bytes, and that
 *     trips eslint's `no-control-regex` only when it's inside a
 *     `.tsx` (the eslint config tolerates control chars in `.ts`);
 *   - keeping the helpers in a non-component file lets
 *     `react-refresh/only-export-components` keep treating
 *     DetectedURLs.tsx as a clean component module;
 *   - unit-testing the regex behaviour is easier when there's no
 *     React import in the test file.
 */

/**
 * Match http(s) URLs in plain text (after ANSI stripping). Stops at
 * whitespace, control chars, and angle brackets so adjacent UI text
 * doesn't get rolled into the match.
 */
// eslint-disable-next-line no-control-regex -- \x00-\x1f are stop chars
const URL_REGEX = /\bhttps?:\/\/[^\s<>"'\x00-\x1f]+/g

export function extractURLs(text: string): string[] {
  const matches = text.match(URL_REGEX)
  if (!matches) return []
  return matches.map(trimTrailingPunctuation).filter((u) => u.length > 0)
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

/**
 * Strip ANSI CSI escape sequences from text so URL extraction isn't
 * confused by colour resets or cursor moves embedded in the stream.
 * Covers the only form we see in PTY output: ESC `[` followed by
 * parameter bytes (digits / `;` / `?`) and a final letter — SGR
 * (colours), DEC private modes, cursor positioning, etc.
 */
// eslint-disable-next-line no-control-regex -- \x1b is the literal ESC byte
const ANSI_CSI_REGEX = /\x1b\[[0-9;?]*[a-zA-Z]/g

export function stripANSI(text: string): string {
  return text.replace(ANSI_CSI_REGEX, '')
}
