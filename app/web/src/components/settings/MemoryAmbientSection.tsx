// MemoryAmbientSection — Settings → Memory · Ambient panel.
//
// Owns three lists with inline Add / Test / Run-now / Delete
// affordances:
//   · Summarizer providers (Anthropic / OpenAI / LM Studio /
//     Ollama / Integration) — backed by /memory-summarizer-providers
//   · Capture rules — backed by /memory-capture-rules
//   · Injection profiles — backed by /memory-injection-profiles
//
// Plus a Token cost panel at the bottom that aggregates the
// summarizer call log per-provider.
//
// Mirrors the BackupsSection control-center pattern: status →
// where things go → schedules → advanced.

import { useEffect, useMemo, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Brain, Loader2, Pencil, Plus, RefreshCw, Wand2 } from 'lucide-react'
import { Trans, useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'

import {
  type CaptureRule,
  type CostSummary,
  type InjectionProfile,
  type InjectionStrategy,
  type ProviderKind,
  type SummarizerProvider,
  type TriggerKind,
  type TargetScope,
  PROVIDER_KIND_LABELS,
  STRATEGY_LABELS,
  TRIGGER_KIND_LABELS,
  createCaptureRule,
  createInjectionProfile,
  createProvider,
  deleteCaptureRule,
  deleteInjectionProfile,
  deleteProvider,
  listCaptureRules,
  listInjectionProfiles,
  listProviders,
  providerCost,
  runCaptureRuleNow,
  testProvider,
  updateProvider,
} from '@/lib/memoryAmbient'
import { probeEmbeddingEndpoint } from '@/lib/memory'
import { APIError } from '@/lib/api'

export function MemoryAmbientSection() {
  return (
    <div className="flex flex-col gap-8">
      <SectionHeader />
      <ProvidersBlock />
      <RulesBlock />
      <ProfilesBlock />
      <CostBlock />
    </div>
  )
}

function SectionHeader() {
  const { t } = useTranslation()
  return (
    <div className="rounded-md border border-border bg-card/30 p-3 text-[12px] flex items-start gap-3">
      <Brain className="size-4 mt-0.5 text-accent" />
      <div className="flex-1">
        <div className="font-medium">{t('web.memoryAmbient.header.title')}</div>
        <div className="text-muted-foreground mt-1">
          {t('web.memoryAmbient.header.body')}
        </div>
      </div>
    </div>
  )
}

// ── providers ────────────────────────────────────────────────────

export function ProvidersBlock() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)
  const { data: providers, isLoading } = useQuery({
    queryKey: ['memory-summarizer-providers'],
    queryFn: listProviders,
  })

  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-[13px] font-medium">{t('web.memoryAmbient.providers.title')}</h3>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button size="sm">
              <Plus className="size-3 mr-1" />
              {t('web.memoryAmbient.providers.addButton')}
            </Button>
          </DialogTrigger>
          <NewProviderDialog
            onCreated={() => {
              setOpen(false)
              qc.invalidateQueries({ queryKey: ['memory-summarizer-providers'] })
            }}
          />
        </Dialog>
      </div>
      <p className="text-[11.5px] text-muted-foreground mb-2">
        {t('web.memoryAmbient.providers.intro')}
      </p>
      {isLoading ? (
        <div className="text-[12px] text-muted-foreground">{t('web.memoryAmbient.loading')}</div>
      ) : !providers || providers.length === 0 ? (
        <div className="rounded-md border border-dashed border-border p-4 text-center text-[12px] text-muted-foreground">
          {t('web.memoryAmbient.providers.empty')}
        </div>
      ) : (
        <div className="flex flex-col gap-1.5">
          {providers.map((p) => (
            <ProviderRow
              key={p.id}
              provider={p}
              onChanged={() =>
                qc.invalidateQueries({
                  queryKey: ['memory-summarizer-providers'],
                })
              }
            />
          ))}
        </div>
      )}
    </div>
  )
}

