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
import { Brain, Plus, Wand2 } from 'lucide-react'

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
  return (
    <div className="rounded-md border border-border bg-card/30 p-3 text-[12px] flex items-start gap-3">
      <Brain className="size-4 mt-0.5 text-accent" />
      <div className="flex-1">
        <div className="font-medium">Ambient memory — auto-capture & inject</div>
        <div className="text-muted-foreground mt-1">
          opendray polls every live agent session every 10 seconds,
          extracts durable facts via a configurable LLM, and dedups
          before storing them in the shared memory pool. Configure
          which LLM does the extraction (Provider), when extraction
          fires (Capture rule), and what — if anything — gets
          prepended to the agent's system prompt at spawn (Injection
          profile).
        </div>
      </div>
    </div>
  )
}

// ── providers ────────────────────────────────────────────────────

function ProvidersBlock() {
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)
  const { data: providers, isLoading } = useQuery({
    queryKey: ['memory-summarizer-providers'],
    queryFn: listProviders,
  })

  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-[13px] font-medium">Summarizer providers</h3>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button size="sm">
              <Plus className="size-3 mr-1" />
              Add provider
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
        At least one enabled provider is required for capture to
        actually fire. Local options (Ollama, LM Studio,
        Integration) keep your transcripts off external networks.
      </p>
      {isLoading ? (
        <div className="text-[12px] text-muted-foreground">Loading…</div>
      ) : !providers || providers.length === 0 ? (
        <div className="rounded-md border border-dashed border-border p-4 text-center text-[12px] text-muted-foreground">
          No providers configured yet.
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
  const [testing, setTesting] = useState(false)

  async function onTest() {
    setTesting(true)
    try {
      const res = await testProvider(provider.id)
      if (res.ok) toast.success(`${provider.name}: connection OK`)
      else toast.error('Test failed', { description: res.error })
    } catch (err) {
      toast.error('Test failed', { description: errMsg(err) })
    } finally {
      setTesting(false)
    }
  }

  async function onDelete() {
    if (!window.confirm(`Delete provider "${provider.name}"?`)) return
    try {
      await deleteProvider(provider.id)
      toast.success('Provider deleted')
      onChanged()
    } catch (err) {
      toast.error('Delete failed', { description: errMsg(err) })
    }
  }

  async function onToggleEnabled() {
    try {
      await updateProvider(provider.id, { enabled: !provider.enabled })
      onChanged()
    } catch (err) {
      toast.error('Update failed', { description: errMsg(err) })
    }
  }

  async function onMakeDefault() {
    try {
      await updateProvider(provider.id, { is_default: true })
      toast.success(`${provider.name} is now the default`)
      onChanged()
    } catch (err) {
      toast.error('Update failed', { description: errMsg(err) })
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
            <span className="text-accent text-[10px]">★ default</span>
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
          Make default
        </Button>
      )}
      <Button
        onClick={onTest}
        variant="outline"
        size="sm"
        className="h-7 text-[11px]"
        disabled={testing}
      >
        {testing ? 'Testing…' : 'Test'}
      </Button>
      <Button
        onClick={onDelete}
        variant="outline"
        size="sm"
        className="h-7 px-2 text-[11px]"
      >
        Delete
      </Button>
    </div>
  )
}

function NewProviderDialog({ onCreated }: { onCreated: () => void }) {
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
      toast.error('Name is required')
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
      toast.success(`Provider ${name} created`)
      onCreated()
    } catch (err) {
      toast.error('Create failed', { description: errMsg(err) })
    } finally {
      setBusy(false)
    }
  }

  return (
    <DialogContent>
      <DialogHeader>
        <DialogTitle>Add summarizer provider</DialogTitle>
      </DialogHeader>
      <div className="flex flex-col gap-3 text-[12px]">
        <Label>Kind</Label>
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
        <Label>Name</Label>
        <Input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g. lmstudio-qwen"
        />
        <Label>Model</Label>
        <Input value={model} onChange={(e) => setModel(e.target.value)} />
        {kind !== 'anthropic' && kind !== 'openai' && kind !== 'integration' && (
          <>
            <Label>Base URL</Label>
            <Input
              value={baseURL}
              onChange={(e) => setBaseURL(e.target.value)}
            />
          </>
        )}
        {kind === 'integration' && (
          <p className="text-[11px] text-muted-foreground">
            Integration providers resolve their base URL from a
            registered integration. Configure that under
            Integrations first; advanced wiring (extra_config) is
            DB-only in this release.
          </p>
        )}
        {requiresAPIKey && (
          <>
            <Label>API key</Label>
            <Input
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              type="password"
              placeholder={kind === 'openai' ? 'sk-…' : 'sk-ant-…'}
            />
            <p className="text-[10.5px] text-muted-foreground">
              Stored encrypted (AES-GCM with the backup master
              passphrase). Never echoed back; only the fingerprint
              is shown after save.
            </p>
          </>
        )}
        <label className="flex items-center gap-2 mt-1">
          <Switch
            checked={isDefault}
            onCheckedChange={setIsDefault}
            className="scale-75"
          />
          Make this the default provider
        </label>
      </div>
      <DialogFooter>
        <Button onClick={submit} disabled={busy}>
          Create
        </Button>
      </DialogFooter>
    </DialogContent>
  )
}

