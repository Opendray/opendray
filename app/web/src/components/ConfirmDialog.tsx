// ConfirmDialog — drop-in replacement for the browser's native
// `confirm()` modal.
//
// Why: Chrome (and others) attach a "Prevent this page from creating
// additional dialogs" checkbox to native confirm() / alert() popups.
// Once the operator ticks that box (often by accident, hunting for
// Cancel), every subsequent confirm() call silently returns false and
// the dialog renders for a frame and vanishes — making destructive
// actions like Remove Session look like "the button is broken." The
// flag is per-browsing-context and can't be unset from the page.
//
// useConfirmDialog() returns { confirm, dialog } where:
//   - confirm({ title, description, confirmLabel, destructive })
//     returns a Promise<boolean> the caller awaits.
//   - dialog is the JSX the caller renders once (anywhere in their
//     tree).
//
// Internally a single Radix-backed Dialog instance is shown/hidden
// based on the most recent confirm() call; the in-flight resolver is
// stashed so OK / Cancel can fulfil the right Promise.

import { useCallback, useRef, useState } from 'react'

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'

export interface ConfirmOptions {
  title: string
  description?: string
  confirmLabel?: string
  cancelLabel?: string
  // When true, paint the confirm button in destructive red. Use for
  // delete/remove flows; leave false (default) for benign confirms.
  destructive?: boolean
}

interface ConfirmState extends ConfirmOptions {
  open: boolean
}

const INITIAL: ConfirmState = { open: false, title: '' }

export function useConfirmDialog() {
  const [state, setState] = useState<ConfirmState>(INITIAL)
  // Stash the resolver from the latest pending Promise so OK / Cancel
  // can fulfil it from the dialog buttons.
  const resolverRef = useRef<((ok: boolean) => void) | null>(null)

  const confirm = useCallback(
    (opts: ConfirmOptions) =>
      new Promise<boolean>((resolve) => {
        resolverRef.current = resolve
        // Defer the state update past the current click event. Without
        // this, Radix Dialog mounts mid-bubble and its
        // onInteractOutside / onPointerDownOutside handler sees the
        // still-bubbling click (which originally hit the Remove
        // button) as a click outside the dialog and closes it the
        // same frame it opened — the operator sees a flash and the
        // session never gets removed. queueMicrotask runs after the
        // click finishes propagating but before the next paint, so
        // there's no visible delay.
        queueMicrotask(() => setState({ ...opts, open: true }))
      }),
    [],
  )

  const settle = useCallback((ok: boolean) => {
    const resolver = resolverRef.current
    resolverRef.current = null
    setState((s) => ({ ...s, open: false }))
    resolver?.(ok)
  }, [])

  const dialog = (
    <Dialog
      open={state.open}
      onOpenChange={(open) => {
        if (!open) settle(false)
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{state.title}</DialogTitle>
          {state.description && (
            <DialogDescription>{state.description}</DialogDescription>
          )}
        </DialogHeader>
        <DialogFooter>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => settle(false)}
            autoFocus
          >
            {state.cancelLabel ?? 'Cancel'}
          </Button>
          <Button
            variant={state.destructive ? 'destructive' : 'accent'}
            size="sm"
            onClick={() => settle(true)}
          >
            {state.confirmLabel ?? 'Confirm'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )

  return { confirm, dialog }
}
