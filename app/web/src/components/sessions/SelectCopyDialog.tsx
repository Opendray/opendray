import { useRef } from 'react'
import { Copy, CopyCheck, TextCursorInput } from 'lucide-react'
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
import { copyText } from '@/lib/clipboard'

interface SelectCopyDialogProps {
  text: string
  open: boolean
  onOpenChange: (v: boolean) => void
}

// SelectCopyDialog renders the terminal output as native, selectable
// DOM text. The live terminal is an xterm <canvas>: on touch devices a
// finger drag can't select canvas text, and while a TUI has mouse
// tracking on (Claude Code / Codex / Antigravity all do), pointer gestures
// are forwarded to the program as mouse events instead of forming a
// selection. Reconstructing the buffer into a <pre> with user-select
// sidesteps both — the OS's own selection (drag on desktop, long-press
// handles on touch) works, so the operator can grab any portion (a
// command, a URL) and copy just that, identically on web and mobile.
export function SelectCopyDialog({
  text,
  open,
  onOpenChange,
}: SelectCopyDialogProps) {
  const { t } = useTranslation()
  const preRef = useRef<HTMLPreElement>(null)

  const copySelection = async () => {
    // Whatever the user highlighted inside the dialog. The <pre> is the
    // only selectable text on screen while the modal is up, so reading
    // the document selection is enough — no need to range-check it
    // against the node.
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
      toast(t('web.sessions.terminal.selectCopyEmpty'))
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
            <TextCursorInput className="size-4 mt-0.5 text-muted-foreground shrink-0" />
            <div className="flex flex-col min-w-0 flex-1">
              <DialogTitle>{t('web.sessions.terminal.selectCopyTitle')}</DialogTitle>
              <span className="text-[11px] text-muted-foreground/70">
                {t('web.sessions.terminal.selectCopyDesc')}
              </span>
            </div>
          </div>
        </DialogHeader>

        <div className="rounded-md border border-border bg-card/30 overflow-hidden flex-1 min-h-0 flex">
          <ScrollArea className="flex-1">
            <pre
              ref={preRef}
              // select-text re-enables selection (xterm's wrapper sets
              // user-select:none in places); whitespace-pre-wrap keeps
              // long lines readable without a horizontal scrollbar.
              className="select-text whitespace-pre-wrap break-words font-mono text-[12px] leading-[1.55] p-3 text-foreground/90"
            >
              {text || ' '}
            </pre>
          </ScrollArea>
        </div>

        <div className="flex items-center justify-end gap-2">
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
