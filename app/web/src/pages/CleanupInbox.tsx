// /memory/cleanup — cross-project cleanup inbox. Lists all pending
// memory_cleanup_decisions across every project so an operator can
// triage cleanly without clicking into each project. Mirrors
// app/mobile/lib/features/memory_cleanup/cleanup_inbox_screen.dart.

import { useMemo } from 'react'
import { Link } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Check, ChevronRight, Loader2, Trash2, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'

import {
  type CleanupDecision,
  approveDecision,
  listCleanupDecisions,
  rejectDecision,
} from '@/lib/memoryCleanup'

export function CleanupInboxPage() {
  const { t } = useTranslation()
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
      const key = `${d.memory_scope}:${d.memory_scope_key || t('web.cleanupInbox.globalScope')}`
      if (!m.has(key)) m.set(key, [])
      m.get(key)!.push(d)
    }
    return [...m.entries()].sort((a, b) =>
      a[0].localeCompare(b[0]),
    )
  }, [query.data, t])

  const refresh = () =>
    qc.invalidateQueries({ queryKey: ['cleanup-decisions'] })

  if (query.isLoading) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 p-6 text-sm">
        <Loader2 className="h-3 w-3 animate-spin" />{' '}
        {t('web.cleanupInbox.loading')}
      </div>
    )
  }

  if ((query.data ?? []).length === 0) {
    return (
      <div className="mx-auto max-w-2xl space-y-2 p-12 text-center">
        <h1 className="text-lg font-semibold">
          {t('web.cleanupInbox.emptyTitle')}
        </h1>
        <p className="text-muted-foreground text-sm">
          {t('web.cleanupInbox.emptyDescription')}
        </p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-4xl space-y-6 p-6">
      <div>
        <h1 className="text-xl font-semibold">{t('web.cleanupInbox.title')}</h1>
        <p className="text-muted-foreground text-sm">
          {t('web.cleanupInbox.subtitle')}
        </p>
      </div>

      {grouped.map(([key, decisions]) => {
        const [scope, scopeKey] = key.split(':', 2)
        const orphan = scope === 'project' && isLikelyOrphanScope(scopeKey)
        return (
          <section key={key} className="space-y-3">
            <header className="flex items-center justify-between border-b pb-2">
              <div className="flex items-center gap-2">
                <Badge variant="outline">{scope}</Badge>
                <span className="font-mono text-xs">{scopeKey}</span>
                {orphan && (
                  <Badge
                    variant="muted"
                    className="text-[9px]"
                    title={t('web.cleanupInbox.orphanTitle')}
                  >
                    {t('web.cleanupInbox.orphanBadge')}
                  </Badge>
                )}
              </div>
              {scope === 'project' && scopeKey && !orphan && (
                <Link
                  to="/memory/project"
                  search={{ cwd: scopeKey }}
                  className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1 text-xs"
                >
                  {t('web.cleanupInbox.openProject')}{' '}
                  <ChevronRight className="h-3 w-3" />
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

// Same heuristic as in pages/Project.tsx — scope_keys with fewer
// than 2 non-empty path segments (e.g. `/Users/`) are bug data
// from old mirror imports, not real projects.
function isLikelyOrphanScope(cwd: string): boolean {
  if (!cwd) return false
  const parts = cwd.split('/').filter((s) => s.length > 0)
  return parts.length < 2
}

function CleanupRow({
  decision,
  onChange,
}: {
  decision: CleanupDecision
  onChange: () => void
}) {
  const { t } = useTranslation()
  const approve = useMutation({
    mutationFn: () => approveDecision(decision.id),
    onSuccess: () => {
      toast.success(
        decision.verdict === 'keep'
          ? t('web.cleanupInbox.approvedKeptToast')
          : t('web.cleanupInbox.approvedExecutedToast', {
              verdict: decision.verdict,
            }),
      )
      onChange()
    },
    onError: (e: Error) => {
      toast.error(t('web.cleanupInbox.approveFailedToast'), {
        description: e.message,
      })
      onChange()
    },
  })
  const reject = useMutation({
    mutationFn: () => rejectDecision(decision.id),
    onSuccess: () => {
      toast.success(t('web.cleanupInbox.rejectedToast'))
      onChange()
    },
    onError: (e: Error) => {
      toast.error(t('web.cleanupInbox.rejectFailedToast'), {
        description: e.message,
      })
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
            {t('web.cleanupInbox.mergeIntoPrefix')}{' '}
            {decision.merge_into.slice(-8)}
          </span>
        )}
      </div>
      <pre className="bg-muted/20 mb-2 max-h-32 overflow-auto rounded p-2 font-mono text-[11px] whitespace-pre-wrap">
        {decision.memory_text_snapshot}
      </pre>
      <p className="text-muted-foreground mb-3 text-xs italic">
        {t('web.cleanupInbox.reasonPrefix')} {decision.reason}
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
            ? t('web.cleanupInbox.confirmKeepButton')
            : t('web.cleanupInbox.executeButton')}
        </Button>
        <Button
          size="sm"
          variant="outline"
          onClick={() => reject.mutate()}
          disabled={approve.isPending || reject.isPending}
        >
          <X className="mr-1 h-3 w-3" />
          {t('web.cleanupInbox.rejectButton')}
        </Button>
      </div>
    </div>
  )
}
