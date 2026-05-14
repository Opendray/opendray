// ProjectScreen — web parity with app/mobile/lib/features/project/
// project_screen.dart. Hosts the cross-CLI project memory UI:
//
//   Goal / Plan / Tech / Activity / Journal / Inbox / Cleanup
//
// Goal + Plan are operator-editable; Tech (project scanner output)
// and Activity (git activity LLM summary) are scanner-managed and
// read-only; Journal is the auto-appended session-end log; Inbox
// queues agent-proposed goal/plan edits for approval; Cleanup
// holds the M13 librarian's pending verdicts.

import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import {
  AlertTriangle,
  Check,
  Inbox,
  Loader2,
  RotateCcw,
  Save,
  Sparkles,
  Trash2,
  X,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Textarea } from '@/components/ui/textarea'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

import {
  type DocKind,
  type ProjectDoc,
  type DocProposal,
  type SessionLogEntry,
  approveProposal,
  listProjectDocs,
  listPendingProposals,
  listSessionLogs,
  putProjectDoc,
  rejectProposal,
  resetProjectMemory,
} from '@/lib/projectDocs'
import {
  type CleanupDecision,
  approveDecision,
  listCleanupDecisions,
  rejectDecision,
  runCleanup,
} from '@/lib/memoryCleanup'
import { deleteMemoriesByScope } from '@/lib/memory'
import { MemoryHealthCard } from '@/components/project/MemoryHealthCard'

interface ProjectScreenProps {
  cwd: string
}

function useDocLabel() {
  const { t } = useTranslation()
  return (kind: DocKind | 'goal' | 'plan'): string =>
    t(`web.project.docLabel.${kind}`)
}

function useVerdictLabel() {
  const { t } = useTranslation()
  return (v: string): string => {
    if (v === 'stale') return t('web.project.verdictLabel.stale')
    if (v === 'duplicate') return t('web.project.verdictLabel.duplicate')
    return t('web.project.verdictLabel.keep')
  }
}