function ProviderRow({
  provider,
  onChanged,
}: {
  provider: SummarizerProvider
  onChanged: () => void
}) {
  const { t } = useTranslation()
  const [testing, setTesting] = useState(false)

  async function onTest() {
    setTesting(true)
    try {
      const res = await testProvider(provider.id)
      if (res.ok) toast.success(t('web.memoryAmbient.providers.row.testOk', { name: provider.name }))
      else toast.error(t('web.memoryAmbient.providers.row.testFailedToast'), { description: res.error })
    } catch (err) {
      toast.error(t('web.memoryAmbient.providers.row.testFailedToast'), { description: errMsg(err) })
    } finally {
      setTesting(false)
    }
  }

  async function onDelete() {
    if (!window.confirm(t('web.memoryAmbient.providers.row.deleteConfirm', { name: provider.name }))) return
    try {
      await deleteProvider(provider.id)
      toast.success(t('web.memoryAmbient.providers.row.deletedToast'))
      onChanged()
    } catch (err) {
      toast.error(t('web.memoryAmbient.providers.row.deleteFailedToast'), { description: errMsg(err) })
    }
  }

  async function onToggleEnabled() {
    try {
      await updateProvider(provider.id, { enabled: !provider.enabled })
      onChanged()
    } catch (err) {
      toast.error(t('web.memoryAmbient.providers.row.updateFailedToast'), { description: errMsg(err) })
    }
  }

  async function onMakeDefault() {
    try {
      await updateProvider(provider.id, { is_default: true })
      toast.success(t('web.memoryAmbient.providers.row.madeDefaultToast', { name: provider.name }))
      onChanged()
    } catch (err) {
      toast.error(t('web.memoryAmbient.providers.row.updateFailedToast'), { description: errMsg(err) })
    }
  }

  return (
    <div className="flex items-center gap-3 p-2.5 rounded-md border border-border bg-card/30">
      <span className="px-2 py-0.5 rounded border border-border text-[10px] uppercase tracking-wide bg-card font-mono">
        {provider.kind}
      </span>
      <div className="flex-1 min-w-0">
        <div className="font-mono text-[11.5px] truncate">
          {provider.name}{' '}
          {provider.is_default && (
            <span className="text-accent text-[10px]">
              {t('web.memoryAmbient.providers.row.defaultBadge')}
            </span>
          )}
        </div>
        <div className="text-[11px] text-muted-foreground truncate">
          {provider.model}
          {provider.base_url && ` · ${provider.base_url}`}
          {provider.api_key_set &&
            provider.api_key_fingerprint &&
            ` · key:${provider.api_key_fingerprint.slice(0, 8)}`}
        </div>
      </div>
      <Switch
        checked={provider.enabled}
        onCheckedChange={onToggleEnabled}
        className="scale-75"
      />
      {!provider.is_default && provider.enabled && (
        <Button
          onClick={onMakeDefault}
          variant="outline"
          size="sm"
          className="h-7 text-[11px]"
        >
          {t('web.memoryAmbient.providers.row.makeDefault')}
        </Button>
      )}
      <ChangeModelDialog provider={provider} onSaved={onChanged} />
      <Button
        onClick={onTest}
        variant="outline"
        size="sm"
        className="h-7 text-[11px]"
        disabled={testing}
      >
        {testing
          ? t('web.memoryAmbient.providers.row.testing')
          : t('web.memoryAmbient.providers.row.test')}
      </Button>
      <Button
        onClick={onDelete}
        variant="outline"
        size="sm"
        className="h-7 px-2 text-[11px]"
      >
        {t('web.memoryAmbient.providers.row.delete')}
      </Button>
    </div>
  )
}

// localModelKinds are the provider kinds whose base_url points at a
// local OpenAI-compatible server we can enumerate models from
// (LM Studio / Ollama). Cloud kinds keep the plain text input — their
// catalogs are huge and key-gated.
function isLocalModelKind(kind: ProviderKind): boolean {
  return kind === 'ollama' || kind === 'lmstudio'
}

