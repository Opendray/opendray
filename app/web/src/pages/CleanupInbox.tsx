// /memory/cleanup — cross-project cleanup inbox. Lists all pending
// memory_cleanup_decisions across every project so an operator can
// triage cleanly without clicking into each project. Mirrors
// app/mobile/lib/features/memory_cleanup/cleanup_inbox_screen.dart.

import { useMemo } from 'react'
import { Link } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Check, ChevronRight, Loader2, Trash2, X } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'

import {
  type CleanupDecision,
  approveDecision,
  listCleanupDecisions,
  rejectDecision,
} from '@/lib/memoryCleanup'

export function CleanupInboxPage() {
  const qc = useQueryClient()

  const query = useQuery({
    queryKey: ['cleanup-decisions', 'all-pending'],
    queryFn: () =>
      listCleanupDecisions({ status: 'pending', limit: 200 }),
    staleTime: 10_000,
  })

  const grouped = useMemo(() => {
    const m = new Map<string, CleanupDecision[]>()
    for (const d of query.data ?? []) {
      const key = `${d.memory_scope}:${d.memory_scope_key || '(global)'}`
      if (!m.has(key)) m.set(key, [])
      m.get(key)!.push(d)
    }
    return [...m.entries()].sort((a, b) =>
      a[0].localeCompare(b[0]),
    )
  }, [query.data])

  const refresh = () =>
    qc.invalidateQueries({ queryKey: ['cleanup-decisions'] })

  if (query.isLoading) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 p-6 text-sm">
        <Loader2 className="h-3 w-3 animate-spin" /> Loading…
      </div>
    )
  }

  if ((query.data ?? []).length === 0) {
    return (
      <div className="mx-auto max-w-2xl space-y-2 p-12 text-center">
        <h1 className="text-lg font-semibold">Cleanup inbox empty</h1>
        <p className="text-muted-foreground text-sm">
          No pending cleanup decisions across any project. The LLM librarian
          either hasn't run yet for the eligible memories, or it found
          everything load-bearing.
        </p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-4xl space-y-6 p-6">
      <div>
        <h1 className="text-xl font-semibold">Cleanup inbox</h1>
        <p className="text-muted-foreground text-sm">
          Cross-project pending decisions from the LLM memory librarian.
          Approving stale → deletes, approving duplicate → merges,
          approving keep → freezes the entry from being re-judged for a while.
        </p>
      </div>

      {grouped.map(([key, decisions]) => {
        const [scope, scopeKey] = key.split(':', 2)
        return (
          <section key={key} className="space-y-3">
            <header className="flex items-center justify-between border-b pb-2">
              <div>
                <Badge variant="outline" className="mr-2">
                  {scope}
                </Badge>
                <span className="font-mono text-xs">{scopeKey}</span>
              </div>
              {scope === 'project' && scopeKey && (
                <Link
                  to="/memory/project"
                  search={{ cwd: scopeKey }}
                  className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1 text-xs"
                >
                  Open project <ChevronRight className="h-3 w-3" />
                </Link>
              )}
            </header>
            {decisions.map((d) => (
              <CleanupRow key={d.id} decision={d} onChange={refresh} />
            ))}
          </section>
        )
      })}
    </div>
  )
}

function CleanupRow({
  decision,
  onChange,
}: {
  decision: CleanupDecision
  onChange: () => void
}) {
  const approve = useMutation({
    mutationFn: () => approveDecision(decision.id),
    onSuccess: () => {
      toast.success(
        decision.verdict === 'keep' ? 'Kept' : `${decision.verdict} executed`,
      )
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

  const verdictVariant =
    decision.verdict === 'stale'
      ? 'danger'
      : decision.verdict === 'duplicate'
        ? 'muted'
        : 'outline'

  return (
    <div className="bg-card rounded-md border p-3">
      <div className="mb-2 flex items-center gap-2">
        <Badge variant={verdictVariant as 'danger' | 'muted' | 'outline'}>
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