export function ProjectScreen({ cwd }: ProjectScreenProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()

  const docsQuery = useQuery({
    queryKey: ['project-docs', cwd],
    queryFn: () => listProjectDocs(cwd),
    enabled: !!cwd,
  })
  const proposalsQuery = useQuery({
    queryKey: ['project-doc-proposals', cwd],
    queryFn: () => listPendingProposals(cwd),
    enabled: !!cwd,
  })
  const logsQuery = useQuery({
    queryKey: ['session-logs', cwd],
    queryFn: () => listSessionLogs(cwd, 50),
    enabled: !!cwd,
  })
  const decisionsQuery = useQuery({
    queryKey: ['cleanup-decisions', 'project', cwd],
    queryFn: () =>
      listCleanupDecisions({
        status: 'pending',
        scope: 'project',
        scope_key: cwd,
        limit: 100,
      }),
    enabled: !!cwd,
  })

  const docsByKind = useMemo(() => {
    const map: Record<DocKind, ProjectDoc | undefined> = {
      goal: undefined,
      plan: undefined,
      tech_stack: undefined,
      recent_activity: undefined,
    }
    for (const d of docsQuery.data ?? []) map[d.kind] = d
    return map
  }, [docsQuery.data])

  const inboxCount = proposalsQuery.data?.length ?? 0
  const cleanupCount = decisionsQuery.data?.length ?? 0
  const docsCount = (docsQuery.data ?? []).length
  const journalCount = (logsQuery.data ?? []).length

  if (!cwd) {
    return (
      <div className="text-muted-foreground p-8 text-center text-sm">
        {t('web.project.noCwd')}
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      <div className="border-b px-4 py-3">
        <div className="flex items-start justify-between gap-3">
          <div>
            <div className="text-muted-foreground font-mono text-xs">{cwd}</div>
            <div className="mt-1 flex items-center gap-3 text-xs">
              <span>
                {t('web.project.header.docsCount', { count: docsCount })}
              </span>
              <span>·</span>
              <span>
                {t('web.project.header.journalEntries', { count: journalCount })}
              </span>
              {inboxCount > 0 && (
                <>
                  <span>·</span>
                  <Badge variant="danger" className="text-[10px]">
                    {t('web.project.header.pendingProposals', {
                      count: inboxCount,
                    })}
                  </Badge>
                </>
              )}
              {cleanupCount > 0 && (
                <>
                  <span>·</span>
                  <Badge variant="muted" className="text-[10px]">
                    {t('web.project.header.cleanupPending', {
                      count: cleanupCount,
                    })}
                  </Badge>
                </>
              )}
            </div>
          </div>
          <ResetButton
            cwd={cwd}
            onDone={() => {
              qc.invalidateQueries({ queryKey: ['project-docs', cwd] })
              qc.invalidateQueries({ queryKey: ['project-doc-proposals', cwd] })
              qc.invalidateQueries({ queryKey: ['session-logs', cwd] })
              qc.invalidateQueries({ queryKey: ['cleanup-decisions'] })
              qc.invalidateQueries({ queryKey: ['memories'] })
            }}
          />
        </div>
      </div>

      <Tabs defaultValue="health" className="flex flex-1 flex-col overflow-hidden">
        <TabsList className="bg-muted/30 mx-4 mt-3 w-fit">
          <TabsTrigger value="health">{t('web.project.tabs.health')}</TabsTrigger>
          <TabsTrigger value="goal">{t('web.project.tabs.goal')}</TabsTrigger>
          <TabsTrigger value="plan">{t('web.project.tabs.plan')}</TabsTrigger>
          <TabsTrigger value="tech">{t('web.project.tabs.tech')}</TabsTrigger>
          <TabsTrigger value="activity">
            {t('web.project.tabs.activity')}
          </TabsTrigger>
          <TabsTrigger value="journal">
            {t('web.project.tabs.journal')}
          </TabsTrigger>
          <TabsTrigger value="inbox" className="relative">
            {t('web.project.tabs.inbox')}
            {inboxCount > 0 && (
              <span className="bg-destructive text-destructive-foreground absolute -top-1 -right-2 flex h-4 min-w-4 items-center justify-center rounded-full px-1 text-[9px] font-bold">
                {inboxCount}
              </span>
            )}
          </TabsTrigger>
          <TabsTrigger value="cleanup" className="relative">
            {t('web.project.tabs.cleanup')}
            {cleanupCount > 0 && (
              <span className="bg-secondary absolute -top-1 -right-2 flex h-4 min-w-4 items-center justify-center rounded-full px-1 text-[9px] font-bold">
                {cleanupCount}
              </span>
            )}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="health" className="flex-1 overflow-auto">
          <MemoryHealthCard cwd={cwd} />
        </TabsContent>

        <TabsContent value="goal" className="flex-1 overflow-auto p-4">
          <DocEditor
            cwd={cwd}
            kind="goal"
            doc={docsByKind.goal}
            onSaved={() => qc.invalidateQueries({ queryKey: ['project-docs', cwd] })}
          />
        </TabsContent>

        <TabsContent value="plan" className="flex-1 overflow-auto p-4">
          <DocEditor
            cwd={cwd}
            kind="plan"
            doc={docsByKind.plan}
            onSaved={() => qc.invalidateQueries({ queryKey: ['project-docs', cwd] })}
          />
        </TabsContent>

        <TabsContent value="tech" className="flex-1 overflow-auto p-4">
          <ReadonlyDocTab doc={docsByKind.tech_stack} kind="tech_stack" />
        </TabsContent>

        <TabsContent value="activity" className="flex-1 overflow-auto p-4">
          <ReadonlyDocTab doc={docsByKind.recent_activity} kind="recent_activity" />
        </TabsContent>

        <TabsContent value="journal" className="flex-1 overflow-auto p-4">
          <JournalTab entries={logsQuery.data ?? []} loading={logsQuery.isLoading} />
        </TabsContent>

        <TabsContent value="inbox" className="flex-1 overflow-auto p-4">
          <InboxTab
            proposals={proposalsQuery.data ?? []}
            loading={proposalsQuery.isLoading}
            onChange={() => {
              qc.invalidateQueries({ queryKey: ['project-doc-proposals', cwd] })
              qc.invalidateQueries({ queryKey: ['project-docs', cwd] })
            }}
          />
        </TabsContent>

        <TabsContent value="cleanup" className="flex-1 overflow-auto p-4">
          <CleanupTab
            cwd={cwd}
            decisions={decisionsQuery.data ?? []}
            loading={decisionsQuery.isLoading}
            onChange={() =>
              qc.invalidateQueries({
                queryKey: ['cleanup-decisions', 'project', cwd],
              })
            }
          />
        </TabsContent>
      </Tabs>
    </div>
  )
}

// ─── Doc editor (goal / plan) ────────────────────────────────

interface DocEditorProps {
  cwd: string
  kind: DocKind
  doc?: ProjectDoc
  onSaved: () => void
}

function DocEditor({ cwd, kind, doc, onSaved }: DocEditorProps) {
  const { t } = useTranslation()
  const labelFor = useDocLabel()
  const [text, setText] = useState(doc?.content ?? '')
  const [dirty, setDirty] = useState(false)
  useMemo(() => {
    if (!dirty) setText(doc?.content ?? '')
  }, [doc?.content, dirty])

  const save = useMutation({
    mutationFn: () =>
      putProjectDoc({ cwd, kind, content: text }),
    onSuccess: () => {
      setDirty(false)
      onSaved()
      toast.success(
        t('web.project.editor.savedToast', { label: labelFor(kind) }),
      )
    },
    onError: (e: Error) =>
      toast.error(t('web.project.editor.saveFailedToast'), {
        description: e.message,
      }),
  })

  return (
    <div className="space-y-3">
      <div className="text-muted-foreground flex items-center justify-between text-xs">
        <span>
          {doc ? (
            <>
              {t('web.project.editor.updatedBy')} <strong>{doc.updated_by}</strong>{' '}
              <span className="ml-1">
                {new Date(doc.updated_at).toLocaleString()}
              </span>
            </>
          ) : (
            <span>
              {t('web.project.editor.noDocSet', { label: labelFor(kind) })}
            </span>
          )}
        </span>
        <Button
          size="sm"
          disabled={!dirty || save.isPending}
          onClick={() => save.mutate()}
        >
          {save.isPending ? (
            <Loader2 className="mr-2 h-3 w-3 animate-spin" />
          ) : (
            <Save className="mr-2 h-3 w-3" />
          )}
          {t('web.project.editor.save')}
        </Button>
      </div>
      <Textarea
        rows={20}
        value={text}
        onChange={(e) => {
          setText(e.target.value)
          setDirty(true)
        }}
        placeholder={
          kind === 'goal'
            ? t('web.project.editor.goalPlaceholder')
            : t('web.project.editor.planPlaceholder')
        }
        className="font-mono text-sm"
      />
    </div>
  )
}

// ─── Read-only doc (tech_stack / recent_activity) ────────────

interface ReadonlyDocTabProps {
  doc?: ProjectDoc
  kind: 'tech_stack' | 'recent_activity'
}

function ReadonlyDocTab({ doc, kind }: ReadonlyDocTabProps) {
  const { t } = useTranslation()
  const kindLabel = t(`web.project.readonly.${kind}.label`)
  const emptyHint = t(`web.project.readonly.${kind}.empty`)
  if (!doc) {
    return (
      <div className="text-muted-foreground text-sm">
        <p className="mb-2">
          {t('web.project.readonly.noneCaptured', { label: kindLabel })}
        </p>
        <p className="text-xs">{emptyHint}</p>
      </div>
    )
  }
  return (
    <div className="space-y-3">
      <div className="text-muted-foreground flex items-center justify-between text-xs">
        <span>
          {t('web.project.readonly.generatedBy')}{' '}
          <strong>{doc.updated_by}</strong> ·{' '}
          {t('web.project.readonly.lastRefresh')}{' '}
          {new Date(doc.updated_at).toLocaleString()}
        </span>
      </div>
      <pre className="bg-muted/30 max-h-[60vh] overflow-auto rounded-md p-3 font-mono text-xs whitespace-pre-wrap">
        {doc.content}
      </pre>
    </div>
  )
}

// ─── Journal tab ─────────────────────────────────────────────

function JournalTab({
  entries,
  loading,
}: {
  entries: SessionLogEntry[]
  loading: boolean
}) {
  const { t } = useTranslation()
  if (loading) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 text-sm">
        <Loader2 className="h-3 w-3 animate-spin" /> {t('web.project.journal.loading')}
      </div>
    )
  }
  if (entries.length === 0) {
    return (
      <p className="text-muted-foreground text-sm">
        {t('web.project.journal.empty')}
      </p>
    )
  }
  return (
    <div className="space-y-3">
      {entries.map((e) => (
        <div key={e.id} className="bg-card rounded-md border p-3">
          <div className="mb-1 flex items-center gap-2 text-xs">
            <Badge variant="outline" className="font-mono">
              {e.kind}
            </Badge>
            {e.session_id && (
              <span className="text-muted-foreground font-mono text-[10px]">
                {e.session_id.slice(-8)}
              </span>
            )}
            <span className="text-muted-foreground ml-auto text-[10px]">
              {new Date(e.created_at).toLocaleString()}
            </span>
          </div>
          {e.title && (
            <div className="mb-1 text-sm font-semibold">{e.title}</div>
          )}
          <pre className="bg-muted/20 max-h-72 overflow-auto rounded p-2 font-mono text-[11px] whitespace-pre-wrap">
            {e.content}
          </pre>
        </div>
      ))}
    </div>
  )
}

