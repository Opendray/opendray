// Cortex — the single home of the experience flywheel:
// Memory (raw episodic facts) → Notes (each project's official doc) →
// Knowledge (cross-project, iterable expertise) → injected back into
// every new session. One module, three rungs, one loop — no more
// silo tabs.

import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
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
import { toast } from 'sonner'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

import { Badge } from '@/components/ui/badge'
import { getCortexStatus } from '@/lib/cortex'
import {
  approveProposal,
  GLOBAL_CWD,
  listPendingProposals,
  listProjects,
  rejectProposal,
  type DocProposal,
} from '@/lib/projectDocs'

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

      {/* Everything waiting on the operator, reviewable right here —
          the rung-card PENDING badges are counts; this is the inbox. */}
      <ProposalInbox />

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

// ProposalInbox lists every pending doc proposal (project notes + KB
// pages) with an inline preview and approve / reject — so a PENDING
// badge on the rung cards is always one click from the actual content,
// regardless of which project it belongs to.
function ProposalInbox() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [openId, setOpenId] = useState<string | null>(null)

  const proposalsQuery = useQuery({
    queryKey: ['cortex-pending-proposals'],
    queryFn: () => listPendingProposals(),
    refetchInterval: 30_000,
  })
  const invalidate = () => {
    qc.invalidateQueries({ queryKey: ['cortex-pending-proposals'] })
    qc.invalidateQueries({ queryKey: ['cortex-status'] })
  }
  const approve = useMutation({
    mutationFn: (id: string) => approveProposal(id),
    onSuccess: () => {
      toast.success(t('web.cortex.home.proposals.approvedToast'))
      invalidate()
    },
    onError: (e: Error) =>
      toast.error(t('web.cortex.home.proposals.failedToast'), {
        description: e.message,
      }),
  })
  const reject = useMutation({
    mutationFn: (id: string) => rejectProposal(id),
    onSuccess: () => {
      toast.success(t('web.cortex.home.proposals.rejectedToast'))
      invalidate()
    },
    onError: (e: Error) =>
      toast.error(t('web.cortex.home.proposals.failedToast'), {
        description: e.message,
      }),
  })

  const proposals = proposalsQuery.data ?? []
  if (proposals.length === 0) return null

  const projectLabel = (p: DocProposal) => {
    if (p.cwd === GLOBAL_CWD) return t('web.cortex.home.proposals.kbLabel')
    const parts = p.cwd.split('/')
    return parts[parts.length - 1] || p.cwd
  }

  return (
    <div className="space-y-1.5">
      <h2 className="flex items-center gap-2 text-sm font-semibold">
        <Inbox className="h-4 w-4" />
        {t('web.cortex.home.proposals.title', { count: proposals.length })}
      </h2>
      <p className="text-muted-foreground text-xs">
        {t('web.cortex.home.proposals.hint')}
      </p>
      {proposals.map((p) => (
        <div key={p.id} className="bg-card rounded-md border">
          <div className="flex items-center gap-2 p-2.5">
            <Badge variant="outline" className="flex-none text-[10px]">
              {projectLabel(p)}
            </Badge>
            <code className="text-muted-foreground flex-none text-[10px]">
              {p.kind}
            </code>
            <span className="text-muted-foreground min-w-0 flex-1 truncate text-xs">
              {p.reason}
            </span>
            <span className="text-muted-foreground/70 flex-none text-[10px]">
              {new Date(p.created_at).toLocaleDateString()}
            </span>
            <button
              onClick={() => setOpenId(openId === p.id ? null : p.id)}
              className="border-border flex-none rounded-md border px-2 py-0.5 text-[11px]"
            >
              {openId === p.id
                ? t('web.cortex.home.proposals.hide')
                : t('web.cortex.home.proposals.preview')}
            </button>
            <button
              onClick={() => approve.mutate(p.id)}
              disabled={approve.isPending || reject.isPending}
              className="flex-none rounded-md bg-emerald-600/80 px-2 py-0.5 text-[11px] text-white disabled:opacity-50"
            >
              {t('web.cortex.home.proposals.approve')}
            </button>
            <button
              onClick={() => reject.mutate(p.id)}
              disabled={approve.isPending || reject.isPending}
              className="flex-none rounded-md border border-red-500/40 px-2 py-0.5 text-[11px] text-red-400 disabled:opacity-50"
            >
              {t('web.cortex.home.proposals.reject')}
            </button>
            {p.cwd === GLOBAL_CWD ? (
              <Link
                to="/cortex/knowledge"
                className="text-muted-foreground hover:text-foreground flex-none"
                title={t('web.cortex.home.proposals.open')}
              >
                <ArrowRight className="h-3.5 w-3.5" />
              </Link>
            ) : (
              <Link
                to="/cortex/project"
                search={{ cwd: p.cwd }}
                className="text-muted-foreground hover:text-foreground flex-none"
                title={t('web.cortex.home.proposals.open')}
              >
                <ArrowRight className="h-3.5 w-3.5" />
              </Link>
            )}
          </div>
          {openId === p.id && (
            <div className="border-border max-h-72 overflow-auto border-t p-3 text-sm">
              <ReactMarkdown remarkPlugins={[remarkGfm]}>
                {p.proposed_content}
              </ReactMarkdown>
            </div>
          )}
        </div>
      ))}
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
