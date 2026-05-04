import { useState } from 'react'
import { Copy, AlertTriangle, Check, X } from 'lucide-react'

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'

interface APIKeyRevealDialogProps {
  open: boolean
  apiKey: string
  title?: string
  description?: string
  onClose: () => void
}

/**
 * One-time API-key reveal.
 *
 * The plaintext key is shown once after registration or rotation.
 * Two ways out:
 *
 *   1. **Done** — the operator copied the key (Copy button auto-
 *      acknowledges, or they tick the "I copied it" box) and is
 *      ready to update every consumer app.
 *   2. **Discard** (X) — the operator changed their mind. The new
 *      key is thrown away; `onClose` still fires so the UI returns
 *      to the integrations list. The key was ALREADY rotated on the
 *      server side, so the previous key is invalid either way.
 */
export function APIKeyRevealDialog({
  open,
  apiKey,
  title = 'API key issued',
  description = 'This is the only time the plaintext key will be shown. Copy it now and update every consumer app — the previous key (if any) no longer authenticates.',
  onClose,
}: APIKeyRevealDialogProps) {
  const [acknowledged, setAcknowledged] = useState(false)
  const [copied, setCopied] = useState(false)

  const reset = () => {
    setAcknowledged(false)
    setCopied(false)
  }

  const close = () => {
    reset()
    onClose()
  }

  const discard = () => {
    if (
      !window.confirm(
        'Discard the new key? Rotation has already invalidated the old key — discarding means you have NO working key for this integration until you rotate again.',
      )
    )
      return
    close()
  }

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(apiKey)
      setCopied(true)
      // Successful copy implies the operator has the key. Auto-ack
      // so they don't also need to click the checkbox.
      setAcknowledged(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      // Clipboard blocked — they can still select+copy from the
      // input. Leave acknowledged untouched in that case.
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        // Backdrop / ESC closes only when acknowledged. The
        // explicit Discard button is the escape hatch otherwise.
        if (!v && acknowledged) close()
      }}
    >
      <DialogContent
        className="max-w-[520px]"
        hideClose
        onEscapeKeyDown={(e) => {
          if (!acknowledged) e.preventDefault()
        }}
        onInteractOutside={(e) => {
          if (!acknowledged) e.preventDefault()
        }}
      >
        {/* Manual close button so the operator always has a way out
            even before they decide whether to keep the key. */}
        <button
          type="button"
          onClick={discard}
          className="absolute right-3 top-3 rounded-sm p-1 text-muted-foreground/70 hover:text-foreground hover:bg-muted/40 transition-colors"
          aria-label="Discard new key"
          title="Discard the new key (rotation already happened — old key is gone too)"
        >
          <X className="size-4" />
        </button>

        <DialogHeader>
          <div className="flex items-center gap-2">
            <AlertTriangle className="size-4 text-state-idle" />
            <DialogTitle>{title}</DialogTitle>
          </div>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>

        <div className="flex items-center gap-2 mt-3">
          <input
            readOnly
            value={apiKey}
            onFocus={(e) => e.currentTarget.select()}
            className="flex-1 font-mono text-[12px] bg-input/40 border border-border rounded-md h-9 px-3"
          />
          <Button variant="outline" size="sm" onClick={copy}>
            {copied ? (
              <>
                <Check className="size-3.5" />
                Copied
              </>
            ) : (
              <>
                <Copy className="size-3.5" />
                Copy
              </>
            )}
          </Button>
        </div>

        <p className="text-[11px] text-muted-foreground/80 mt-3 leading-snug">
          <strong>Update every consumer app with this new key.</strong>
          {' '}The previous key has been invalidated server-side and
          will return <code>401 unauthorized</code> on the next
          request.
        </p>

        <label className="flex items-start gap-2 mt-3 text-[12px] text-muted-foreground cursor-pointer">
          <input
            type="checkbox"
            checked={acknowledged}
            onChange={(e) => setAcknowledged(e.target.checked)}
            className="mt-0.5 accent-accent"
          />
          <span>
            I have copied the key and will update my consumer apps. I
            understand opendray will not display it again.
          </span>
        </label>

        <DialogFooter>
          <Button
            variant="ghost"
            size="sm"
            onClick={discard}
            className="text-muted-foreground hover:text-destructive"
          >
            Discard
          </Button>
          <Button
            variant="accent"
            size="sm"
            disabled={!acknowledged}
            onClick={close}
          >
            Done
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
