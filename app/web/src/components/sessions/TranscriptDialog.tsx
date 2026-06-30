import { useCallback, useEffect, useRef, useState } from 'react'
import { Copy, CopyCheck, RefreshCw, ScrollText } from 'lucide-react'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { api, APIError } from '@/lib/api'
import { copyText } from '@/lib/clipboard'

interface TranscriptDialogProps {
  sessionId: string | null
  open: boolean
  onOpenChange: (v: boolean) => void
}

// Strip terminal control sequences so the ring-buffer bytes read as
// plain text. The grok TUI ignores wheel input in its alt-screen, so
// users have no way to scroll its own viewport — this dialog gives them
// a flat transcript of everything the PTY has emitted, regardless of
// whether the CLI implements scrollback.
//
// We keep the regex narrow on purpose: CSI, OSC, single-shift, charset
// designators, and bare C1 escapes cover what real CLIs emit. We do NOT
// try to fold redraw frames into a deduplicated chat history — a TUI
// that home-and-rewrites each tick will repeat lines in the output.
// That's still vastly more useful than no scroll at all, and a future
// pass can layer per-provider parsing (grok chat_history.jsonl etc.)
// on top of the same dialog.
/* eslint-disable no-control-regex */
function stripAnsi(input: string): string {
  return input
    .replace(/\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)/g, '') // OSC ... BEL | ST
    .replace(/\x1b\[[\x30-\x3f]*[\x20-\x2f]*[\x40-\x7e]/g, '') // CSI
    .replace(/\x1b[PX^_][^\x1b]*\x1b\\/g, '') // DCS / SOS / PM / APC
    .replace(/\x1b[()*+\-./][\x20-\x2f]*[A-Za-z0-9]/g, '') // charset designators
    .replace(/\x1b[=>78cDEHMNOZ]/g, '') // common 2-byte controls
    .replace(/\x1b/g, '') // any lone ESC the patterns above missed
    .replace(/[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]/g, '') // strip remaining C0 (keep \t\n\r)
}
/* eslint-enable no-control-regex */

export function TranscriptDialog({
  sessionId,
  open,
  onOpenChange,
}: TranscriptDialogProps) {
  const { t } = useTranslation()
  const [text, setText] = useState<string>('')
  const [loading, setLoading] = useState(false)
  const [errored, setErrored] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)

  const fetchBuffer = useCallback(async () => {
    if (!sessionId) return
    setLoading(true)
    setErrored(false)
    try {
      const res = await api(`/api/v1/sessions/${sessionId}/buffer`, {
        raw: true,
      })
      if (!(res instanceof Response)) throw new Error('not a Response')
      const buf = await res.arrayBuffer()
      // The ring buffer can carry mid-character UTF-8 splits at the
      // wrap boundary; { fatal: false } lets the decoder emit U+FFFD
      // for the byte that got chopped rather than throwing.
      const decoded = new TextDecoder('utf-8', { fatal: false }).decode(buf)
      setText(stripAnsi(decoded))
    } catch (err) {
      setErrored(true)
      const msg =
        err instanceof APIError ? `HTTP ${err.status}` : (err as Error).message
      toast.error(t('web.sessions.terminal.transcriptFetchFailed'), {
        description: msg,
      })
    } finally {
      setLoading(false)
    }
  }, [sessionId, t])

  useEffect(() => {
    if (open && sessionId) {
      // The "open" transition genuinely needs to set state inside the
      // effect: the dialog is the external system whose lifecycle drives
      // this fetch, and fetchBuffer guards its own updates.
      // eslint-disable-next-line react-hooks/set-state-in-effect
      void fetchBuffer()
    } else if (!open) {
      // Drop the text so reopening for a different session doesn't
      // flash stale content while the new fetch is in flight.
      setText('')
      setErrored(false)
    }
  }, [open, sessionId, fetchBuffer])

  // Scroll to bottom after a refresh so the user lands on the latest
  // output, matching the mental model of "viewing the live tail".
  useEffect(() => {
    if (!open || loading) return
    const viewport = scrollRef.current?.querySelector(
      '[data-radix-scroll-area-viewport]',
    ) as HTMLElement | null
    if (viewport) viewport.scrollTop = viewport.scrollHeight
  }, [text, open, loading])

  const copySelection = async () => {
    const sel = window.getSelection()?.toString() ?? ''
    if (!sel.trim()) {
      toast(t('web.sessions.terminal.selectCopyNoSelection'))
      return
    }
    if (await copyText(sel)) {
      toast.success(t('web.sessions.terminal.copiedToast'))
    } else {
      toast.error(t('web.sessions.terminal.copyFailedToast'))
    }
  }

  const copyAll = async () => {
    if (!text.trim()) {
      toast(t('web.sessions.terminal.transcriptEmpty'))
      return
    }
    if (await copyText(text)) {
      toast.success(t('web.sessions.terminal.copiedToast'))
    } else {
      toast.error(t('web.sessions.terminal.copyFailedToast'))
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[min(92vw,1100px)] w-[min(92vw,1100px)] h-[min(85vh,800px)] gap-2 flex flex-col">
        <DialogHeader className="shrink-0">
          <div className="flex items-start gap-2 min-w-0">
            <ScrollText className="size-4 mt-0.5 text-muted-foreground shrink-0" />
            <div className="flex flex-col min-w-0 flex-1">
              <DialogTitle>
                {t('web.sessions.terminal.transcriptTitle')}
              </DialogTitle>
              <span className="text-[11px] text-muted-foreground/70">
                {t('web.sessions.terminal.transcriptDesc')}
              </span>
            </div>
          </div>
        </DialogHeader>

        <div className="rounded-md border border-border bg-card/30 overflow-hidden flex-1 min-h-0 flex">
          <ScrollArea ref={scrollRef} className="flex-1">
            <pre className="select-text whitespace-pre-wrap break-words font-mono text-[12px] leading-[1.55] p-3 text-foreground/90">
              {loading
                ? t('web.sessions.terminal.transcriptLoading')
                : errored
                  ? t('web.sessions.terminal.transcriptFetchFailed')
                  : text.trim()
                    ? text
                    : t('web.sessions.terminal.transcriptEmpty')}
            </pre>
          </ScrollArea>
        </div>

        <div className="flex items-center justify-end gap-2">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => void fetchBuffer()}
            disabled={loading || !sessionId}
            className="text-[11px] gap-1.5"
          >
            <RefreshCw className={`size-3 ${loading ? 'animate-spin' : ''}`} />
            {t('web.sessions.terminal.transcriptRefresh')}
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={copySelection}
            className="text-[11px] gap-1.5"
          >
            <Copy className="size-3" />
            {t('web.sessions.terminal.selectCopyCopySelection')}
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={copyAll}
            className="text-[11px] gap-1.5"
          >
            <CopyCheck className="size-3" />
            {t('web.sessions.terminal.selectCopyCopyAll')}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => onOpenChange(false)}
            className="text-[11px]"
          >
            {t('common.close')}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
