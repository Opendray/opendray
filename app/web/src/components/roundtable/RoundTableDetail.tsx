import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Loader2, Send, Sparkles, Trash2, Users } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import {
  closeRoundTable,
  deleteRoundTable,
  getRoundTable,
  postMessage,
  summarizeRoundTable,
  type Message,
  type SeatProvider,
} from '@/lib/roundtable'

// Per-vendor accent for chat bubbles — the cross-vendor identity at a glance.
const SEAT_BUBBLE: Record<string, string> = {
  claude: 'border-orange-500/30 bg-orange-500/10',
  codex: 'border-emerald-500/30 bg-emerald-500/10',
  antigravity: 'border-sky-500/30 bg-sky-500/10',
}

export function RoundTableDetail({
  id,
  onDeleted,
}: {
  id: string
  onDeleted?: () => void
}) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [draft, setDraft] = useState('')
  const scrollRef = useRef<HTMLDivElement>(null)

  const query = useQuery({
    queryKey: ['round-table', id],
    queryFn: () => getRoundTable(id),
    // A group chat feels live: poll while the chat is active. Members reply
    // asynchronously, so this is how their messages appear.
    refetchInterval: (q) =>
      q.state.data?.round_table.status === 'active' ? 3000 : false,
  })

  const messages = query.data?.messages ?? []

  // Auto-scroll to the newest message.
  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight })
  }, [messages.length])

  const send = useMutation({
    mutationFn: (content: string) => postMessage(id, content),
    onSuccess: () => {
      setDraft('')
      qc.invalidateQueries({ queryKey: ['round-table', id] })
      qc.invalidateQueries({ queryKey: ['round-tables'] })
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const summarize = useMutation({
    mutationFn: () => summarizeRoundTable(id),
    onSuccess: () => {
      toast.success(t('web.roundTable.detail.summarizing'))
      qc.invalidateQueries({ queryKey: ['round-table', id] })
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const close = useMutation({
    mutationFn: () => closeRoundTable(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['round-table', id] })
      qc.invalidateQueries({ queryKey: ['round-tables'] })
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const remove = useMutation({
    mutationFn: () => deleteRoundTable(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['round-tables'] })
      onDeleted?.()
    },
    onError: (e: Error) => toast.error(e.message),
  })

  if (query.isLoading) {
    return (
      <div className="flex items-center gap-2 p-6 text-sm text-muted-foreground">
        <Loader2 className="size-4 animate-spin" />
        {t('web.roundTable.detail.loading')}
      </div>
    )
  }
  if (query.isError || !query.data) {
    return (
      <div className="p-6 text-sm text-state-failed">
        {t('web.roundTable.detail.loadFailed')}
      </div>
    )
  }

  const { round_table: rt } = query.data
  const closed = rt.status === 'closed'

  // Still awaiting replies? The last message is an operator turn whose
  // @mentions haven't all answered yet.
  const awaiting = computeAwaiting(messages)

  const insertMention = (token: string) =>
    setDraft((d) => (d ? `${d.trimEnd()} @${token} ` : `@${token} `))

  const submit = () => {
    const content = draft.trim()
    if (content && !send.isPending) send.mutate(content)
  }

  return (
    <div className="flex h-full min-h-0 flex-col">
      {/* Header */}
      <div className="flex items-start justify-between gap-4 pb-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <h2 className="truncate text-sm font-medium">
              {rt.topic || t('web.roundTable.untitled')}
            </h2>
            <Badge variant={closed ? 'outline' : 'success'}>
              {t(`web.roundTable.status.${rt.status}`)}
            </Badge>
          </div>
          <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
            {rt.seats.map((s) => (
              <Badge key={s.provider} variant="muted" className="capitalize">
                {s.provider}
              </Badge>
            ))}
            {rt.cwd && (
              <span className="truncate font-mono text-[11px] text-muted-foreground">
                {rt.cwd}
              </span>
            )}
          </div>
        </div>
        <div className="flex shrink-0 gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={summarize.isPending || messages.length === 0}
            onClick={() => summarize.mutate()}
          >
            <Sparkles className="size-3.5" />
            {t('web.roundTable.detail.summarize')}
          </Button>
          {!closed && (
            <Button
              variant="ghost"
              size="sm"
              disabled={close.isPending}
              onClick={() => close.mutate()}
            >
              {t('web.roundTable.detail.close')}
            </Button>
          )}
          <Button
            variant="ghost"
            size="sm"
            disabled={remove.isPending}
            onClick={() => {
              if (window.confirm(t('web.roundTable.detail.deleteConfirm'))) {
                remove.mutate()
              }
            }}
            title={t('web.roundTable.detail.delete')}
          >
            <Trash2 className="size-3.5" />
          </Button>
        </div>
      </div>

      {/* Messages */}
      <div
        ref={scrollRef}
        className="flex-1 min-h-0 overflow-y-auto rounded-lg border border-border bg-card/20 p-3"
      >
        {messages.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center gap-2 text-center text-sm text-muted-foreground">
            <Users className="size-6" />
            <div>{t('web.roundTable.detail.emptyThread')}</div>
            <div className="text-xs">{t('web.roundTable.detail.emptyHint')}</div>
          </div>
        ) : (
          <div className="flex flex-col gap-2.5">
            {messages.map((m) => (
              <MessageBubble key={m.id} m={m} />
            ))}
            {awaiting && (
              <div className="flex items-center gap-2 px-1 text-xs text-muted-foreground">
                <Loader2 className="size-3.5 animate-spin" />
                {t('web.roundTable.detail.replying')}
              </div>
            )}
          </div>
        )}
      </div>

      {/* Composer */}
      {!closed && (
        <div className="pt-3">
          <div className="mb-1.5 flex flex-wrap items-center gap-1.5">
            <span className="text-[11px] text-muted-foreground">
              {t('web.roundTable.detail.mentionHint')}
            </span>
            {rt.seats.map((s) => (
              <button
                key={s.provider}
                type="button"
                onClick={() => insertMention(s.provider)}
                className="rounded-full border border-border bg-card px-2 py-0.5 text-[11px] text-muted-foreground transition-colors hover:text-foreground"
              >
                @{s.provider}
              </button>
            ))}
            {rt.seats.length > 1 && (
              <button
                type="button"
                onClick={() => insertMention('all')}
                className="rounded-full border border-border bg-card px-2 py-0.5 text-[11px] text-muted-foreground transition-colors hover:text-foreground"
              >
                @all
              </button>
            )}
          </div>
          <div className="flex items-end gap-2">
            <textarea
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
                  e.preventDefault()
                  submit()
                }
              }}
              rows={2}
              placeholder={t('web.roundTable.detail.composerPlaceholder')}
              className="flex-1 resize-none rounded-md border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
            />
            <Button
              size="sm"
              disabled={!draft.trim() || send.isPending}
              onClick={submit}
            >
              {send.isPending ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <Send className="size-3.5" />
              )}
              {t('web.roundTable.detail.send')}
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}

function MessageBubble({ m }: { m: Message }) {
  const { t } = useTranslation()
  const isOperator = m.role === 'operator'
  const isSystem = m.role === 'system'
  const isSummary = m.kind === 'summary'

  if (isSystem) {
    return (
      <div className="mx-auto max-w-[85%] rounded-md border border-state-failed/30 bg-state-failed/5 px-3 py-1.5 text-center text-[12px] text-muted-foreground">
        {m.content}
      </div>
    )
  }

  return (
    <div className={cn('flex', isOperator ? 'justify-end' : 'justify-start')}>
      <div
        className={cn(
          'max-w-[85%] rounded-lg border px-3 py-2',
          isOperator
            ? 'border-primary/30 bg-primary/10'
            : isSummary
              ? 'border-accent/40 bg-accent/10'
              : (SEAT_BUBBLE[m.seat_provider ?? ''] ??
                'border-border bg-card'),
        )}
      >
        <div className="mb-0.5 flex items-center gap-1.5">
          <span className="text-[11px] font-medium capitalize">
            {isOperator ? t('web.roundTable.you') : m.seat_provider}
          </span>
          {isSummary && (
            <Badge variant="accent">{t('web.roundTable.summary')}</Badge>
          )}
        </div>
        <p className="whitespace-pre-wrap text-[13px] leading-relaxed">
          {m.content}
        </p>
      </div>
    </div>
  )
}

// computeAwaiting reports whether the last operator message still has
// @mentioned members that haven't replied yet (so the UI shows a typing
// indicator through the sequential round).
function computeAwaiting(messages: Message[]): boolean {
  let lastOp = -1
  for (let i = messages.length - 1; i >= 0; i--) {
    if (messages[i].role === 'operator') {
      lastOp = i
      break
    }
  }
  if (lastOp < 0) return false
  const mentions = messages[lastOp].mentions ?? []
  if (mentions.length === 0) return false
  const replied = new Set<string>()
  for (let i = lastOp + 1; i < messages.length; i++) {
    const m = messages[i]
    if ((m.role === 'seat' || m.role === 'system') && m.seat_provider) {
      replied.add(m.seat_provider)
    }
  }
  return mentions.some((p: string) => !replied.has(p as SeatProvider))
}
