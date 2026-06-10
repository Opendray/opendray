// CurationChat — the conversational maintenance channel (Cortex Phase 4).
// Bind it to any doc section or knowledge page and the operator can ask
// the AI to update/restructure/re-draft it. AI replies may carry a
// structured revision: applied directly (ai-maintained, unlocked) or
// filed as a proposal (human-locked) — shown as a badge on the message.
// One click escalates to a full agent session for codebase-grounded work.

import { useEffect, useMemo, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import {
  ArrowUpRight,
  Bot,
  Check,
  Inbox,
  Loader2,
  MessageSquarePlus,
  Send,
  X,
} from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { useTranslation } from 'react-i18next'
import { useNavigate } from '@tanstack/react-router'

import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Textarea } from '@/components/ui/textarea'
import {
  type ConversationTargetKind,
  closeConversation,
  createConversation,
  escalateConversation,
  getConversation,
  listConversations,
  sendConversationMessage,
} from '@/lib/cortex'

interface CurationChatProps {
  targetKind: ConversationTargetKind
  targetCwd: string
  targetSlug: string
  /** Called after a revision was applied/proposed so the host can refetch the doc. */
  onRevision?: () => void
}

export function CurationChat({
  targetKind,
  targetCwd,
  targetSlug,
  onRevision,
}: CurationChatProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [draft, setDraft] = useState('')
  const bottomRef = useRef<HTMLDivElement>(null)

  // The open conversation for this target (newest wins); created lazily
  // on the first send.
  const convsQuery = useQuery({
    queryKey: ['cortex-conversations', targetCwd, targetSlug],
    queryFn: () => listConversations(targetCwd, targetSlug),
  })
  const active = useMemo(
    () => (convsQuery.data ?? []).find((c) => c.status !== 'closed'),
    [convsQuery.data],
  )

  const detailQuery = useQuery({
    queryKey: ['cortex-conversation', active?.id],
    queryFn: () => getConversation(active!.id),
    enabled: !!active,
    // While the last turn is the operator's, the AI reply is still in
    // flight — poll until it lands (the backend also emits
    // cortex.conversation.reply on the event bus).
    refetchInterval: (q) => {
      const msgs = q.state.data?.messages
      if (!msgs || msgs.length === 0) return false
      return msgs[msgs.length - 1].role === 'operator' ? 2500 : false
    },
  })
  const messages = detailQuery.data?.messages ?? []
  const awaitingReply =
    messages.length > 0 && messages[messages.length - 1].role === 'operator'

  // Refetch the host doc whenever a revision message lands.
  const lastRevision = useMemo(() => {
    for (let i = messages.length - 1; i >= 0; i--) {
      const m = messages[i]
      if (m.revision_action) return m.id
    }
    return ''
  }, [messages])
  const seenRevision = useRef('')
  useEffect(() => {
    if (lastRevision && lastRevision !== seenRevision.current) {
      seenRevision.current = lastRevision
      onRevision?.()
    }
  }, [lastRevision, onRevision])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages.length])

  const send = useMutation({
    mutationFn: async (text: string) => {
      let conv = active
      if (!conv) {
        conv = await createConversation({
          target_kind: targetKind,
          target_cwd: targetCwd,
          target_slug: targetSlug,
        })
      }
      await sendConversationMessage(conv.id, text)
      return conv.id
    },
    onSuccess: (id) => {
      setDraft('')
      qc.invalidateQueries({ queryKey: ['cortex-conversations', targetCwd, targetSlug] })
      qc.invalidateQueries({ queryKey: ['cortex-conversation', id] })
    },
    onError: (e: Error) =>
      toast.error(t('web.cortex.chat.sendFailed'), { description: e.message }),
  })

  const escalate = useMutation({
    mutationFn: () => escalateConversation(active!.id),
    onSuccess: (conv) => {
      toast.success(t('web.cortex.chat.escalatedToast'))
      qc.invalidateQueries({ queryKey: ['cortex-conversation', conv.id] })
      if (conv.escalated_session_id) {
        navigate({ to: '/sessions', search: { open: conv.escalated_session_id } as any })
      }
    },
    onError: (e: Error) =>
      toast.error(t('web.cortex.chat.escalateFailed'), { description: e.message }),
  })

  const close = useMutation({
    mutationFn: () => closeConversation(active!.id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['cortex-conversations', targetCwd, targetSlug] })
    },
  })

  return (
    <div className="bg-muted/10 flex max-h-[28rem] flex-col rounded-md border">
      <div className="flex items-center justify-between gap-2 border-b px-3 py-2">
        <span className="text-muted-foreground flex items-center gap-1.5 text-xs font-medium">
          <Bot className="h-3.5 w-3.5" />
          {t('web.cortex.chat.title')}
        </span>
        <div className="flex items-center gap-1.5">
          {active && (
            <>
              <Button
                size="sm"
                variant="ghost"
                className="h-6 px-2 text-[11px]"
                disabled={escalate.isPending || active.status === 'escalated'}
                title={t('web.cortex.chat.escalateHint')}
                onClick={() => escalate.mutate()}
              >
                {escalate.isPending ? (
                  <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                ) : (
                  <ArrowUpRight className="mr-1 h-3 w-3" />
                )}
                {active.status === 'escalated'
                  ? t('web.cortex.chat.escalated')
                  : t('web.cortex.chat.escalate')}
              </Button>
              <Button
                size="sm"
                variant="ghost"
                className="h-6 px-2 text-[11px]"
                onClick={() => close.mutate()}
                title={t('web.cortex.chat.closeHint')}
              >
                <X className="h-3 w-3" />
              </Button>
            </>
          )}
        </div>
      </div>

      <div className="flex-1 space-y-2.5 overflow-auto px-3 py-2.5">
        {messages.length === 0 && (
          <div className="text-muted-foreground flex flex-col items-center gap-1.5 py-6 text-center text-xs">
            <MessageSquarePlus className="h-5 w-5 opacity-50" />
            <p>{t('web.cortex.chat.emptyHint')}</p>
          </div>
        )}
        {messages.map((m) => (
          <div
            key={m.id}
            className={
              m.role === 'operator'
                ? 'ml-8 rounded-md bg-primary/10 px-2.5 py-1.5'
                : m.role === 'system'
                  ? 'rounded-md border border-dashed px-2.5 py-1.5 opacity-70'
                  : 'mr-8 rounded-md bg-card px-2.5 py-1.5'
            }
          >
            <div className="text-sm leading-relaxed [&_p]:my-1">
              <ReactMarkdown remarkPlugins={[remarkGfm]}>{m.content}</ReactMarkdown>
            </div>
            {m.revision_action === 'applied' && (
              <Badge variant="success" className="mt-1 text-[10px]">
                <Check className="mr-1 h-2.5 w-2.5" />
                {t('web.cortex.chat.revisionApplied')}
              </Badge>
            )}
            {m.revision_action === 'proposed' && (
              <Badge variant="warning" className="mt-1 text-[10px]">
                <Inbox className="mr-1 h-2.5 w-2.5" />
                {t('web.cortex.chat.revisionProposed')}
              </Badge>
            )}
          </div>
        ))}
        {awaitingReply && (
          <div className="text-muted-foreground mr-8 flex items-center gap-2 rounded-md bg-card px-2.5 py-1.5 text-xs">
            <Loader2 className="h-3 w-3 animate-spin" />
            {t('web.cortex.chat.thinking')}
          </div>
        )}
        <div ref={bottomRef} />
      </div>

      <div className="flex items-end gap-2 border-t px-3 py-2">
        <Textarea
          rows={2}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && (e.metaKey || e.ctrlKey) && draft.trim()) {
              send.mutate(draft.trim())
            }
          }}
          placeholder={t('web.cortex.chat.placeholder')}
          className="min-h-0 flex-1 text-sm"
        />
        <Button
          size="sm"
          disabled={!draft.trim() || send.isPending || awaitingReply}
          onClick={() => send.mutate(draft.trim())}
        >
          {send.isPending ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <Send className="h-3.5 w-3.5" />
          )}
        </Button>
      </div>
    </div>
  )
}
