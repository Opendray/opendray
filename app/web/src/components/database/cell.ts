// cellText turns a JSON cell value from a DBResultSet into display text.
// Objects/arrays (jsonb) are compacted to JSON; null/undefined render as
// empty (callers show a distinct NULL marker).
export function cellText(v: unknown): string {
  if (v === null || v === undefined) return ''
  if (typeof v === 'object') return JSON.stringify(v)
  return String(v)
}