// LocalModelSelect — pick a model that actually exists on the local
// endpoint instead of typing a name blind, mirroring the Agent
// workers' ModelPicker. Probes <base_url>/models (with the Ollama
// /api/tags fallback the backend probe already implements); endpoint
// unreachable → plain input with a hint, so nothing is ever blocked
// on the probe.
function LocalModelSelect({
  baseURL,
  value,
  onChange,
}: {
  baseURL: string
  value: string
  onChange: (v: string) => void
}) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [customMode, setCustomMode] = useState(false)
  const probe = useQuery({
    queryKey: ['endpoint-models', baseURL],
    queryFn: () => probeEmbeddingEndpoint(baseURL),
    enabled: baseURL.trim() !== '',
    staleTime: 30_000,
    retry: false,
  })
  const reachable = probe.data?.reachable === true
  const models = reachable ? (probe.data?.models ?? []) : []

  // Default to the first advertised model when the field is empty —
  // saves the "type a guess, watch the call fail" round-trip.
  useEffect(() => {
    if (reachable && models.length > 0 && value === '') {
      onChange(models[0])
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [reachable, models.length])

  if (probe.isLoading && baseURL.trim() !== '') {
    return (
      <div className="flex items-center gap-2">
        <Input value={value} onChange={(e) => onChange(e.target.value)} />
        <Loader2 className="size-3 shrink-0 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!reachable || models.length === 0 || customMode) {
    return (
      <div className="flex flex-col gap-1">
        <div className="flex items-center gap-2">
          <Input
            value={value}
            onChange={(e) => onChange(e.target.value)}
            autoFocus={customMode}
          />
          {reachable && models.length > 0 && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="h-8 shrink-0 text-[11px]"
              onClick={() => setCustomMode(false)}
            >
              {t('web.memoryAmbient.providers.modelSelect.backToList')}
            </Button>
          )}
        </div>
        {baseURL.trim() !== '' && !reachable && (
          <p className="text-[10.5px] text-muted-foreground">
            {t('web.memoryAmbient.providers.modelSelect.unreachable')}
          </p>
        )}
        {reachable && models.length === 0 && (
          <p className="text-[10.5px] text-muted-foreground">
            {t('web.memoryAmbient.providers.modelSelect.none')}
          </p>
        )}
      </div>
    )
  }

  const listed = models.includes(value)
  return (
    <div className="flex items-center gap-2">
      <select
        value={listed ? value : '__keep__'}
        onChange={(e) => {
          if (e.target.value === '__custom__') {
            setCustomMode(true)
            return
          }
          if (e.target.value !== '__keep__') onChange(e.target.value)
        }}
        className="h-8 min-w-0 flex-1 rounded-md border border-border bg-background px-2"
      >
        {!listed && value !== '' && (
          <option value="__keep__">
            {value} · {t('web.memoryAmbient.providers.modelSelect.notOnEndpoint')}
          </option>
        )}
        {models.map((m) => (
          <option key={m} value={m}>
            {m}
          </option>
        ))}
        <option value="__custom__">
          {t('web.memoryAmbient.providers.modelSelect.custom')}
        </option>
      </select>
      <Button
        type="button"
        variant="outline"
        size="sm"
        className="h-8 w-8 shrink-0 p-0"
        title={t('web.memoryAmbient.providers.modelSelect.refresh')}
        onClick={() =>
          qc.invalidateQueries({ queryKey: ['endpoint-models', baseURL] })
        }
      >
        <RefreshCw className="size-3" />
      </Button>
    </div>
  )
}

// ChangeModelDialog — per-row model switch for an existing provider.
// Before this the only way to change a provider's model was delete +
// recreate; with local endpoints the picker lists what's installed.
function ChangeModelDialog({
  provider,
  onSaved,
}: {
  provider: SummarizerProvider
  onSaved: () => void
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [model, setModel] = useState(provider.model)
  const [busy, setBusy] = useState(false)

  async function submit() {
    setBusy(true)
    try {
      await updateProvider(provider.id, { model })
      toast.success(
        t('web.memoryAmbient.providers.modelSelect.savedToast', {
          name: provider.name,
          model,
        }),
      )
      setOpen(false)
      onSaved()
    } catch (err) {
      toast.error(t('web.memoryAmbient.providers.row.updateFailedToast'), {
        description: errMsg(err),
      })
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        setOpen(v)
        if (v) setModel(provider.model)
      }}
    >
      <DialogTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          className="h-7 px-2 text-[11px]"
          title={t('web.memoryAmbient.providers.modelSelect.editTitle')}
        >
          <Pencil className="size-3" />
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {t('web.memoryAmbient.providers.modelSelect.dialogTitle', {
              name: provider.name,
            })}
          </DialogTitle>
        </DialogHeader>
        <div className="flex flex-col gap-3 text-[12px]">
          <Label>{t('web.memoryAmbient.providers.dialog.modelLabel')}</Label>
          {isLocalModelKind(provider.kind) && provider.base_url ? (
            <LocalModelSelect
              baseURL={provider.base_url}
              value={model}
              onChange={setModel}
            />
          ) : (
            <Input value={model} onChange={(e) => setModel(e.target.value)} />
          )}
        </div>
        <DialogFooter>
          <Button onClick={submit} disabled={busy || model.trim() === ''}>
            {t('web.memoryAmbient.providers.modelSelect.save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function NewProviderDialog({ onCreated }: { onCreated: () => void }) {
  const { t } = useTranslation()
  const [kind, setKind] = useState<ProviderKind>('ollama')
  const [name, setName] = useState('')
  const [model, setModel] = useState('')
  const [baseURL, setBaseURL] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [isDefault, setIsDefault] = useState(false)
  const [busy, setBusy] = useState(false)

  // Sensible defaults per kind.
  useEffect(() => {
    switch (kind) {
      case 'ollama':
        setBaseURL('http://localhost:11434')
        setModel('qwen2.5:7b')
        break
      case 'lmstudio':
        setBaseURL('http://localhost:1234/v1')
        setModel('qwen2.5-7b-instruct')
        break
      case 'anthropic':
        setBaseURL('')
        setModel('claude-haiku-4-5')
        break
      case 'openai':
        setBaseURL('')
        setModel('gpt-4o-mini')
        break
      case 'integration':
        setBaseURL('')
        setModel('via-integration')
        break
    }
  }, [kind])

  const requiresAPIKey = kind === 'anthropic' || kind === 'openai'

  async function submit() {
    if (!name) {
      toast.error(t('web.memoryAmbient.providers.dialog.nameRequiredToast'))
      return
    }
    setBusy(true)
    try {
      await createProvider({
        kind,
        name,
        model,
        base_url: baseURL || undefined,
        api_key: requiresAPIKey ? apiKey : undefined,
        is_default: isDefault,
        enabled: true,
      })
      toast.success(t('web.memoryAmbient.providers.dialog.createdToast', { name }))
      onCreated()
    } catch (err) {
      toast.error(t('web.memoryAmbient.providers.dialog.createFailedToast'), { description: errMsg(err) })
    } finally {
      setBusy(false)
    }
  }

  return (
    <DialogContent>
      <DialogHeader>
        <DialogTitle>{t('web.memoryAmbient.providers.dialog.title')}</DialogTitle>
      </DialogHeader>
      <div className="flex flex-col gap-3 text-[12px]">
        <Label>{t('web.memoryAmbient.providers.dialog.kindLabel')}</Label>
        <select
          value={kind}
          onChange={(e) => setKind(e.target.value as ProviderKind)}
          className="h-8 rounded-md border border-border bg-background px-2"
        >
          {(Object.keys(PROVIDER_KIND_LABELS) as ProviderKind[]).map((k) => (
            <option key={k} value={k}>
              {PROVIDER_KIND_LABELS[k]}
            </option>
          ))}
        </select>
        <Label>{t('web.memoryAmbient.providers.dialog.nameLabel')}</Label>
        <Input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder={t('web.memoryAmbient.providers.dialog.namePlaceholder')}
        />
        {kind !== 'anthropic' && kind !== 'openai' && kind !== 'integration' && (
          <>
            <Label>{t('web.memoryAmbient.providers.dialog.baseUrlLabel')}</Label>
            <Input
              value={baseURL}
              onChange={(e) => setBaseURL(e.target.value)}
            />
          </>
        )}
        <Label>{t('web.memoryAmbient.providers.dialog.modelLabel')}</Label>
        {isLocalModelKind(kind) ? (
          <LocalModelSelect baseURL={baseURL} value={model} onChange={setModel} />
        ) : (
          <Input value={model} onChange={(e) => setModel(e.target.value)} />
        )}
        {kind === 'integration' && (
          <p className="text-[11px] text-muted-foreground">
            {t('web.memoryAmbient.providers.dialog.integrationNote')}
          </p>
        )}
        {requiresAPIKey && (
          <>
            <Label>{t('web.memoryAmbient.providers.dialog.apiKeyLabel')}</Label>
            <Input
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              type="password"
              placeholder={kind === 'openai' ? 'sk-…' : 'sk-ant-…'}
            />
            <p className="text-[10.5px] text-muted-foreground">
              {t('web.memoryAmbient.providers.dialog.apiKeyHint')}
            </p>
          </>
        )}
        <label className="flex items-center gap-2 mt-1">
          <Switch
            checked={isDefault}
            onCheckedChange={setIsDefault}
            className="scale-75"
          />
          {t('web.memoryAmbient.providers.dialog.makeDefaultLabel')}
        </label>
      </div>
      <DialogFooter>
        <Button onClick={submit} disabled={busy}>
          {t('web.memoryAmbient.providers.dialog.create')}
        </Button>
      </DialogFooter>
    </DialogContent>
  )
}

// ── capture rules ────────────────────────────────────────────────

export function RulesBlock() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)
  const { data: rules, isLoading } = useQuery({
    queryKey: ['memory-capture-rules'],
    queryFn: listCaptureRules,
  })

  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-[13px] font-medium">{t('web.memoryAmbient.rules.title')}</h3>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button size="sm">
              <Plus className="size-3 mr-1" />
              {t('web.memoryAmbient.rules.addButton')}
            </Button>
          </DialogTrigger>
          <NewRuleDialog
            onCreated={() => {
              setOpen(false)
              qc.invalidateQueries({ queryKey: ['memory-capture-rules'] })
            }}
          />
        </Dialog>
      </div>
      <p className="text-[11.5px] text-muted-foreground mb-2">
        {t('web.memoryAmbient.rules.intro')}
      </p>
      {isLoading ? (
        <div className="text-[12px] text-muted-foreground">{t('web.memoryAmbient.loading')}</div>
      ) : !rules || rules.length === 0 ? (
        <div className="rounded-md border border-dashed border-border p-4 text-center text-[12px] text-muted-foreground">
          {t('web.memoryAmbient.rules.empty')}
        </div>
      ) : (
        <div className="flex flex-col gap-1.5">
          {rules.map((r) => (
            <RuleRow
              key={r.id}
              rule={r}
              onChanged={() =>
                qc.invalidateQueries({ queryKey: ['memory-capture-rules'] })
              }
            />
          ))}
        </div>
      )}
    </div>
  )
}

function RuleRow({
  rule,
  onChanged,
}: {
  rule: CaptureRule
  onChanged: () => void
}) {
  const { t } = useTranslation()
  const [running, setRunning] = useState(false)

  async function onRunNow() {
    setRunning(true)
    try {
      const res = await runCaptureRuleNow(rule.id)
      toast.success(t('web.memoryAmbient.rules.row.firedToast', { sessions: res.sessions_invoked }))
    } catch (err) {
      toast.error(t('web.memoryAmbient.rules.row.runNowFailedToast'), { description: errMsg(err) })
    } finally {
      setRunning(false)
    }
  }

  async function onDelete() {
    if (!window.confirm(t('web.memoryAmbient.rules.row.deleteConfirm', { name: rule.name }))) return
    try {
      await deleteCaptureRule(rule.id)
      toast.success(t('web.memoryAmbient.rules.row.deletedToast'))
      onChanged()
    } catch (err) {
      toast.error(t('web.memoryAmbient.rules.row.deleteFailedToast'), { description: errMsg(err) })
    }
  }

  const triggerSummary = useMemo(() => {
    const cfg = rule.trigger_config || {}
    switch (rule.trigger_kind) {
      case 'after_messages':
        return t('web.memoryAmbient.rules.row.summary.afterMessages', { n: (cfg as any).n ?? 6 })
      case 'on_idle':
        return t('web.memoryAmbient.rules.row.summary.onIdle', { seconds: (cfg as any).seconds ?? 60 })
      case 'k_chars':
        return t('web.memoryAmbient.rules.row.summary.kChars', { k: (cfg as any).k ?? 4000 })
      case 'manual':
        return t('web.memoryAmbient.rules.row.summary.manual')
    }
  }, [rule, t])

  return (
    <div className="flex items-center gap-3 p-2.5 rounded-md border border-border bg-card/30">
      <span className="px-2 py-0.5 rounded border border-border text-[10px] uppercase tracking-wide bg-card font-mono">
        {rule.trigger_kind}
      </span>
      <div className="flex-1 min-w-0">
        <div className="font-mono text-[11.5px] truncate">
          {rule.name}
          {!rule.session_id && (
            <span className="ml-2 text-[10px] text-muted-foreground">
              {t('web.memoryAmbient.rules.row.globalDefault')}
            </span>
          )}
        </div>
        <div className="text-[11px] text-muted-foreground truncate">
          {triggerSummary} · {t('web.memoryAmbient.rules.row.scopeLabel')}{rule.target_scope} ·{' '}
          {t('web.memoryAmbient.rules.row.dedupLabel')}
          {rule.dedup_threshold.toFixed(2)}
        </div>
      </div>
      <Button
        onClick={onRunNow}
        variant="outline"
        size="sm"
        className="h-7 text-[11px]"
        disabled={running}
      >
        <Wand2 className="size-3 mr-1" />
        {running
          ? t('web.memoryAmbient.rules.row.running')
          : t('web.memoryAmbient.rules.row.runNow')}
      </Button>
      <Button
        onClick={onDelete}
        variant="outline"
        size="sm"
        className="h-7 px-2 text-[11px]"
      >
        {t('web.memoryAmbient.rules.row.delete')}
      </Button>
    </div>
  )
}

function NewRuleDialog({ onCreated }: { onCreated: () => void }) {
  const { t } = useTranslation()
  const [name, setName] = useState('global-default')
  const [triggerKind, setTriggerKind] = useState<TriggerKind>('after_messages')
  const [n, setN] = useState(6)
  const [seconds, setSeconds] = useState(60)
  const [k, setK] = useState(4000)
  const [targetScope, setTargetScope] = useState<TargetScope>('project')
  const [dedup, setDedup] = useState(0.85)
  const [busy, setBusy] = useState(false)

  async function submit() {
    if (!name) {
      toast.error(t('web.memoryAmbient.rules.dialog.nameRequiredToast'))
      return
    }
    let trigger_config: Record<string, unknown> = {}
    switch (triggerKind) {
      case 'after_messages':
        trigger_config = { n }
        break
      case 'on_idle':
        trigger_config = { seconds }
        break
      case 'k_chars':
        trigger_config = { k }
        break
      case 'manual':
        trigger_config = {}
        break
    }
    setBusy(true)
    try {
      await createCaptureRule({
        name,
        trigger_kind: triggerKind,
        trigger_config,
        target_scope: targetScope,
        dedup_threshold: dedup,
        enabled: true,
      })
      toast.success(t('web.memoryAmbient.rules.dialog.createdToast', { name }))
      onCreated()
    } catch (err) {
      toast.error(t('web.memoryAmbient.rules.dialog.createFailedToast'), { description: errMsg(err) })
    } finally {
      setBusy(false)
    }
  }

  return (
    <DialogContent>
      <DialogHeader>
        <DialogTitle>{t('web.memoryAmbient.rules.dialog.title')}</DialogTitle>
      </DialogHeader>
      <div className="flex flex-col gap-3 text-[12px]">
        <Label>{t('web.memoryAmbient.rules.dialog.nameLabel')}</Label>
        <Input value={name} onChange={(e) => setName(e.target.value)} />
        <Label>{t('web.memoryAmbient.rules.dialog.triggerLabel')}</Label>
        <select
          value={triggerKind}
          onChange={(e) => setTriggerKind(e.target.value as TriggerKind)}
          className="h-8 rounded-md border border-border bg-background px-2"
        >
          {(Object.keys(TRIGGER_KIND_LABELS) as TriggerKind[]).map((k) => (
            <option key={k} value={k}>
              {TRIGGER_KIND_LABELS[k]}
            </option>
          ))}
        </select>
        {triggerKind === 'after_messages' && (
          <>
            <Label>{t('web.memoryAmbient.rules.dialog.nLabel')}</Label>
            <Input
              type="number"
              value={n}
              onChange={(e) => setN(parseInt(e.target.value) || 6)}
            />
          </>
        )}
        {triggerKind === 'on_idle' && (
          <>
            <Label>{t('web.memoryAmbient.rules.dialog.idleLabel')}</Label>
            <Input
              type="number"
              value={seconds}
              onChange={(e) => setSeconds(parseInt(e.target.value) || 60)}
            />
          </>
        )}
        {triggerKind === 'k_chars' && (
          <>
            <Label>{t('web.memoryAmbient.rules.dialog.kLabel')}</Label>
            <Input
              type="number"
              value={k}
              onChange={(e) => setK(parseInt(e.target.value) || 4000)}
            />
          </>
        )}
        <Label>{t('web.memoryAmbient.rules.dialog.scopeLabel')}</Label>
        <select
          value={targetScope}
          onChange={(e) => setTargetScope(e.target.value as TargetScope)}
          className="h-8 rounded-md border border-border bg-background px-2"
        >
          <option value="project">{t('web.memoryAmbient.rules.dialog.scopeProject')}</option>
          <option value="global">{t('web.memoryAmbient.rules.dialog.scopeGlobal')}</option>
        </select>
        <Label>{t('web.memoryAmbient.rules.dialog.dedupLabel')}</Label>
        <Input
          type="number"
          step="0.05"
          min="0"
          max="1"
          value={dedup}
          onChange={(e) => setDedup(parseFloat(e.target.value) || 0.85)}
        />
        <p className="text-[10.5px] text-muted-foreground">
          {t('web.memoryAmbient.rules.dialog.dedupHint')}
        </p>
      </div>
      <DialogFooter>
        <Button onClick={submit} disabled={busy}>
          {t('web.memoryAmbient.rules.dialog.create')}
        </Button>
      </DialogFooter>
    </DialogContent>
  )
}

// ── injection profiles ───────────────────────────────────────────

export function ProfilesBlock() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)
  const { data: profiles, isLoading } = useQuery({
    queryKey: ['memory-injection-profiles'],
    queryFn: listInjectionProfiles,
  })

  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-[13px] font-medium">{t('web.memoryAmbient.profiles.title')}</h3>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button size="sm">
              <Plus className="size-3 mr-1" />
              {t('web.memoryAmbient.profiles.addButton')}
            </Button>
          </DialogTrigger>
          <NewProfileDialog
            onCreated={() => {
              setOpen(false)
              qc.invalidateQueries({ queryKey: ['memory-injection-profiles'] })
            }}
          />
        </Dialog>
      </div>
      <p className="text-[11.5px] text-muted-foreground mb-2">
        {t('web.memoryAmbient.profiles.intro')}
      </p>
      {isLoading ? (
        <div className="text-[12px] text-muted-foreground">{t('web.memoryAmbient.loading')}</div>
      ) : !profiles || profiles.length === 0 ? (
        <div className="rounded-md border border-dashed border-border p-4 text-center text-[12px] text-muted-foreground">
          {t('web.memoryAmbient.profiles.empty')}
        </div>
      ) : (
        <div className="flex flex-col gap-1.5">
          {profiles.map((p) => (
            <ProfileRow
              key={p.id}
              profile={p}
              onChanged={() =>
                qc.invalidateQueries({
                  queryKey: ['memory-injection-profiles'],
                })
              }
            />
          ))}
        </div>
      )}
    </div>
  )
}

