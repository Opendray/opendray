/**
 * DetectedURLs — floating "N links" badge above the xterm terminal.
 *
 * The xterm `WebLinksAddon` already makes single-line URLs clickable,
 * but OAuth URLs printed by AI CLIs (claude / codex / gemini) are
 * often 400-600 chars long and visually wrap across multiple lines.
 * Each wrap segment becomes a separate hyperlink hit-box that's
 * (a) tiny and easy to miss on touch, and (b) opens just the partial
 * fragment of the URL it covers.
 *
 * This component sidesteps the wrap problem: the parent Terminal
 * scans PTY output for URLs, dedupes them, and renders this badge
 * when at least one URL has been seen. Tapping the badge opens a
 * Dialog with each URL as a full Open/Copy pair — works the same on
 * desktop and mobile (Dialog goes near-fullscreen below ~640px).
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

export function DetectedURLs({ urls }: DetectedURLsProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)

  if (urls.length === 0) return null

  const handleOpen = (url: string) => {
    // `noopener,noreferrer` so the opened tab can't reach back into
    // the admin SPA via window.opener — defence-in-depth even though
    // the URL is one the operator literally just saw printed in
    // their own session.
    window.open(url, '_blank', 'noopener,noreferrer')
  }

  const handleCopy = async (url: string) => {
    try {
      await navigator.clipboard.writeText(url)
      toast.success(t('web.sessions.terminal.urls.copiedToast'))
    } catch {
      // Some mobile browsers gate clipboard.writeText to https / secure
      // contexts — if it throws, ask the user to long-press the URL
      // in the dialog body (which has `select-all` for that purpose).
      toast.error(t('web.sessions.terminal.urls.copyFailedToast'))
    }
  }

  const count = urls.length
  const buttonKey =
    count === 1
      ? 'web.sessions.terminal.urls.buttonLabel'
      : 'web.sessions.terminal.urls.buttonLabel_plural'

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
          <span className="text-xs font-medium">
            {t(buttonKey, { count })}
          </span>
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
            {/* Reverse so the most recently seen URL is on top — that's
             * almost always the one the operator is trying to act on.
             * (Newest-first matches the auth flow: idle → CLI prints
             * URL → user looks at the dialog → wants the one just
             * printed.) */}
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
                    <Button
                      size="sm"
                      className="flex-1"
                      onClick={() => handleOpen(url)}
                    >
                      <ExternalLink className="mr-1.5 size-3.5" />
                      {t('web.sessions.terminal.urls.openButton')}
                    </Button>
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

// URL extraction lives in `./url-extractor.ts` so this file remains
// a clean component module (react-refresh expects that).
