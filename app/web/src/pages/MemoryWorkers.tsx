// /memory/workers — M25 per-task worker settings.
//
// Surfaces the four memory-system LLM touchpoints (gatekeeper,
// cleaner, gitactivity, transcript) as a configurable list:
// each row picks summarizer vs agent + the underlying provider,
// runs a connectivity test, and shows recent invocation latency.

import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import {
  AlertTriangle,
  CheckCircle2,
  ChevronRight,
  Loader2,
  Play,
  Save,
} from 'lucide-react'
import { Trans, useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import {
  type AgentProviderID,
  type CallSummary,
  type WorkerConfig,
  type WorkerKind,
  type TaskKind,
  listMemoryWorkerCalls,
  listMemoryWorkers,
  taskAgentSupported,
  testMemoryWorker,
  upsertMemoryWorker,
} from '@/lib/memoryWorkers'
import { listProviders as listSummarizerProviders } from '@/lib/memoryAmbient'
import { listClaudeAccounts } from '@/lib/claudeAccounts'

function useTaskLabels() {
  const { t } = useTranslation()
  return {
    label: (task: TaskKind) => t(`web.memoryWorkers.tasks.${task}.label`),
    description: (task: TaskKind) =>
      t(`web.memoryWorkers.tasks.${task}.description`),
  }
}

export function MemoryWorkersPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const workersQuery = useQuery({
    queryKey: ['memory-workers'],
    queryFn: listMemoryWorkers,
  })
  const summarizersQuery = useQuery({
    queryKey: ['memory-summarizer-providers'],
    queryFn: listSummarizerProviders,
    staleTime: 60_000,
  })
  const accountsQuery = useQuery({
    queryKey: ['claude-accounts'],
    queryFn: listClaudeAccounts,
    staleTime: 60_000,
  })
  const callsQuery = useQuery({
    queryKey: ['memory-worker-calls'],
    queryFn: () => listMemoryWorkerCalls({ limit: 200 }),
    staleTime: 15_000,
    refetchInterval: 30_000,
  })

  const refresh = () =>
    qc.invalidateQueries({ queryKey: ['memory-workers'] })

  if (workersQuery.isLoading) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 p-8 text-sm">
        <Loader2 className="size-3 animate-spin" />
        {t('web.memoryWorkers.loading')}
      </div>
    )
  }

  // Empty-data state: most common reason is the server hasn't yet
  // applied migration 0029 (memory_workers table) — surface it
  // clearly so operators don't stare at a half-empty page wondering
  // what went wrong.
  if (workersQuery.isError) {
    return (
      <div className="mx-auto max-w-2xl space-y-3 p-6 text-sm">
        <h1 className="text-xl font-semibold">
          {t('web.memoryWorkers.title')}
        </h1>
        <div className="bg-destructive/10 text-destructive rounded-md border p-3 text-xs">
          <strong>{t('web.memoryWorkers.errorTitle')}</strong>{' '}
          {t('web.memoryWorkers.errorDescription')}
        </div>
        <pre className="bg-muted/30 overflow-auto rounded p-2 font-mono text-[10px]">
          {String(workersQuery.error)}
        </pre>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-3xl space-y-6 p-6">
      <header>
        <h1 className="text-xl font-semibold">
          {t('web.memoryWorkers.title')}
        </h1>
        <p className="text-muted-foreground mt-1 text-sm">
          <Trans
            i18nKey="web.memoryWorkers.intro"
            components={{ 1: <strong />, 3: <strong />, 5: <code /> }}
          />
        </p>
      </header>

      <div className="space-y-4">
        {(workersQuery.data ?? []).map((w) => (
          <WorkerCard
            key={w.task}
            config={w}
            summarizers={summarizersQuery.data ?? []}
            accounts={accountsQuery.data ?? []}
            calls={(callsQuery.data ?? []).filter((c) => c.task === w.task)}
            onSaved={refresh}
          />
        ))}
      </div>
    </div>
  )
}

interface WorkerCardProps {
  config: WorkerConfig
  summarizers: { id: string; name: string; kind: string }[]
  accounts: { id: string; display_name?: string; name?: string }[]
  calls: CallSummary[]
  onSaved: () => void
}