function ProfileRow({
  profile,
  onChanged,
}: {
  profile: InjectionProfile
  onChanged: () => void
}) {
  const { t } = useTranslation()
  async function onDelete() {
    if (!window.confirm(t('web.memoryAmbient.profiles.row.deleteConfirm'))) return
    try {
      await deleteInjectionProfile(profile.id)
      toast.success(t('web.memoryAmbient.profiles.row.deletedToast'))
      onChanged()
    } catch (err) {
      toast.error(t('web.memoryAmbient.profiles.row.deleteFailedToast'), { description: errMsg(err) })
    }
  }
  return (
    <div className="flex items-center gap-3 p-2.5 rounded-md border border-border bg-card/30">
      <span className="px-2 py-0.5 rounded border border-border text-[10px] uppercase tracking-wide bg-card font-mono">
        {profile.strategy_kind}
      </span>
      <div className="flex-1 min-w-0">
        <div className="font-mono text-[11.5px] truncate">
          {profile.id}
          {!profile.session_id && (
            <span className="ml-2 text-[10px] text-muted-foreground">
              {t('web.memoryAmbient.profiles.row.globalDefault')}
            </span>
          )}
        </div>
        <div className="text-[11px] text-muted-foreground truncate">
          {STRATEGY_LABELS[profile.strategy_kind]}
          {profile.config &&
            (profile.config as any).k != null &&
            ` · k=${(profile.config as any).k}`}
        </div>
      </div>
      <Button
        onClick={onDelete}
        variant="outline"
        size="sm"
        className="h-7 px-2 text-[11px]"
      >
        {t('web.memoryAmbient.profiles.row.delete')}
      </Button>
    </div>
  )
}