// ─── Inbox (proposal approve/reject) ─────────────────────────

function InboxTab({
  proposals,
  loading,
  onChange,
}: {
  proposals: DocProposal[]
  loading: boolean
  onChange: () => void
}) {
  const { t } = useTranslation()
  if (loading) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 text-sm">
        <Loader2 className="h-3 w-3 animate-spin" /> {t('web.project.inbox.loading')}
      </div>
    )
  }
  if (proposals.length === 0) {
    return (
      <div className="text-muted-foreground flex flex-col items-center gap-2 py-12 text-sm">
        <Inbox className="text-muted-foreground/50 h-8 w-8" />
        <p>{t('web.project.inbox.emptyTitle')}</p>
        <p className="text-xs">{t('web.project.inbox.emptyHint')}</p>
      </div>
    )
  }
  return (
    <div className="space-y-3">
      {proposals.map((p) => (
        <ProposalCard key={p.id} proposal={p} onChange={onChange} />
      ))}
    </div>
  )
}

interface ProposalCardProps {
  proposal: DocProposal
  onChange: () => void
}

function ProposalCard({ proposal, onChange }: ProposalCardProps) {
  const { t } = useTranslation()
  const labelFor = useDocLabel()
  const [confirmOpen, setConfirmOpen] = useState(false)
  const label = labelFor(proposal.kind)

  const approve = useMutation({
    mutationFn: () => approveProposal(proposal.id),
    onSuccess: () => {
      toast.success(
        t('web.project.inbox.approvedToast', { label }),
      )
      onChange()
      setConfirmOpen(false)
    },
    onError: (e: Error) => {
      toast.error(t('web.project.inbox.approveFailedToast'), {
        description: e.message,
      })
      // refresh anyway — likely 409 already-decided
      onChange()
      setConfirmOpen(false)
    },
  })
  const reject = useMutation({
    mutationFn: () => rejectProposal(proposal.id),
    onSuccess: () => {
      toast.success(t('web.project.inbox.rejectedToast'))
      onChange()
    },
    onError: (e: Error) => {
      toast.error(t('web.project.inbox.rejectFailedToast'), {
        description: e.message,
      })
      onChange()
    },
  })

  return (
    <div className="bg-card rounded-md border p-3">
      <div className="mb-2 flex items-center gap-2">
        <Badge variant="default">{label}</Badge>
        <span className="text-muted-foreground text-[11px]">
          {new Date(proposal.created_at).toLocaleString()}
        </span>
        {proposal.proposed_by_session && (
          <span className="text-muted-foreground font-mono text-[10px]">
            {t('web.project.inbox.sessionPrefix')}{' '}
            {proposal.proposed_by_session.slice(-8)}
          </span>
        )}
      </div>
      {proposal.reason && (
        <p className="mb-3 text-sm">{proposal.reason}</p>
      )}
      <div className="text-destructive bg-destructive/10 mb-3 flex items-start gap-2 rounded-md p-2 text-xs">
        <AlertTriangle className="mt-0.5 h-3 w-3 flex-none" />
        <div>
          <strong>{t('web.project.inbox.warning', { label })}</strong>{' '}
          {t('web.project.inbox.warningSuffix')}
        </div>
      </div>
      <div className="mb-3 grid grid-cols-1 gap-2 md:grid-cols-2">
        <DiffBlock
          label={t('web.project.inbox.current')}
          body={proposal.prior_content ?? t('web.project.inbox.emptyBody')}
        />
        <DiffBlock
          label={t('web.project.inbox.proposed')}
          body={proposal.proposed_content}
          highlight
        />
      </div>
      <div className="flex gap-2">
        <Button
          size="sm"
          variant="default"
          onClick={() => setConfirmOpen(true)}
          disabled={approve.isPending || reject.isPending}
        >
          <Check className="mr-1 h-3 w-3" />
          {t('web.project.inbox.approve')}
        </Button>
        <Button
          size="sm"
          variant="outline"
          onClick={() => reject.mutate()}
          disabled={approve.isPending || reject.isPending}
        >
          <X className="mr-1 h-3 w-3" />
          {t('web.project.inbox.reject')}
        </Button>
      </div>

      <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {t('web.project.inbox.confirmDialogTitle', { label })}
            </DialogTitle>
            <DialogDescription>
              {t('web.project.inbox.confirmDialogDescription', { label })}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setConfirmOpen(false)}
              disabled={approve.isPending}
            >
              {t('web.project.inbox.confirmCancel')}
            </Button>
            <Button
              onClick={() => approve.mutate()}
              disabled={approve.isPending}
            >
              {approve.isPending ? (
                <Loader2 className="mr-1 h-3 w-3 animate-spin" />
              ) : (
                <Check className="mr-1 h-3 w-3" />
              )}
              {t('web.project.inbox.confirmReplace')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function DiffBlock({
  label,
  body,
  highlight,
}: {
  label: string
  body: string
  highlight?: boolean
}) {
  return (
    <div className="flex flex-col">
      <div className="text-muted-foreground mb-1 text-[10px] font-semibold tracking-wide uppercase">
        {label}
      </div>
      <pre
        className={`max-h-48 overflow-auto rounded-md border p-2 font-mono text-[11px] whitespace-pre-wrap ${
          highlight ? 'border-primary/40 bg-primary/5' : 'bg-muted/20'
        }`}
      >
        {body}
      </pre>
    </div>
  )
}

// ─── Cleanup tab ─────────────────────────────────────────────

function CleanupTab({
  cwd,
  decisions,
  loading,
  onChange,
}: {
  cwd: string
  decisions: CleanupDecision[]
  loading: boolean
  onChange: () => void
}) {
  const { t } = useTranslation()
  const runMutation = useMutation({
    mutationFn: () => runCleanup({ scope: 'project', scope_key: cwd }),
    onSuccess: (res) => {
      toast.success(
        t('web.project.cleanup.runSucceededToast', {
          decided: res.decided,
          scanned: res.scanned,
        }),
      )
      onChange()
    },
    onError: (e: Error) =>
      toast.error(t('web.project.cleanup.runFailedToast'), {
        description: e.message,
      }),
  })

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <p className="text-muted-foreground text-xs">
          {t('web.project.cleanup.hint')}
        </p>
        <Button
          size="sm"
          variant="outline"
          onClick={() => runMutation.mutate()}
          disabled={runMutation.isPending}
        >
          {runMutation.isPending ? (
            <Loader2 className="mr-1 h-3 w-3 animate-spin" />
          ) : (
            <Sparkles className="mr-1 h-3 w-3" />
          )}
          {t('web.project.cleanup.runNow')}
        </Button>
      </div>
      {loading ? (
        <Loader2 className="h-3 w-3 animate-spin" />
      ) : decisions.length === 0 ? (
        <p className="text-muted-foreground py-8 text-center text-sm">
          {t('web.project.cleanup.empty')}
        </p>
      ) : (
        decisions.map((d) => (
          <CleanupDecisionCard key={d.id} decision={d} onChange={onChange} />
        ))
      )}
    </div>
  )
}

function CleanupDecisionCard({
  decision,
  onChange,
}: {
  decision: CleanupDecision
  onChange: () => void
}) {
  const { t } = useTranslation()
  const labelForVerdict = useVerdictLabel()
  const approve = useMutation({
    mutationFn: () => approveDecision(decision.id),
    onSuccess: () => {
      toast.success(
        t('web.project.cleanup.approvedExecutedToast', {
          label: labelForVerdict(decision.verdict),
        }),
      )
      onChange()
    },
    onError: (e: Error) => {
      toast.error(t('web.project.cleanup.approveFailedToast'), {
        description: e.message,
      })
      onChange()
    },
  })
  const reject = useMutation({
    mutationFn: () => rejectDecision(decision.id),
    onSuccess: () => {
      toast.success(t('web.project.cleanup.rejectedToast'))
      onChange()
    },
    onError: (e: Error) => {
      toast.error(t('web.project.cleanup.rejectFailedToast'), {
        description: e.message,
      })
      onChange()
    },
  })

  const verdictColor =
    decision.verdict === 'stale'
      ? 'danger'
      : decision.verdict === 'duplicate'
        ? 'muted'
        : 'outline'

  return (
    <div className="bg-card rounded-md border p-3">
      <div className="mb-2 flex items-center gap-2">
        <Badge variant={verdictColor as 'danger' | 'muted' | 'outline'}>
          {decision.verdict}
        </Badge>
        <span className="text-muted-foreground text-[11px]">
          {new Date(decision.created_at).toLocaleString()}
        </span>
        {decision.merge_into && (
          <span className="text-muted-foreground font-mono text-[10px]">
            {t('web.project.cleanup.mergeIntoPrefix')}{' '}
            {decision.merge_into.slice(-8)}
          </span>
        )}
      </div>
      <pre className="bg-muted/20 mb-2 max-h-32 overflow-auto rounded p-2 font-mono text-[11px] whitespace-pre-wrap">
        {decision.memory_text_snapshot}
      </pre>
      <p className="text-muted-foreground mb-3 text-xs italic">
        {t('web.project.cleanup.reasonPrefix')} {decision.reason}
      </p>
      <div className="flex gap-2">
        <Button
          size="sm"
          variant="default"
          onClick={() => approve.mutate()}
          disabled={approve.isPending || reject.isPending}
        >
          {approve.isPending ? (
            <Loader2 className="mr-1 h-3 w-3 animate-spin" />
          ) : decision.verdict === 'keep' ? (
            <Check className="mr-1 h-3 w-3" />
          ) : (
            <Trash2 className="mr-1 h-3 w-3" />
          )}
          {decision.verdict === 'keep'
            ? t('web.project.cleanup.confirmKeepButton')
            : t('web.project.cleanup.executeButton')}
        </Button>
        <Button
          size="sm"
          variant="outline"
          onClick={() => reject.mutate()}
          disabled={approve.isPending || reject.isPending}
        >
          <X className="mr-1 h-3 w-3" />
          {t('web.project.cleanup.rejectButton')}
        </Button>
      </div>
    </div>
  )
}

// ─── Reset project memory ────────────────────────────────────

interface ResetButtonProps {
  cwd: string
  onDone: () => void
}

function ResetButton({ cwd, onDone }: ResetButtonProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [includeScanner, setIncludeScanner] = useState(false)
  const [includeMemories, setIncludeMemories] = useState(false)
  const [busy, setBusy] = useState(false)

  const handleReset = async () => {
    setBusy(true)
    try {
      const counts = await resetProjectMemory({
        cwd,
        include_scanner_docs: includeScanner,
        include_cleanup_decisions: true,
      })
      let memoryCount = 0
      if (includeMemories) {
        memoryCount = await deleteMemoriesByScope('project', cwd)
      }
      const parts = [
        t('web.project.reset.summary.docs', { count: counts.project_docs }),
        t('web.project.reset.summary.journal', { count: counts.session_logs }),
        t('web.project.reset.summary.proposals', {
          count: counts.project_doc_proposals,
        }),
        t('web.project.reset.summary.cleanup', {
          count: counts.memory_cleanup_decisions,
        }),
      ]
      if (includeMemories)
        parts.push(t('web.project.reset.summary.memories', { count: memoryCount }))
      toast.success(
        t('web.project.reset.successToast', { summary: parts.join(' · ') }),
      )
      onDone()
      setOpen(false)
    } catch (e) {
      toast.error(t('web.project.reset.failedToast'), {
        description: e instanceof Error ? e.message : String(e),
      })
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <Button
        size="sm"
        variant="outline"
        className="text-destructive hover:text-destructive flex-none"
        onClick={() => setOpen(true)}
      >
        <RotateCcw className="mr-1 h-3 w-3" />
        {t('web.project.reset.button')}
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('web.project.reset.dialogTitle')}</DialogTitle>
            <DialogDescription>
              {t('web.project.reset.dialogDescription')}
            </DialogDescription>
          </DialogHeader>
          <div className="text-destructive bg-destructive/10 mb-3 flex items-start gap-2 rounded-md p-3 text-xs">
            <AlertTriangle className="mt-0.5 h-3 w-3 flex-none" />
            <div>
              <strong className="font-mono">{cwd}</strong>
              <br />
              {t('web.project.reset.alwaysDeleted')}
            </div>
          </div>
          <div className="space-y-2 text-sm">
            <label className="flex cursor-pointer items-start gap-2">
              <input
                type="checkbox"
                checked={includeScanner}
                onChange={(e) => setIncludeScanner(e.target.checked)}
                className="mt-0.5"
              />
              <span>
                <strong>{t('web.project.reset.alsoDeleteScannerLabel')}</strong>{' '}
                {t('web.project.reset.alsoDeleteScannerSuffix')}
                <br />
                <span className="text-muted-foreground text-xs">
                  {t('web.project.reset.alsoDeleteScannerHint')}
                </span>
              </span>
            </label>
            <label className="flex cursor-pointer items-start gap-2">
              <input
                type="checkbox"
                checked={includeMemories}
                onChange={(e) => setIncludeMemories(e.target.checked)}
                className="mt-0.5"
              />
              <span>
                <strong>{t('web.project.reset.alsoDeleteMemoriesLabel')}</strong>{' '}
                {t('web.project.reset.alsoDeleteMemoriesSuffix')}
                <br />
                <span className="text-muted-foreground text-xs">
                  {t('web.project.reset.alsoDeleteMemoriesHint')}
                </span>
              </span>
            </label>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setOpen(false)}
              disabled={busy}
            >
              {t('web.project.reset.cancel')}
            </Button>
            <Button
              variant="default"
              className="bg-destructive hover:bg-destructive/90"
              onClick={handleReset}
              disabled={busy}
            >
              {busy ? (
                <Loader2 className="mr-1 h-3 w-3 animate-spin" />
              ) : (
                <Trash2 className="mr-1 h-3 w-3" />
              )}
              {t('web.project.reset.deleteForever')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
