import type { Terminal as XTerm } from '@xterm/xterm'

// terminalBufferText reconstructs the terminal's full buffer (scrollback
// + viewport) as clean plain text.
//
// We deliberately do NOT use term.selectAll()+getSelection(): that path
// keeps every cell's trailing padding, so each line comes back padded to
// the terminal width with spaces, and a session that only printed a few
// lines copies as a wall of blank rows. Instead we walk the buffer and
// translateToString(true) each line — `true` trims trailing whitespace —
// then drop the trailing run of empty lines so a freshly-started session
// doesn't copy thousands of blanks below its output.
export function terminalBufferText(term: XTerm | null | undefined): string {
  if (!term) return ''
  const buf = term.buffer.active
  const lines: string[] = []
  for (let i = 0; i < buf.length; i++) {
    // trimRight (the translateToString arg) strips the per-cell padding
    // xterm keeps to the right of the last printed glyph on each row.
    lines.push(buf.getLine(i)?.translateToString(true) ?? '')
  }
  // Drop trailing blank lines — the scrollback past the last printed row
  // is just empty cells we don't want in the clipboard.
  let end = lines.length
  while (end > 0 && lines[end - 1].trim() === '') end--
  return lines.slice(0, end).join('\n')
}