function WorkerCard({
  config,
  summarizers,
  accounts,
  calls,
  onSaved,
}: WorkerCardProps) {
  const { t } = useTranslation()
  const taskLabels = useTaskLabels()
  const [kind, setKind] = useState<WorkerKind>(config.kind)
  const [summarizerId, setSummarizerId] = useState(config.summarizer_id ?? '')
  const [providerId, setProviderId] = useState<AgentProviderID | ''>(
    (config.provider_id as AgentProviderID) ?? '',
  )
  const [accountId, setAccountId] = useState(config.account_id ?? '')
  const [enabled, setEnabled] = useState(config.enabled)

  const agentAllowed = taskAgentSupported(config.task)
  const dirty =
    kind !== config.kind ||
    summarizerId !== (config.summarizer_id ?? '') ||
    providerId !== ((config.provider_id as string) ?? '') ||
    accountId !== (config.account_id ?? '') ||
    enabled !== config.enabled

  const save = useMutation({
    mutationFn: () =>
      upsertMemoryWorker(config.task, {
        kind,
        summarizer_id: kind === 'summarizer' ? summarizerId : '',
        provider_id: kind === 'agent' ? providerId || undefined : '',
        account_id:
          kind === 'agent' && providerId === 'claude' ? accountId : '',
        enabled,
      }),
    onSuccess: () => {
      toast.success(
        t('web.memoryWorkers.savedToast', { label: taskLabels.label(config.task) }),
      )
      onSaved()
    },
    onError: (e: Error) =>
      toast.error(t('web.memoryWorkers.saveFailedToast'), {
        description: e.message,
      }),
  })

  const test = useMutation({
    mutationFn: () => testMemoryWorker(config.task),
    onSuccess: (res) => {
      if (res.ok) {
        toast.success(
          t('web.memoryWorkers.testOkToast', {
            label: taskLabels.label(config.task),
            ms: res.duration_ms,
          }),
          { description: res.preview ? truncate(res.preview, 200) : '' },
        )
      } else {
        toast.error(
          t('web.memoryWorkers.testFailedToast', {
            label: taskLabels.label(config.task),
          }),
          {
            description: res.error ?? t('web.memoryWorkers.unknownError'),
          },
        )
      }
    },
    onError: (e: Error) =>
      toast.error(t('web.memoryWorkers.testCallFailedToast'), {
        description: e.message,
      }),
  })

  const recentMetrics = useMemo(() => computeMetrics(calls), [calls])

  return (
    <div className="bg-card space-y-3 rounded-md border p-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="flex items-center gap-2">
            <h2 className="text-base font-semibold">
              {taskLabels.label(config.task)}
            </h2>
            <Badge variant={enabled ? 'success' : 'muted'} className="text-[9px]">
              {enabled
                ? t('web.memoryWorkers.enabledBadge')
                : t('web.memoryWorkers.disabledBadge')}
            </Badge>
            {!agentAllowed && (
              <Badge variant="warning" className="text-[9px]">
                {t('web.memoryWorkers.summarizerOnlyBadge')}
              </Badge>
            )}
          </div>
          <p className="text-muted-foreground mt-1 text-xs">
            {taskLabels.description(config.task)}
          </p>
        </div>
        <div className="text-muted-foreground flex flex-col items-end text-[10px]">
          <span>
            {t('web.memoryWorkers.callsCount', { count: recentMetrics.count })}
          </span>
          {recentMetrics.count > 0 && (
            <>
              <span>
                {t('web.memoryWorkers.avgMs', { ms: recentMetrics.avgMs })}
              </span>
              {recentMetrics.errorCount > 0 && (
                <span className="text-destructive">
                  {t('web.memoryWorkers.errorsCount', {
                    count: recentMetrics.errorCount,
                  })}
                </span>
              )}
            </>
          )}
        </div>
      </div>

      <div className="space-y-2">
        <div className="grid grid-cols-1 gap-2 md:grid-cols-3">
          <div>
            <label className="text-muted-foreground mb-1 block text-[10px] tracking-wide uppercase">
              {t('web.memoryWorkers.workerLabel')}
            </label>
            <Select
              value={kind}
              onValueChange={(v) => setKind(v as WorkerKind)}
              disabled={!agentAllowed}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="summarizer">
                  {t('web.memoryWorkers.summarizerHttp')}
                </SelectItem>
                {agentAllowed && (
                  <SelectItem value="agent">
                    {t('web.memoryWorkers.agentCliPrint')}
                  </SelectItem>
                )}
              </SelectContent>
            </Select>
          </div>

          {kind === 'summarizer' && (
            <div className="md:col-span-2">
              <label className="text-muted-foreground mb-1 block text-[10px] tracking-wide uppercase">
                {t('web.memoryWorkers.summarizerProviderLabel')}
              </label>
              <Select
                value={summarizerId || '__default__'}
                onValueChange={(v) =>
                  setSummarizerId(v === '__default__' ? '' : v)
                }
              >
                <SelectTrigger>
                  <SelectValue
                    placeholder={t('web.memoryWorkers.registryDefault')}
                  />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__default__">
                    {t('web.memoryWorkers.registryDefault')}
                  </SelectItem>
                  {summarizers.map((s) => (
                    <SelectItem key={s.id} value={s.id}>
                      {s.name} · {s.kind}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}

          {kind === 'agent' && (
            <>
              <div>
                <label className="text-muted-foreground mb-1 block text-[10px] tracking-wide uppercase">
                  {t('web.memoryWorkers.cliLabel')}
                </label>
                <Select
                  value={providerId || ''}
                  onValueChange={(v) =>
                    setProviderId(v as AgentProviderID | '')
                  }
                >
                  <SelectTrigger>
                    <SelectValue
                      placeholder={t('web.memoryWorkers.selectPlaceholder')}
                    />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="claude">
                      {t('web.memoryWorkers.cliClaude')}
                    </SelectItem>
                    <SelectItem value="gemini">
                      {t('web.memoryWorkers.cliGemini')}
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
              {providerId === 'claude' && (
                <div>
                  <label className="text-muted-foreground mb-1 block text-[10px] tracking-wide uppercase">
                    {t('web.memoryWorkers.claudeAccountLabel')}
                  </label>
                  <Select
                    value={accountId || '__default__'}
                    onValueChange={(v) =>
                      setAccountId(v === '__default__' ? '' : v)
                    }
                  >
                    <SelectTrigger>
                      <SelectValue
                        placeholder={t('web.memoryWorkers.claudeAccountDefault')}
                      />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="__default__">
                        {t('web.memoryWorkers.claudeAccountDefault')}
                      </SelectItem>
                      {accounts.map((a) => (
                        <SelectItem key={a.id} value={a.id}>
                          {a.display_name || a.name || a.id}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              )}
            </>
          )}
        </div>

        {kind === 'agent' && (
          <div className="text-muted-foreground bg-muted/30 flex items-start gap-2 rounded-md p-2 text-xs">
            <AlertTriangle className="mt-0.5 size-3 flex-none" />
            <div>
              <Trans
                i18nKey="web.memoryWorkers.agentWarning"
                components={{ 1: <strong />, 3: <strong /> }}
              />
            </div>
          </div>
        )}
      </div>

      <div className="flex items-center gap-2 pt-1">
        <label className="text-muted-foreground flex cursor-pointer items-center gap-1 text-xs">
          <input
            type="checkbox"
            checked={enabled}
            onChange={(e) => setEnabled(e.target.checked)}
          />
          {t('web.memoryWorkers.enabledCheckbox')}
        </label>
        <div className="ml-auto flex gap-2">
          <Button
            size="sm"
            variant="outline"
            onClick={() => test.mutate()}
            disabled={test.isPending}
          >
            {test.isPending ? (
              <Loader2 className="mr-1 size-3 animate-spin" />
            ) : (
              <Play className="mr-1 size-3" />
            )}
            {t('web.memoryWorkers.testButton')}
          </Button>
          <Button
            size="sm"
            disabled={!dirty || save.isPending}
            onClick={() => save.mutate()}
          >
            {save.isPending ? (
              <Loader2 className="mr-1 size-3 animate-spin" />
            ) : (
              <Save className="mr-1 size-3" />
            )}
            {t('web.memoryWorkers.saveButton')}
          </Button>
        </div>
      </div>

      {calls.length > 0 && (
        <details className="text-xs">
          <summary className="text-muted-foreground hover:text-foreground inline-flex cursor-pointer items-center gap-1">
            <ChevronRight className="size-3 transition-transform" />
            {t('web.memoryWorkers.recentCalls', { count: calls.length })}
          </summary>
          <div className="mt-2 max-h-48 overflow-auto rounded-md border">
            <table className="w-full text-[11px]">
              <thead className="bg-muted/30">
                <tr>
                  <th className="px-2 py-1 text-left">
                    {t('web.memoryWorkers.tableWhen')}
                  </th>
                  <th className="px-2 py-1 text-left">
                    {t('web.memoryWorkers.tableWorker')}
                  </th>
                  <th className="px-2 py-1 text-right">
                    {t('web.memoryWorkers.tableMs')}
                  </th>
                  <th className="px-2 py-1">
                    {t('web.memoryWorkers.tableOk')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {calls.slice(0, 25).map((c) => (
                  <tr key={c.id} className="border-t">
                    <td className="text-muted-foreground px-2 py-1 font-mono">
                      {new Date(c.started_at).toLocaleTimeString()}
                    </td>
                    <td className="px-2 py-1">
                      {c.worker_kind}
                      {c.provider_id ? ` · ${c.provider_id}` : ''}
                    </td>
                    <td className="px-2 py-1 text-right font-mono">
                      {c.duration_ms}
                    </td>
                    <td className="px-2 py-1 text-center">
                      {c.success ? (
                        <CheckCircle2 className="text-state-running mx-auto size-3" />
                      ) : (
                        <AlertTriangle className="text-state-failed mx-auto size-3" />
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </details>
      )}
    </div>
  )
}

function computeMetrics(calls: CallSummary[]) {
  if (calls.length === 0) return { count: 0, avgMs: 0, errorCount: 0 }
  const cutoff = Date.now() - 24 * 3600_000
  const recent = calls.filter((c) => new Date(c.started_at).getTime() > cutoff)
  let totalMs = 0
  let errors = 0
  for (const c of recent) {
    totalMs += c.duration_ms
    if (!c.success) errors++
  }
  return {
    count: recent.length,
    avgMs: recent.length ? Math.round(totalMs / recent.length) : 0,
    errorCount: errors,
  }
}

function truncate(s: string, n: number): string {
  return s.length > n ? s.slice(0, n) + '…' : s
}
