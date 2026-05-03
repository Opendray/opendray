import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  Activity as ActivityIcon,
  ArrowDownToLine,
  ArrowUpFromLine,
  Loader2,
  Plug,
  RefreshCw,
} from 'lucide-react'
import { formatDistanceToNowStrict } from 'date-fns'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  listIntegrationCalls,
  type IntegrationCall,
} from '@/lib/integrationCalls'
import { listIntegrations } from '@/lib/integrations'
import { cn } from '@/lib/utils'

type DirectionFilter = 'all' | 'inbound' | 'outbound'
type StatusFilter = 'all' | '2' | '3' | '4' | '5'

// Activity — per-call audit log of every API request made by a
// registered integration. Two flows are recorded:
//
//   inbound  — third-party app → opendray  (via its API key)
//   outbound — admin → opendray /proxy/{prefix}/* → integration
//
// Calls made by the admin UI directly (e.g. you spawning a session
// from the Sessions page) are NOT recorded here — they are not the
// gateway use case this view is designed to surface.
//
// TODO(adr-0010): KPI cards row above the table — calls/min,
// error rate, p95 latency, top integration. Trigger: first real
// integration accumulates ≥100 calls/day for a week. Backed by a
// /integrations/_calls/summary endpoint (also deferred).
//
// TODO(adr-0010): "Load older" pagination button — currently we
// show the latest 100 only. Trigger: when next_cursor is
// consistently present after first page refresh.
//
// TODO(adr-0010): deep link from the Integrations page row →
// this page with integration_id pre-filled. Trigger: when
// Integrations gets a detail page, OR when 2+ active integrations
// make the unfiltered view noisy.
export function ActivityPage() {
  const [integrationID, setIntegrationID] = useState<string>('')
  const [direction, setDirection] = useState<DirectionFilter>('all')
  const [status, setStatus] = useState<StatusFilter>('all')

  const integrations = useQuery({
    queryKey: ['integrations'],
    queryFn: listIntegrations,
    staleTime: 30_000,
  })

  const calls = useQuery({
    queryKey: ['integration-calls', { integrationID, direction, status }],
    queryFn: () =>
      listIntegrationCalls({
        integration_id: integrationID || undefined,
        direction: direction === 'all' ? undefined : direction,
        status_class:
          status === 'all'
            ? undefined
            : (parseInt(status, 10) as 2 | 3 | 4 | 5),
        limit: 100,
      }),
    refetchInterval: 5_000,
  })

  const intgrName = (id: string) =>
    integrations.data?.find((i) => i.id === id)?.name ?? id.slice(0, 12)

  const entries = calls.data?.entries ?? []

  return (
    <div className="h-full flex flex-col bg-background">
      <header className="border-b border-border px-6 py-4 flex flex-wrap items-start gap-3">
        <div className="flex-1 min-w-[260px]">
          <h1 className="text-[16px] font-semibold tracking-tight flex items-center gap-2">
            <ActivityIcon className="size-4 text-muted-foreground" />
            Activity
          </h1>
          <p className="text-[12px] text-muted-foreground max-w-2xl">
            Per-call audit of API requests made by registered integrations.
            Includes both inbound calls (a third-party app calling opendray
            with its API key) and outbound proxied calls (admin → opendray
            proxy → integration). Calls made directly by this admin UI are
            not recorded.
          </p>
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => calls.refetch()}
          disabled={calls.isFetching}
          title="Refresh"
        >
          {calls.isFetching ? (
            <Loader2 className="size-3.5 animate-spin" />
          ) : (
            <RefreshCw className="size-3.5" />
          )}
          Refresh
        </Button>
      </header>

      <div className="border-b border-border px-6 py-3 flex flex-wrap items-center gap-2">
        <Filter label="Integration">
          <Select
            value={integrationID || '__all__'}
            onValueChange={(v) =>
              setIntegrationID(v === '__all__' ? '' : v)
            }
          >
            <SelectTrigger className="h-8 w-[220px] text-[12px]">
              <SelectValue placeholder="All integrations" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__all__">All integrations</SelectItem>
              {integrations.data?.map((i) => (
                <SelectItem key={i.id} value={i.id}>
                  {i.name}{' '}
                  <span className="text-muted-foreground/60 font-mono">
                    /{i.route_prefix}
                  </span>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Filter>

        <Filter label="Direction">
          <Select
            value={direction}
            onValueChange={(v) => setDirection(v as DirectionFilter)}
          >
            <SelectTrigger className="h-8 w-[140px] text-[12px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All</SelectItem>
              <SelectItem value="inbound">Inbound</SelectItem>
              <SelectItem value="outbound">Outbound</SelectItem>
            </SelectContent>
          </Select>
        </Filter>

        <Filter label="Status">
          <Select
            value={status}
            onValueChange={(v) => setStatus(v as StatusFilter)}
          >
            <SelectTrigger className="h-8 w-[140px] text-[12px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All statuses</SelectItem>
              <SelectItem value="2">2xx success</SelectItem>
              <SelectItem value="3">3xx redirect</SelectItem>
              <SelectItem value="4">4xx client error</SelectItem>
              <SelectItem value="5">5xx server error</SelectItem>
            </SelectContent>
          </Select>
        </Filter>

        <div className="flex-1" />
        <span className="text-[10px] text-muted-foreground/70 font-mono tabular-nums">
          {entries.length} {entries.length === 1 ? 'call' : 'calls'}
        </span>
      </div>

      <ScrollArea className="flex-1">
        {calls.isLoading ? (
          <div className="flex items-center justify-center py-16 gap-2 text-[12px] text-muted-foreground">
            <Loader2 className="size-3.5 animate-spin" />
            Loading…
          </div>
        ) : entries.length === 0 ? (
          <EmptyState
            hasIntegrations={(integrations.data?.length ?? 0) > 0}
            hasFilter={!!integrationID || direction !== 'all' || status !== 'all'}
          />
        ) : (
          <CallsTable calls={entries} integrationName={intgrName} />
        )}
      </ScrollArea>
    </div>
  )
}

// ── presentation ──────────────────────────────────────────────────
function Filter({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <label className="flex items-center gap-1.5">
      <span className="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium">
        {label}
      </span>
      {children}
    </label>
  )
}

function CallsTable({
  calls,
  integrationName,
}: {
  calls: IntegrationCall[]
  integrationName: (id: string) => string
}) {
  return (
    <table className="w-full text-[12px] border-collapse">
      <thead>
        <tr className="border-b border-border bg-card/40">
          <Th>Time</Th>
          <Th>Integration</Th>
          <Th className="w-[50px]" title="Direction" />
          <Th>Method</Th>
          <Th>Path</Th>
          <Th className="text-right">Status</Th>
          <Th className="text-right">Duration</Th>
        </tr>
      </thead>
      <tbody>
        {calls.map((c) => (
          <CallRow
            key={c.id}
            call={c}
            integrationName={integrationName(c.integration_id)}
          />
        ))}
      </tbody>
    </table>
  )
}

function Th({
  children,
  className,
  title,
}: {
  children?: React.ReactNode
  className?: string
  title?: string
}) {
  return (
    <th
      className={cn(
        'text-left px-3 py-1.5 text-[10px] uppercase tracking-wider font-medium text-muted-foreground/70',
        className,
      )}
      title={title}
    >
      {children}
    </th>
  )
}

function CallRow({
  call,
  integrationName,
}: {
  call: IntegrationCall
  integrationName: string
}) {
  const t = new Date(call.ts)
  const time = isNaN(t.getTime()) ? '--' : t.toTimeString().slice(0, 8)
  const ageLabel = isNaN(t.getTime())
    ? ''
    : compactRel(formatDistanceToNowStrict(t, { addSuffix: false }))

  const statusClass =
    call.status_code >= 500
      ? 'danger'
      : call.status_code >= 400
        ? 'warning'
        : call.status_code >= 300
          ? 'muted'
          : 'success'

  return (
    <tr className="border-b border-border/60 hover:bg-card/40 transition-colors">
      <td className="px-3 py-1.5 text-muted-foreground/80 font-mono tabular-nums whitespace-nowrap">
        {time}
        <span className="text-[10px] text-muted-foreground/40 ml-1.5">
          {ageLabel}
        </span>
      </td>
      <td className="px-3 py-1.5 truncate max-w-[180px]">{integrationName}</td>
      <td className="px-3 py-1.5">
        {call.direction === 'inbound' ? (
          <ArrowDownToLine
            className="size-3 text-state-running"
            aria-label="inbound"
          />
        ) : (
          <ArrowUpFromLine
            className="size-3 text-accent"
            aria-label="outbound"
          />
        )}
      </td>
      <td className="px-3 py-1.5 font-mono text-foreground/80">
        {call.method}
      </td>
      <td className="px-3 py-1.5 font-mono text-foreground/70 truncate max-w-[420px]">
        {call.path}
      </td>
      <td className="px-3 py-1.5 text-right">
        <Badge variant={statusClass}>{call.status_code}</Badge>
      </td>
      <td className="px-3 py-1.5 text-right font-mono tabular-nums text-muted-foreground/80 whitespace-nowrap">
        {formatDuration(call.duration_ms)}
      </td>
    </tr>
  )
}

function EmptyState({
  hasIntegrations,
  hasFilter,
}: {
  hasIntegrations: boolean
  hasFilter: boolean
}) {
  if (hasFilter) {
    return (
      <div className="flex flex-col items-center justify-center py-16 gap-2 text-[12px] text-muted-foreground">
        <span>No calls match these filters.</span>
      </div>
    )
  }
  return (
    <div className="flex flex-col items-center justify-center py-16 px-6 gap-3 text-center">
      <Plug className="size-6 text-muted-foreground/40" />
      <div className="text-[13px] font-medium text-foreground">
        No API calls recorded yet
      </div>
      <p className="text-[12px] text-muted-foreground max-w-md leading-relaxed">
        When a third-party app calls opendray with its integration API key,
        every request is logged here.
      </p>
      <ol className="text-[12px] text-muted-foreground/80 max-w-md leading-relaxed flex flex-col gap-1 list-decimal list-inside text-left">
        <li>
          {hasIntegrations
            ? 'Use an existing integration\'s API key in your third-party app'
            : 'Register an integration in Integrations → New'}
        </li>
        <li>
          Call any endpoint, e.g.{' '}
          <code className="text-foreground/80 font-mono text-[11px]">
            POST /api/v1/sessions
          </code>
        </li>
        <li>Calls appear here within seconds</li>
      </ol>
      <p className="text-[11px] text-muted-foreground/60 max-w-md leading-relaxed mt-1">
        Calls you make from this admin UI are not logged — only
        integration-attributed traffic is recorded.
      </p>
    </div>
  )
}

// ── helpers ──────────────────────────────────────────────────────
function formatDuration(ms: number): string {
  if (ms < 1) return '<1ms'
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(2)}s`
}

function compactRel(s: string): string {
  return s
    .replace(/ seconds?/, 's')
    .replace(/ minutes?/, 'm')
    .replace(/ hours?/, 'h')
    .replace(/ days?/, 'd')
}