function NewProfileDialog({ onCreated }: { onCreated: () => void }) {
  const { t } = useTranslation()
  const [strategy, setStrategy] = useState<InjectionStrategy>('top_k_recent')
  const [k, setK] = useState(5)
  const [busy, setBusy] = useState(false)

  const usesK =
    strategy === 'top_k_recent' || strategy === 'top_k_relevant'

  async function submit() {
    setBusy(true)
    try {
      await createInjectionProfile({
        strategy_kind: strategy,
        config: usesK ? { k } : {},
      })
      toast.success(t('web.memoryAmbient.profiles.dialog.createdToast'))
      onCreated()
    } catch (err) {
      toast.error(t('web.memoryAmbient.profiles.dialog.createFailedToast'), { description: errMsg(err) })
    } finally {
      setBusy(false)
    }
  }

  return (
    <DialogContent>
      <DialogHeader>
        <DialogTitle>{t('web.memoryAmbient.profiles.dialog.title')}</DialogTitle>
      </DialogHeader>
      <div className="flex flex-col gap-3 text-[12px]">
        <Label>{t('web.memoryAmbient.profiles.dialog.strategyLabel')}</Label>
        <select
          value={strategy}
          onChange={(e) => setStrategy(e.target.value as InjectionStrategy)}
          className="h-8 rounded-md border border-border bg-background px-2"
        >
          {(Object.keys(STRATEGY_LABELS) as InjectionStrategy[]).map((k) => (
            <option key={k} value={k}>
              {STRATEGY_LABELS[k]}
            </option>
          ))}
        </select>
        {usesK && (
          <>
            <Label>{t('web.memoryAmbient.profiles.dialog.kLabel')}</Label>
            <Input
              type="number"
              min="1"
              max="50"
              value={k}
              onChange={(e) => setK(parseInt(e.target.value) || 5)}
            />
          </>
        )}
        <p className="text-[10.5px] text-muted-foreground">
          {t('web.memoryAmbient.profiles.dialog.hint')}
        </p>
      </div>
      <DialogFooter>
        <Button onClick={submit} disabled={busy}>
          {t('web.memoryAmbient.profiles.dialog.create')}
        </Button>
      </DialogFooter>
    </DialogContent>
  )
}