// ── capture rules ────────────────────────────────────────────────

function RulesBlock() {
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)
  const { data: rules, isLoading } = useQuery({
    queryKey: ['memory-capture-rules'],
    queryFn: listCaptureRules,
  })

  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-[13px] font-medium">Capture rules</h3>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button size="sm">
              <Plus className="size-3 mr-1" />
              Add rule
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
        Each rule says "when this trigger fires, summarize new
        transcript messages and store the durable facts." Per-
        session rules override the global default. v1 ships
        4 trigger kinds.
      </p>
      {isLoading ? (
        <div className="text-[12px] text-muted-foreground">Loading…</div>
      ) : !rules || rules.length === 0 ? (
        <div className="rounded-md border border-dashed border-border p-4 text-center text-[12px] text-muted-foreground">
          No capture rules yet. Add one to enable auto-capture.
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
  const [running, setRunning] = useState(false)

  async function onRunNow() {
    setRunning(true)
    try {
      const res = await runCaptureRuleNow(rule.id)
      toast.success(`Rule fired across ${res.sessions_invoked} session(s)`)
    } catch (err) {
      toast.error('Run-now failed', { description: errMsg(err) })
    } finally {
      setRunning(false)
    }
  }

  async function onDelete() {
    if (!window.confirm(`Delete rule "${rule.name}"?`)) return
    try {
      await deleteCaptureRule(rule.id)
      toast.success('Rule deleted')
      onChanged()
    } catch (err) {
      toast.error('Delete failed', { description: errMsg(err) })
    }
  }

  const triggerSummary = useMemo(() => {
    const cfg = rule.trigger_config || {}
    switch (rule.trigger_kind) {
      case 'after_messages':
        return `every ${(cfg as any).n ?? 6} messages`
      case 'on_idle':
        return `idle ≥ ${(cfg as any).seconds ?? 60}s`
      case 'k_chars':
        return `≥ ${(cfg as any).k ?? 4000} chars`
      case 'manual':
        return 'manual only'
    }
  }, [rule])

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
              global default
            </span>
          )}
        </div>
        <div className="text-[11px] text-muted-foreground truncate">
          {triggerSummary} · scope:{rule.target_scope} · dedup:
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
        {running ? 'Running…' : 'Run now'}
      </Button>
      <Button
        onClick={onDelete}
        variant="outline"
        size="sm"
        className="h-7 px-2 text-[11px]"
      >
        Delete
      </Button>
    </div>
  )
}

function NewRuleDialog({ onCreated }: { onCreated: () => void }) {
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
      toast.error('Name is required')
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
      toast.success(`Rule ${name} created`)
      onCreated()
    } catch (err) {
      toast.error('Create failed', { description: errMsg(err) })
    } finally {
      setBusy(false)
    }
  }

  return (
    <DialogContent>
      <DialogHeader>
        <DialogTitle>Add capture rule</DialogTitle>
      </DialogHeader>
      <div className="flex flex-col gap-3 text-[12px]">
        <Label>Name</Label>
        <Input value={name} onChange={(e) => setName(e.target.value)} />
        <Label>Trigger</Label>
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
            <Label>N (messages)</Label>
            <Input
              type="number"
              value={n}
              onChange={(e) => setN(parseInt(e.target.value) || 6)}
            />
          </>
        )}
        {triggerKind === 'on_idle' && (
          <>
            <Label>Idle seconds</Label>
            <Input
              type="number"
              value={seconds}
              onChange={(e) => setSeconds(parseInt(e.target.value) || 60)}
            />
          </>
        )}
        {triggerKind === 'k_chars' && (
          <>
            <Label>K (characters)</Label>
            <Input
              type="number"
              value={k}
              onChange={(e) => setK(parseInt(e.target.value) || 4000)}
            />
          </>
        )}
        <Label>Target scope</Label>
        <select
          value={targetScope}
          onChange={(e) => setTargetScope(e.target.value as TargetScope)}
          className="h-8 rounded-md border border-border bg-background px-2"
        >
          <option value="session">session</option>
          <option value="project">project (recommended)</option>
          <option value="global">global</option>
        </select>
        <Label>Dedup threshold (0.0 – 1.0)</Label>
        <Input
          type="number"
          step="0.05"
          min="0"
          max="1"
          value={dedup}
          onChange={(e) => setDedup(parseFloat(e.target.value) || 0.85)}
        />
        <p className="text-[10.5px] text-muted-foreground">
          Higher = stricter de-duplication. 0.85 is the recommended
          sweet spot.
        </p>
      </div>
      <DialogFooter>
        <Button onClick={submit} disabled={busy}>
          Create
        </Button>
      </DialogFooter>
    </DialogContent>
  )
}

