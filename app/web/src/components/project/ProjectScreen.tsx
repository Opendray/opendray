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
  Save,
  Sparkles,
  Trash2,
  X,
} from 'lucide-react'

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
} from '@/lib/projectDocs'
import {
  type CleanupDecision,
  approveDecision,
  listCleanupDecisions,
  rejectDecision,
  runCleanup,
} from '@/lib/memoryCleanup'

interface ProjectScreenProps {
  cwd: string
}

export function ProjectScreen({ cwd }: ProjectScreenProps) {
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

  if (!cwd) {
    return (
      <div className="text-muted-foreground p-8 text-center text-sm">
        Pick a project to manage its memory.
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      <div className="border-b px-4 py-3">
        <div className="text-muted-foreground font-mono text-xs">{cwd}</div>
        <div className="mt-1 flex items-center gap-3 text-xs">
          <span>{(docsQuery.data ?? []).length} docs</span>
          <span>·</span>
          <span>{(logsQuery.data ?? []).length} journal entries</span>
          {inboxCount > 0 && (
            <>
              <span>·</span>
              <Badge variant="danger" className="text-[10px]">
                {inboxCount} pending proposal{inboxCount > 1 ? 's' : ''}
              </Badge>
            </>
          )}
          {cleanupCount > 0 && (
            <>
              <span>·</span>
              <Badge variant="muted" className="text-[10px]">
                {cleanupCount} cleanup pending
              </Badge>
            </>
          )}
        </div>
      </div>

      <Tabs defaultValue="goal" className="flex flex-1 flex-col overflow-hidden">
        <TabsList className="bg-muted/30 mx-4 mt-3 w-fit">
          <TabsTrigger value="goal">Goal</TabsTrigger>
          <TabsTrigger value="plan">Plan</TabsTrigger>
          <TabsTrigger value="tech">Tech</TabsTrigger>
          <TabsTrigger value="activity">Activity</TabsTrigger>
          <TabsTrigger value="journal">Journal</TabsTrigger>
          <TabsTrigger value="inbox" className="relative">
            Inbox
            {inboxCount > 0 && (
              <span className="bg-destructive text-destructive-foreground absolute -top-1 -right-2 flex h-4 min-w-4 items-center justify-center rounded-full px-1 text-[9px] font-bold">
                {inboxCount}
              </span>
            )}
          </TabsTrigger>
          <TabsTrigger value="cleanup" className="relative">
            Cleanup
            {cleanupCount > 0 && (
              <span className="bg-secondary absolute -top-1 -right-2 flex h-4 min-w-4 items-center justify-center rounded-full px-1 text-[9px] font-bold">
                {cleanupCount}
              </span>
            )}
          </TabsTrigger>
        </TabsList>

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
          <ReadonlyDocTab
            doc={docsByKind.tech_stack}
            kindLabel="Tech stack & structure"
            emptyHint="Run a Claude session in this project — scanner refreshes on every spawn."
          />
        </TabsContent>

        <TabsContent value="activity" className="flex-1 overflow-auto p-4">
          <ReadonlyDocTab
            doc={docsByKind.recent_activity}
            kindLabel="Recent activity (git → LLM)"
            emptyHint="The git activity summariser runs every 24h; check back after the next scheduler tick."
          />
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
      toast.success(`${labelFor(kind)} saved`)
    },
    onError: (e: Error) =>
      toast.error('Save failed', { description: e.message }),
  })

  return (
    <div className="space-y-3">
      <div className="text-muted-foreground flex items-center justify-between text-xs">
        <span>
          {doc ? (
            <>
              Updated by <strong>{doc.updated_by}</strong>{' '}
              <span className="ml-1">
                {new Date(doc.updated_at).toLocaleString()}
              </span>
            </>
          ) : (
            <span>No {labelFor(kind).toLowerCase()} set yet.</span>
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
          Save
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
            ? 'What are we building? One paragraph. Read by every agent on spawn.'
            : 'Active plan — what we are doing right now and what is next. Updated as work progresses.'
        }
        className="font-mono text-sm"
      />
    </div>
  )
}

// ─── Read-only doc (tech_stack / recent_activity) ────────────

interface ReadonlyDocTabProps {
  doc?: ProjectDoc
  kindLabel: string
  emptyHint: string
}

