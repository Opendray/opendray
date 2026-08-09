import { Paperclip, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'


export interface AttachmentItem {
  path: string
  name: string
}

// A staging tray of pending image attachments, anchored to the bottom of
// the terminal pane. Renders nothing when empty (no layout impact on the
// xterm fit()). Esc-to-clear is owned by Terminal.tsx; this is presentational.
export function AttachmentTray({
  items,
  onRemove,
  onClear,
}: {
  items: AttachmentItem[]
  onRemove: (index: number) => void
  onClear: () => void
}) {
  const { t } = useTranslation()
  if (items.length === 0) return null
  return (
    <div className="absolute inset-x-0 bottom-0 z-10 flex items-center gap-2 overflow-x-auto border-t border-border bg-card/95 px-2 py-1.5 backdrop-blur">
      <div className="flex min-w-0 items-center gap-1.5">
        {items.map((item, i) => (
          <span
            key={`${item.path}-${i}`}
            className="flex max-w-[180px] items-center gap-1 rounded-md border border-border bg-background px-1.5 py-0.5 text-[11px]"
          >
            <Paperclip className="size-3 shrink-0 text-muted-foreground" />
            {/* The position, not the filename: this is the label the
                agent's input line will carry, and a chip reading
                `screenshot 2026-08-09 at 14.03.11.png` is no more use
                than the path was. The real name stays in the tooltip. */}
            <span className="truncate font-mono" title={item.name}>
              [image{i + 1}]
            </span>
            <button
              type="button"
              onClick={() => onRemove(i)}
              aria-label={t('web.sessions.terminal.attachRemove', {
                name: item.name,
              })}
              className="shrink-0 rounded p-0.5 text-muted-foreground hover:bg-card hover:text-foreground"
            >
              <X className="size-3" />
            </button>
          </span>
        ))}
      </div>
      <div className="ml-auto flex shrink-0 items-center gap-2">
        {/* There is no "insert" any more — the paths go in with the
            message. Saying so is the whole affordance. */}
        <span className="text-[11px] text-muted-foreground/70">
          {t('web.sessions.terminal.attachOnSend')}
        </span>
        <button
          type="button"
          onClick={onClear}
          className="rounded-md px-2 py-0.5 text-[11px] text-muted-foreground hover:bg-background hover:text-foreground"
        >
          {t('web.sessions.terminal.attachClear')}
        </button>
      </div>
    </div>
  )
}
