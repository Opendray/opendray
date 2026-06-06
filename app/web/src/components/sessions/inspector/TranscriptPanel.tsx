import { useQuery } from '@tanstack/react-query'
import { Copy, MessageSquare, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { fetchSessionTranscript } from '@/lib/sessions'
import type { Session } from '@/lib/types'
import { copyText } from '@/lib/clipboard'

// TranscriptPanel — Inspector "Conversation" tab. Renders the session's
// reconstructed conversation (user prompts + assistant prose) from the
// CLI's JSONL as a scrollable list. This is the scrollback the live
// terminal can't give: the agent CLIs run a full-screen alternate-screen
// TUI, which has no scrollback. The parent Inspector wraps this in a
// ScrollArea, so the list itself just flows.
export function TranscriptPanel({ session }: { session: Session }) {
  const { t } = useTranslation()
  const { data, isLoading, isError } = useQuery({
    queryKey: ['session-transcript', session.id],
    queryFn: () => fetchSessionTranscript(session.id),
    refetchInterval: 10_000,
  })

  const turns = data?.turns ?? []

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12 text-muted-foreground">
        <Loader2 className="size-4 animate-spin" />
      </div>
    )
  }
  if (isError) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 py-12 text-center text-xs text-muted-foreground">
        <MessageSquare className="size-6 opacity-40" />
        {t('web.sessions.inspector.transcript.loadFailed')}
      </div>
    )
  }
  if (turns.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 py-12 text-center text-xs text-muted-foreground">
        <MessageSquare className="size-6 opacity-40" />
        {t('web.sessions.inspector.transcript.empty')}
      </div>
    )
  }

  const copyAll = async () => {
    const text = turns
      .map((tn) => `${tn.role.toUpperCase()}: ${tn.text}`)
      .join('\n\n')
    await copyText(text)
    toast.success(t('web.sessions.inspector.transcript.copied'))
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between gap-2">
        <span className="text-[11px] text-muted-foreground">
          {t('web.sessions.inspector.transcript.count', { count: turns.length })}
        </span>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-6 gap-1 text-[11px]"
          onClick={copyAll}
        >
          <Copy className="size-3" />
          {t('web.sessions.inspector.transcript.copyAll')}
        </Button>
      </div>
      {turns.map((turn, i) => (
        <div key={i} className="rounded-md border border-border bg-card/30 p-2">
          <span
            className={
              turn.role === 'user'
                ? 'text-[10px] font-semibold uppercase tracking-wider text-accent'
                : 'text-[10px] font-semibold uppercase tracking-wider text-muted-foreground'
            }
          >
            {turn.role}
          </span>
          <div className="mt-1 whitespace-pre-wrap break-words text-[12px] leading-relaxed">
            {turn.text}
          </div>
        </div>
      ))}
    </div>
  )
}
