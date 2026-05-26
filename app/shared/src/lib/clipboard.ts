// copyText writes `text` to the clipboard, returning whether it
// succeeded.
//
// The async Clipboard API (navigator.clipboard.writeText) is only
// exposed in a *secure context* — https, or http on localhost. The
// opendray dashboard is frequently reached over plain http on a LAN
// IP (e.g. http://192.168.x.x:8770 from an iPad), where
// navigator.clipboard is `undefined` and every writeText call throws
// or no-ops. That's why "Copy" buttons silently fail on mobile.
//
// So we try the modern API when it's actually available, then fall
// back to the legacy execCommand('copy') path, which works in
// non-secure contexts (incl. iOS Safari) as long as it runs inside a
// user gesture — which every caller here does (a tap/click handler).
export async function copyText(text: string): Promise<boolean> {
  if (
    typeof navigator !== 'undefined' &&
    navigator.clipboard &&
    typeof window !== 'undefined' &&
    window.isSecureContext
  ) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      // Permission denied / transient failure — fall through to legacy.
    }
  }
  return legacyCopy(text)
}

// legacyCopy stages the text in an off-screen <textarea>, selects it,
// and runs document.execCommand('copy'). iOS Safari needs the node to
// be actually rendered (not display:none) and needs an explicit
// setSelectionRange — a bare .select() doesn't take on iOS.
function legacyCopy(text: string): boolean {
  if (typeof document === 'undefined') return false
  const ta = document.createElement('textarea')
  ta.value = text
  ta.setAttribute('readonly', '')
  // Keep it in the layout but invisible and inert.
  ta.style.position = 'fixed'
  ta.style.top = '0'
  ta.style.left = '0'
  ta.style.width = '1px'
  ta.style.height = '1px'
  ta.style.padding = '0'
  ta.style.border = 'none'
  ta.style.outline = 'none'
  ta.style.boxShadow = 'none'
  ta.style.background = 'transparent'
  ta.style.opacity = '0'
  document.body.appendChild(ta)

  // Preserve whatever the user had selected so a programmatic copy
  // doesn't clobber their selection.
  const selection = document.getSelection()
  const saved =
    selection && selection.rangeCount > 0 ? selection.getRangeAt(0) : null

  let ok = false
  try {
    ta.focus()
    ta.select()
    ta.setSelectionRange(0, text.length)
    ok = document.execCommand('copy')
  } catch {
    ok = false
  } finally {
    document.body.removeChild(ta)
    if (saved && selection) {
      selection.removeAllRanges()
      selection.addRange(saved)
    }
  }
  return ok
}