// ── token cost panel ─────────────────────────────────────────────

export function CostBlock() {
  const { t } = useTranslation()
  const { data: providers } = useQuery({
    queryKey: ['memory-summarizer-providers'],
    queryFn: listProviders,
  })

  const enabledProviders = useMemo(
    () => (providers ?? []).filter((p) => p.enabled),
    [providers],
  )

  return (
    <div>
      <h3 className="text-[13px] font-medium mb-2">{t('web.memoryAmbient.cost.title')}</h3>
      <p className="text-[11.5px] text-muted-foreground mb-2">
        <Trans
          i18nKey="web.memoryAmbient.cost.intro"
          components={{ 1: <code className="text-foreground" /> }}
        />
      </p>
      {enabledProviders.length === 0 ? (
        <div className="rounded-md border border-dashed border-border p-4 text-center text-[12px] text-muted-foreground">
          {t('web.memoryAmbient.cost.empty')}
        </div>
      ) : (
        <div className="rounded-md border border-border overflow-hidden">
          <table className="w-full text-[11.5px]">
            <thead className="bg-card/50 text-[11px] text-muted-foreground">
              <tr>
                <th className="px-3 py-1.5 text-left font-medium">{t('web.memoryAmbient.cost.columns.provider')}</th>
                <th className="px-3 py-1.5 text-right font-medium">{t('web.memoryAmbient.cost.columns.calls')}</th>
                <th className="px-3 py-1.5 text-right font-medium">{t('web.memoryAmbient.cost.columns.inTokens')}</th>
                <th className="px-3 py-1.5 text-right font-medium">{t('web.memoryAmbient.cost.columns.outTokens')}</th>
                <th className="px-3 py-1.5 text-right font-medium">{t('web.memoryAmbient.cost.columns.usdEst')}</th>
              </tr>
            </thead>
            <tbody>
              {enabledProviders.map((p) => (
                <CostRow key={p.id} provider={p} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

function CostRow({ provider }: { provider: SummarizerProvider }) {
  const { data: cost } = useQuery({
    queryKey: ['memory-summarizer-cost', provider.id],
    queryFn: () => providerCost(provider.id),
  })

  return (
    <tr className="border-t border-border/40">
      <td className="px-3 py-1.5">
        <span className="font-mono">{provider.name}</span>
        <span className="ml-2 text-[10px] text-muted-foreground">
          {provider.kind}
        </span>
      </td>
      <td className="px-3 py-1.5 text-right font-mono">
        {cost?.Calls ?? '–'}
      </td>
      <td className="px-3 py-1.5 text-right font-mono">
        {cost?.InputTokens?.toLocaleString() ?? '–'}
      </td>
      <td className="px-3 py-1.5 text-right font-mono">
        {cost?.OutputTokens?.toLocaleString() ?? '–'}
      </td>
      <td className="px-3 py-1.5 text-right font-mono">
        {cost?.EstimatedUSD != null
          ? `$${cost.EstimatedUSD.toFixed(4)}`
          : '–'}
      </td>
    </tr>
  )
}

// ── helpers ──────────────────────────────────────────────────────

function errMsg(err: unknown): string {
  if (err instanceof APIError) {
    if (
      err.body &&
      typeof err.body === 'object' &&
      'error' in err.body &&
      typeof (err.body as { error: unknown }).error === 'string'
    ) {
      return (err.body as { error: string }).error
    }
  }
  if (err instanceof Error) return err.message
  return 'Unknown error'
}

// silence unused import — keep CostSummary in the type-tree
export type _unusedCostSummary = CostSummary
