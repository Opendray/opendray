import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

import { APIError } from '@/lib/api'
import {
  listKnowledgeNodes,
  searchKnowledge,
  getKnowledgeGraph,
  promoteKnowledgeNode,
  skillifyKnowledgeNode,
  deleteKnowledgeNode,
  draftKB,
  type KnowledgeNode,
  type KnowledgeSearchHit,
  type KnowledgeKind,
  type KnowledgeScope,
} from '@/lib/knowledge'
import {
  getProjectDoc,
  putProjectDoc,
  GLOBAL_CWD,
  type DocKind,
} from '@/lib/projectDocs'

// ── shared bits ───────────────────────────────────────────────

function TabBtn({
  active,
  onClick,
  children,
}: {
  active: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      onClick={onClick}
      className={`rounded-t-md px-3 py-1.5 text-sm ${
        active
          ? 'bg-card font-medium border-x border-t border-border'
          : 'text-muted-foreground hover:text-foreground'
      }`}
    >
      {children}
    </button>
  )
}

// strip the drafter's hidden signature marker before display
function stripSig(s: string): string {
  return s
    .split('\n')
    .filter((l) => !l.includes('kb-sig:'))
    .join('\n')
    .trim()
}

// explicit markdown styling so we don't depend on the typography plugin
const MD = {
  h1: (p: any) => <h1 className="mt-4 mb-2 text-lg font-semibold" {...p} />,
  h2: (p: any) => (
    <h2 className="mt-4 mb-1.5 text-base font-semibold border-b border-border pb-1" {...p} />
  ),
  h3: (p: any) => <h3 className="mt-3 mb-1 text-sm font-semibold" {...p} />,
  p: (p: any) => <p className="my-1.5 text-sm leading-relaxed" {...p} />,
  ul: (p: any) => <ul className="my-1.5 ml-5 list-disc space-y-0.5 text-sm" {...p} />,
  ol: (p: any) => <ol className="my-1.5 ml-5 list-decimal space-y-0.5 text-sm" {...p} />,
  li: (p: any) => <li className="leading-relaxed" {...p} />,
  code: (p: any) => (
    <code className="rounded bg-muted px-1 py-0.5 text-[12px] font-mono" {...p} />
  ),
  strong: (p: any) => <strong className="font-semibold" {...p} />,
  a: (p: any) => <a className="text-primary underline" {...p} />,
  hr: () => <hr className="my-3 border-border" />,
}

// ── Knowledge Base (curated pages) ────────────────────────────

interface KBPage {
  kind: DocKind
  cwd: string
}

