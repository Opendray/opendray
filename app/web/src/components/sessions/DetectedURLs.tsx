/**
 * DetectedURLs — floating "open the latest URL" affordance on the
 * terminal pane.
 *
 * Design:
 *   - The badge is ALWAYS a direct `<a target="_blank">` anchor to
 *     the **most recently seen** URL. AI-CLI auth flows
 *     (`claude login`, `gemini auth login`, `codex login`) emit the
 *     OAuth URL as the last thing they print, so "most recent" is
 *     correct ~100 % of the time for the use case the badge exists
 *     to rescue. One tap → browser opens the URL.
 *   - A small secondary "⋯" button next to the main link opens a
 *     dialog listing every URL seen this session, with per-row
 *     Open / Copy buttons. Used only when the operator actually
 *     wants a URL other than the latest one.
 *
 * Both the primary anchor and the per-row Open buttons in the dialog
 * are real `<a target="_blank">` anchors (not `window.open()` from
 * click handlers). Some mobile-Safari popup-blocker configs gate
 * button-driven `window.open` even from a direct click; anchor
 * navigation isn't blocked.
 */

import { useState } from 'react'
import { ExternalLink, MoreHorizontal } from 'lucide-react'
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

// The badge anchor — primary tap target, full link semantics.
const PRIMARY_ANCHOR_CLASS =
  'inline-flex h-8 items-center gap-1.5 rounded-l-md border border-r-0 ' +
  'border-border bg-secondary px-2.5 text-xs font-medium ' +
  'text-secondary-foreground shadow-sm transition-colors ' +
  'hover:bg-secondary/80 focus-visible:outline-none focus-visible:ring-2 ' +
  'focus-visible:ring-ring'

// The "⋯" button — secondary tap target, opens dialog when N ≥ 2.
const SECONDARY_BUTTON_CLASS =
  'inline-flex h-8 w-8 items-center justify-center rounded-r-md border ' +
  'border-border bg-secondary text-secondary-foreground shadow-sm ' +
  'transition-colors hover:bg-secondary/80 focus-visible:outline-none ' +
  'focus-visible:ring-2 focus-visible:ring-ring'

export function DetectedURLs({ urls }: DetectedURLsProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)

  if (urls.length === 0) return null

  const count = urls.length
  const latest = urls[urls.length - 1]
  const buttonKey =
    count === 1
      ? 'web.sessions.terminal.urls.buttonLabel'
      : 'web.sessions.terminal.urls.buttonLabel_plural'
  const buttonText = t(buttonKey, { count })

  // ── N = 1: single anchor badge ──────────────────────────────────
  if (count === 1) {
    return (
      <a
        href={latest}
        target="_blank"
        rel="noopener noreferrer"
        className={`${PRIMARY_ANCHOR_CLASS} rounded-md border-r absolute top-2 right-2 z-10`}
        title={t('web.sessions.terminal.urls.tooltip')}
        aria-label={t('web.sessions.terminal.urls.tooltip')}
      >
        <ExternalLink className="size-3.5" />
        {buttonText}
      </a>
    )
  }

  // ── N ≥ 2: primary anchor (open latest) + secondary ⋯ (open dialog) ──
  const handleCopy = async (url: string) => {
    try {
      await navigator.clipboard.writeText(url)
      toast.success(t('web.sessions.terminal.urls.copiedToast'))
    } catch {
      toast.error(t('web.sessions.terminal.urls.copyFailedToast'))
    }
  }

  return (
    <div className="absolute top-2 right-2 z-10 inline-flex">
      {/* Primary: tap → open the most recent URL (OAuth) directly */}
      <a
        href={latest}
        target="_blank"
        rel="noopener noreferrer"
        className={PRIMARY_ANCHOR_CLASS}
        title={t('web.sessions.terminal.urls.tapToOpenLatest')}
        aria-label={t('web.sessions.terminal.urls.tapToOpenLatest')}
      >
        <ExternalLink className="size-3.5" />
        {buttonText}
      </a>

      {/* Secondary: open dialog for the rare "I want an older URL" case */}
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogTrigger asChild>
          <button
            type="button"
            className={SECONDARY_BUTTON_CLASS}
            title={t('web.sessions.terminal.urls.openListTooltip')}
            aria-label={t('web.sessions.terminal.urls.openListTooltip')}
          >
            <MoreHorizontal className="size-3.5" />
          </button>
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
              {/* Reverse so newest is on top (matches the primary anchor's target) */}
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
    </div>
  )
}
