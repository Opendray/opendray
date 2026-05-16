import { useState, type FormEvent, useEffect } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Plus,
  Loader2,
  Send,
  Trash2,
  MessageSquare,
  RefreshCw,
  Copy,
  Code2,
  Check,
  Pencil,
} from 'lucide-react'
import { toast } from 'sonner'
import { Trans, useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  Select,
  SelectTrigger,
  SelectContent,
  SelectItem,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Textarea } from '@/components/ui/textarea'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { BrandIcon } from '@/components/BrandIcon'
import { BrandAvatar } from '@/components/BrandAvatar'
import {
  listChannels,
  createChannel,
  deleteChannel,
  testChannel,
  listChannelKinds,
  updateChannel,
} from '@/lib/channels'
import type { Channel } from '@/lib/types'
import { cn } from '@/lib/utils'
import {
  KIND_DEFS,
  buildWebhookURL,
  getKindDef,
  type KindDef,
  type KindField,
} from '@/lib/channelKinds'

const BRIDGE_CAPABILITIES = [
  'text',
  'card',
  'buttons',
  'image',
  'file',
  'typing',
  'update_message',
  'reply_to_message',
] as const

function generateBridgeToken(): string {
  const bytes = new Uint8Array(24)
  crypto.getRandomValues(bytes)
  return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('')
}

export function ChannelsPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [createOpen, setCreateOpen] = useState(false)
  const [editingChannel, setEditingChannel] = useState<Channel | null>(null)
  const [setupChannel, setSetupChannel] = useState<Channel | null>(null)

  const { data: channels, isLoading } = useQuery({
    queryKey: ['channels'],
    queryFn: listChannels,
    refetchInterval: 6_000,
  })

  return (
    <div className="h-full flex flex-col bg-background">
      <header className="border-b border-border px-6 py-4 flex items-center gap-3">
        <div className="flex-1">
          <h1 className="text-[16px] font-semibold tracking-tight">
            {t('web.channels.title')}
          </h1>
          <p className="text-[12px] text-muted-foreground">
            <Trans
              i18nKey="web.channels.subtitle"
              components={{ 1: <code /> }}
            />
          </p>
        </div>
        <Button
          variant="accent"
          size="sm"
          onClick={() => setCreateOpen(true)}
        >
          <Plus className="size-3.5" /> {t('web.channels.newButton')}
        </Button>
      </header>

      <ScrollArea className="flex-1">
        <div className="p-6 max-w-[960px] flex flex-col gap-3">
          {isLoading && (
            <div className="flex items-center gap-2 text-[12px] text-muted-foreground">
              <Loader2 className="size-3.5 animate-spin" />
              {t('web.channels.loading')}
            </div>
          )}
          {!isLoading && (channels?.length ?? 0) === 0 && (
            <div className="flex flex-col items-center justify-center py-16 gap-3 text-center">
              <MessageSquare
                className="size-10 text-muted-foreground/40"
                strokeWidth={1.5}
              />
              <h2 className="text-[14px] font-semibold">
                {t('web.channels.empty.title')}
              </h2>
              <p className="text-[12px] text-muted-foreground max-w-[420px]">
                <Trans
                  i18nKey="web.channels.empty.description"
                  components={{ 1: <code /> }}
                />
              </p>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setCreateOpen(true)}
              >
                <Plus className="size-3.5" /> {t('web.channels.newButton')}
              </Button>
            </div>
          )}
          {(channels ?? []).map((c) => (
            <ChannelCard
              key={c.id}
              channel={c}
              onTest={() => {
                testChannel(c.id)
                  .then(() => toast.success(t('web.channels.toasts.testSent')))
                  .catch((err: Error) =>
                    toast.error(t('web.channels.toasts.testFailed'), {
                      description: err.message,
                    }),
                  )
              }}
              onToggle={(enabled) => {
                updateChannel(c.id, { enabled })
                  .then(() => qc.invalidateQueries({ queryKey: ['channels'] }))
                  .catch((err: Error) => toast.error(err.message))
              }}
              onDelete={() => {
                if (
                  !confirm(t('web.channels.toasts.deleteConfirm', { id: c.id }))
                )
                  return
                deleteChannel(c.id)
                  .then(() => {
                    qc.invalidateQueries({ queryKey: ['channels'] })
                    toast.success(t('web.channels.toasts.deleted'))
                  })
                  .catch((err: Error) => toast.error(err.message))
              }}
              onSetup={() => setSetupChannel(c)}
              onEdit={() => setEditingChannel(c)}
            />
          ))}
        </div>
      </ScrollArea>

      <ChannelDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onCreated={(c) => {
          if (c.kind === 'bridge') setSetupChannel(c)
        }}
      />
      <ChannelDialog
        open={editingChannel !== null}
        onOpenChange={(v) => {
          if (!v) setEditingChannel(null)
        }}
        editing={editingChannel}
      />
      <BridgeSetupDialog
        channel={setupChannel}
        open={setupChannel !== null}
        onOpenChange={(v) => {
          if (!v) setSetupChannel(null)
        }}
      />
    </div>
  )
}

