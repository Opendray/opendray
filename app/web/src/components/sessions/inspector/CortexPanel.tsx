// CortexPanel — compact Cortex project-workspace glance inside the
// session inspector. Mirrors the mobile 🏁 shortcut that jumps from
// session detail to the project workspace, but adds inline stats so
// the operator doesn't always have to navigate away.
//
// "Open Cortex workspace" lands on /cortex/project — the unified
// project doc (overview / goal / plan / tech / activity) plus journal,
// inbox, and memory hygiene. All data is scoped to the session's cwd.

import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { ArrowUpRight, Inbox, Loader2, NotebookPen, Target } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  listPendingProposals,
  listProjectDocs,
  listSessionLogs,
  type DocKind,
  type ProjectDoc,
} from '@/lib/projectDocs'
import { listArchived } from '@/lib/memory'

interface CortexPanelProps {
  cwd: string
}

export function CortexPanel({ cwd }: CortexPanelProps) {
  const { t } = useTranslation()
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
  const archivedQ = useQuery({
    queryKey: ['archived-memories', 'project', cwd],
    queryFn: () => listArchived('project', cwd, 200),
    enabled: !!cwd,
    staleTime: 30_000,
  })

  if (!cwd) {
    return (
      <p className="text-muted-foreground text-xs">
        {t('web.sessions.inspector.cortexPanel.noCwd')}
      </p>
    )
  }

  const docsByKind: Partial<Record<DocKind, ProjectDoc>> = {}
  for (const d of docsQ.data ?? []) docsByKind[d.kind] = d

  const goal = docsByKind.goal?.content?.trim() ?? ''
  const plan = docsByKind.plan?.content?.trim() ?? ''
  const journalCount = logsQ.data?.length ?? 0
  const inboxCount = proposalsQ.data?.length ?? 0
  const archivedCount = archivedQ.data?.length ?? 0
  const latestJournal = logsQ.data?.[0]

  return (
    <div className="space-y-3 text-xs">
      <Button asChild size="sm" className="h-8 w-full justify-between">
        <Link to="/cortex/project" search={{ cwd }}>
          <span className="flex items-center gap-1.5">
            <Target className="size-3" />
            {t('web.sessions.inspector.cortexPanel.open')}
          </span>
          <ArrowUpRight className="size-3" />
        </Link>
      </Button>

      <div className="grid grid-cols-2 gap-1.5">
        <StatCell
          label={t('web.sessions.inspector.cortexPanel.docs')}
          value={(docsQ.data ?? []).length}
          loading={docsQ.isLoading}
        />
        <StatCell
          label={t('web.sessions.inspector.cortexPanel.journal')}
          value={journalCount}
          loading={logsQ.isLoading}
        />
        <StatCell
          label={t('web.sessions.inspector.cortexPanel.inbox')}
          value={inboxCount}
          loading={proposalsQ.isLoading}
          danger={inboxCount > 0}
          dangerLabel={t('web.sessions.inspector.cortexPanel.pending')}
        />
        <StatCell
          label={t('web.sessions.inspector.cortexPanel.archived')}
          value={archivedCount}
          loading={archivedQ.isLoading}
        />
      </div>

      {goal && (
        <SectionPreview
          icon={<Target className="size-3" />}
          label={t('web.sessions.inspector.cortexPanel.goal')}
          body={goal}
        />
      )}
      {plan && (
        <SectionPreview
          icon={<NotebookPen className="size-3" />}
          label={t('web.sessions.inspector.cortexPanel.plan')}
          body={plan}
        />
      )}

      {latestJournal && (
        <div className="bg-card space-y-1 rounded-md border p-2">
          <div className="text-muted-foreground flex items-center gap-1 text-[10px] tracking-wide uppercase">
            <Inbox className="size-2.5" />
            {t('web.sessions.inspector.cortexPanel.latestJournal')}
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
          {t('web.sessions.inspector.cortexPanel.empty')}
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
  dangerLabel,
}: {
  label: string
  value: number
  loading?: boolean
  danger?: boolean
  dangerLabel?: string
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
                {dangerLabel}
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
