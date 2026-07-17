import { useEffect, useRef, useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { launchKBLibrarian } from '@/lib/cortex'
import { listAgentModels, type AgentProviderID } from '@/lib/memoryWorkers'
import { listClaudeAccounts } from '@/lib/claudeAccounts'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

import {
  listKnowledgeNodes,
  getKnowledgeGraph,
  getKnowledgeGraphAll,
  listRetirementCandidates,
  candidateScore,
  skillifyKnowledgeNode,
  setKnowledgeNodeEnabled,
  deleteKnowledgeNode,
  draftKB,
  type KnowledgeNode,
  type KnowledgeEdge,
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
import { Loader2, Plus, Sparkles } from 'lucide-react'
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
  const [librarianOpen, setLibrarianOpen] = useState(false)

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
        {/* Cross-page KB admin — launch the Librarian agent session that can
            organize/create/edit any KB page (vs the per-page Discuss chat). */}
        <button
          onClick={() => setLibrarianOpen(true)}
          className="text-muted-foreground hover:text-foreground flex items-center gap-1 rounded px-2 py-1.5 text-left text-xs"
          title={t('web.knowledge.kb.librarian.hint')}
        >
          <Sparkles className="h-3 w-3" />
          {t('web.knowledge.kb.librarian.button')}
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

      {librarianOpen && (
        <LaunchLibrarianDialog onClose={() => setLibrarianOpen(false)} />
      )}
    </div>
  )
}