function ChannelCard({
  channel,
  onTest,
  onToggle,
  onDelete,
  onSetup,
  onEdit,
}: {
  channel: Channel
  onTest: () => void
  onToggle: (enabled: boolean) => void
  onDelete: () => void
  onSetup: () => void
  onEdit: () => void
}) {
  const { t } = useTranslation()
  const cfg = channel.config as Record<string, unknown>
  const def = getKindDef(channel.kind)
  const isBridge = channel.kind === 'bridge'

  // Pull a token to show as a masked preview. Bridge uses its own
  // `token`; named kinds list candidates in their KindDef; fall back to
  // the legacy `bot_token`.
  const tokenCandidates: string[] = isBridge
    ? ['token']
    : (def?.tokenFields ?? ['bot_token'])
  let rawToken = ''
  for (const key of tokenCandidates) {
    const v = cfg[key]
    if (typeof v === 'string' && v) {
      rawToken = v
      break
    }
  }
  const tokenPreview = rawToken
    ? `${rawToken.slice(0, 6)}…${rawToken.slice(-4)}`
    : '—'

  const displayKind =
    isBridge && cfg.name
      ? `${cfg.name} ${t('web.channels.card.bridgeSuffix')}`
      : (def?.label ?? channel.kind)
  const webhookURL = def?.webhookBased ? buildWebhookURL(channel.id) : ''

  return (
    <div className="border border-border rounded-md p-4 bg-card/30 flex flex-col gap-3">
      <div className="flex items-start gap-3">
        <BrandAvatar
          iconKey={def?.iconKey}
          fallbackLetter={(def?.label ?? channel.kind).charAt(0)}
          size={28}
          title={def?.label ?? channel.kind}
        />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-[13px] font-semibold">{displayKind}</span>
            <Badge variant="outline" className="font-mono normal-case">
              {channel.id}
            </Badge>
            {channel.running ? (
              <Badge variant="success">{t('web.channels.card.running')}</Badge>
            ) : channel.enabled ? (
              <Badge variant="warning">{t('web.channels.card.starting')}</Badge>
            ) : (
              <Badge variant="muted">{t('web.channels.card.disabled')}</Badge>
            )}
            {channel.muted && (
              <Badge variant="warning">{t('web.channels.card.muted')}</Badge>
            )}
          </div>
          <div className="text-[11px] text-muted-foreground/80 font-mono mt-1 flex flex-wrap gap-x-4 gap-y-0.5">
            <span>
              {t('web.channels.card.tokenLabel')} {tokenPreview}
            </span>
            {typeof cfg.chat_id === 'string' || typeof cfg.chat_id === 'number' ? (
              <span>
                {t('web.channels.card.chatIdLabel')} {String(cfg.chat_id)}
              </span>
            ) : null}
            {typeof cfg.channel_id === 'string' && cfg.channel_id ? (
              <span>
                {t('web.channels.card.channelIdLabel')} {String(cfg.channel_id)}
              </span>
            ) : null}
            {Array.isArray(cfg.notify_on) && (cfg.notify_on as string[]).length > 0 && (
              <span>
                {t('web.channels.card.notifyOnLabel')}{' '}
                {(cfg.notify_on as string[]).join(', ')}
              </span>
            )}
          </div>
          {webhookURL && (
            <div className="mt-2 flex items-center gap-2 text-[11px] font-mono">
              <span className="text-muted-foreground">
                {t('web.channels.card.webhookLabel')}
              </span>
              <code className="truncate flex-1 text-muted-foreground/90">{webhookURL}</code>
              <Button
                variant="ghost"
                size="sm"
                className="h-6 px-2 text-[11px]"
                onClick={() => {
                  navigator.clipboard.writeText(webhookURL)
                  toast.success(t('web.channels.card.webhookCopiedToast'))
                }}
                title={t('web.channels.card.copyWebhookTooltip')}
              >
                <Copy className="size-3" />
              </Button>
            </div>
          )}
          {channel.capabilities && channel.capabilities.length > 0 && (
            <div className="mt-2 flex flex-wrap gap-1">
              {channel.capabilities.map((c) => (
                <Badge key={c} variant="muted" className="font-mono text-[10px]">
                  {c}
                </Badge>
              ))}
            </div>
          )}
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <Switch checked={channel.enabled} onCheckedChange={onToggle} />
          {isBridge && (
            <Button
              variant="outline"
              size="sm"
              onClick={onSetup}
              title={t('web.channels.card.setupTooltip')}
            >
              <Code2 className="size-3.5" /> {t('web.channels.card.setup')}
            </Button>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={onTest}
            disabled={!channel.running || isBridge}
            title={
              isBridge
                ? t('web.channels.card.testBridgeTooltip')
                : !channel.running
                  ? t('web.channels.card.testNotRunningTooltip')
                  : undefined
            }
          >
            <Send className="size-3.5" /> {t('web.channels.card.test')}
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={onEdit}
            aria-label={t('web.channels.card.editAria')}
            title={t('web.channels.card.editTooltip')}
          >
            <Pencil className="size-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={onDelete}
            aria-label={t('web.channels.card.deleteAria')}
            className="text-muted-foreground hover:text-destructive"
          >
            <Trash2 className="size-3.5" />
          </Button>
        </div>
      </div>
    </div>
  )
}

function ChannelDialog({
  open,
  onOpenChange,
  onCreated,
  editing,
}: {
  open: boolean
  onOpenChange: (v: boolean) => void
  onCreated?: (channel: Channel) => void
  // When set, dialog is in EDIT mode: kind is locked, values are
  // pre-filled from the channel's existing config, and submit calls
  // updateChannel(id, …) instead of createChannel(…).
  editing?: Channel | null
}) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const isEdit = !!editing
  const { data: kinds } = useQuery({
    queryKey: ['channel-kinds'],
    queryFn: listChannelKinds,
    enabled: open && !isEdit,
  })

  const [kind, setKind] = useState<string>('telegram')
  const [values, setValues] = useState<Record<string, string>>({})
  const [bridgeName, setBridgeName] = useState('')
  const [bridgeToken, setBridgeToken] = useState('')
  const [bridgeCaps, setBridgeCaps] = useState<string[]>([])

  const [enabled, setEnabled] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const def = getKindDef(kind)

  // Hydrate dialog state every time it opens.
  useEffect(() => {
    if (!open) return
    setError(null)
    if (editing) {
      setKind(editing.kind)
      setEnabled(editing.enabled)
      const cfg = editing.config as Record<string, unknown>
      if (editing.kind === 'bridge') {
        setBridgeName(typeof cfg.name === 'string' ? cfg.name : '')
        setBridgeToken(typeof cfg.token === 'string' ? cfg.token : '')
        const caps = Array.isArray(cfg.accept_capabilities) ? (cfg.accept_capabilities as string[]) : []
        setBridgeCaps(caps)
        setValues({})
      } else {
        const d = getKindDef(editing.kind)
        setValues(d ? valuesFromConfig(d, cfg) : {})
        setBridgeName('')
        setBridgeToken(generateBridgeToken())
        setBridgeCaps([])
      }
    } else {
      // Create mode — keep current `kind`, just (re)seed defaults.
      setEnabled(true)
      setBridgeName('')
      if (!bridgeToken) setBridgeToken(generateBridgeToken())
      setBridgeCaps([])
      // Values reset is handled by the kind-change effect below.
    }
  }, [open, editing])

  useEffect(() => {
    if (kinds && kinds.length > 0 && !kinds.includes(kind) && !isEdit) {
      setKind(kinds[0])
    }
  }, [kinds, kind, isEdit])

  // When the operator picks a different kind in CREATE mode, reset
  // values to that kind's defaults. Editing mode never triggers this
  // (kind is locked).
  useEffect(() => {
    if (isEdit || !def) return
    setValues({
      // Empty notify_on = "all topics" — explicit per-channel filter
      // is the user's job.
      notify_on: '',
      // Default mode `once` matches the backend default and is what
      // the user expects ("notify me once when work needs me").
      notify_mode: DEFAULT_NOTIFY_MODE,
      notify_cooldown_s: DEFAULT_COOLDOWN,
      notify_include_snippet: 'true',
      notify_snippet_max_chars: DEFAULT_SNIPPET_CAP,
    })
  }, [def, isEdit])

  const create = useMutation({
    mutationFn: createChannel,
    onSuccess: (channel) => {
      qc.invalidateQueries({ queryKey: ['channels'] })
      toast.success(t('web.channels.toasts.created'))
      onOpenChange(false)
      onCreated?.(channel)
    },
    onError: (e: Error) => setError(e.message),
  })

  const update = useMutation({
    mutationFn: (vars: { id: string; config: Record<string, unknown>; enabled: boolean }) =>
      updateChannel(vars.id, { config: vars.config, enabled: vars.enabled }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['channels'] })
      toast.success(t('web.channels.toasts.updated'))
      onOpenChange(false)
    },
    onError: (e: Error) => setError(e.message),
  })

  const submit = (e: FormEvent) => {
    e.preventDefault()
    setError(null)

    let config: Record<string, unknown>
    if (kind === 'bridge') {
      if (!bridgeName.trim()) {
        setError(t('web.channels.dialog.nameRequired'))
        return
      }
      if (!bridgeToken.trim()) {
        setError(t('web.channels.dialog.tokenRequired'))
        return
      }
      config = {
        name: bridgeName.trim(),
        token: bridgeToken.trim(),
      }
      if (bridgeCaps.length > 0) {
        config.accept_capabilities = bridgeCaps
      }
    } else if (def) {
      try {
        config = buildConfigFromValues(def, values, t)
      } catch (err) {
        setError((err as Error).message)
        return
      }
    } else {
      setError(t('web.channels.dialog.unknownKind', { kind }))
      return
    }

    if (editing) {
      update.mutate({ id: editing.id, config, enabled })
    } else {
      create.mutate({ kind, config, enabled })
    }
  }

  const toggleBridgeCap = (cap: string) => {
    setBridgeCaps((prev) =>
      prev.includes(cap) ? prev.filter((c) => c !== cap) : [...prev, cap],
    )
  }

  const orderedKinds = orderKinds(kinds ?? [])
  const pending = create.isPending || update.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[520px] max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {isEdit
              ? t('web.channels.dialog.editTitle')
              : t('web.channels.dialog.createTitle')}
          </DialogTitle>
          <DialogDescription>
            {kind === 'bridge'
              ? t('web.channels.dialog.descriptionBridge')
              : (def?.description ?? t('web.channels.dialog.descriptionDefault'))}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="flex flex-col gap-4 mt-2">
          <div className="space-y-1.5">
            <Label htmlFor="kind">{t('web.channels.dialog.kindLabel')}</Label>
            {isEdit ? (
              <div className="flex items-center gap-2 px-3 py-2 border border-border rounded-md bg-muted/20 text-[12px]">
                <BrandIcon
                  iconKey={def?.iconKey}
                  size={14}
                  title={def?.label ?? kind}
                />
                <span className="font-mono">{def?.label ?? kind}</span>
                <span className="text-muted-foreground/70">
                  {t('web.channels.dialog.kindImmutable')}
                </span>
              </div>
            ) : (
              <Select value={kind} onValueChange={setKind}>
                <SelectTrigger id="kind">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {orderedKinds.map((k) => {
                    const d = getKindDef(k)
                    return (
                      <SelectItem key={k} value={k}>
                        <span className="inline-flex items-center gap-2">
                          {d?.iconKey ? (
                            <BrandIcon iconKey={d.iconKey} size={14} title={d.label} />
                          ) : (
                            <span>{d?.emoji ?? '📨'}</span>
                          )}
                          <span>{d ? d.label : k}</span>
                        </span>
                      </SelectItem>
                    )
                  })}
                </SelectContent>
              </Select>
            )}
          </div>

          {kind === 'bridge' ? (
            <BridgeFields
              name={bridgeName}
              setName={setBridgeName}
              token={bridgeToken}
              setToken={setBridgeToken}
              caps={bridgeCaps}
              toggleCap={toggleBridgeCap}
            />
          ) : def ? (
            <>
              <KindFields
                def={def}
                values={values}
                setValue={(name, val) => setValues((prev) => ({ ...prev, [name]: val }))}
              />
              <NotificationFields
                values={values}
                setValue={(name, val) => setValues((prev) => ({ ...prev, [name]: val }))}
              />
            </>
          ) : null}

          {!isEdit && def?.afterCreateHint && (
            <div className="text-[11px] text-muted-foreground bg-muted/30 border border-border rounded-md px-3 py-2 leading-relaxed">
              {def.afterCreateHint}
            </div>
          )}

          <div className="flex items-center gap-2">
            <Switch
              id="enabled"
              checked={enabled}
              onCheckedChange={setEnabled}
            />
            <Label htmlFor="enabled" className="!text-[12px]">
              {t('web.channels.dialog.enabledLabel')}
              {kind === 'bridge'
                ? t('web.channels.dialog.enabledBridgeHint')
                : def?.webhookBased
                  ? t('web.channels.dialog.enabledWebhookHint')
                  : t('web.channels.dialog.enabledDefaultHint')}
            </Label>
          </div>

          {error && (
            <div className={cn('text-[12px] text-destructive bg-destructive/10 border border-destructive/30 rounded-md px-3 py-2')}>
              {error}
            </div>
          )}

          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => onOpenChange(false)}
              disabled={pending}
            >
              {t('web.channels.dialog.cancel')}
            </Button>
            <Button
              type="submit"
              variant="accent"
              size="sm"
              disabled={pending}
            >
              {pending && <Loader2 className="size-3.5 animate-spin" />}
              {isEdit
                ? pending
                  ? t('web.channels.dialog.saving')
                  : t('web.channels.dialog.save')
                : pending
                  ? t('web.channels.dialog.creating')
                  : t('web.channels.dialog.create')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

// valuesFromConfig is the inverse of buildConfigFromValues: given a
// channel's persisted config, produce the flat string-keyed map that
// the dialog form binds to.
function valuesFromConfig(
  def: KindDef,
  config: Record<string, unknown>,
): Record<string, string> {
  const out: Record<string, string> = {}
  for (const field of def.fields) {
    const raw = config[field.name]
    if (raw == null) continue
    if (Array.isArray(raw)) {
      out[field.name] = raw.map((v) => String(v)).join('\n')
    } else if (typeof raw === 'number' || typeof raw === 'string' || typeof raw === 'boolean') {
      out[field.name] = String(raw)
    }
  }
  // Populate synthetic notification fields. Backend stores notify_on
  // as a string array (or absent); cooldown as a number (or absent
  // → DEFAULT_COOLDOWN).
  if (Array.isArray(config.notify_on)) {
    out.notify_on = (config.notify_on as unknown[])
      .map((v) => String(v))
      .filter((s) => s.length > 0)
      .join(',')
  } else {
    out.notify_on = ''
  }
  if (typeof config.notify_mode === 'string' && config.notify_mode !== '') {
    out.notify_mode = config.notify_mode as string
  } else if (typeof config.notify_cooldown_s === 'number' && config.notify_cooldown_s > 0) {
    // Legacy channels (saved before notify_mode existed) with a
    // non-zero cooldown infer mode = cooldown.
    out.notify_mode = 'cooldown'
  } else {
    out.notify_mode = DEFAULT_NOTIFY_MODE
  }
  if (typeof config.notify_cooldown_s === 'number' && config.notify_cooldown_s > 0) {
    out.notify_cooldown_s = String(config.notify_cooldown_s)
  } else {
    out.notify_cooldown_s = DEFAULT_COOLDOWN
  }
  // Snippet defaults: include=true unless explicitly false in config.
  out.notify_include_snippet = config.notify_include_snippet === false ? 'false' : 'true'
  if (typeof config.notify_snippet_max_chars === 'number') {
    out.notify_snippet_max_chars = String(config.notify_snippet_max_chars)
  } else {
    out.notify_snippet_max_chars = DEFAULT_SNIPPET_CAP
  }
  return out
}

// SESSION_TOPICS is the canonical list users can opt in/out of.
// Order matches the order they're rendered as checkboxes.
const SESSION_TOPICS = ['session.started', 'session.idle', 'session.ended'] as const

const NOTIFY_MODE_VALUES = ['once', 'cooldown', 'every'] as const
const COOLDOWN_VALUES = ['60', '300', '900', '1800', '3600'] as const
const SNIPPET_CAP_VALUES = ['0', '1000', '3000', '6000', '12000'] as const

const DEFAULT_NOTIFY_MODE = 'once'
const DEFAULT_COOLDOWN = '300'
const DEFAULT_SNIPPET_CAP = '0'

// NotificationFields renders the per-channel notification controls
// shared by every non-bridge kind: a checkbox row to choose which
// session.* topics to forward, plus a cooldown select that throttles
// duplicate notifications for the same (topic, session) within the
// chosen window.
function NotificationFields({
  values,
  setValue,
}: {
  values: Record<string, string>
  setValue: (name: string, val: string) => void
}) {
  const { t } = useTranslation()
  // Stored as comma-separated topics. Empty string = all topics
  // (matches the backend's notify_on=[] = "any" semantics).
  const selected = parseTopicList(values.notify_on ?? '')
  const isAllSelected = selected.length === 0 || selected.length === SESSION_TOPICS.length

  const mode = values.notify_mode ?? DEFAULT_NOTIFY_MODE
  const cooldown = values.notify_cooldown_s ?? DEFAULT_COOLDOWN
  // Snippet stored as the literal string "false" when off, anything
  // else (including missing) means "include snippet".
  const includeSnippet = values.notify_include_snippet !== 'false'
  const snippetCap = values.notify_snippet_max_chars ?? DEFAULT_SNIPPET_CAP

  const modeHintMap: Record<string, string> = {
    once: t('web.channels.notifications.modes.onceHint'),
    cooldown: t('web.channels.notifications.modes.cooldownHint'),
    every: t('web.channels.notifications.modes.everyHint'),
  }
  const modeLabelMap: Record<string, string> = {
    once: t('web.channels.notifications.modes.onceLabel'),
    cooldown: t('web.channels.notifications.modes.cooldownLabel'),
    every: t('web.channels.notifications.modes.everyLabel'),
  }
  const modeHint = modeHintMap[mode] ?? ''

  const toggleTopic = (topic: string) => {
    // First click: convert "all" → explicit list of remaining ones.
    const base = isAllSelected ? [...SESSION_TOPICS] : selected
    const next = base.includes(topic)
      ? base.filter((t) => t !== topic)
      : [...base, topic]
    // If the result is "all three", store as empty (= match all)
    // so the channel keeps receiving any topics added in future.
    if (next.length === SESSION_TOPICS.length) {
      setValue('notify_on', '')
    } else {
      setValue('notify_on', next.join(','))
    }
  }

  return (
    <div className="space-y-2 border border-border rounded-md p-3 bg-muted/10">
      <div className="text-[12px] font-semibold text-muted-foreground/90">
        {t('web.channels.notifications.sectionTitle')}
      </div>

      <div className="space-y-1.5">
        <Label className="!text-[11px] text-muted-foreground/80">
          {t('web.channels.notifications.notifyOnLabel')}
        </Label>
        <div className="flex flex-wrap gap-1.5">
          {SESSION_TOPICS.map((topic) => {
            const checked = isAllSelected || selected.includes(topic)
            return (
              <Badge
                key={topic}
                variant={checked ? 'success' : 'outline'}
                className="cursor-pointer font-mono text-[11px]"
                onClick={() => toggleTopic(topic)}
              >
                {checked ? <Check className="size-3" /> : null} {topic}
              </Badge>
            )
          })}
        </div>
        <p className="text-[11px] text-muted-foreground/80">
          {isAllSelected
            ? t('web.channels.notifications.hintAll')
            : selected.length === 0
              ? t('web.channels.notifications.hintNone')
              : t('web.channels.notifications.hintSome', {
                  selected: selected.length,
                  total: SESSION_TOPICS.length,
                })}
        </p>
      </div>

      <div className="space-y-1.5">
        <Label htmlFor="notify_mode" className="!text-[11px] text-muted-foreground/80">
          {t('web.channels.notifications.repeatPolicyLabel')}
        </Label>
        <Select value={mode} onValueChange={(v) => setValue('notify_mode', v)}>
          <SelectTrigger id="notify_mode">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {NOTIFY_MODE_VALUES.map((value) => (
              <SelectItem key={value} value={value}>
                {modeLabelMap[value]}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {modeHint && (
          <p className="text-[11px] text-muted-foreground/80">{modeHint}</p>
        )}
        {mode === 'cooldown' && (
          <div className="mt-2">
            <Label
              htmlFor="notify_cooldown_s"
              className="!text-[11px] text-muted-foreground/80"
            >
              {t('web.channels.notifications.cooldownLabel')}
            </Label>
            <Select
              value={cooldown}
              onValueChange={(v) => setValue('notify_cooldown_s', v)}
            >
              <SelectTrigger id="notify_cooldown_s" className="mt-1">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {COOLDOWN_VALUES.map((value) => (
                  <SelectItem key={value} value={value}>
                    {t(`web.channels.notifications.cooldowns.${value}`)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}
        {mode === 'once' && (
          <p className="text-[11px] text-muted-foreground/60 italic">
            {t('web.channels.notifications.onceReplyHint')}
          </p>
        )}
      </div>

      <div className="space-y-1.5">
        <Label className="!text-[11px] text-muted-foreground/80">
          {t('web.channels.notifications.terminalSnippetLabel')}
        </Label>
        <div className="flex items-center gap-2">
          <Switch
            id="notify_include_snippet"
            checked={includeSnippet}
            onCheckedChange={(v) =>
              setValue('notify_include_snippet', v ? 'true' : 'false')
            }
          />
          <Label htmlFor="notify_include_snippet" className="!text-[11px]">
            {t('web.channels.notifications.embedSnippetLabel')}
          </Label>
        </div>
        {includeSnippet && (
          <Select
            value={snippetCap}
            onValueChange={(v) => setValue('notify_snippet_max_chars', v)}
          >
            <SelectTrigger id="notify_snippet_max_chars" className="mt-1">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {SNIPPET_CAP_VALUES.map((value) => (
                <SelectItem key={value} value={value}>
                  {t(`web.channels.notifications.snippetCaps.${value}`)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
        <p className="text-[11px] text-muted-foreground/80">
          {t('web.channels.notifications.snippetExplainer')}
        </p>
      </div>
    </div>
  )
}

function parseTopicList(raw: string): string[] {
  return raw
    .split(/[\n,]/)
    .map((s) => s.trim())
    .filter((s) => s.length > 0)
}

// KindFields renders the form fields for a non-bridge channel kind
// straight from its KindDef. Lays the first field as autoFocus.
function KindFields({
  def,
  values,
  setValue,
}: {
  def: KindDef
  values: Record<string, string>
  setValue: (name: string, val: string) => void
}) {
  return (
    <>
      {def.fields.map((field, idx) => (
        <KindFieldRow
          key={field.name}
          field={field}
          autoFocus={idx === 0}
          value={values[field.name] ?? ''}
          onChange={(v) => setValue(field.name, v)}
        />
      ))}
    </>
  )
}

function KindFieldRow({
  field,
  autoFocus,
  value,
  onChange,
}: {
  field: KindField
  autoFocus?: boolean
  value: string
  onChange: (v: string) => void
}) {
  const id = `field_${field.name}`
  const isMono = field.type === 'password' || field.name.endsWith('_id') || field.name.endsWith('_token')
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>{field.label}</Label>
      {field.type === 'textarea' ? (
        <Textarea
          id={id}
          rows={3}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={field.placeholder}
          className="font-mono text-[12px]"
        />
      ) : (
        <Input
          id={id}
          type={field.type === 'password' ? 'password' : 'text'}
          autoComplete="off"
          autoFocus={autoFocus}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={field.placeholder}
          required={field.required}
          className={isMono ? 'font-mono' : undefined}
        />
      )}
      {field.hint && (
        <p className="text-[11px] text-muted-foreground/80">{field.hint}</p>
      )}
    </div>
  )
}

// buildConfigFromValues turns a flat {fieldName: stringValue} record
// into the channel-config JSON the backend expects. Required fields
// are validated; optional fields are dropped when empty. Field-name
// conventions for special types:
//   - uids                       → string[]    (split on newline)
//   - topic_ids                  → number[]    (split on newline, parse int)
//   - chat_id                    → number when numeric, else string
//   - notify_on (synthetic)      → string[]    (split on , or \n)
//   - notify_cooldown_s (synth)  → number (seconds)
function buildConfigFromValues(
  def: KindDef,
  values: Record<string, string>,
  t: TFunction,
): Record<string, unknown> {
  const cfg: Record<string, unknown> = {}
  for (const field of def.fields) {
    const raw = (values[field.name] ?? '').trim()
    if (!raw) {
      if (field.required) {
        throw new Error(
          t('web.channels.dialog.fieldRequired', { label: field.label }),
        )
      }
      continue
    }
    if (field.name === 'uids') {
      const lines = raw
        .split(/\n/)
        .map((s) => s.trim())
        .filter((s) => s.length > 0)
      if (lines.length > 0) cfg[field.name] = lines
      continue
    }
    if (field.name === 'topic_ids') {
      const ids = raw
        .split(/\n/)
        .map((s) => s.trim())
        .filter((s) => s.length > 0)
        .map((s) => {
          const n = Number(s)
          if (!Number.isFinite(n) || !/^\d+$/.test(s)) {
            throw new Error(
              t('web.channels.dialog.topicIdsNumeric', { value: s }),
            )
          }
          return n
        })
      if (ids.length > 0) cfg.topic_ids = ids
      continue
    }
    if (field.name === 'chat_id') {
      const n = Number(raw)
      cfg.chat_id = Number.isFinite(n) && /^-?\d+$/.test(raw) ? n : raw
      continue
    }
    cfg[field.name] = raw
  }
  // Synthetic notification fields — every non-bridge channel receives
  // these, regardless of whether they're declared in def.fields.
  const topics = parseTopicList(values.notify_on ?? '')
  if (topics.length > 0 && topics.length < SESSION_TOPICS.length) {
    // Persist explicit selection. All-three-selected is omitted to
    // mean "any topic" (matches the backend default).
    cfg.notify_on = topics
  }
  // Repeat policy: persist mode unless it's the default ("once" = no
  // emit so existing channels picked up by the new code keep their
  // implicit default behavior).
  const mode = (values.notify_mode ?? '').trim()
  if (mode !== '' && mode !== DEFAULT_NOTIFY_MODE) {
    cfg.notify_mode = mode
  }
  // Cooldown only matters when mode=cooldown.
  if (mode === 'cooldown') {
    const cooldown = (values.notify_cooldown_s ?? '').trim()
    if (cooldown !== '') {
      const n = Number(cooldown)
      if (!Number.isFinite(n) || n < 0) {
        throw new Error(t('web.channels.dialog.cooldownInvalid'))
      }
      cfg.notify_cooldown_s = n
    }
  }
  // Snippet-include is persisted as a real bool. Only emit the field
  // when the operator explicitly turned it OFF — backend default is
  // already true.
  if (values.notify_include_snippet === 'false') {
    cfg.notify_include_snippet = false
  }
  const cap = (values.notify_snippet_max_chars ?? '').trim()
  if (cap !== '' && cap !== DEFAULT_SNIPPET_CAP) {
    const n = Number(cap)
    if (!Number.isFinite(n) || n < 0) {
      throw new Error(t('web.channels.dialog.snippetCapInvalid'))
    }
    // 0 = no cap (channel impl handles platform-specific chunking).
    cfg.notify_snippet_max_chars = n
  }
  return cfg
}

// Server-registered kinds that we explicitly hide from the
// create-flow dropdown. Channels already in this state still
// render in the channel list — only the *new channel* option is
// suppressed. Used for kinds whose adapter still ships
// server-side (so existing rows keep working) but whose UX we
// no longer want operators to discover.
const HIDDEN_KINDS = new Set(['wechat'])

// orderKinds produces the dropdown order: native kinds first
// (KIND_DEFS order), then unknown kinds, then bridge last.
// Anything in HIDDEN_KINDS is dropped from the unknown bucket so
// retired entries (e.g. personal-WeChat / WxPusher) don't reappear
// just because the server still registers the adapter.
function orderKinds(kinds: string[]): string[] {
  const known = KIND_DEFS.map((k) => k.kind)
  const set = new Set(kinds)
  const out: string[] = []
  for (const k of known) {
    if (set.has(k)) out.push(k)
  }
  for (const k of kinds) {
    if (!known.includes(k) && k !== 'bridge' && !HIDDEN_KINDS.has(k)) {
      out.push(k)
    }
  }
  if (set.has('bridge')) out.push('bridge')
  return out
}

function BridgeFields({
  name,
  setName,
  token,
  setToken,
  caps,
  toggleCap,
}: {
  name: string
  setName: (v: string) => void
  token: string
  setToken: (v: string) => void
  caps: string[]
  toggleCap: (cap: string) => void
}) {
  const { t } = useTranslation()
  return (
    <>
      <div className="space-y-1.5">
        <Label htmlFor="bridge_name">{t('web.channels.bridge.nameLabel')}</Label>
        <Input
          id="bridge_name"
          autoComplete="off"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder={t('web.channels.bridge.namePlaceholder')}
          required
          autoFocus
        />
        <p className="text-[11px] text-muted-foreground/80">
          {t('web.channels.bridge.nameHint')}
        </p>
      </div>
      <div className="space-y-1.5">
        <Label htmlFor="bridge_token">
          {t('web.channels.bridge.tokenLabel')}
        </Label>
        <div className="flex gap-2">
          <Input
            id="bridge_token"
            value={token}
            onChange={(e) => setToken(e.target.value)}
            className="font-mono text-[11px]"
            required
          />
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => setToken(generateBridgeToken())}
            title={t('web.channels.bridge.regenerateTooltip')}
          >
            <RefreshCw className="size-3.5" />
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => {
              navigator.clipboard.writeText(token)
              toast.success(t('web.channels.bridge.tokenCopiedToast'))
            }}
            title={t('web.channels.bridge.copyTooltip')}
          >
            <Copy className="size-3.5" />
          </Button>
        </div>
        <p className="text-[11px] text-muted-foreground/80">
          <Trans
            i18nKey="web.channels.bridge.tokenHint"
            components={{ 1: <code /> }}
          />
        </p>
      </div>
      <div className="space-y-1.5">
        <Label>{t('web.channels.bridge.capsLabel')}</Label>
        <div className="flex flex-wrap gap-1">
          {BRIDGE_CAPABILITIES.map((cap) => {
            const active = caps.includes(cap)
            return (
              <Badge
                key={cap}
                variant={active ? 'success' : 'outline'}
                className="cursor-pointer font-mono"
                onClick={() => toggleCap(cap)}
              >
                {cap}
              </Badge>
            )
          })}
        </div>
        <p className="text-[11px] text-muted-foreground/80">
          {t('web.channels.bridge.capsHint')}
        </p>
      </div>
      <div className="text-[11px] text-muted-foreground bg-muted/30 border border-border rounded-md px-3 py-2 leading-relaxed">
        <Trans
          i18nKey="web.channels.bridge.afterCreate"
          components={{ 1: <strong /> }}
        />
      </div>
    </>
  )
}

function BridgeSetupDialog({
  channel,
  open,
  onOpenChange,
}: {
  channel: Channel | null
  open: boolean
  onOpenChange: (v: boolean) => void
}) {
  const { t } = useTranslation()
  if (!channel || channel.kind !== 'bridge') return null
  const cfg = channel.config as {
    name?: string
    token?: string
    accept_capabilities?: string[]
  }
  const name = cfg.name ?? 'my-platform'
  const token = cfg.token ?? ''
  const declaredCaps =
    cfg.accept_capabilities && cfg.accept_capabilities.length > 0
      ? cfg.accept_capabilities
      : ['text', 'card', 'buttons']

  // Build the WS URL from the current page origin so it works in both
  // dev (Vite proxy at :5173) and prod (opendray's embedded SPA at
  // :8770/admin). window.location.origin is http(s)://host[:port].
  const httpURL =
    typeof window !== 'undefined' ? window.location.origin : 'http://localhost:5173'
  const wsURL = httpURL.replace(/^http/, 'ws') + '/api/v1/channels/bridge/ws'

  const pythonCode = pythonAdapterTemplate({
    wsURL,
    token,
    name,
    capabilities: declaredCaps,
  })
  const nodeCode = nodeAdapterTemplate({
    wsURL,
    token,
    name,
    capabilities: declaredCaps,
  })
  const wscatCmd = wscatTemplate({ wsURL, token, name, capabilities: declaredCaps })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[720px]">
        <DialogHeader>
          <DialogTitle>
            <Trans
              i18nKey="web.channels.setup.title"
              values={{ name }}
              components={{
                1: <span className="font-mono" />,
              }}
            >
              {`Adapter setup — `}
              <span className="font-mono">{name}</span>
            </Trans>
          </DialogTitle>
          <DialogDescription>
            {t('web.channels.setup.description')}
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4 mt-2">
          <CopyRow label={t('web.channels.setup.wsUrlLabel')} value={wsURL} />
          <CopyRow label={t('web.channels.setup.tokenLabel')} value={token} secret />
          <div className="text-[11px] text-muted-foreground leading-relaxed bg-muted/20 border border-border rounded-md px-3 py-2">
            <Trans
              i18nKey="web.channels.setup.authInfo"
              values={{
                frame: `{"type":"register","platform":"…","capabilities":[…]}`,
              }}
              components={{
                1: <strong />,
                3: <code />,
                5: <code />,
                7: <code />,
                9: <code />,
                11: <code />,
              }}
            />
          </div>

          <Tabs defaultValue="python" className="w-full">
            <TabsList>
              <TabsTrigger value="python">Python</TabsTrigger>
              <TabsTrigger value="node">Node.js</TabsTrigger>
              <TabsTrigger value="wscat">wscat</TabsTrigger>
            </TabsList>
            <TabsContent value="python" className="mt-3">
              <CodeBlock filename="adapter.py" code={pythonCode} />
              <p className="text-[11px] text-muted-foreground/80 mt-2">
                <Trans
                  i18nKey="web.channels.setup.pythonInstall"
                  components={{ 1: <code />, 3: <code /> }}
                />
              </p>
            </TabsContent>
            <TabsContent value="node" className="mt-3">
              <CodeBlock filename="adapter.mjs" code={nodeCode} />
              <p className="text-[11px] text-muted-foreground/80 mt-2">
                <Trans
                  i18nKey="web.channels.setup.nodeInstall"
                  components={{ 1: <code />, 3: <code /> }}
                />
              </p>
            </TabsContent>
            <TabsContent value="wscat" className="mt-3">
              <CodeBlock filename="shell" code={wscatCmd} />
              <p className="text-[11px] text-muted-foreground/80 mt-2">
                <Trans
                  i18nKey="web.channels.setup.wscatInstall"
                  components={{ 1: <code /> }}
                />
              </p>
            </TabsContent>
          </Tabs>
        </div>

        <DialogFooter>
          <Button variant="ghost" size="sm" onClick={() => onOpenChange(false)}>
            {t('web.channels.setup.close')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function CopyRow({
  label,
  value,
  secret = false,
}: {
  label: string
  value: string
  secret?: boolean
}) {
  const { t } = useTranslation()
  const [revealed, setRevealed] = useState(!secret)
  const [copied, setCopied] = useState(false)
  const display = revealed ? value : '•'.repeat(Math.min(value.length, 40))
  return (
    <div className="space-y-1.5">
      <Label>{label}</Label>
      <div className="flex gap-2">
        <Input value={display} readOnly className="font-mono text-[11px]" />
        {secret && (
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => setRevealed((v) => !v)}
          >
            {revealed
              ? t('web.channels.setup.copyHide')
              : t('web.channels.setup.copyShow')}
          </Button>
        )}
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => {
            navigator.clipboard.writeText(value)
            setCopied(true)
            setTimeout(() => setCopied(false), 1500)
            toast.success(t('web.channels.setup.copyLabelToast', { label }))
          }}
        >
          {copied ? <Check className="size-3.5" /> : <Copy className="size-3.5" />}
        </Button>
      </div>
    </div>
  )
}

function CodeBlock({ filename, code }: { filename: string; code: string }) {
  const { t } = useTranslation()
  const [copied, setCopied] = useState(false)
  return (
    <div className="border border-border rounded-md overflow-hidden bg-card/50">
      <div className="flex items-center justify-between px-3 py-1.5 bg-muted/40 border-b border-border">
        <span className="text-[11px] font-mono text-muted-foreground">
          {filename}
        </span>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-7 text-[11px]"
          onClick={() => {
            navigator.clipboard.writeText(code)
            setCopied(true)
            setTimeout(() => setCopied(false), 1500)
            toast.success(t('web.channels.setup.codeCopiedToast'))
          }}
        >
          {copied ? <Check className="size-3.5" /> : <Copy className="size-3.5" />}
          {copied
            ? t('web.channels.setup.copied')
            : t('web.channels.setup.copyCode')}
        </Button>
      </div>
      <pre className="text-[11px] font-mono p-3 overflow-x-auto leading-snug max-h-[320px]">
        {code}
      </pre>
    </div>
  )
}

function pythonAdapterTemplate(p: {
  wsURL: string
  token: string
  name: string
  capabilities: string[]
}): string {
  const caps = JSON.stringify(p.capabilities)
  return `import asyncio, json, websockets

URL   = "${p.wsURL}"
TOKEN = "${p.token}"
NAME  = "${p.name}"
CAPS  = ${caps}

async def main():
    async with websockets.connect(
        URL, additional_headers={"X-Bridge-Token": TOKEN}
    ) as ws:
        # 1) Register — first frame must be type="register".
        await ws.send(json.dumps({
            "type": "register",
            "platform": NAME,
            "capabilities": CAPS,
        }))
        ack = json.loads(await ws.recv())
        assert ack.get("ok"), f"register rejected: {ack}"
        print("registered as", NAME)

        # 2) Demo: pretend a user just messaged us.
        await ws.send(json.dumps({
            "type": "message",
            "session_key": f"{NAME}:demo:user1",
            "conversation_id": "demo",
            "user_id": "user1",
            "user_name": "Alice",
            "text": "hello opendray",
            "reply_ctx": "msg-001",
        }))

        # 3) Pump outbound frames from opendray.
        async for raw in ws:
            frame = json.loads(raw)
            t = frame.get("type")
            if t == "ping":
                await ws.send(json.dumps({"type": "pong"}))
                continue
            print("from opendray:", t, frame)
            # TODO: render frame to your platform UI.
            # text:        frame["text"]
            # send_card:   frame["card"]   -> markdown + buttons
            # send_buttons:frame["buttons"]-> [[{text, value}, ...], ...]
            # update_message: edit msg with frame["preview_handle"]

asyncio.run(main())
`
}

function nodeAdapterTemplate(p: {
  wsURL: string
  token: string
  name: string
  capabilities: string[]
}): string {
  const caps = JSON.stringify(p.capabilities)
  return `import WebSocket from 'ws'

const URL   = '${p.wsURL}'
const TOKEN = '${p.token}'
const NAME  = '${p.name}'
const CAPS  = ${caps}

const ws = new WebSocket(URL, { headers: { 'X-Bridge-Token': TOKEN } })

ws.on('open', () => {
  ws.send(JSON.stringify({
    type: 'register',
    platform: NAME,
    capabilities: CAPS,
  }))
})

ws.on('message', (raw) => {
  const frame = JSON.parse(raw.toString())
  if (frame.type === 'register_ack') {
    if (!frame.ok) { console.error('register rejected', frame); ws.close(); return }
    console.log('registered as', NAME)
    // Demo: pretend a user just messaged us.
    ws.send(JSON.stringify({
      type: 'message',
      session_key: \`\${NAME}:demo:user1\`,
      conversation_id: 'demo',
      user_id: 'user1',
      user_name: 'Alice',
      text: 'hello opendray',
      reply_ctx: 'msg-001',
    }))
    return
  }
  if (frame.type === 'ping') { ws.send(JSON.stringify({ type: 'pong' })); return }
  console.log('from opendray:', frame.type, frame)
  // TODO: render frame to your platform UI.
})

ws.on('close', () => console.log('disconnected'))
ws.on('error', (err) => console.error(err))
`
}

function wscatTemplate(p: {
  wsURL: string
  token: string
  name: string
  capabilities: string[]
}): string {
  const reg = JSON.stringify({
    type: 'register',
    platform: p.name,
    capabilities: p.capabilities,
  })
  return `# Connect:
wscat -c "${p.wsURL}" -H "X-Bridge-Token: ${p.token}"

# Then paste this register frame:
${reg}

# Pretend the user replied "/help":
{"type":"message","session_key":"${p.name}:demo:user1","conversation_id":"demo","user_id":"user1","text":"/help","reply_ctx":"msg-001"}
`
}