// ── injection profiles ───────────────────────────────────────────

function ProfilesBlock() {
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)
  const { data: profiles, isLoading } = useQuery({
    queryKey: ['memory-injection-profiles'],
    queryFn: listInjectionProfiles,
  })

  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-[13px] font-medium">Injection profiles</h3>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button size="sm">
              <Plus className="size-3 mr-1" />
              Add profile
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
        At spawn time opendray prepends a markdown banner of
        recent project memories to the agent's system prompt — IF
        a profile is configured. Without a profile, the model
        still uses memory_search on demand.
      </p>
      {isLoading ? (
        <div className="text-[12px] text-muted-foreground">Loading…</div>
      ) : !profiles || profiles.length === 0 ? (
        <div className="rounded-md border border-dashed border-border p-4 text-center text-[12px] text-muted-foreground">
          No injection profile. Memories are not auto-injected at
          spawn — model still uses memory_search.
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
  async function onDelete() {
    if (!window.confirm(`Delete this injection profile?`)) return
    try {
      await deleteInjectionProfile(profile.id)
      toast.success('Profile deleted')
      onChanged()
    } catch (err) {
      toast.error('Delete failed', { description: errMsg(err) })
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
              global default
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
        Delete
      </Button>
    </div>
  )
}

function NewProfileDialog({ onCreated }: { onCreated: () => void }) {
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
      toast.success('Profile created')
      onCreated()
    } catch (err) {
      toast.error('Create failed', { description: errMsg(err) })
    } finally {
      setBusy(false)
    }
  }

  return (
    <DialogContent>
      <DialogHeader>
        <DialogTitle>Add injection profile</DialogTitle>
      </DialogHeader>
      <div className="flex flex-col gap-3 text-[12px]">
        <Label>Strategy</Label>
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
            <Label>K (top memories to inject)</Label>
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
          One profile per session_id (or global default).
          Per-session profiles can be added later via API; UI
          currently only manages the global default.
        </p>
      </div>
      <DialogFooter>
        <Button onClick={submit} disabled={busy}>
          Create
        </Button>
      </DialogFooter>
    </DialogContent>
  )
}

// ── token cost panel ─────────────────────────────────────────────

function CostBlock() {
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
      <h3 className="text-[13px] font-medium mb-2">Token cost (all-time)</h3>
      <p className="text-[11.5px] text-muted-foreground mb-2">
        Per-provider summary aggregated from{' '}
        <code className="text-foreground">memory_summarizer_calls</code>.
        Local providers (Ollama, LM Studio, Integration) are
        priced as $0 — operator owns hardware cost.
      </p>
      {enabledProviders.length === 0 ? (
        <div className="rounded-md border border-dashed border-border p-4 text-center text-[12px] text-muted-foreground">
          No enabled providers — no cost data.
        </div>
      ) : (
        <div className="rounded-md border border-border overflow-hidden">
          <table className="w-full text-[11.5px]">
            <thead className="bg-card/50 text-[11px] text-muted-foreground">
              <tr>
                <th className="px-3 py-1.5 text-left font-medium">Provider</th>
                <th className="px-3 py-1.5 text-right font-medium">Calls</th>
                <th className="px-3 py-1.5 text-right font-medium">In tokens</th>
                <th className="px-3 py-1.5 text-right font-medium">Out tokens</th>
                <th className="px-3 py-1.5 text-right font-medium">USD est.</th>
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
