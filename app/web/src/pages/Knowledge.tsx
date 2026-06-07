import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { APIError } from '@/lib/api'
import {
  listKnowledgeNodes,
  searchKnowledge,
  getKnowledgeGraph,
  promoteKnowledgeNode,
  skillifyKnowledgeNode,
  type KnowledgeNode,
  type KnowledgeSearchHit,
} from '@/lib/knowledge'

const KIND_STYLES: Record<string, string> = {
  entity: 'bg-blue-500/15 text-blue-400',
  fact: 'bg-zinc-500/15 text-zinc-300',
  playbook: 'bg-amber-500/15 text-amber-400',
  skill: 'bg-emerald-500/15 text-emerald-400',
}

function KindBadge({ kind }: { kind: string }) {
  return (
    <span
      className={`rounded px-1.5 py-0.5 text-[10px] uppercase tracking-wide ${
        KIND_STYLES[kind] ?? 'bg-zinc-500/15 text-zinc-300'
      }`}
    >
      {kind}
    </span>
  )
}

export function KnowledgePage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [query, setQuery] = useState('')
  const [cwd, setCwd] = useState('')
  const [hits, setHits] = useState<KnowledgeSearchHit[] | null>(null)
  const [selected, setSelected] = useState<KnowledgeNode | null>(null)

  const browse = useQuery({
    queryKey: ['knowledge-nodes'],
    queryFn: () => listKnowledgeNodes(),
  })

  const graph = useQuery({
    queryKey: ['knowledge-graph', selected?.id],
    queryFn: () => getKnowledgeGraph(selected!.id),
    enabled: !!selected,
  })

  const runSearch = async () => {
    if (!query.trim()) {
      setHits(null)
      return
    }
    try {
      setHits(await searchKnowledge(query.trim(), cwd.trim(), 20))
    } catch (e) {
      toast.error(e instanceof APIError ? e.message : String(e))
    }
  }

  const promote = useMutation({
    mutationFn: (id: string) => promoteKnowledgeNode(id, 'global'),
    onSuccess: () => {
      toast.success(t('web.knowledge.promoted'))
      qc.invalidateQueries({ queryKey: ['knowledge-nodes'] })
    },
    onError: () => toast.error(t('web.knowledge.actionFailed')),
  })

  const skillify = useMutation({
    mutationFn: (id: string) => skillifyKnowledgeNode(id),
    onSuccess: (n) => {
      toast.success(t('web.knowledge.skillified', { title: n.title }))
      qc.invalidateQueries({ queryKey: ['knowledge-nodes'] })
    },
    onError: () => toast.error(t('web.knowledge.actionFailed')),
  })

  const nodes = hits ? hits.map((h) => h.node) : (browse.data ?? [])

  return (
    <div className="flex flex-col h-full">
      <header className="px-4 py-3 border-b border-border">
        <h1 className="text-lg font-semibold">{t('web.knowledge.title')}</h1>
        <p className="text-sm text-muted-foreground">
          {t('web.knowledge.subtitle')}
        </p>
        <div className="mt-3 flex gap-2">
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && runSearch()}
            placeholder={t('web.knowledge.searchPlaceholder')}
            className="flex-1 rounded-md border border-border bg-card px-3 py-1.5 text-sm"
          />
          <input
            value={cwd}
            onChange={(e) => setCwd(e.target.value)}
            placeholder={t('web.knowledge.cwdPlaceholder')}
            className="w-72 rounded-md border border-border bg-card px-3 py-1.5 text-sm"
          />
          <button
            onClick={runSearch}
            className="rounded-md bg-primary px-3 py-1.5 text-sm text-primary-foreground"
          >
            {t('web.knowledge.search')}
          </button>
          {hits && (
            <button
              onClick={() => {
                setHits(null)
                setQuery('')
              }}
              className="rounded-md border border-border px-3 py-1.5 text-sm"
            >
              {t('web.knowledge.browse')}
            </button>
          )}
        </div>
      </header>

      <div className="flex flex-1 min-h-0">
        <div className="w-1/2 overflow-auto border-r border-border">
          {nodes.length === 0 ? (
            <p className="p-4 text-sm text-muted-foreground">
              {browse.isLoading
                ? '…'
                : hits
                  ? t('web.knowledge.noResults')
                  : t('web.knowledge.empty')}
            </p>
          ) : (
            <ul className="divide-y divide-border">
              {nodes.map((n) => (
                <li key={n.id}>
                  <button
                    onClick={() => setSelected(n)}
                    className={`w-full text-left px-4 py-2 hover:bg-card ${
                      selected?.id === n.id ? 'bg-card' : ''
                    }`}
                  >
                    <div className="flex items-center gap-2">
                      <KindBadge kind={n.kind} />
                      <span className="text-sm truncate">{n.title}</span>
                    </div>
                    <div className="text-[11px] text-muted-foreground">
                      {n.scope}
                      {n.scope_key ? ` · ${n.scope_key}` : ''}
                    </div>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        <div className="w-1/2 overflow-auto p-4">
          {!selected ? (
            <p className="text-sm text-muted-foreground">
              {t('web.knowledge.selectHint')}
            </p>
          ) : (
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <KindBadge kind={selected.kind} />
                <h2 className="text-base font-medium">{selected.title}</h2>
              </div>
              <div className="text-xs text-muted-foreground">
                {selected.scope}
                {selected.scope_key ? ` · ${selected.scope_key}` : ''} ·{' '}
                {selected.maturity}
              </div>
              {selected.body && (
                <pre className="whitespace-pre-wrap rounded-md bg-card p-3 text-sm">
                  {selected.body}
                </pre>
              )}
              <div className="flex gap-2">
                {selected.kind === 'playbook' && (
                  <button
                    onClick={() => skillify.mutate(selected.id)}
                    disabled={skillify.isPending}
                    className="rounded-md bg-emerald-600/80 px-3 py-1.5 text-sm text-white disabled:opacity-50"
                  >
                    {t('web.knowledge.skillify')}
                  </button>
                )}
                {selected.scope !== 'global' && (
                  <button
                    onClick={() => promote.mutate(selected.id)}
                    disabled={promote.isPending}
                    className="rounded-md border border-border px-3 py-1.5 text-sm disabled:opacity-50"
                  >
                    {t('web.knowledge.promote')}
                  </button>
                )}
              </div>
              <div>
                <h3 className="text-sm font-medium mb-1">
                  {t('web.knowledge.neighbors')}
                </h3>
                {graph.data?.neighbors?.length ? (
                  <ul className="space-y-1">
                    {graph.data.neighbors.map((nb, i) => (
                      <li key={i}>
                        <button
                          onClick={() => setSelected(nb.node)}
                          className="text-left text-sm hover:underline"
                        >
                          <span className="text-[11px] text-muted-foreground">
                            {nb.direction === 'out' ? '→' : '←'} {nb.edge_type}:{' '}
                          </span>
                          {nb.node.title}
                        </button>
                      </li>
                    ))}
                  </ul>
                ) : (
                  <p className="text-xs text-muted-foreground">—</p>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
