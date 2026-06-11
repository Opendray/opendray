import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

import {
  listKnowledgeNodes,
  getKnowledgeGraph,
  listImpactEntities,
  listRetirementCandidates,
  candidateScore,
  skillifyKnowledgeNode,
  setKnowledgeNodeEnabled,
  deleteKnowledgeNode,
  draftKB,
  type KnowledgeNode,
  type RetirementReason,
} from '@/lib/knowledge'
import {
  getProjectDoc,
  putProjectDoc,
  listPendingProposals,
  approveProposal,
  rejectProposal,
  listBlueprintSections,
  putBlueprintSection,
  deleteBlueprintSection,
  GLOBAL_CWD,
  type BlueprintSection,
  type DocKind,
  type DocProposal,
} from '@/lib/projectDocs'
import { CurationChat } from '@/components/cortex/CurationChat'
import { Switch } from '@/components/ui/switch'
import { Loader2, Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

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

// ── Knowledge Base (the cross-project compounding asset) ──────

// Knowledge's two natures (Experience Flywheel §2). Foundational pages
// are binding guardrails injected into every project; Emergent pages
// are distilled guidance. The PAGE SET is dynamic since the knowledge
// blueprint: the classic four are pinned, and the operator (or AI)
// can add new kb_* pages so every knowledge domain gets its own
// fine-grained, individually-indexed document.
const CLASSIC_KB_KINDS = new Set([
  'kb_infrastructure',
  'kb_conventions',
  'kb_lessons',
  'kb_reusable',
])

function kbPageLabel(sec: BlueprintSection, t: (k: string) => string): string {
  return CLASSIC_KB_KINDS.has(sec.slug)
    ? t(`web.knowledge.kb.kinds.${sec.slug}`)
    : sec.title
}

function NavSection({
  label,
  hint,
  sections,
  sel,
  onSelect,
}: {
  label: string
  hint: string
  sections: BlueprintSection[]
  sel: DocKind
  onSelect: (k: DocKind) => void
}) {
  const { t } = useTranslation()
  return (
    <>
      <p className="text-muted-foreground px-2 pt-3 pb-0.5 text-[11px] font-medium uppercase tracking-wide">
        {label}
      </p>
      <p className="text-muted-foreground/70 px-2 pb-1 text-[10px] leading-tight">
        {hint}
      </p>
      {sections.map((sec) => (
        <button
          key={sec.slug}
          onClick={() => onSelect(sec.slug)}
          className={`block w-full rounded px-2 py-1.5 text-left text-sm ${
            sel === sec.slug ? 'bg-primary text-primary-foreground' : 'hover:bg-card'
          }`}
          title={sec.description}
        >
          {kbPageLabel(sec, t)}
          {!sec.inject && (
            <span className="text-muted-foreground/60 ml-1.5 text-[9px] uppercase">
              {t('web.knowledge.kb.onDemand')}
            </span>
          )}
        </button>
      ))}
    </>
  )
}

// NewPageDialog creates a kb_* knowledge page (a blueprint section
// under the global cwd). Fine-grained pages keep each knowledge domain
// individually indexable instead of growing the four classics forever.
function NewPageDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean
  onOpenChange: (v: boolean) => void
  onCreated: (slug: string) => void
}) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [slug, setSlug] = useState('')
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [nature, setNature] = useState<'foundational' | 'emergent'>('emergent')
  const [inject, setInject] = useState(false)

  const fullSlug = 'kb_' + slug.trim()
  const valid = /^kb_[a-z0-9][a-z0-9_]{0,44}$/.test(fullSlug) && title.trim() !== ''

  const create = useMutation({
    mutationFn: () =>
      putBlueprintSection({
        cwd: GLOBAL_CWD,
        slug: fullSlug,
        title: title.trim(),
        description: description.trim(),
        position: 99,
        maintainer_mode: 'ai',
        prompt_hint: '',
        pinned: false,
        inject,
        nature,
      }),
    onSuccess: (sec) => {
      toast.success(t('web.knowledge.kb.newPage.createdToast'))
      qc.invalidateQueries({ queryKey: ['kb-blueprint'] })
      onOpenChange(false)
      setSlug('')
      setTitle('')
      setDescription('')
      onCreated(sec.slug)
    },
    onError: (e: Error) =>
      toast.error(t('web.knowledge.actionFailed'), { description: e.message }),
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('web.knowledge.kb.newPage.title')}</DialogTitle>
          <DialogDescription>
            {t('web.knowledge.kb.newPage.description')}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-3">
          <div className="flex items-center gap-1">
            <span className="text-muted-foreground font-mono text-sm">kb_</span>
            <Input
              value={slug}
              onChange={(e) => setSlug(e.target.value)}
              placeholder={t('web.knowledge.kb.newPage.slugPlaceholder')}
              className="h-8 font-mono text-sm"
            />
          </div>
          <Input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder={t('web.knowledge.kb.newPage.titlePlaceholder')}
            className="h-8 text-sm"
          />
          <Input
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder={t('web.knowledge.kb.newPage.descPlaceholder')}
            className="h-8 text-sm"
          />
          <div className="flex items-center gap-4 text-sm">
            <Select
              value={nature}
              onValueChange={(v) => setNature(v as 'foundational' | 'emergent')}
            >
              <SelectTrigger className="h-8 w-44 text-sm">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="foundational">
                  {t('web.knowledge.kb.foundational')}
                </SelectItem>
                <SelectItem value="emergent">
                  {t('web.knowledge.kb.emergent')}
                </SelectItem>
              </SelectContent>
            </Select>
            <label className="text-muted-foreground flex items-center gap-1.5 text-xs">
              <input
                type="checkbox"
                checked={inject}
                onChange={(e) => setInject(e.target.checked)}
              />
              {t('web.knowledge.kb.newPage.inject')}
            </label>
          </div>
          <p className="text-muted-foreground text-[11px]">
            {t('web.knowledge.kb.newPage.injectHint')}
          </p>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('web.knowledge.kb.cancel')}
          </Button>
          <Button disabled={!valid || create.isPending} onClick={() => create.mutate()}>
            {create.isPending && <Loader2 className="mr-1 h-3 w-3 animate-spin" />}
            {t('web.knowledge.kb.newPage.create')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function KnowledgeBaseView() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [sel, setSel] = useState<DocKind>('kb_infrastructure')
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState('')
  const [showProposal, setShowProposal] = useState(false)
  const [chatOpen, setChatOpen] = useState(false)
  const [newPageOpen, setNewPageOpen] = useState(false)

  // The knowledge blueprint: the page set is data, not a constant.
  const blueprint = useQuery({
    queryKey: ['kb-blueprint'],
    queryFn: () => listBlueprintSections(GLOBAL_CWD),
  })
  const kbSections = blueprint.data ?? []
  const foundationalSections = kbSections.filter((s) => s.nature === 'foundational')
  const emergentSections = kbSections.filter((s) => s.nature !== 'foundational')
  const selSection = kbSections.find((s) => s.slug === sel)

  const doc = useQuery({
    queryKey: ['kb-doc', GLOBAL_CWD, sel],
    queryFn: () => getProjectDoc(GLOBAL_CWD, sel),
  })

  const removePage = useMutation({
    mutationFn: () => deleteBlueprintSection(GLOBAL_CWD, sel),
    onSuccess: () => {
      toast.success(t('web.knowledge.kb.pageRemovedToast'))
      qc.invalidateQueries({ queryKey: ['kb-blueprint'] })
      setSel('kb_infrastructure')
    },
    onError: (e: Error) =>
      toast.error(t('web.knowledge.actionFailed'), { description: e.message }),
  })
  // B3 — pending AI update proposals for the locked global pages.
  const proposals = useQuery({
    queryKey: ['kb-proposals', GLOBAL_CWD],
    queryFn: () => listPendingProposals(GLOBAL_CWD),
  })
  const pending: DocProposal | undefined = (proposals.data ?? []).find(
    (p) => p.kind === sel,
  )

  const invalidate = () => {
    qc.invalidateQueries({ queryKey: ['kb-doc'] })
    qc.invalidateQueries({ queryKey: ['kb-proposals'] })
  }

  const save = useMutation({
    mutationFn: () => putProjectDoc({ cwd: GLOBAL_CWD, kind: sel, content: draft }),
    onSuccess: () => {
      setEditing(false)
      toast.success(t('web.knowledge.kb.saved'))
      invalidate()
    },
    onError: () => toast.error(t('web.knowledge.actionFailed')),
  })

  const unlock = useMutation({
    mutationFn: () =>
      putProjectDoc({
        cwd: GLOBAL_CWD,
        kind: sel,
        content: stripSig(doc.data?.content ?? ''),
        updatedBy: 'agent',
      }),
    onSuccess: () => {
      toast.success(t('web.knowledge.kb.unlocked'))
      invalidate()
    },
    onError: () => toast.error(t('web.knowledge.actionFailed')),
  })

  const regen = useMutation({
    mutationFn: () => draftKB(),
    onSuccess: () => toast.success(t('web.knowledge.kb.regenerating')),
    onError: () => toast.error(t('web.knowledge.actionFailed')),
  })

  const approve = useMutation({
    mutationFn: () => approveProposal(pending!.id),
    onSuccess: () => {
      setShowProposal(false)
      toast.success(t('web.knowledge.kb.proposal.approved'))
      invalidate()
    },
    onError: () => toast.error(t('web.knowledge.actionFailed')),
  })
  const reject = useMutation({
    mutationFn: () => rejectProposal(pending!.id),
    onSuccess: () => {
      setShowProposal(false)
      toast.success(t('web.knowledge.kb.proposal.rejected'))
      invalidate()
    },
    onError: () => toast.error(t('web.knowledge.actionFailed')),
  })

  const select = (k: DocKind) => {
    setSel(k)
    setEditing(false)
    setShowProposal(false)
    setChatOpen(false)
  }

  const content = stripSig(doc.data?.content ?? '')
  const exists = !!doc.data?.id
  const locked = doc.data?.updated_by === 'operator'
  const foundational = selSection?.nature === 'foundational'

  return (
    <div className="border-border flex min-h-0 flex-1 rounded-b-md rounded-tr-md border">
      {/* nav — two natures, page set from the knowledge blueprint */}
      <div className="border-border flex w-64 shrink-0 flex-col overflow-auto border-r p-2">
        <NavSection
          label={t('web.knowledge.kb.foundational')}
          hint={t('web.knowledge.kb.foundationalHint')}
          sections={foundationalSections}
          sel={sel}
          onSelect={select}
        />
        <NavSection
          label={t('web.knowledge.kb.emergent')}
          hint={t('web.knowledge.kb.emergentHint')}
          sections={emergentSections}
          sel={sel}
          onSelect={select}
        />
        <button
          onClick={() => setNewPageOpen(true)}
          className="text-muted-foreground hover:text-foreground mt-2 flex items-center gap-1 rounded px-2 py-1.5 text-left text-xs"
        >
          <Plus className="h-3 w-3" />
          {t('web.knowledge.kb.newPage.button')}
        </button>
      </div>

      {/* content */}
      <div className="flex min-h-0 flex-1 flex-col">
        <div className="border-border flex items-center gap-2 border-b px-4 py-2">
          <h2 className="text-sm font-medium">
            {selSection ? kbPageLabel(selSection, t) : sel}
          </h2>
          <span
            className={`rounded px-1.5 py-0.5 text-[10px] ${
              foundational
                ? 'bg-amber-500/15 text-amber-400'
                : 'bg-blue-500/15 text-blue-400'
            }`}
          >
            {foundational
              ? t('web.knowledge.kb.bindingBadge')
              : t('web.knowledge.kb.referenceBadge')}
          </span>
          {exists && (
            <span
              className={`rounded px-1.5 py-0.5 text-[10px] ${
                locked
                  ? 'bg-emerald-500/15 text-emerald-400'
                  : 'bg-zinc-500/15 text-zinc-300'
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
                className="border-border rounded-md border px-2.5 py-1 text-xs"
              >
                {t('web.knowledge.kb.edit')}
              </button>
            )}
            {!editing && locked && (
              <button
                onClick={() => unlock.mutate()}
                disabled={unlock.isPending}
                className="border-border rounded-md border px-2.5 py-1 text-xs disabled:opacity-50"
              >
                {t('web.knowledge.kb.unlock')}
              </button>
            )}
            {!editing && (
              <button
                onClick={() => regen.mutate()}
                disabled={regen.isPending}
                className="border-border rounded-md border px-2.5 py-1 text-xs disabled:opacity-50"
              >
                {t('web.knowledge.kb.regenerate')}
              </button>
            )}
            {!editing && (
              <button
                onClick={() => setChatOpen((v) => !v)}
                className={`rounded-md border px-2.5 py-1 text-xs ${
                  chatOpen
                    ? 'border-primary bg-primary/15 text-primary'
                    : 'border-border'
                }`}
                title={t('web.knowledge.kb.discussHint')}
              >
                {t('web.knowledge.kb.discuss')}
              </button>
            )}
            {!editing && selSection && !selSection.pinned && (
              <button
                onClick={() => removePage.mutate()}
                disabled={removePage.isPending}
                className="rounded-md border border-red-500/40 px-2.5 py-1 text-xs text-red-400 disabled:opacity-50"
                title={t('web.knowledge.kb.removePageHint')}
              >
                {t('web.knowledge.kb.removePage')}
              </button>
            )}
          </div>
        </div>

        {/* Governance conversation — discuss + re-draft this page with
            the AI (重新制定方针). Locked pages get proposals, never
            silent overwrites. */}
        {chatOpen && !editing && (
          <div className="border-b p-3">
            <CurationChat
              targetKind="kb_page"
              targetCwd={GLOBAL_CWD}
              targetSlug={sel}
              onRevision={invalidate}
            />
          </div>
        )}

        {/* B3 — AI proposed an update to this (locked) page */}
        {pending && !editing && (
          <div className="border-b border-amber-500/30 bg-amber-500/10 px-4 py-2 text-xs">
            <div className="flex items-center gap-2">
              <span className="flex-1 text-amber-300">
                {t('web.knowledge.kb.proposal.text')}
              </span>
              <button
                onClick={() => setShowProposal((v) => !v)}
                className="border-border rounded-md border px-2 py-0.5"
              >
                {showProposal
                  ? t('web.knowledge.kb.proposal.hide')
                  : t('web.knowledge.kb.proposal.preview')}
              </button>
              <button
                onClick={() => approve.mutate()}
                disabled={approve.isPending}
                className="rounded-md bg-emerald-600/80 px-2 py-0.5 text-white disabled:opacity-50"
              >
                {t('web.knowledge.kb.proposal.approve')}
              </button>
              <button
                onClick={() => reject.mutate()}
                disabled={reject.isPending}
                className="rounded-md border border-red-500/40 px-2 py-0.5 text-red-400 disabled:opacity-50"
              >
                {t('web.knowledge.kb.proposal.reject')}
              </button>
            </div>
            {showProposal && (
              <div className="bg-card mt-2 max-h-72 overflow-auto rounded-md p-3">
                <ReactMarkdown remarkPlugins={[remarkGfm]} components={MD}>
                  {stripSig(pending.proposed_content)}
                </ReactMarkdown>
              </div>
            )}
          </div>
        )}

        <div className="flex-1 overflow-auto p-4">
          {editing ? (
            <div className="flex h-full flex-col gap-2">
              <textarea
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                className="border-border bg-card flex-1 resize-none rounded-md p-3 font-mono text-sm"
              />
              <div className="flex gap-2">
                <button
                  onClick={() => save.mutate()}
                  disabled={save.isPending}
                  className="bg-primary text-primary-foreground rounded-md px-3 py-1.5 text-sm disabled:opacity-50"
                >
                  {t('web.knowledge.kb.save')}
                </button>
                <button
                  onClick={() => setEditing(false)}
                  className="border-border rounded-md border px-3 py-1.5 text-sm"
                >
                  {t('web.knowledge.kb.cancel')}
                </button>
                <span className="text-muted-foreground self-center text-[11px]">
                  {t('web.knowledge.kb.editHint')}
                </span>
              </div>
            </div>
          ) : doc.isLoading ? (
            <p className="text-muted-foreground text-sm">…</p>
          ) : content ? (
            <ReactMarkdown remarkPlugins={[remarkGfm]} components={MD}>
              {content}
            </ReactMarkdown>
          ) : (
            <p className="text-muted-foreground text-sm">
              {t('web.knowledge.kb.empty')}
            </p>
          )}
        </div>
      </div>

      <NewPageDialog
        open={newPageOpen}
        onOpenChange={setNewPageOpen}
        onCreated={(slug) => select(slug)}
      />
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

function ImpactView() {
  const { t } = useTranslation()
  const [selId, setSelId] = useState<string | null>(null)
  const [filter, setFilter] = useState('')

  const entitiesQuery = useQuery({
    queryKey: ['knowledge-impact'],
    queryFn: () => listImpactEntities(),
  })
  const detailQuery = useQuery({
    queryKey: ['knowledge-impact-detail', selId],
    queryFn: () => getKnowledgeGraph(selId!),
    enabled: !!selId,
  })

  const entities = (entitiesQuery.data ?? []).filter(
    (e) =>
      !filter.trim() ||
      e.node.title.toLowerCase().includes(filter.trim().toLowerCase()),
  )
  const neighbors = detailQuery.data?.neighbors ?? []
  const grouped: Record<string, typeof neighbors> = {}
  for (const n of neighbors) {
    const k = n.node.kind
    grouped[k] = grouped[k] ? [...grouped[k], n] : [n]
  }
  const groupOrder = ['entity', 'playbook', 'skill', 'fact']

  return (
    <div className="border-border flex min-h-0 flex-1 rounded-b-md rounded-tr-md border">
      {/* entity list — sorted by blast radius */}
      <div className="border-border flex w-72 shrink-0 flex-col border-r">
        <div className="border-border border-b p-2">
          <p className="text-muted-foreground mb-1.5 px-1 text-[11px] leading-snug">
            {t('web.knowledge.impact.intro')}
          </p>
          <Input
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            placeholder={t('web.knowledge.impact.filter')}
            className="h-7 text-xs"
          />
        </div>
        <div className="min-h-0 flex-1 overflow-auto p-1.5">
          {entitiesQuery.isLoading ? (
            <Loader2 className="m-4 h-4 w-4 animate-spin" />
          ) : entities.length === 0 ? (
            <p className="text-muted-foreground p-4 text-center text-xs">
              {t('web.knowledge.impact.empty')}
            </p>
          ) : (
            entities.map((e) => (
              <button
                key={e.node.id}
                onClick={() => setSelId(e.node.id)}
                className={`flex w-full items-center gap-2 rounded px-2 py-1.5 text-left text-sm ${
                  selId === e.node.id
                    ? 'bg-primary text-primary-foreground'
                    : 'hover:bg-card'
                }`}
              >
                <span className="min-w-0 flex-1 truncate">{e.node.title}</span>
                {e.node.entity_type && (
                  <span className="text-[9px] uppercase opacity-60">
                    {e.node.entity_type}
                  </span>
                )}
                <span
                  className={`rounded px-1.5 py-0.5 text-[10px] ${
                    selId === e.node.id ? 'bg-black/20' : 'bg-muted'
                  }`}
                  title={t('web.knowledge.impact.degreeHint')}
                >
                  {e.degree}
                </span>
              </button>
            ))
          )}
        </div>
      </div>

      {/* blast radius of the selected entity */}
      <div className="min-h-0 flex-1 overflow-auto p-4">
        {!selId ? (
          <p className="text-muted-foreground py-10 text-center text-sm">
            {t('web.knowledge.impact.pickOne')}
          </p>
        ) : detailQuery.isLoading ? (
          <Loader2 className="h-4 w-4 animate-spin" />
        ) : (
          <>
            <h2 className="mb-1 text-base font-semibold">
              {detailQuery.data?.node.title}
            </h2>
            <p className="text-muted-foreground mb-4 text-xs">
              {t('web.knowledge.impact.blastRadius', {
                count: neighbors.length,
              })}
            </p>
            {neighbors.length === 0 && (
              <p className="text-muted-foreground text-sm">
                {t('web.knowledge.impact.noLinks')}
              </p>
            )}
            {groupOrder
              .filter((k) => grouped[k]?.length)
              .map((k) => (
                <section key={k} className="mb-4">
                  <h3 className="mb-1.5 flex items-center gap-2 text-sm font-semibold">
                    <KindBadge kind={k} />
                    <span className="text-muted-foreground text-[11px] font-normal">
                      {grouped[k].length}
                    </span>
                  </h3>
                  <div className="space-y-1">
                    {grouped[k].map((n) => (
                      <div
                        key={n.node.id}
                        className="bg-card flex items-center gap-2 rounded-md border px-2.5 py-1.5"
                      >
                        <span className="min-w-0 flex-1 truncate text-sm">
                          {n.node.kind === 'entity' &&
                          n.node.entity_type === 'project'
                            ? n.node.scope_key || n.node.title
                            : n.node.title}
                        </span>
                        <span className="text-muted-foreground text-[10px]">
                          {n.edge_type}
                        </span>
                      </div>
                    ))}
                  </div>
                </section>
              ))}
          </>
        )}
      </div>
    </div>
  )
}

export function KnowledgePage() {
  const { t } = useTranslation()
  const [tab, setTab] = useState<'kb' | 'distill' | 'impact'>('kb')

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
          <TabBtn active={tab === 'distill'} onClick={() => setTab('distill')}>
            {t('web.knowledge.distill.tab')}
          </TabBtn>
          <TabBtn active={tab === 'impact'} onClick={() => setTab('impact')}>
            {t('web.knowledge.impact.tab')}
          </TabBtn>
        </div>
      </header>
      <div className="flex flex-1 flex-col min-h-0 px-4 pb-4">
        {tab === 'kb' ? (
          <KnowledgeBaseView />
        ) : tab === 'distill' ? (
          <DistillationView />
        ) : (
          <ImpactView />
        )}
      </div>
    </div>
  )
}

// ── Distillation workbench — repeated experience → tested skills ──
//
// The experience compiler mines session episodes ACROSS projects and only
// drafts a candidate when the same procedure SUCCEEDED in ≥2 sessions
// (repetition + success evidence — never a single session). Candidates are
// ranked by recurrence × the procedure's manual time cost, so what saves
// the most operator time distills first. Where the procedure is fully
// mechanical the candidate carries an executable run.sh (with a validation
// step) — promotion ships it next to SKILL.md and registers a custom task.
// The outcome loop then watches every promoted skill: sessions that load
// it report success/failure, and the retirement list proposes dropping
// what the evidence says isn't working.

// Candidate provenance written by the experience compiler.
function compilerStats(n: KnowledgeNode) {
  const p = n.provenance ?? {}
  return {
    recurrence: typeof p.recurrence === 'number' ? p.recurrence : 0,
    estMinutes: typeof p.est_minutes === 'number' ? p.est_minutes : 0,
    projects: Array.isArray(p.projects) ? p.projects.length : 0,
    compiled: typeof p.script === 'string' && p.script !== '',
  }
}

function RetireReasonBadge({ reason }: { reason: RetirementReason }) {
  const { t } = useTranslation()
  const styles: Record<RetirementReason, string> = {
    never_used: 'bg-zinc-500/20 text-zinc-300',
    low_success: 'bg-red-500/15 text-red-400',
    dormant: 'bg-amber-500/15 text-amber-400',
  }
  return (
    <span
      className={`rounded px-1.5 py-0.5 text-[9px] ${styles[reason]}`}
      title={t(`web.knowledge.distill.retirement.${reason}Hint`)}
    >
      {t(`web.knowledge.distill.retirement.${reason}`)}
    </span>
  )
}

function DistillationView() {
  const { t } = useTranslation()
  const qc = useQueryClient()

  const playbooksQuery = useQuery({
    queryKey: ['knowledge-nodes', 'playbook'],
    queryFn: () => listKnowledgeNodes({ kind: 'playbook' }),
  })
  const skillsQuery = useQuery({
    queryKey: ['knowledge-nodes', 'skill'],
    queryFn: () => listKnowledgeNodes({ kind: 'skill' }),
  })
  const retirementQuery = useQuery({
    queryKey: ['knowledge-retirement'],
    queryFn: () => listRetirementCandidates(),
  })
  const refresh = () => {
    qc.invalidateQueries({ queryKey: ['knowledge-nodes', 'playbook'] })
    qc.invalidateQueries({ queryKey: ['knowledge-nodes', 'skill'] })
    qc.invalidateQueries({ queryKey: ['knowledge-retirement'] })
  }

  const skillify = useMutation({
    mutationFn: (id: string) => skillifyKnowledgeNode(id),
    onSuccess: () => {
      toast.success(t('web.knowledge.distill.skillifiedToast'))
      refresh()
    },
    onError: () => toast.error(t('web.knowledge.actionFailed')),
  })
  const toggle = useMutation({
    mutationFn: (input: { id: string; enabled: boolean }) =>
      setKnowledgeNodeEnabled(input.id, input.enabled),
    onSuccess: (n) => {
      toast.success(
        n.enabled
          ? t('web.knowledge.distill.enabledToast')
          : t('web.knowledge.distill.disabledToast'),
      )
      refresh()
    },
    onError: () => toast.error(t('web.knowledge.actionFailed')),
  })
  const remove = useMutation({
    mutationFn: (id: string) => deleteKnowledgeNode(id),
    onSuccess: () => {
      toast.success(t('web.knowledge.distill.removedToast'))
      refresh()
    },
    onError: () => toast.error(t('web.knowledge.actionFailed')),
  })

  // Rank candidates by what saves the most operator time: recurrence ×
  // manual time cost (the compiler's score). Ties / legacy rows fall back
  // to recency via the API's updated_at ordering.
  const playbooks = [...(playbooksQuery.data ?? [])].sort(
    (a, b) => candidateScore(b) - candidateScore(a),
  )
  const skills = skillsQuery.data ?? []
  const retireReasons = new Map(
    (retirementQuery.data ?? []).map((c) => [c.node.id, c.reason]),
  )

  return (
    <div className="border-border min-h-0 flex-1 overflow-auto rounded-b-md rounded-tr-md border p-4">
      <p className="text-muted-foreground mb-4 max-w-3xl text-xs leading-relaxed">
        {t('web.knowledge.distill.intro')}
      </p>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {/* Playbooks — distilled, awaiting promotion */}
        <section>
          <h2 className="mb-2 flex items-center gap-2 text-sm font-semibold">
            {t('web.knowledge.distill.playbooks')}
            <span className="text-muted-foreground text-[11px] font-normal">
              {playbooks.length}
            </span>
          </h2>
          <p className="text-muted-foreground/80 mb-2 text-[11px]">
            {t('web.knowledge.distill.playbooksHint')}
          </p>
          {playbooksQuery.isLoading ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : playbooks.length === 0 ? (
            <p className="text-muted-foreground py-6 text-center text-xs">
              {t('web.knowledge.distill.playbooksEmpty')}
            </p>
          ) : (
            <div className="space-y-2">
              {playbooks.map((n) => {
                const stats = compilerStats(n)
                return (
                  <div key={n.id} className="bg-card rounded-md border p-3">
                    <div className="mb-1.5 flex items-start justify-between gap-2">
                      <span className="text-sm font-medium">{n.title}</span>
                      <span className="flex flex-none gap-1">
                        {stats.compiled && (
                          <span
                            className="rounded bg-emerald-500/15 px-1.5 py-0.5 text-[9px] text-emerald-400"
                            title={t('web.knowledge.distill.compiledHint')}
                          >
                            {t('web.knowledge.distill.compiledBadge')}
                          </span>
                        )}
                        {n.scope === 'global' && (
                          <span className="rounded bg-blue-500/15 px-1.5 py-0.5 text-[9px] text-blue-400">
                            global
                          </span>
                        )}
                      </span>
                    </div>
                    {stats.recurrence > 0 && (
                      <p
                        className="text-muted-foreground mb-1 text-[11px]"
                        title={t('web.knowledge.distill.scoreHint')}
                      >
                        {t('web.knowledge.distill.recurrence', {
                          count: stats.recurrence,
                        })}
                        {' · '}
                        {t('web.knowledge.distill.timeCost', {
                          minutes: stats.estMinutes,
                        })}
                        {stats.projects > 1 &&
                          ` · ${t('web.knowledge.distill.projectSpan', {
                            count: stats.projects,
                          })}`}
                      </p>
                    )}
                    {typeof n.provenance?.summary === 'string' && (
                      <p className="text-muted-foreground mb-2 line-clamp-3 text-xs">
                        {String(n.provenance.summary)}
                      </p>
                    )}
                    <div className="flex gap-2">
                      <button
                        onClick={() => skillify.mutate(n.id)}
                        disabled={skillify.isPending}
                        className="bg-primary text-primary-foreground rounded-md px-2.5 py-1 text-xs disabled:opacity-50"
                        title={t('web.knowledge.distill.skillifyHint')}
                      >
                        {t('web.knowledge.distill.skillify')}
                      </button>
                      <button
                        onClick={() => remove.mutate(n.id)}
                        disabled={remove.isPending}
                        className="rounded-md border border-red-500/40 px-2.5 py-1 text-xs text-red-400 disabled:opacity-50"
                      >
                        {t('web.knowledge.distill.discard')}
                      </button>
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </section>

        {/* Skills — promoted, injected into every spawn */}
        <section>
          <h2 className="mb-2 flex items-center gap-2 text-sm font-semibold">
            {t('web.knowledge.distill.skills')}
            <span className="text-muted-foreground text-[11px] font-normal">
              {skills.length}
            </span>
          </h2>
          <p className="text-muted-foreground/80 mb-2 text-[11px]">
            {t('web.knowledge.distill.skillsHint')}
          </p>
          {skillsQuery.isLoading ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : skills.length === 0 ? (
            <p className="text-muted-foreground py-6 text-center text-xs">
              {t('web.knowledge.distill.skillsEmpty')}
            </p>
          ) : (
            <div className="space-y-2">
              {skills.map((n) => {
                const enabled = n.enabled !== false
                const retireReason = retireReasons.get(n.id)
                const outcomes = (n.success_count ?? 0) + (n.failure_count ?? 0)
                const compiled = compilerStats(n).compiled
                return (
                  <div
                    key={n.id}
                    className={`bg-card rounded-md border p-3 ${enabled ? '' : 'opacity-60'}`}
                  >
                    <div className="mb-1 flex items-start justify-between gap-2">
                      <span className="text-sm font-medium">{n.title}</span>
                      <span className="flex flex-none items-center gap-1.5">
                        {retireReason && <RetireReasonBadge reason={retireReason} />}
                        {compiled && (
                          <span
                            className="rounded bg-emerald-500/15 px-1.5 py-0.5 text-[9px] text-emerald-400"
                            title={t('web.knowledge.distill.compiledHint')}
                          >
                            {t('web.knowledge.distill.compiledBadge')}
                          </span>
                        )}
                        {enabled ? (
                          <span className="rounded bg-emerald-500/15 px-1.5 py-0.5 text-[9px] text-emerald-400">
                            {t('web.knowledge.distill.injectedBadge')}
                          </span>
                        ) : (
                          <span className="rounded bg-zinc-500/15 px-1.5 py-0.5 text-[9px] text-zinc-400">
                            {t('web.knowledge.distill.disabledBadge')}
                          </span>
                        )}
                        <Switch
                          checked={enabled}
                          disabled={toggle.isPending}
                          onCheckedChange={(v) =>
                            toggle.mutate({ id: n.id, enabled: v })
                          }
                          title={t('web.knowledge.distill.toggleHint')}
                        />
                      </span>
                    </div>
                    <p className="text-muted-foreground text-[11px]">
                      {t('web.knowledge.distill.usage', {
                        count: n.use_count ?? 0,
                      })}
                      {outcomes > 0 &&
                        ` · ${t('web.knowledge.distill.outcomes', {
                          ok: n.success_count ?? 0,
                          failed: n.failure_count ?? 0,
                        })}`}
                      {n.last_used_at &&
                        ` · ${t('web.knowledge.distill.lastUsed', {
                          date: new Date(n.last_used_at).toLocaleDateString(),
                        })}`}
                    </p>
                    <button
                      onClick={() => remove.mutate(n.id)}
                      disabled={remove.isPending}
                      className="text-muted-foreground hover:text-destructive mt-1 text-[11px]"
                    >
                      {t('web.knowledge.distill.retire')}
                    </button>
                  </div>
                )
              })}
            </div>
          )}
        </section>
      </div>
    </div>
  )
}
