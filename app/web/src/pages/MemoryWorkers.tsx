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
  listMemoryWorkerCalls,
  listMemoryWorkers,
  taskAgentSupported,
  taskDescription,
  taskLabel,
  testMemoryWorker,
  upsertMemoryWorker,
} from '@/lib/memoryWorkers'
import { listProviders as listSummarizerProviders } from '@/lib/memoryAmbient'
import { listClaudeAccounts } from '@/lib/claudeAccounts'

export function MemoryWorkersPage() {
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
        Loading worker config…
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
        <h1 className="text-xl font-semibold">Memory workers</h1>
        <div className="bg-destructive/10 text-destructive rounded-md border p-3 text-xs">
          <strong>Endpoint not reachable.</strong> The
          /api/v1/memory/workers routes are new in M25 — the
          opendray binary may need a restart to mount them and
          run migration 0029.
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
        <h1 className="text-xl font-semibold">Memory workers</h1>
        <p className="text-muted-foreground mt-1 text-sm">
          Each memory-system LLM touchpoint can be served independently
          by the local <strong>summarizer</strong> endpoint
          (LM Studio / OpenAI-compat) or by spawning a headless{' '}
          <strong>Claude / Gemini agent</strong> in <code>--print</code>{' '}
          mode. High-quality narrative tasks (gitactivity, transcript)
          benefit from agent workers; high-frequency tasks (gatekeeper)
          stay on the local endpoint by design.
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
      toast.success(`${taskLabel(config.task)} updated`)
      onSaved()
    },
    onError: (e: Error) =>
      toast.error('Save failed', { description: e.message }),
  })

  const test = useMutation({
    mutationFn: () => testMemoryWorker(config.task),
    onSuccess: (res) => {
      if (res.ok) {
        toast.success(
          `${taskLabel(config.task)} OK — ${res.duration_ms}ms`,
          { description: res.preview ? truncate(res.preview, 200) : '' },
        )
      } else {
        toast.error(`${taskLabel(config.task)} failed`, {
          description: res.error ?? 'unknown error',
        })
      }
    },
    onError: (e: Error) =>
      toast.error('Test call failed', { description: e.message }),
  })

  const recentMetrics = useMemo(() => computeMetrics(calls), [calls])

  return (
    <div className="bg-card space-y-3 rounded-md border p-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="flex items-center gap-2">
            <h2 className="text-base font-semibold">
              {taskLabel(config.task)}
            </h2>
            <Badge variant={enabled ? 'success' : 'muted'} className="text-[9px]">
              {enabled ? 'enabled' : 'disabled'}
            </Badge>
            {!agentAllowed && (
              <Badge variant="warning" className="text-[9px]">
                summarizer-only
              </Badge>
            )}
          </div>
          <p className="text-muted-foreground mt-1 text-xs">
            {taskDescription(config.task)}
          </p>
        </div>
        <div className="text-muted-foreground flex flex-col items-end text-[10px]">
          <span>{recentMetrics.count} calls · 24h</span>
          {recentMetrics.count > 0 && (
            <>
              <span>avg {recentMetrics.avgMs}ms</span>
              {recentMetrics.errorCount > 0 && (
                <span className="text-destructive">
                  {recentMetrics.errorCount} errors
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
              Worker
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
                <SelectItem value="summarizer">Summarizer (HTTP)</SelectItem>
                {agentAllowed && (
                  <SelectItem value="agent">Agent (CLI --print)</SelectItem>
                )}
              </SelectContent>
            </Select>
          </div>

          {kind === 'summarizer' && (
            <div className="md:col-span-2">
              <label className="text-muted-foreground mb-1 block text-[10px] tracking-wide uppercase">
                Summarizer provider
              </label>
              <Select
                value={summarizerId || '__default__'}
                onValueChange={(v) =>
                  setSummarizerId(v === '__default__' ? '' : v)
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder="Registry default" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__default__">
                    Registry default
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
                  CLI
                </label>
                <Select
                  value={providerId || ''}
                  onValueChange={(v) =>
                    setProviderId(v as AgentProviderID | '')
                  }
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="claude">Claude</SelectItem>
                    <SelectItem value="gemini">Gemini</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              {providerId === 'claude' && (
                <div>
                  <label className="text-muted-foreground mb-1 block text-[10px] tracking-wide uppercase">
                    Claude account
                  </label>
                  <Select
                    value={accountId || '__default__'}
                    onValueChange={(v) =>
                      setAccountId(v === '__default__' ? '' : v)
                    }
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Default" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="__default__">Default</SelectItem>
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
              Agent mode spawns a headless CLI per call. Latency rises
              from <strong>~1s</strong> (summarizer) to{' '}
              <strong>~5-15s</strong>; cost shifts from CPU to your
              Claude/Gemini quota.
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
          Enabled
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
            Test
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
            Save
          </Button>
        </div>
      </div>

      {calls.length > 0 && (
        <details className="text-xs">
          <summary className="text-muted-foreground hover:text-foreground inline-flex cursor-pointer items-center gap-1">
            <ChevronRight className="size-3 transition-transform" />
            Recent calls ({calls.length})
          </summary>
          <div className="mt-2 max-h-48 overflow-auto rounded-md border">
            <table className="w-full text-[11px]">
              <thead className="bg-muted/30">
                <tr>
                  <th className="px-2 py-1 text-left">when</th>
                  <th className="px-2 py-1 text-left">worker</th>
                  <th className="px-2 py-1 text-right">ms</th>
                  <th className="px-2 py-1">ok</th>
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