function KnowledgeBaseView() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [handbookCwd, setHandbookCwd] = useState('')
  const [sel, setSel] = useState<KBPage>({
    kind: 'kb_infrastructure',
    cwd: GLOBAL_CWD,
  })
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState('')

  const doc = useQuery({
    queryKey: ['kb-doc', sel.cwd, sel.kind],
    queryFn: () => getProjectDoc(sel.cwd, sel.kind),
  })

  const save = useMutation({
    mutationFn: () =>
      putProjectDoc({ cwd: sel.cwd, kind: sel.kind, content: draft }),
    onSuccess: () => {
      setEditing(false)
      toast.success(t('web.knowledge.kb.saved'))
      qc.invalidateQueries({ queryKey: ['kb-doc'] })
    },
    onError: () => toast.error(t('web.knowledge.actionFailed')),
  })

  const unlock = useMutation({
    mutationFn: () =>
      putProjectDoc({
        cwd: sel.cwd,
        kind: sel.kind,
        content: stripSig(doc.data?.content ?? ''),
        updatedBy: 'agent',
      }),
    onSuccess: () => {
      toast.success(t('web.knowledge.kb.unlocked'))
      qc.invalidateQueries({ queryKey: ['kb-doc'] })
    },
    onError: () => toast.error(t('web.knowledge.actionFailed')),
  })

  const regen = useMutation({
    mutationFn: () => draftKB(),
    onSuccess: () => toast.success(t('web.knowledge.kb.regenerating')),
    onError: () => toast.error(t('web.knowledge.actionFailed')),
  })

  const select = (p: KBPage) => {
    setSel(p)
    setEditing(false)
  }

  const content = stripSig(doc.data?.content ?? '')
  const exists = !!doc.data?.id
  const locked = doc.data?.updated_by === 'operator'

  const globalKinds: DocKind[] = [
    'kb_infrastructure',
    'kb_conventions',
    'kb_lessons',
    'kb_reusable',
  ]

  return (
    <div className="flex flex-1 min-h-0 border border-border rounded-b-md rounded-tr-md">
      {/* nav */}
      <div className="w-64 shrink-0 overflow-auto border-r border-border p-2">
        <p className="px-2 pb-1 pt-1 text-[11px] uppercase tracking-wide text-muted-foreground">
          {t('web.knowledge.kb.global')}
        </p>
        {globalKinds.map((k) => (
          <button
            key={k}
            onClick={() => select({ kind: k, cwd: GLOBAL_CWD })}
            className={`block w-full rounded px-2 py-1.5 text-left text-sm ${
              sel.cwd === GLOBAL_CWD && sel.kind === k
                ? 'bg-primary text-primary-foreground'
                : 'hover:bg-card'
            }`}
          >
            {t(`web.knowledge.kb.kinds.${k}`)}
          </button>
        ))}

        <p className="px-2 pb-1 pt-3 text-[11px] uppercase tracking-wide text-muted-foreground">
          {t('web.knowledge.kb.projectHandbook')}
        </p>
        <input
          value={handbookCwd}
          onChange={(e) => setHandbookCwd(e.target.value)}
          placeholder={t('web.knowledge.cwdPlaceholder')}
          className="mb-1 w-full rounded-md border border-border bg-card px-2 py-1 text-xs"
        />
        <button
          disabled={!handbookCwd.trim()}
          onClick={() =>
            select({ kind: 'kb_handbook', cwd: handbookCwd.trim() })
          }
          className={`block w-full rounded px-2 py-1.5 text-left text-sm disabled:opacity-40 ${
            sel.kind === 'kb_handbook' && sel.cwd === handbookCwd.trim()
              ? 'bg-primary text-primary-foreground'
              : 'hover:bg-card'
          }`}
        >
          {t('web.knowledge.kb.kinds.kb_handbook')}
        </button>
      </div>

      {/* content */}
      <div className="flex flex-1 flex-col min-h-0">
        <div className="flex items-center gap-2 border-b border-border px-4 py-2">
          <h2 className="text-sm font-medium">
            {sel.kind === 'kb_handbook'
              ? `${t('web.knowledge.kb.kinds.kb_handbook')} · ${sel.cwd.split('/').pop()}`
              : t(`web.knowledge.kb.kinds.${sel.kind}`)}
          </h2>
          {exists && (
            <span
              className={`rounded px-1.5 py-0.5 text-[10px] ${
                locked
                  ? 'bg-amber-500/15 text-amber-400'
                  : 'bg-emerald-500/15 text-emerald-400'
              }`}
            >
              {locked ? t('web.knowledge.kb.locked') : t('web.knowledge.kb.aiDrafted')}
            </span>
          )}
          <div className="ml-auto flex gap-2">
            {!editing && (
              <button
                onClick={() => {
                  setDraft(content)
                  setEditing(true)
                }}
                className="rounded-md border border-border px-2.5 py-1 text-xs"
              >
                {t('web.knowledge.kb.edit')}
              </button>
            )}
            {!editing && locked && (
              <button
                onClick={() => unlock.mutate()}
                disabled={unlock.isPending}
                className="rounded-md border border-border px-2.5 py-1 text-xs disabled:opacity-50"
              >
                {t('web.knowledge.kb.unlock')}
              </button>
            )}
            {!editing && (
              <button
                onClick={() => regen.mutate()}
                disabled={regen.isPending}
                className="rounded-md border border-border px-2.5 py-1 text-xs disabled:opacity-50"
              >
                {t('web.knowledge.kb.regenerate')}
              </button>
            )}
          </div>
        </div>

        <div className="flex-1 overflow-auto p-4">
          {editing ? (
            <div className="flex h-full flex-col gap-2">
              <textarea
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                className="flex-1 resize-none rounded-md border border-border bg-card p-3 font-mono text-sm"
              />
              <div className="flex gap-2">
                <button
                  onClick={() => save.mutate()}
                  disabled={save.isPending}
                  className="rounded-md bg-primary px-3 py-1.5 text-sm text-primary-foreground disabled:opacity-50"
                >
                  {t('web.knowledge.kb.save')}
                </button>
                <button
                  onClick={() => setEditing(false)}
                  className="rounded-md border border-border px-3 py-1.5 text-sm"
                >
                  {t('web.knowledge.kb.cancel')}
                </button>
                <span className="self-center text-[11px] text-muted-foreground">
                  {t('web.knowledge.kb.editHint')}
                </span>
              </div>
            </div>
          ) : doc.isLoading ? (
            <p className="text-sm text-muted-foreground">…</p>
          ) : content ? (
            <ReactMarkdown remarkPlugins={[remarkGfm]} components={MD}>
              {content}
            </ReactMarkdown>
          ) : (
            <p className="text-sm text-muted-foreground">
              {t('web.knowledge.kb.empty')}
            </p>
          )}
        </div>
      </div>
    </div>
  )
}

