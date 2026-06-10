// Cortex — the single home of the experience flywheel:
// Memory (raw episodic facts) → Notes (each project's official doc) →
// Knowledge (cross-project, iterable expertise) → injected back into
// every new session. One module, three rungs, one loop — no more
// silo tabs.

import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import {
  ArrowRight,
  BookOpenText,
  Brain,
  Inbox,
  Network,
  RefreshCcwDot,
  Settings,
  ShieldQuestion,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { getCortexStatus } from '@/lib/cortex'
import { listProjects } from '@/lib/projectDocs'

export function CortexPage() {
  const { t } = useTranslation()
  const statusQuery = useQuery({
    queryKey: ['cortex-status'],
    queryFn: getCortexStatus,
    refetchInterval: 30_000,
  })
  const projectsQuery = useQuery({
    queryKey: ['known-projects'],
    queryFn: () => listProjects(),
    staleTime: 30_000,
  })
  const s = statusQuery.data
  const activeProjects = (projectsQuery.data ?? []).filter(
    (p) => p.status === 'active',
  )

  return (
    <div className="mx-auto max-w-4xl space-y-6 p-6">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="flex items-center gap-2 text-xl font-semibold">
            <RefreshCcwDot className="h-5 w-5" />
            {t('web.cortex.home.title')}
          </h1>
          <p className="text-muted-foreground mt-1 text-sm">
            {t('web.cortex.home.subtitle')}
          </p>
        </div>
        <Link
          to="/cortex/settings"
          className="text-muted-foreground hover:text-foreground flex flex-none items-center gap-1.5 rounded-md border px-3 py-1.5 text-xs"
        >
          <Settings className="h-3.5 w-3.5" />
          {t('web.cortex.home.settings')}
        </Link>
      </div>

      {/* The loop, rung by rung. Each card is the entry into that layer. */}
      <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
        <RungCard
          to="/cortex/memory"
          icon={<Brain className="h-4 w-4" />}
          title={t('web.cortex.home.memory.title')}
          description={t('web.cortex.home.memory.description')}
          badges={
            <>
              {s?.memory.enabled === false && (
                <Badge variant="muted" className="text-[10px]">
                  {t('web.cortex.home.disabled')}
                </Badge>
              )}
              {(s?.memory.quarantine_count ?? 0) > 0 && (
                <Badge variant="warning" className="text-[10px]">
                  <ShieldQuestion className="mr-1 h-2.5 w-2.5" />
                  {t('web.cortex.home.memory.quarantine', {
                    count: s!.memory.quarantine_count,
                  })}
                </Badge>
              )}
            </>
          }
        />
        <RungCard
          to="/cortex/project"
          icon={<BookOpenText className="h-4 w-4" />}
          title={t('web.cortex.home.notes.title')}
          description={t('web.cortex.home.notes.description')}
          badges={
            <>
              <Badge variant="outline" className="text-[10px]">
                {t('web.cortex.home.notes.projects', {
                  count: s?.notes.active_projects ?? 0,
                })}
              </Badge>
              {(s?.notes.pending_proposals ?? 0) > 0 && (
                <Badge variant="danger" className="text-[10px]">
                  <Inbox className="mr-1 h-2.5 w-2.5" />
                  {t('web.cortex.home.pendingProposals', {
                    count: s!.notes.pending_proposals,
                  })}
                </Badge>
              )}
            </>
          }
        />
        <RungCard
          to="/cortex/knowledge"
          icon={<Network className="h-4 w-4" />}
          title={t('web.cortex.home.knowledge.title')}
          description={t('web.cortex.home.knowledge.description')}
          badges={
            <>
              {s?.knowledge.enabled === false && (
                <Badge variant="muted" className="text-[10px]">
                  {t('web.cortex.home.disabled')}
                </Badge>
              )}
              {(s?.knowledge.pending_proposals ?? 0) > 0 && (
                <Badge variant="danger" className="text-[10px]">
                  <Inbox className="mr-1 h-2.5 w-2.5" />
                  {t('web.cortex.home.pendingProposals', {
                    count: s!.knowledge.pending_proposals,
                  })}
                </Badge>
              )}
            </>
          }
        />
      </div>

      <p className="text-muted-foreground text-center text-xs">
        {t('web.cortex.home.loopHint')}
      </p>

      {/* Active projects — jump straight into a workspace. */}
      {activeProjects.length > 0 && (
        <div className="space-y-1.5">
          <h2 className="text-sm font-semibold">
            {t('web.cortex.home.activeProjects')}
          </h2>
          {activeProjects.map((p) => (
            <Link
              key={p.cwd}
              to="/cortex/project"
              search={{ cwd: p.cwd }}
              className="hover:bg-muted/50 flex items-center gap-2 rounded-md border p-2.5"
            >
              <span className="flex-1 truncate font-mono text-xs">{p.cwd}</span>
              {p.suggest_archive && (
                <Badge variant="warning" className="text-[10px]">
                  {t('web.cortex.home.idle', { days: p.idle_days })}
                </Badge>
              )}
              <ArrowRight className="text-muted-foreground h-3.5 w-3.5" />
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}

function RungCard({
  to,
  icon,
  title,
  description,
  badges,
}: {
  to: string
  icon: React.ReactNode
  title: string
  description: string
  badges?: React.ReactNode
}) {
  return (
    <Link
      to={to}
      className="bg-card hover:border-primary/40 flex flex-col gap-2 rounded-lg border p-4 transition-colors"
    >
      <div className="flex items-center gap-2 text-sm font-semibold">
        {icon}
        {title}
      </div>
      <p className="text-muted-foreground flex-1 text-xs leading-relaxed">
        {description}
      </p>
      <div className="flex flex-wrap gap-1.5">{badges}</div>
    </Link>
  )
}
