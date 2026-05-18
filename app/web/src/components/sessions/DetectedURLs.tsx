/**
 * DetectedURLs — floating "links" badge above the xterm terminal.
 *
 * Two render paths so the common-case auth flow (one URL, one tap)
 * doesn't get gated behind a dialog:
 *
 *   N = 1 URL : the badge is itself an `<a target="_blank">` — one
 *               tap and the OAuth URL opens in the browser. No
 *               dialog, no "Open" button to find. This is what
 *               most AI-CLI auth flows produce, and the badge
 *               existed mainly to rescue it.
 *
 *   N ≥ 2 URLs: the badge opens a dialog listing every URL with
 *               its own `<a target="_blank">` Open button and a
 *               clipboard Copy fallback. Same UI as before.
 *
 * Both render paths use real `<a target="_blank">` anchors instead
 * of `window.open(url)` from a click handler — that matters on
 * mobile Safari and some popup-blocker configs where button-driven
 * window.open gets gated and silently no-ops.
 */

import { useState } from 'react'
import { ExternalLink, Link2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from 'shared-ui/primitives/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from 'shared-ui/primitives/dialog'
import { ScrollArea } from 'shared-ui/primitives/scroll-area'

interface DetectedURLsProps {
  urls: string[]
}

const BADGE_BASE_CLASS =
  'absolute top-2 right-2 z-10 inline-flex h-8 items-center gap-1.5 ' +
  'rounded-md border border-border bg-secondary px-2.5 text-xs ' +
  'font-medium text-secondary-foreground shadow-sm transition-colors ' +
  'hover:bg-secondary/80 focus-visible:outline-none focus-visible:ring-2 ' +
  'focus-visible:ring-ring'

export function DetectedURLs({ urls }: DetectedURLsProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)

  if (urls.length === 0) return null

  const count = urls.length
  const buttonKey =
    count === 1
      ? 'web.sessions.terminal.urls.buttonLabel'
      : 'web.sessions.terminal.urls.buttonLabel_plural'
  const buttonText = t(buttonKey, { count })

  // ── One URL: badge IS the link. One tap → browser. ────────────────
  if (count === 1) {
    return (
      <a
        href={urls[0]}
        target="_blank"
        rel="noopener noreferrer"
        className={BADGE_BASE_CLASS}
        title={t('web.sessions.terminal.urls.tooltip')}
        aria-label={t('web.sessions.terminal.urls.tooltip')}
      >
        <ExternalLink className="size-3.5" />
        {buttonText}
      </a>
    )
  }

  // ── Multiple URLs: badge opens dialog so user can pick. ───────────
  const handleCopy = async (url: string) => {
    try {
      await navigator.clipboard.writeText(url)
      toast.success(t('web.sessions.terminal.urls.copiedToast'))
    } catch {
      toast.error(t('web.sessions.terminal.urls.copyFailedToast'))
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button
          size="sm"
          variant="secondary"
          className="absolute top-2 right-2 z-10 h-8 gap-1.5 px-2.5 shadow-sm"
          title={t('web.sessions.terminal.urls.tooltip')}
          aria-label={t('web.sessions.terminal.urls.tooltip')}
        >
          <Link2 className="size-3.5" />
          <span className="text-xs font-medium">{buttonText}</span>
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-[min(640px,95vw)]">
        <DialogHeader>
          <DialogTitle>
            {t('web.sessions.terminal.urls.dialogTitle')}
          </DialogTitle>
          <DialogDescription>
            {t('web.sessions.terminal.urls.dialogDesc')}
          </DialogDescription>
        </DialogHeader>
        <ScrollArea className="max-h-[60vh]">
          <div className="space-y-2 pr-3">
            {urls
              .slice()
              .reverse()
              .map((url) => (
                <div
                  key={url}
                  className="rounded-md border bg-muted/30 p-3"
                >
                  <div className="mb-2 select-all break-all font-mono text-xs">
                    {url}
                  </div>
                  <div className="flex gap-2">
                    <a
                      href={url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex h-9 flex-1 items-center justify-center gap-1.5 rounded-md bg-primary text-sm font-medium text-primary-foreground shadow-sm hover:bg-primary/90"
                      onClick={() => setOpen(false)}
                    >
                      <ExternalLink className="size-3.5" />
                      {t('web.sessions.terminal.urls.openButton')}
                    </a>
                    <Button
                      size="sm"
                      variant="outline"
                      className="flex-1"
                      onClick={() => handleCopy(url)}
                    >
                      {t('web.sessions.terminal.urls.copyButton')}
                    </Button>
                  </div>
                </div>
              ))}
          </div>
        </ScrollArea>
      </DialogContent>
    </Dialog>
  )
}