// ── Graph browser (the raw node graph — secondary) ────────────

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

function GraphBrowser() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [query, setQuery] = useState('')
  const [cwd, setCwd] = useState('')
  const [hits, setHits] = useState<KnowledgeSearchHit[] | null>(null)
  const [selected, setSelected] = useState<KnowledgeNode | null>(null)
  const [kind, setKind] = useState<'all' | KnowledgeKind>('entity')
  const [scope, setScope] = useState<'all' | KnowledgeScope>('all')

  const browse = useQuery({
    queryKey: ['knowledge-nodes', kind, scope, cwd],
    queryFn: () =>
      listKnowledgeNodes({
        kind: kind === 'all' ? undefined : kind,
        scope: scope === 'all' ? undefined : scope,
        scopeKey: cwd.trim() || undefined,
      }),
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

  const del = useMutation({
    mutationFn: (id: string) => deleteKnowledgeNode(id),
    onSuccess: () => {
      toast.success(t('web.knowledge.deleted'))
      setSelected(null)
      qc.invalidateQueries({ queryKey: ['knowledge-nodes'] })
    },
    onError: () => toast.error(t('web.knowledge.actionFailed')),
  })

  const nodes = hits ? hits.map((h) => h.node) : (browse.data ?? [])

  return (
    <div className="flex flex-1 flex-col min-h-0 border border-border rounded-b-md rounded-tr-md">
      <header className="border-b border-border px-4 py-3">
        <div className="flex gap-2">
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
        <div className="mt-2 flex flex-wrap gap-1">
          {/* 'fact' retired (P-G): facts live in Memory, not the graph. */}
          {(['all', 'entity', 'playbook', 'skill'] as const).map((k) => (
            <button
              key={k}
              onClick={() => setKind(k)}
              className={`rounded px-2 py-1 text-xs ${
                kind === k
                  ? 'bg-primary text-primary-foreground'
                  : 'border border-border text-muted-foreground'
              }`}
            >
              {t(`web.knowledge.kinds.${k}`)}
            </button>
          ))}
        </div>
        <div className="mt-1 flex flex-wrap items-center gap-1">
          <span className="mr-1 text-[11px] text-muted-foreground">
            {t('web.knowledge.scope')}
          </span>
          {(['all', 'global', 'project', 'domain'] as const).map((sc) => (
            <button
              key={sc}
              onClick={() => setScope(sc)}
              className={`rounded px-2 py-1 text-xs ${
                scope === sc
                  ? 'bg-primary text-primary-foreground'
                  : 'border border-border text-muted-foreground'
              }`}
            >
              {t(`web.knowledge.scopes.${sc}`)}
            </button>
          ))}
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
                <button
                  onClick={() => {
                    if (window.confirm(t('web.knowledge.deleteConfirm')))
                      del.mutate(selected.id)
                  }}
                  disabled={del.isPending}
                  className="rounded-md border border-red-500/40 px-3 py-1.5 text-sm text-red-400 disabled:opacity-50"
                >
                  {t('web.knowledge.delete')}
                </button>
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

// ── page ──────────────────────────────────────────────────────

export function KnowledgePage() {
  const { t } = useTranslation()
  const [tab, setTab] = useState<'kb' | 'graph'>('kb')

  return (
    <div className="flex h-full flex-col">
      <header className="px-4 pt-3">
        <h1 className="text-lg font-semibold">{t('web.knowledge.title')}</h1>
        <p className="text-sm text-muted-foreground">
          {t('web.knowledge.subtitle')}
        </p>
        <div className="mt-2 flex items-center gap-1">
          <TabBtn active={tab === 'kb'} onClick={() => setTab('kb')}>
            {t('web.knowledge.kb.tab')}
          </TabBtn>
          <TabBtn active={tab === 'graph'} onClick={() => setTab('graph')}>
            {t('web.knowledge.kb.graphTab')}
          </TabBtn>
        </div>
      </header>
      <div className="flex flex-1 flex-col min-h-0 px-4 pb-4">
        {tab === 'kb' ? <KnowledgeBaseView /> : <GraphBrowser />}
      </div>
    </div>
  )
}
