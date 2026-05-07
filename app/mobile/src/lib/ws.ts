// Mobile WebSocket URL builder.
//
// shared/lib/ws.ts ships an `wsURL(path, token)` helper that assumes
// same-origin (uses `window.location.host`). Mobile is never
// same-origin — the gateway URL is user-entered at first launch — so
// we construct absolute WS URLs explicitly here. The `BinaryWS` class
// from shared/lib/ws.ts is URL-agnostic and gets reused as-is.

export function mobileWSURL(
  serverURL: string,
  path: string,
  token: string,
): string {
  // Convert http:// → ws://, https:// → wss://. Strip trailing slash
  // on serverURL so we don't double up the path separator.
  const trimmed = serverURL.replace(/\/+$/, '')
  const wsBase = trimmed
    .replace(/^http:\/\//i, 'ws://')
    .replace(/^https:\/\//i, 'wss://')
  const prefixed = path.startsWith('/') ? path : `/${path}`
  const sep = prefixed.includes('?') ? '&' : '?'
  return `${wsBase}${prefixed}${sep}token=${encodeURIComponent(token)}`
}
