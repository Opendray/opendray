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
  setConversationProvider,
} from '@/lib/cortex'
import { type AgentProviderID, listAgentModels } from '@/lib/memoryWorkers'
import { listProviders } from '@/lib/memoryAmbient'
import { probeEmbeddingEndpoint } from '@/lib/memory'

// Cloud-agent providers selectable for a discussion. Mirrors the backend's
// curation override set (claude | gemini | codex); '' = global curation worker.
const CURATION_PROVIDERS: { id: AgentProviderID; label: string }[] = [
  { id: 'claude', label: 'Claude' },
  { id: 'gemini', label: 'Gemini' },
  { id: 'codex', label: 'Codex' },
]

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

  // The picker selection encodes which model backs the discussion:
  //   ''           → global `curation` worker config
  //   'agent:<id>' → cloud-agent CLI (claude/gemini/codex) + a model
  //   'local:<id>' → a configured summarizer/HTTP provider (local model)
  // Seeded from the active conversation once it loads (sync effect below).
  const [selection, setSelection] = useState('')
  const [model, setModel] = useState('')
  const agentProvider: AgentProviderID | '' = selection.startsWith('agent:')
    ? (selection.slice('agent:'.length) as AgentProviderID)
    : ''

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

  // Configured local/HTTP models (summarizer providers) selectable here.
  const summarizersQuery = useQuery({
    queryKey: ['memory-summarizer-providers'],
    queryFn: listProviders,
    staleTime: 5 * 60 * 1000,
  })
  const localProviders = (summarizersQuery.data ?? []).filter((p) => p.enabled)

  // Seed the picker from the active conversation when it changes (so a
  // reopened thread shows its pinned model), without clobbering an
  // in-progress selection on the same conversation.
  const syncedConvId = useRef<string | undefined>(undefined)
  useEffect(() => {
    if (active && active.id !== syncedConvId.current) {
      syncedConvId.current = active.id
      if (active.summarizer_id) setSelection(`local:${active.summarizer_id}`)
      else if (active.provider_id) setSelection(`agent:${active.provider_id}`)
      else setSelection('')
      setModel(active.model ?? '')
    }
  }, [active])

  // Model catalog for the selected cloud-agent provider.
  const modelsQuery = useQuery({
    queryKey: ['agent-models', agentProvider],
    queryFn: () => listAgentModels(agentProvider as AgentProviderID),
    enabled: agentProvider !== '',
    staleTime: 60 * 60 * 1000,
  })

  // The selected local provider (when 'local:<id>') + the models its
  // endpoint actually serves (probed live; LM Studio/Ollama expose many).
  const localSelected = selection.startsWith('local:')
    ? localProviders.find((p) => p.id === selection.slice('local:'.length))
    : undefined
  const localModelsQuery = useQuery({
    queryKey: ['endpoint-models', localSelected?.base_url],
    queryFn: () => probeEmbeddingEndpoint(localSelected?.base_url ?? ''),
    enabled: !!localSelected?.base_url,
    staleTime: 5 * 60 * 1000,
  })
  const localModels = localModelsQuery.data?.reachable
    ? (localModelsQuery.data.models ?? [])
    : []

  // Map a selection + agent model to the backend override shape.
  const overrideFor = (sel: string, m: string) => {
    if (sel.startsWith('agent:'))
      return { provider_id: sel.slice('agent:'.length), model: m }
    if (sel.startsWith('local:'))
      return { summarizer_id: sel.slice('local:'.length) }
    return {}
  }

  // Persist an override change. When no conversation exists yet the
  // selection is held locally and applied at creation time (see send).
  const persistOverride = useMutation({
    mutationFn: (next: { sel: string; model: string }) =>
      setConversationProvider(active!.id, overrideFor(next.sel, next.model)),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['cortex-conversations', targetCwd, targetSlug] })
      if (active) qc.invalidateQueries({ queryKey: ['cortex-conversation', active.id] })
    },
    onError: (e: Error) =>
      toast.error(t('web.cortex.chat.modelChangeFailed'), { description: e.message }),
  })

  const changeSelection = (sel: string) => {
    setSelection(sel)
    setModel('')
    if (active) persistOverride.mutate({ sel, model: '' })
  }
  const changeModel = (m: string) => {
    setModel(m)
    if (active) persistOverride.mutate({ sel: selection, model: m })
  }

  const send = useMutation({
    mutationFn: async (text: string) => {
      let conv = active
      if (!conv) {
        conv = await createConversation({
          target_kind: targetKind,
          target_cwd: targetCwd,
          target_slug: targetSlug,
          ...overrideFor(selection, model),
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

      <div className="text-muted-foreground flex items-center gap-2 border-b px-3 py-1.5 text-[11px]">
        <span className="shrink-0">{t('web.cortex.chat.modelLabel')}</span>
        <select
          value={selection}
          onChange={(e) => changeSelection(e.target.value)}
          className="bg-background min-w-0 flex-1 rounded border px-1.5 py-0.5 text-[11px]"
          title={t('web.cortex.chat.modelHint')}
        >
          <option value="">{t('web.cortex.chat.modelGlobalDefault')}</option>
          <optgroup label={t('web.cortex.chat.modelGroupCloud')}>
            {CURATION_PROVIDERS.map((p) => (
              <option key={p.id} value={`agent:${p.id}`}>
                {p.label}
              </option>
            ))}
          </optgroup>
          {localProviders.length > 0 && (
            <optgroup label={t('web.cortex.chat.modelGroupLocal')}>
              {localProviders.map((p) => (
                <option key={p.id} value={`local:${p.id}`}>
                  {p.name}
                  {p.model ? ` · ${p.model}` : ''}
                </option>
              ))}
            </optgroup>
          )}
        </select>
        {agentProvider !== '' && (
          <select
            value={model}
            onChange={(e) => changeModel(e.target.value)}
            className="bg-background min-w-0 flex-1 rounded border px-1.5 py-0.5 text-[11px]"
          >
            <option value="">{t('web.cortex.chat.modelCliDefault')}</option>
            {(modelsQuery.data ?? []).map((m) => (
              <option key={m.id} value={m.id}>
                {m.label}
              </option>
            ))}
          </select>
        )}
        {selection.startsWith('local:') && (
          <select
            value={model}
            onChange={(e) => changeModel(e.target.value)}
            className="bg-background min-w-0 flex-1 rounded border px-1.5 py-0.5 text-[11px]"
            title={
              localModelsQuery.isError
                ? t('web.cortex.chat.modelProbeFailed')
                : undefined
            }
          >
            <option value="">
              {localSelected?.model
                ? `${t('web.cortex.chat.modelProviderDefault')} · ${localSelected.model}`
                : t('web.cortex.chat.modelCliDefault')}
            </option>
            {localModels.map((m) => (
              <option key={m} value={m}>
                {m}
              </option>
            ))}
          </select>
        )}
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
