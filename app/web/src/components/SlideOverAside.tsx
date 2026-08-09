import { useEffect, type ReactNode } from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'

import { cn } from '@/lib/utils'

interface SlideOverAsideProps {
  /** Which edge the panel is docked to. */
  side: 'left' | 'right'
  /**
   * When false the panel renders inline as a normal flex column — the
   * desktop layout. When true it becomes an overlay drawer.
   */
  compact: boolean
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Accessible name for the edge handle that pulls the panel in. */
  label: string
  /**
   * Hide the edge handle. Use when the caller already offers another way
   * to open the panel (e.g. a toolbar button) and a permanent handle
   * floating over the content would just be in the way.
   */
  hideHandle?: boolean
  children: ReactNode
}

/**
 * A side panel that is an inline column on wide screens and a slide-over
 * drawer on narrow ones.
 *
 * Positioning is `absolute`, not `fixed`: the drawer must stay inside the
 * page area so the app topbar — which carries the nav toggle — is never
 * covered by a page-level panel. **The nearest positioned ancestor must
 * therefore be the page/shell container**, i.e. give it `relative`.
 */
export function SlideOverAside({
  side,
  compact,
  open,
  onOpenChange,
  label,
  hideHandle,
  children,
}: SlideOverAsideProps) {
  // Escape closes the drawer. Only while it is actually an overlay —
  // inline columns are not dismissible, and swallowing Escape there
  // would steal it from dialogs and the terminal.
  useEffect(() => {
    if (!compact || !open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onOpenChange(false)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [compact, open, onOpenChange])

  if (!compact) return open ? <>{children}</> : null

  const closedTransform =
    side === 'left' ? '-translate-x-full' : 'translate-x-full'
  const HandleIcon = side === 'left' ? ChevronRight : ChevronLeft

  return (
    <>
      <div
        className={cn(
          // The panels inside are sized for desktop (w-60…w-80) and are
          // `shrink-0`, so on a narrow phone they would cover the whole
          // screen and leave no backdrop to tap. max-w-full on the child
          // clamps them regardless of flex-shrink, keeping a strip of
          // backdrop visible as the "tap here to dismiss" affordance.
          'absolute inset-y-0 z-50 flex max-w-[85%] [&>*]:max-w-full transition-transform duration-200 ease-out',
          side === 'left' ? 'left-0' : 'right-0',
          open ? 'translate-x-0' : closedTransform,
        )}
        // Keep the closed drawer out of the tab order and off screen
        // readers — it is visually gone, so it must be gone for keyboard
        // and AT users too.
        inert={!open ? true : undefined}
      >
        {children}
      </div>

      {open && (
        <div
          className="absolute inset-0 z-40 bg-black/50"
          onClick={() => onOpenChange(false)}
          aria-hidden
        />
      )}

      {!open && !hideHandle && (
        <button
          type="button"
          onClick={() => onOpenChange(true)}
          aria-label={label}
          className={cn(
            // Opaque and accent-tinted, not a translucent hairline: the
            // first version read as part of the page background and
            // people simply did not find it. Also a 44px-tall,
            // 24px-wide target — the old 20px strip was below the
            // comfortable touch minimum.
            'absolute top-1/2 -translate-y-1/2 z-30 h-14 w-6 border border-accent/40 bg-card text-accent flex items-center justify-center shadow-md active:bg-accent active:text-accent-foreground',
            side === 'left'
              ? 'left-0 rounded-r-md border-l-0'
              : 'right-0 rounded-l-md border-r-0',
          )}
        >
          <HandleIcon className="size-4" />
        </button>
      )}
    </>
  )
}
