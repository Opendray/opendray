// MemoryPanel — compact Project-memory glance inside the session
// inspector. Mirrors the mobile 🏁 icon that jumps from session
// detail to ProjectScreen, but adds inline stats so the operator
// doesn't always have to navigate away.
//
// All data is scoped to the session's cwd. Edits / approvals
// happen on the full /memory/project page (kept simple here).

import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { ArrowUpRight, Inbox, Loader2, NotebookPen, Target } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  listPendingProposals,
  listProjectDocs,
  listSessionLogs,
  type DocKind,
  type ProjectDoc,
} from '@/lib/projectDocs'
import { listCleanupDecisions } from '@/lib/memoryCleanup'

interface MemoryPanelProps {
  cwd: string
}

export function MemoryPanel({ cwd }: MemoryPanelProps) {
  const docsQ = useQuery({
    queryKey: ['project-docs', cwd],
    queryFn: () => listProjectDocs(cwd),
    enabled: !!cwd,
    staleTime: 30_000,
  })
  const proposalsQ = useQuery({
    queryKey: ['project-doc-proposals', cwd],
    queryFn: () => listPendingProposals(cwd),
    enabled: !!cwd,
    staleTime: 30_000,
  })
  const logsQ = useQuery({
    queryKey: ['session-logs', cwd, 3],
    queryFn: () => listSessionLogs(cwd, 3),
    enabled: !!cwd,
    staleTime: 30_000,
  })
  const cleanupQ = useQuery({
    queryKey: ['cleanup-decisions', 'project', cwd],
    queryFn: () =>
      listCleanupDecisions({
        status: 'pending',
        scope: 'project',
        scope_key: cwd,
        limit: 100,
      }),
    enabled: !!cwd,
    staleTime: 30_000,
  })

  if (!cwd) {
    return (
      <p className="text-muted-foreground text-xs">
        Session has no cwd — memory features need a working directory.
      </p>
    )
  }

  const docsByKind: Partial<Record<DocKind, ProjectDoc>> = {}
  for (const d of docsQ.data ?? []) docsByKind[d.kind] = d

  const goal = docsByKind.goal?.content?.trim() ?? ''
  const plan = docsByKind.plan?.content?.trim() ?? ''
  const journalCount = logsQ.data?.length ?? 0
  const inboxCount = proposalsQ.data?.length ?? 0
  const cleanupCount = cleanupQ.data?.length ?? 0
  const latestJournal = logsQ.data?.[0]

  return (
    <div className="space-y-3 text-xs">
      <Button asChild size="sm" className="h-8 w-full justify-between">
        <Link to="/memory/project" search={{ cwd }}>
          <span className="flex items-center gap-1.5">
            <Target className="size-3" />
            Open project memory
          </span>
          <ArrowUpRight className="size-3" />
        </Link>
      </Button>

      <div className="grid grid-cols-2 gap-1.5">
        <StatCell
          label="Docs"
          value={(docsQ.data ?? []).length}
          loading={docsQ.isLoading}
        />
        <StatCell
          label="Journal"
          value={journalCount}
          loading={logsQ.isLoading}
        />
        <StatCell
          label="Inbox"
          value={inboxCount}
          loading={proposalsQ.isLoading}
          danger={inboxCount > 0}
        />
        <StatCell
          label="Cleanup"
          value={cleanupCount}
          loading={cleanupQ.isLoading}
          danger={cleanupCount > 0}
        />
      </div>

      {goal && (
        <SectionPreview
          icon={<Target className="size-3" />}
          label="Goal"
          body={goal}
        />
      )}
      {plan && (
        <SectionPreview
          icon={<NotebookPen className="size-3" />}
          label="Plan"
          body={plan}
        />
      )}

      {latestJournal && (
        <div className="bg-card space-y-1 rounded-md border p-2">
          <div className="text-muted-foreground flex items-center gap-1 text-[10px] tracking-wide uppercase">
            <Inbox className="size-2.5" />
            Latest journal
            <span className="ml-auto font-mono">
              {new Date(latestJournal.created_at).toLocaleDateString()}
            </span>
          </div>
          {latestJournal.title && (
            <div className="text-foreground line-clamp-1 text-[11px] font-semibold">
              {latestJournal.title}
            </div>
          )}
          <p className="text-muted-foreground line-clamp-4 text-[10px] leading-relaxed">
            {latestJournal.content}
          </p>
        </div>
      )}

      {!goal && !plan && journalCount === 0 && !docsQ.isLoading && (
        <p className="text-muted-foreground py-2 text-center text-[11px]">
          No memory captured yet for this project. Spawn a session or set a
          goal to populate.
        </p>
      )}
    </div>
  )
}

function StatCell({
  label,
  value,
  loading,
  danger,
}: {
  label: string
  value: number
  loading?: boolean
  danger?: boolean
}) {
  return (
    <div className="bg-card rounded-md border p-2">
      <div className="text-muted-foreground text-[9px] tracking-wide uppercase">
        {label}
      </div>
      <div className="mt-0.5 flex items-baseline gap-1">
        {loading ? (
          <Loader2 className="size-3 animate-spin" />
        ) : (
          <>
            <span className="text-sm font-semibold">{value}</span>
            {danger && value > 0 && (
              <Badge variant="danger" className="text-[9px]">
                pending
              </Badge>
            )}
          </>
        )}
      </div>
    </div>
  )
}

function SectionPreview({
  icon,
  label,
  body,
}: {
  icon: React.ReactNode
  label: string
  body: string
}) {
  return (
    <div className="bg-card rounded-md border p-2">
      <div className="text-muted-foreground flex items-center gap-1 text-[10px] tracking-wide uppercase">
        {icon}
        {label}
      </div>
      <p className="text-foreground mt-1 line-clamp-3 text-[11px] leading-relaxed">
        {body}
      </p>
    </div>
  )
}