// LaunchLibrarianDialog picks the cloud agent + model (+ Claude account) that
// backs the KB Librarian session, then spawns it and navigates to the session.
// Mounted only while open so its state resets each time.
function LaunchLibrarianDialog({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  // Provider is a plain string (not AgentProviderID) so the Librarian can
  // offer every worker-backed CLI — grok/opencode included — independent of
  // the discuss-chat's narrower union.
  const [provider, setProvider] = useState('claude')
  const [model, setModel] = useState('')
  const [account, setAccount] = useState('')

  const modelsQuery = useQuery({
    queryKey: ['agent-models', provider],
    queryFn: () => listAgentModels(provider as AgentProviderID),
    staleTime: 60 * 60 * 1000,
  })
  const accountsQuery = useQuery({
    queryKey: ['claude-accounts'],
    queryFn: listClaudeAccounts,
    enabled: provider === 'claude',
  })
  const accounts = (accountsQuery.data ?? []).filter((a) => a.enabled)

  const launch = useMutation({
    mutationFn: () =>
      launchKBLibrarian({
        provider,
        model: model.trim() || undefined,
        claude_account_id:
          provider === 'claude' ? account.trim() || undefined : undefined,
      }),
    onSuccess: ({ session_id }) => {
      toast.success(t('web.knowledge.kb.librarian.launchedToast'))
      onClose()
      navigate({ to: '/sessions', search: { open: session_id } })
    },
    onError: (e: Error) =>
      toast.error(t('web.knowledge.actionFailed'), { description: e.message }),
  })

  // Every worker-backed CLI (matches the backend session providers + the
  // memory MCP's KB tools attach to any of them). grok/opencode use a single
  // host login, so no per-agent account picker — only claude is multi-account.
  const LIB_PROVIDERS: { id: string; label: string }[] = [
    { id: 'claude', label: 'Claude' },
    { id: 'codex', label: 'Codex' },
    { id: 'antigravity', label: 'Antigravity' },
    { id: 'grok', label: 'Grok' },
    { id: 'opencode', label: 'OpenCode' },
  ]
  const MODEL_DEFAULT = '__default__'
  const ACCOUNT_DEFAULT = '__default__'

  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('web.knowledge.kb.librarian.title')}</DialogTitle>
          <DialogDescription>
            {t('web.knowledge.kb.librarian.dialogHint')}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-3">
          <div className="flex flex-col gap-1.5">
            <span className="text-xs text-muted-foreground">
              {t('web.knowledge.kb.librarian.provider')}
            </span>
            <Select
              value={provider}
              onValueChange={(v) => {
                setProvider(v)
                setModel('')
                setAccount('')
              }}
            >
              <SelectTrigger className="h-8 text-sm">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {LIB_PROVIDERS.map((p) => (
                  <SelectItem key={p.id} value={p.id}>
                    {p.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="flex flex-col gap-1.5">
            <span className="text-xs text-muted-foreground">
              {t('web.cortex.chat.modelLabel')}
            </span>
            <Select
              value={model === '' ? MODEL_DEFAULT : model}
              onValueChange={(v) => setModel(v === MODEL_DEFAULT ? '' : v)}
            >
              <SelectTrigger className="h-8 text-sm">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={MODEL_DEFAULT}>
                  {t('web.cortex.chat.modelCliDefault')}
                </SelectItem>
                {(modelsQuery.data ?? []).map((m) => (
                  <SelectItem key={m.id} value={m.id}>
                    {m.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {provider === 'claude' && accounts.length > 0 && (
            <div className="flex flex-col gap-1.5">
              <span className="text-xs text-muted-foreground">
                {t('web.knowledge.kb.librarian.account')}
              </span>
              <Select
                value={account === '' ? ACCOUNT_DEFAULT : account}
                onValueChange={(v) =>
                  setAccount(v === ACCOUNT_DEFAULT ? '' : v)
                }
              >
                <SelectTrigger className="h-8 text-sm">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={ACCOUNT_DEFAULT}>
                    {t('web.cortex.chat.accountDefault')}
                  </SelectItem>
                  {accounts.map((a) => (
                    <SelectItem key={a.id} value={a.id}>
                      {a.display_name || a.name || a.id}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {t('web.knowledge.kb.cancel')}
          </Button>
          <Button disabled={launch.isPending} onClick={() => launch.mutate()}>
            {launch.isPending && <Loader2 className="mr-1 h-3 w-3 animate-spin" />}
            {t('web.knowledge.kb.librarian.launch')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ── Graph view (Obsidian-style force-directed network) ────────

const KIND_STYLES: Record<string, string> = {
  entity: 'bg-blue-500/15 text-blue-400',
  project: 'bg-violet-500/15 text-violet-400',
  fact: 'bg-zinc-500/15 text-zinc-300',
  playbook: 'bg-amber-500/15 text-amber-400',
  skill: 'bg-emerald-500/15 text-emerald-400',
}

// Canvas can't consume tailwind classes — same palette as KIND_STYLES,
// resolved to the -400 hex values.
const KIND_COLORS: Record<string, string> = {
  entity: '#60a5fa',
  project: '#a78bfa',
  fact: '#d4d4d8',
  playbook: '#fbbf24',
  skill: '#34d399',
}

// Projects are entities, but they're the anchors everything hangs off —
// give them their own color so the map reads at a glance.
function nodeColorKey(n: KnowledgeNode): string {
  return n.kind === 'entity' && n.entity_type === 'project' ? 'project' : n.kind
}

function nodeDisplayTitle(n: KnowledgeNode): string {
  if (n.kind === 'entity' && n.entity_type === 'project' && n.scope_key) {
    const parts = n.scope_key.split('/')
    return parts[parts.length - 1] || n.title
  }
  return n.title
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

interface SimNode {
  id: string
  x: number
  y: number
  vx: number
  vy: number
  r: number
  color: string
  label: string
  degree: number
  dragged: boolean
}

interface SimEdge {
  a: number // index into nodes
  b: number
}

// Hand-rolled force simulation — node repulsion + edge springs + center
// gravity, cooled by an alpha that decays to rest and re-heats on
// interaction. ~500 nodes is fine for the O(n²) repulsion pass because
// the sim stops ticking once cooled.
const SIM = {
  repulsion: 1600,
  spring: 0.05,
  springLength: 70,
  gravity: 0.03,
  damping: 0.82,
  alphaDecay: 0.985,
  alphaMin: 0.02,
}

function GraphCanvas({
  nodes,
  edges,
  selectedId,
  onSelect,
}: {
  nodes: KnowledgeNode[]
  edges: KnowledgeEdge[]
  selectedId: string | null
  onSelect: (id: string | null) => void
}) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null)
  const onSelectRef = useRef(onSelect)
  onSelectRef.current = onSelect
  const selectedRef = useRef(selectedId)

  // The whole sim lives in refs — React renders the canvas exactly once
  // and rAF drives everything else.
  const simRef = useRef<{
    nodes: SimNode[]
    edges: SimEdge[]
    byId: Map<string, number>
    alpha: number
    view: { x: number; y: number; k: number }
    hovered: number
  }>({
    nodes: [],
    edges: [],
    byId: new Map(),
    alpha: 0,
    view: { x: 0, y: 0, k: 1 },
    hovered: -1,
  })

  // (Re)build sim state when the data changes; keep positions of nodes
  // that survived so a refetch doesn't scramble the layout.
  useEffect(() => {
    const sim = simRef.current
    const prev = new Map(sim.nodes.map((n) => [n.id, n]))
    const degree = new Map<string, number>()
    for (const e of edges) {
      degree.set(e.src_id, (degree.get(e.src_id) ?? 0) + 1)
      degree.set(e.dst_id, (degree.get(e.dst_id) ?? 0) + 1)
    }
    sim.nodes = nodes.map((n, i) => {
      const d = degree.get(n.id) ?? 0
      // Phyllotaxis spiral for fresh nodes — deterministic, evenly spread.
      const angle = i * 2.39996
      const radius = 22 * Math.sqrt(i + 1)
      const old = prev.get(n.id)
      return {
        id: n.id,
        x: old?.x ?? Math.cos(angle) * radius,
        y: old?.y ?? Math.sin(angle) * radius,
        vx: 0,
        vy: 0,
        r: 4 + Math.min(11, Math.sqrt(d) * 2),
        color: KIND_COLORS[nodeColorKey(n)] ?? KIND_COLORS.fact,
        label: nodeDisplayTitle(n),
        degree: d,
        dragged: false,
      }
    })
    sim.byId = new Map(sim.nodes.map((n, i) => [n.id, i]))
    sim.edges = edges
      .map((e) => ({
        a: sim.byId.get(e.src_id) ?? -1,
        b: sim.byId.get(e.dst_id) ?? -1,
      }))
      .filter((e) => e.a >= 0 && e.b >= 0)
    sim.alpha = 1
  }, [nodes, edges])

  useEffect(() => {
    selectedRef.current = selectedId
    simRef.current.alpha = Math.max(simRef.current.alpha, SIM.alphaMin)
  }, [selectedId])

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    const sim = simRef.current

    let raf = 0
    let width = 0
    let height = 0

    const resize = () => {
      const dpr = window.devicePixelRatio || 1
      width = canvas.clientWidth
      height = canvas.clientHeight
      canvas.width = Math.max(1, Math.round(width * dpr))
      canvas.height = Math.max(1, Math.round(height * dpr))
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
    }
    const ro = new ResizeObserver(resize)
    ro.observe(canvas)
    resize()

    const toWorld = (sx: number, sy: number) => ({
      x: (sx - width / 2 - sim.view.x) / sim.view.k,
      y: (sy - height / 2 - sim.view.y) / sim.view.k,
    })
    const hitTest = (sx: number, sy: number): number => {
      const p = toWorld(sx, sy)
      for (let i = sim.nodes.length - 1; i >= 0; i--) {
        const n = sim.nodes[i]
        const dx = n.x - p.x
        const dy = n.y - p.y
        const hr = n.r + 3 / sim.view.k
        if (dx * dx + dy * dy <= hr * hr) return i
      }
      return -1
    }

    const step = () => {
      const ns = sim.nodes
      if (sim.alpha > SIM.alphaMin && ns.length > 0) {
        sim.alpha *= SIM.alphaDecay
        const a = sim.alpha
        // pairwise repulsion
        for (let i = 0; i < ns.length; i++) {
          const ni = ns[i]
          for (let j = i + 1; j < ns.length; j++) {
            const nj = ns[j]
            let dx = ni.x - nj.x
            let dy = nj.y === ni.y && dx === 0 ? 0.1 : ni.y - nj.y
            const d2 = Math.max(dx * dx + dy * dy, 64)
            const f = (SIM.repulsion * a) / d2
            const d = Math.sqrt(d2)
            dx /= d
            dy /= d
            ni.vx += dx * f
            ni.vy += dy * f
            nj.vx -= dx * f
            nj.vy -= dy * f
          }
        }
        // edge springs
        for (const e of sim.edges) {
          const na = ns[e.a]
          const nb = ns[e.b]
          const dx = nb.x - na.x
          const dy = nb.y - na.y
          const d = Math.max(Math.sqrt(dx * dx + dy * dy), 1)
          const f = SIM.spring * a * (d - SIM.springLength)
          const fx = (dx / d) * f
          const fy = (dy / d) * f
          na.vx += fx
          na.vy += fy
          nb.vx -= fx
          nb.vy -= fy
        }
        // center gravity + integration
        for (const n of ns) {
          n.vx -= n.x * SIM.gravity * a
          n.vy -= n.y * SIM.gravity * a
          if (!n.dragged) {
            n.vx *= SIM.damping
            n.vy *= SIM.damping
            n.x += n.vx
            n.y += n.vy
          } else {
            n.vx = 0
            n.vy = 0
          }
        }
      }
      draw()
      raf = requestAnimationFrame(step)
    }

    const draw = () => {
      ctx.clearRect(0, 0, width, height)
      ctx.save()
      ctx.translate(width / 2 + sim.view.x, height / 2 + sim.view.y)
      ctx.scale(sim.view.k, sim.view.k)

      const fg = getComputedStyle(canvas).color || '#a1a1aa'
      const selIdx = selectedRef.current
        ? (sim.byId.get(selectedRef.current) ?? -1)
        : -1

      // edges first
      ctx.lineWidth = 1 / sim.view.k
      ctx.strokeStyle = fg
      ctx.globalAlpha = 0.16
      ctx.beginPath()
      for (const e of sim.edges) {
        const a = sim.nodes[e.a]
        const b = sim.nodes[e.b]
        ctx.moveTo(a.x, a.y)
        ctx.lineTo(b.x, b.y)
      }
      ctx.stroke()
      ctx.globalAlpha = 1

      // nodes
      for (let i = 0; i < sim.nodes.length; i++) {
        const n = sim.nodes[i]
        ctx.beginPath()
        ctx.arc(n.x, n.y, n.r, 0, Math.PI * 2)
        ctx.fillStyle = n.color
        ctx.fill()
        if (i === selIdx || i === sim.hovered) {
          ctx.lineWidth = 2 / sim.view.k
          ctx.strokeStyle = fg
          ctx.stroke()
        }
      }

      // labels appear as you zoom in; hubs + the active node always show
      const fontPx = Math.max(10 / sim.view.k, 4)
      ctx.font = `${fontPx}px ui-sans-serif, system-ui, sans-serif`
      ctx.textAlign = 'center'
      ctx.textBaseline = 'top'
      ctx.fillStyle = fg
      for (let i = 0; i < sim.nodes.length; i++) {
        const n = sim.nodes[i]
        const show =
          i === selIdx || i === sim.hovered || sim.view.k >= 1.2 || n.degree >= 5
        if (!show) continue
        ctx.globalAlpha = i === selIdx || i === sim.hovered ? 1 : 0.75
        ctx.fillText(n.label.slice(0, 42), n.x, n.y + n.r + 3 / sim.view.k)
      }
      ctx.globalAlpha = 1
      ctx.restore()
    }

    // ── interaction: drag nodes, pan background, wheel zoom ──
    let pointerDown = false
    let dragIdx = -1
    let moved = false
    let lastX = 0
    let lastY = 0

    const onPointerDown = (ev: PointerEvent) => {
      const rect = canvas.getBoundingClientRect()
      const sx = ev.clientX - rect.left
      const sy = ev.clientY - rect.top
      pointerDown = true
      moved = false
      lastX = sx
      lastY = sy
      dragIdx = hitTest(sx, sy)
      if (dragIdx >= 0) sim.nodes[dragIdx].dragged = true
      canvas.setPointerCapture(ev.pointerId)
    }
    const onPointerMove = (ev: PointerEvent) => {
      const rect = canvas.getBoundingClientRect()
      const sx = ev.clientX - rect.left
      const sy = ev.clientY - rect.top
      if (!pointerDown) {
        const h = hitTest(sx, sy)
        if (h !== sim.hovered) {
          sim.hovered = h
          canvas.style.cursor = h >= 0 ? 'pointer' : 'grab'
        }
        return
      }
      if (Math.abs(sx - lastX) + Math.abs(sy - lastY) > 3) moved = true
      if (dragIdx >= 0) {
        const p = toWorld(sx, sy)
        const n = sim.nodes[dragIdx]
        n.x = p.x
        n.y = p.y
        sim.alpha = Math.max(sim.alpha, 0.3)
      } else if (moved) {
        sim.view.x += sx - lastX
        sim.view.y += sy - lastY
        lastX = sx
        lastY = sy
      }
      if (dragIdx >= 0) {
        lastX = sx
        lastY = sy
      }
    }
    const onPointerUp = (ev: PointerEvent) => {
      if (!pointerDown) return
      pointerDown = false
      if (dragIdx >= 0) sim.nodes[dragIdx].dragged = false
      if (!moved) {
        const rect = canvas.getBoundingClientRect()
        const i = hitTest(ev.clientX - rect.left, ev.clientY - rect.top)
        onSelectRef.current(i >= 0 ? sim.nodes[i].id : null)
      }
      dragIdx = -1
    }
    const onWheel = (ev: WheelEvent) => {
      ev.preventDefault()
      const rect = canvas.getBoundingClientRect()
      const sx = ev.clientX - rect.left
      const sy = ev.clientY - rect.top
      const before = toWorld(sx, sy)
      sim.view.k = Math.min(
        5,
        Math.max(0.15, sim.view.k * Math.exp(-ev.deltaY * 0.0015)),
      )
      // keep the point under the cursor stationary while zooming
      sim.view.x = sx - width / 2 - before.x * sim.view.k
      sim.view.y = sy - height / 2 - before.y * sim.view.k
    }

    canvas.addEventListener('pointerdown', onPointerDown)
    canvas.addEventListener('pointermove', onPointerMove)
    canvas.addEventListener('pointerup', onPointerUp)
    canvas.addEventListener('wheel', onWheel, { passive: false })
    raf = requestAnimationFrame(step)

    return () => {
      cancelAnimationFrame(raf)
      ro.disconnect()
      canvas.removeEventListener('pointerdown', onPointerDown)
      canvas.removeEventListener('pointermove', onPointerMove)
      canvas.removeEventListener('pointerup', onPointerUp)
      canvas.removeEventListener('wheel', onWheel)
    }
  }, [])

  return (
    <canvas
      ref={canvasRef}
      className="text-muted-foreground block h-full w-full"
    />
  )
}

function GraphView() {
  const { t } = useTranslation()
  const [selId, setSelId] = useState<string | null>(null)

  const graphQuery = useQuery({
    queryKey: ['knowledge-graph-all'],
    queryFn: () => getKnowledgeGraphAll(),
  })
  const detailQuery = useQuery({
    queryKey: ['knowledge-graph-detail', selId],
    queryFn: () => getKnowledgeGraph(selId!),
    enabled: !!selId,
  })

  const nodes = graphQuery.data?.nodes ?? []
  const edges = graphQuery.data?.edges ?? []
  const neighbors = detailQuery.data?.neighbors ?? []
  const grouped: Record<string, typeof neighbors> = {}
  for (const n of neighbors) {
    const k = n.node.kind
    grouped[k] = grouped[k] ? [...grouped[k], n] : [n]
  }
  const groupOrder = ['entity', 'playbook', 'skill', 'fact']
  const selNode = detailQuery.data?.node

  return (
    <div className="border-border flex min-h-0 flex-1 flex-col rounded-b-md rounded-tr-md border">
      <p className="text-muted-foreground border-border border-b px-3 py-2 text-[11px] leading-snug">
        {t('web.knowledge.graph.intro')}
      </p>
      <div className="flex min-h-0 flex-1">
        <div className="relative min-h-0 flex-1 overflow-hidden">
          {graphQuery.isLoading ? (
            <Loader2 className="m-4 h-4 w-4 animate-spin" />
          ) : nodes.length === 0 ? (
            <div className="flex h-full items-center justify-center p-8">
              <p className="text-muted-foreground max-w-md text-center text-sm leading-relaxed">
                {t('web.knowledge.graph.empty')}
              </p>
            </div>
          ) : (
            <>
              <GraphCanvas
                nodes={nodes}
                edges={edges}
                selectedId={selId}
                onSelect={setSelId}
              />
              {/* legend + controls hint, floating over the canvas */}
              <div className="bg-card/80 border-border pointer-events-none absolute bottom-2 left-2 rounded-md border px-2.5 py-1.5 backdrop-blur">
                <div className="flex items-center gap-3">
                  {['project', 'entity', 'playbook', 'skill'].map((k) => (
                    <span key={k} className="flex items-center gap-1 text-[10px]">
                      <span
                        className="inline-block h-2 w-2 rounded-full"
                        style={{ backgroundColor: KIND_COLORS[k] }}
                      />
                      {t(`web.knowledge.graph.legend.${k}`)}
                    </span>
                  ))}
                </div>
                <p className="text-muted-foreground/80 mt-1 text-[10px]">
                  {t('web.knowledge.graph.hint')}
                </p>
              </div>
            </>
          )}
        </div>

        {/* side panel — the selected node's neighborhood, grouped by kind */}
        {selId && (
          <div className="border-border flex w-80 shrink-0 flex-col border-l">
            <div className="border-border flex items-center gap-2 border-b px-3 py-2">
              <h2 className="min-w-0 flex-1 truncate text-sm font-semibold">
                {selNode ? nodeDisplayTitle(selNode) : '…'}
              </h2>
              {selNode && <KindBadge kind={nodeColorKey(selNode)} />}
              <button
                onClick={() => setSelId(null)}
                className="text-muted-foreground hover:text-foreground text-xs"
              >
                ✕
              </button>
            </div>
            <div className="min-h-0 flex-1 overflow-auto p-3">
              {detailQuery.isLoading ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <>
                  {selNode?.body && (
                    <p className="text-muted-foreground mb-3 line-clamp-4 text-xs">
                      {selNode.body}
                    </p>
                  )}
                  <p className="text-muted-foreground mb-3 text-xs">
                    {t('web.knowledge.graph.connections', {
                      count: neighbors.length,
                    })}
                  </p>
                  {neighbors.length === 0 && (
                    <p className="text-muted-foreground text-sm">
                      {t('web.knowledge.graph.noLinks')}
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
                            <button
                              key={n.node.id}
                              onClick={() => setSelId(n.node.id)}
                              className="bg-card hover:bg-card/60 flex w-full items-center gap-2 rounded-md border px-2.5 py-1.5 text-left"
                            >
                              <span className="min-w-0 flex-1 truncate text-sm">
                                {nodeDisplayTitle(n.node)}
                              </span>
                              <span className="text-muted-foreground text-[10px]">
                                {n.edge_type}
                              </span>
                            </button>
                          ))}
                        </div>
                      </section>
                    ))}
                </>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export function KnowledgePage() {
  const { t } = useTranslation()
  const [tab, setTab] = useState<'kb' | 'distill' | 'graph'>('kb')

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
          <TabBtn active={tab === 'graph'} onClick={() => setTab('graph')}>
            {t('web.knowledge.graph.tab')}
          </TabBtn>
        </div>
      </header>
      <div className="flex flex-1 flex-col min-h-0 px-4 pb-4">
        {tab === 'kb' ? (
          <KnowledgeBaseView />
        ) : tab === 'distill' ? (
          <DistillationView />
        ) : (
          <GraphView />
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

// The vault directory name the skill sink rendered this skill under —
// recorded by skillify as provenance.skill_id (legacy rows fall back to
// nothing and the dialog just omits the slug).
function skillSlugOf(n: KnowledgeNode): string {
  const v = n.provenance?.skill_id
  return typeof v === 'string' ? v : ''
}

// SKILL.md bodies carry YAML frontmatter — noise in a rendered preview.
function stripFrontmatter(md: string): string {
  const m = md.match(/^---\n[\s\S]*?\n---\n?/)
  return m ? md.slice(m[0].length) : md
}

// NodePreviewDialog shows a playbook / skill's full body (the actual
// procedure) so the operator can judge it before promoting, retiring or
// trusting it. Skills link through to Plugins → Agent Skills, where the
// rendered SKILL.md lives once the skill is enabled.
function NodePreviewDialog({
  node,
  onClose,
}: {
  node: KnowledgeNode | null
  onClose: () => void
}) {
  const { t } = useTranslation()
  if (!node) return null
  const slug = skillSlugOf(node)
  return (
    <Dialog open onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="flex max-h-[80vh] flex-col overflow-hidden sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 pr-6">
            <span className="min-w-0 truncate">{node.title}</span>
            <KindBadge kind={node.kind} />
          </DialogTitle>
          {node.kind === 'skill' && (
            <DialogDescription className="flex flex-wrap items-center gap-2">
              {slug && (
                <code className="bg-muted rounded px-1 py-0.5 text-[11px]">
                  {slug}
                </code>
              )}
              {node.enabled !== false ? (
                <Link
                  to="/plugins"
                  className="text-primary text-xs underline underline-offset-2"
                  title={t('web.knowledge.distill.agentSkillsHint')}
                >
                  {t('web.knowledge.distill.inAgentSkills')}
                </Link>
              ) : (
                <span className="text-muted-foreground text-xs">
                  {t('web.knowledge.distill.notInVault')}
                </span>
              )}
            </DialogDescription>
          )}
        </DialogHeader>
        <div className="border-border min-h-0 flex-1 overflow-auto rounded-md border p-3">
          <ReactMarkdown remarkPlugins={[remarkGfm]} components={MD}>
            {stripFrontmatter(node.body)}
          </ReactMarkdown>
        </div>
      </DialogContent>
    </Dialog>
  )
}

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
  const [preview, setPreview] = useState<KnowledgeNode | null>(null)

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
                      <button
                        onClick={() => setPreview(n)}
                        className="text-left text-sm font-medium hover:underline"
                        title={t('web.knowledge.distill.viewHint')}
                      >
                        {n.title}
                      </button>
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
                      <button
                        onClick={() => setPreview(n)}
                        className="text-left text-sm font-medium hover:underline"
                        title={t('web.knowledge.distill.viewHint')}
                      >
                        {n.title}
                      </button>
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
                    <div className="mt-1 flex items-center gap-3">
                      <button
                        onClick={() => remove.mutate(n.id)}
                        disabled={remove.isPending}
                        className="text-muted-foreground hover:text-destructive text-[11px]"
                      >
                        {t('web.knowledge.distill.retire')}
                      </button>
                      {enabled && (
                        <Link
                          to="/plugins"
                          className="text-primary text-[11px] underline-offset-2 hover:underline"
                          title={t('web.knowledge.distill.agentSkillsHint')}
                        >
                          {t('web.knowledge.distill.inAgentSkills')} →
                        </Link>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </section>
      </div>

      <NodePreviewDialog node={preview} onClose={() => setPreview(null)} />
    </div>
  )
}