function ReadonlyDocTab({ doc, kindLabel, emptyHint }: ReadonlyDocTabProps) {
  if (!doc) {
    return (
      <div className="text-muted-foreground text-sm">
        <p className="mb-2">No {kindLabel} captured yet.</p>
        <p className="text-xs">{emptyHint}</p>
      </div>
    )
  }
  return (
    <div className="space-y-3">
      <div className="text-muted-foreground flex items-center justify-between text-xs">
        <span>
          Generated by <strong>{doc.updated_by}</strong> · last refresh{' '}
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
  if (loading) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 text-sm">
        <Loader2 className="h-3 w-3 animate-spin" /> Loading…
      </div>
    )
  }
  if (entries.length === 0) {
    return (
      <p className="text-muted-foreground text-sm">
        No journal entries yet. Each session-end appends one automatically.
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
  if (loading) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 text-sm">
        <Loader2 className="h-3 w-3 animate-spin" /> Loading…
      </div>
    )
  }
  if (proposals.length === 0) {
    return (
      <div className="text-muted-foreground flex flex-col items-center gap-2 py-12 text-sm">
        <Inbox className="text-muted-foreground/50 h-8 w-8" />
        <p>Inbox empty.</p>
        <p className="text-xs">
          Agents file proposals here via `project_goal_set` / `project_plan_set`
          MCP tools.
        </p>
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
  const [confirmOpen, setConfirmOpen] = useState(false)
  const approve = useMutation({
    mutationFn: () => approveProposal(proposal.id),
    onSuccess: () => {
      toast.success(`${labelFor(proposal.kind)} updated`)
      onChange()
      setConfirmOpen(false)
    },
    onError: (e: Error) => {
      toast.error('Approve failed', { description: e.message })
      // refresh anyway — likely 409 already-decided
      onChange()
      setConfirmOpen(false)
    },
  })
  const reject = useMutation({
    mutationFn: () => rejectProposal(proposal.id),
    onSuccess: () => {
      toast.success('Rejected')
      onChange()
    },
    onError: (e: Error) => {
      toast.error('Reject failed', { description: e.message })
      onChange()
    },
  })

  return (
    <div className="bg-card rounded-md border p-3">
      <div className="mb-2 flex items-center gap-2">
        <Badge variant="default">{labelFor(proposal.kind)}</Badge>
        <span className="text-muted-foreground text-[11px]">
          {new Date(proposal.created_at).toLocaleString()}
        </span>
        {proposal.proposed_by_session && (
          <span className="text-muted-foreground font-mono text-[10px]">
            ses {proposal.proposed_by_session.slice(-8)}
          </span>
        )}
      </div>
      {proposal.reason && (
        <p className="mb-3 text-sm">{proposal.reason}</p>
      )}
      <div className="text-destructive bg-destructive/10 mb-3 flex items-start gap-2 rounded-md p-2 text-xs">
        <AlertTriangle className="mt-0.5 h-3 w-3 flex-none" />
        <div>
          <strong>Approve will REPLACE the current {labelFor(proposal.kind).toLowerCase()} entirely.</strong>{' '}
          Review the diff below; this isn't a merge.
        </div>
      </div>
      <div className="mb-3 grid grid-cols-1 gap-2 md:grid-cols-2">
        <DiffBlock
          label="Current"
          body={proposal.prior_content ?? '(empty)'}
        />
        <DiffBlock
          label="Proposed"
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
          Approve
        </Button>
        <Button
          size="sm"
          variant="outline"
          onClick={() => reject.mutate()}
          disabled={approve.isPending || reject.isPending}
        >
          <X className="mr-1 h-3 w-3" />
          Reject
        </Button>
      </div>

      <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              Replace {labelFor(proposal.kind).toLowerCase()}?
            </DialogTitle>
            <DialogDescription>
              The current {labelFor(proposal.kind).toLowerCase()} will be
              overwritten with the proposed content. This cannot be undone via
              this UI (you can manually edit it back).
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setConfirmOpen(false)}
              disabled={approve.isPending}
            >
              Cancel
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
              Confirm replace
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
  const runMutation = useMutation({
    mutationFn: () => runCleanup({ scope: 'project', scope_key: cwd }),
    onSuccess: (res) => {
      toast.success(
        `Cleanup run: ${res.decided} decisions queued (${res.scanned} scanned)`,
      )
      onChange()
    },
    onError: (e: Error) =>
      toast.error('Cleanup run failed', { description: e.message }),
  })

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <p className="text-muted-foreground text-xs">
          The LLM librarian proposes keep / stale / duplicate verdicts for
          this project's memories. You approve before anything is deleted.
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
          Run cleanup now
        </Button>
      </div>
      {loading ? (
        <Loader2 className="h-3 w-3 animate-spin" />
      ) : decisions.length === 0 ? (
        <p className="text-muted-foreground py-8 text-center text-sm">
          No pending decisions. Either nothing aged into eligibility or the
          last run found everything load-bearing.
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
  const approve = useMutation({
    mutationFn: () => approveDecision(decision.id),
    onSuccess: () => {
      toast.success(`${labelForVerdict(decision.verdict)} executed`)
      onChange()
    },
    onError: (e: Error) => {
      toast.error('Approve failed', { description: e.message })
      onChange()
    },
  })
  const reject = useMutation({
    mutationFn: () => rejectDecision(decision.id),
    onSuccess: () => {
      toast.success('Rejected — memory kept')
      onChange()
    },
    onError: (e: Error) => {
      toast.error('Reject failed', { description: e.message })
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
            → merge into {decision.merge_into.slice(-8)}
          </span>
        )}
      </div>
      <pre className="bg-muted/20 mb-2 max-h-32 overflow-auto rounded p-2 font-mono text-[11px] whitespace-pre-wrap">
        {decision.memory_text_snapshot}
      </pre>
      <p className="text-muted-foreground mb-3 text-xs italic">
        Reason: {decision.reason}
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
          {decision.verdict === 'keep' ? 'Confirm keep' : 'Execute'}
        </Button>
        <Button
          size="sm"
          variant="outline"
          onClick={() => reject.mutate()}
          disabled={approve.isPending || reject.isPending}
        >
          <X className="mr-1 h-3 w-3" />
          Reject
        </Button>
      </div>
    </div>
  )
}

// ─── helpers ─────────────────────────────────────────────────

function labelFor(kind: DocKind | 'goal' | 'plan'): string {
  switch (kind) {
    case 'goal':
      return 'Goal'
    case 'plan':
      return 'Plan'
    case 'tech_stack':
      return 'Tech stack'
    case 'recent_activity':
      return 'Recent activity'
  }
  return kind
}

function labelForVerdict(v: string): string {
  if (v === 'stale') return 'Delete'
  if (v === 'duplicate') return 'Merge'
  return 'Keep'
}
