// ProjectScreen — the Cortex project workspace (Notes rung of the
// flywheel). The tab set is no longer hardcoded: it renders the
// project's doc BLUEPRINT — overview front page + whatever sections
// this project's blueprint declares (a mobile app, a service, and a
// CLI can each carry a different section set). Per section: an
// editor / readonly view by maintainer mode, plus a curation chat to
// actively ask the AI for updates. Journal + Inbox + memory Hygiene
// complete the workspace. variant='memory' keeps the standalone
// hygiene view for the legacy route until mobile parity lands.

import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import {
  AlertTriangle,
  Archive,
  ArrowRight,
  Check,
  Inbox,
  LayoutList,
  Loader2,
  Lock,
  MessageSquare,
  Pause,
  Pencil,
  Play,
  RefreshCw,
  RotateCcw,
  Save,
  Trash2,
  Unlock,
  X,
} from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { useTranslation } from 'react-i18next'
import { useNavigate } from '@tanstack/react-router'

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
  type BlueprintSection,
  type DocKind,
  type ProjectDoc,
  type DocProposal,
  type SessionLogEntry,
  type ProjectStatus,
  approveProposal,
  listBlueprintSections,
  listProjectDocs,
  listPendingProposals,
  listProjects,
  listSessionLogs,
  putProjectDoc,
  rejectProposal,
  resetProjectMemory,
  setProjectLifecycle,
} from '@/lib/projectDocs'
import {
  type MemoryRecord,
  deleteMemoriesByScope,
  listArchived,
  restoreMemory,
} from '@/lib/memory'
import { draftKB } from '@/lib/knowledge'
import { MemoryHealthCard } from '@/components/project/MemoryHealthCard'
import { ConflictsPanel } from '@/components/project/ConflictsPanel'
import { JournalStalePanel } from '@/components/project/JournalStalePanel'
import { CurationChat } from '@/components/cortex/CurationChat'
import { BlueprintEditor } from '@/components/cortex/BlueprintEditor'

// strip the drafter's hidden signature marker before display/edit
function stripSig(s: string): string {
  return s
    .split('\n')
    .filter((l) => !l.includes('kb-sig:'))
    .join('\n')
    .trim()
}

// explicit markdown styling (no dependency on the typography plugin)
const MD = {
  h1: (p: any) => <h1 className="mt-4 mb-2 text-lg font-semibold" {...p} />,
  h2: (p: any) => (
    <h2 className="border-border mt-4 mb-1.5 border-b pb-1 text-base font-semibold" {...p} />
  ),
  h3: (p: any) => <h3 className="mt-3 mb-1 text-sm font-semibold" {...p} />,
  p: (p: any) => <p className="my-1.5 text-sm leading-relaxed" {...p} />,
  ul: (p: any) => <ul className="my-1.5 ml-5 list-disc space-y-0.5 text-sm" {...p} />,
  ol: (p: any) => <ol className="my-1.5 ml-5 list-decimal space-y-0.5 text-sm" {...p} />,
  li: (p: any) => <li className="leading-relaxed" {...p} />,
  code: (p: any) => (
    <code className="bg-muted rounded px-1 py-0.5 font-mono text-[12px]" {...p} />
  ),
  strong: (p: any) => <strong className="font-semibold" {...p} />,
  a: (p: any) => <a className="text-primary underline" {...p} />,
  hr: () => <hr className="border-border my-3" />,
}

// ProjectScreen renders one rung of the flywheel for a project (cwd).
// variant='notes' → the project's official doc (goal/plan/tech/activity/
// journal/inbox); variant='memory' → its memory hygiene (health/conflicts/
// archived). Deconflated per the Experience Flywheel architecture so Notes
// and Memory never embed each other.
interface ProjectScreenProps {
  cwd: string
  variant?: 'notes' | 'memory'
}

const CLASSIC_LABEL_KINDS = new Set([
  'overview',
  'goal',
  'plan',
  'tech_stack',
  'recent_activity',
  'kb_infrastructure',
  'kb_conventions',
  'kb_lessons',
  'kb_reusable',
])

function useDocLabel(section?: BlueprintSection) {
  const { t } = useTranslation()
  return (kind: DocKind | 'goal' | 'plan'): string => {
    if (CLASSIC_LABEL_KINDS.has(kind)) return t(`web.project.docLabel.${kind}`)
    // Custom blueprint sections label themselves.
    return section?.slug === kind ? section.title : String(kind)
  }
}

export function ProjectScreen({ cwd, variant = 'notes' }: ProjectScreenProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState(
    variant === 'memory' ? 'health' : 'overview',
  )
  const [blueprintOpen, setBlueprintOpen] = useState(false)
  // A conflict that implicates goal/plan is resolved by editing the doc — which
  // lives on the Cortex project workspace. From the memory variant, jump there.
  const jumpToDoc = (tab: string) =>
    variant === 'memory'
      ? navigate({ to: '/cortex/project', search: { cwd } })
      : setActiveTab(tab)

  // The blueprint IS the tab set (Cortex Phase 3) — lazily seeded with
  // the classic overview/goal/plan/tech/activity on first visit.
  const blueprintQuery = useQuery({
    queryKey: ['blueprint', cwd],
    queryFn: () => listBlueprintSections(cwd),
    enabled: !!cwd && variant === 'notes',
  })
  const sections = useMemo(
    () => blueprintQuery.data ?? [],
    [blueprintQuery.data],
  )
  // If the active section tab disappears (blueprint edit), fall back
  // to the overview front page.
  useEffect(() => {
    if (variant !== 'notes' || sections.length === 0) return
    const fixed = ['journal', 'inbox', 'hygiene']
    if (fixed.includes(activeTab)) return
    if (!sections.some((s) => s.slug === activeTab)) setActiveTab('overview')
  }, [sections, activeTab, variant])

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
  const archivedQuery = useQuery({
    queryKey: ['archived-memories', 'project', cwd],
    queryFn: () => listArchived('project', cwd, 200),
    enabled: !!cwd,
  })

  const docsByKind = useMemo(() => {
    // Keyed by section slug — the blueprint decides which of these the
    // UI shows. kb_* docs live in the Knowledge layer, never here.
    const map: Record<string, ProjectDoc | undefined> = {}
    for (const d of docsQuery.data ?? []) map[d.kind] = d
    return map
  }, [docsQuery.data])

  // Which doc kinds have a pending AI proposal — drives the per-doc banner so
  // the AI-drive (P-B) is visible right on the goal/plan tab, not just Inbox.
  const pendingByKind = useMemo(() => {
    const s = new Set<string>()
    for (const p of proposalsQuery.data ?? []) s.add(p.kind)
    return s
  }, [proposalsQuery.data])

  const inboxCount = proposalsQuery.data?.length ?? 0
  const archivedCount = archivedQuery.data?.length ?? 0
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
              {archivedCount > 0 && (
                <>
                  <span>·</span>
                  <Badge variant="muted" className="text-[10px]">
                    {t('web.project.header.archivedCount', {
                      count: archivedCount,
                    })}
                  </Badge>
                </>
              )}
            </div>
          </div>
          <div className="flex flex-none items-center gap-2">
            {variant === 'notes' && (
              <Button
                size="sm"
                variant="outline"
                className="flex-none"
                onClick={() => setBlueprintOpen(true)}
                title={t('web.cortex.blueprint.openHint')}
              >
                <LayoutList className="mr-1 h-3 w-3" />
                {t('web.cortex.blueprint.open')}
              </Button>
            )}
            <LifecycleControl cwd={cwd} />
            <ResetButton
              cwd={cwd}
              onDone={() => {
                qc.invalidateQueries({ queryKey: ['project-docs', cwd] })
                qc.invalidateQueries({ queryKey: ['project-doc-proposals', cwd] })
                qc.invalidateQueries({ queryKey: ['session-logs', cwd] })
                qc.invalidateQueries({ queryKey: ['archived-memories'] })
                qc.invalidateQueries({ queryKey: ['memories'] })
              }}
            />
          </div>
        </div>
      </div>

      <Tabs
        value={activeTab}
        onValueChange={setActiveTab}
        className="flex flex-1 flex-col overflow-hidden"
      >
        <TabsList className="bg-muted/30 mx-4 mt-3 w-fit max-w-[calc(100%-2rem)] flex-wrap">
          {variant === 'notes' ? (
            <>
              {sections.map((sec) => (
                <TabsTrigger key={sec.slug} value={sec.slug}>
                  {sectionTabLabel(sec, t)}
                </TabsTrigger>
              ))}
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
              <TabsTrigger value="hygiene">
                {t('web.project.tabs.hygiene')}
              </TabsTrigger>
            </>
          ) : (
            <>
              <TabsTrigger value="health">
                {t('web.project.tabs.health')}
              </TabsTrigger>
              <TabsTrigger value="conflicts">
                {t('web.project.tabs.conflicts')}
              </TabsTrigger>
              <TabsTrigger value="archived" className="relative">
                {t('web.project.tabs.archived')}
                {archivedCount > 0 && (
                  <span className="bg-secondary absolute -top-1 -right-2 flex h-4 min-w-4 items-center justify-center rounded-full px-1 text-[9px] font-bold">
                    {archivedCount}
                  </span>
                )}
              </TabsTrigger>
            </>
          )}
        </TabsList>

        <TabsContent value="health" className="flex-1 overflow-auto">
          <MemoryHealthCard cwd={cwd} />
        </TabsContent>

        {variant === 'notes' &&
          sections.map((sec) => (
            <TabsContent
              key={sec.slug}
              value={sec.slug}
              className="flex-1 overflow-auto p-4"
            >
              <SectionTab
                cwd={cwd}
                section={sec}
                doc={docsByKind[sec.slug]}
                hasPending={pendingByKind.has(sec.slug)}
                onGoToInbox={() => setActiveTab('inbox')}
                onSaved={() =>
                  qc.invalidateQueries({ queryKey: ['project-docs', cwd] })
                }
              />
            </TabsContent>
          ))}

        <TabsContent value="hygiene" className="flex-1 overflow-auto">
          <MemoryHealthCard cwd={cwd} />
          <div className="p-4 pt-0">
            <h3 className="mb-2 text-sm font-semibold">
              {t('web.project.tabs.conflicts')}
            </h3>
            <ConflictsPanel cwd={cwd} onJumpTab={jumpToDoc} />
            <h3 className="mt-4 mb-2 text-sm font-semibold">
              {t('web.project.tabs.archived')}
            </h3>
            <ArchivedTab
              records={archivedQuery.data ?? []}
              loading={archivedQuery.isLoading}
              onChange={() =>
                qc.invalidateQueries({
                  queryKey: ['archived-memories', 'project', cwd],
                })
              }
            />
          </div>
        </TabsContent>

        <TabsContent value="journal" className="flex-1 overflow-auto p-4">
          <JournalStalePanel cwd={cwd} />
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

        <TabsContent value="conflicts" className="flex-1 overflow-auto">
          <ConflictsPanel cwd={cwd} onJumpTab={jumpToDoc} />
        </TabsContent>

        <TabsContent value="archived" className="flex-1 overflow-auto p-4">
          <ArchivedTab
            records={archivedQuery.data ?? []}
            loading={archivedQuery.isLoading}
            onChange={() =>
              qc.invalidateQueries({
                queryKey: ['archived-memories', 'project', cwd],
              })
            }
          />
        </TabsContent>
      </Tabs>

      {variant === 'notes' && (
        <BlueprintEditor
          cwd={cwd}
          open={blueprintOpen}
          onOpenChange={setBlueprintOpen}
          onApplied={() => {
            qc.invalidateQueries({ queryKey: ['blueprint', cwd] })
            qc.invalidateQueries({ queryKey: ['project-docs', cwd] })
          }}
        />
      )}
    </div>
  )
}

// sectionTabLabel prefers the localized label for the classic slugs
// (so zh/es operators keep translated tabs) and falls back to the
// section's own stored title for custom sections.
const CLASSIC_TAB_KEY: Record<string, string> = {
  overview: 'web.project.tabs.overview',
  goal: 'web.project.tabs.goal',
  plan: 'web.project.tabs.plan',
  tech_stack: 'web.project.tabs.tech',
  recent_activity: 'web.project.tabs.activity',
}
function sectionTabLabel(
  sec: BlueprintSection,
  t: (k: string) => string,
): string {
  const key = CLASSIC_TAB_KEY[sec.slug]
  return key ? t(key) : sec.title
}

// ─── SectionTab — one blueprint section, rendered by maintainer mode ──

function SectionTab({
  cwd,
  section,
  doc,
  hasPending,
  onGoToInbox,
  onSaved,
}: {
  cwd: string
  section: BlueprintSection
  doc?: ProjectDoc
  hasPending: boolean
  onGoToInbox: () => void
  onSaved: () => void
}) {
  const { t } = useTranslation()
  const [chatOpen, setChatOpen] = useState(false)

  let body: React.ReactNode
  if (section.slug === 'overview') {
    body = (
      <OverviewTab
        cwd={cwd}
        doc={doc}
        hasPending={hasPending}
        onGoToInbox={onGoToInbox}
        onSaved={onSaved}
      />
    )
  } else if (section.maintainer_mode === 'scanner') {
    body = <ReadonlyDocTab doc={doc} kind={section.slug} section={section} />
  } else {
    body = (
      <DocEditor
        cwd={cwd}
        kind={section.slug}
        doc={doc}
        section={section}
        hasPending={hasPending}
        onGoToInbox={onGoToInbox}
        onSaved={onSaved}
      />
    )
  }

  return (
    <div className="space-y-3">
      {body}
      <div>
        <Button
          size="sm"
          variant={chatOpen ? 'default' : 'outline'}
          onClick={() => setChatOpen((v) => !v)}
        >
          <MessageSquare className="mr-1 h-3 w-3" />
          {chatOpen ? t('web.cortex.chat.hide') : t('web.cortex.chat.show')}
        </Button>
      </div>
      {chatOpen && (
        <CurationChat
          targetKind="doc_section"
          targetCwd={cwd}
          targetSlug={section.slug}
          onRevision={onSaved}
        />
      )}
    </div>
  )
}

// ─── Doc self-description (A: make each note explain itself) ──

// How each doc is kept current — derived from the blueprint section's
// maintainer mode so the operator can see at a glance who owns the page.
type Maintainer = 'coauthored' | 'auto' | 'human'
const MAINTAINER_STYLE: Record<Maintainer, string> = {
  coauthored: 'bg-blue-500/15 text-blue-400',
  auto: 'bg-zinc-500/15 text-zinc-300',
  human: 'bg-emerald-500/15 text-emerald-400',
}
const CLASSIC_PURPOSE_KEY: Record<string, string> = {
  overview: 'web.project.docMeta.purpose.overview',
  goal: 'web.project.docMeta.purpose.goal',
  plan: 'web.project.docMeta.purpose.plan',
  tech_stack: 'web.project.docMeta.purpose.tech_stack',
  recent_activity: 'web.project.docMeta.purpose.recent_activity',
}

function maintainerOf(kind: DocKind, section?: BlueprintSection): Maintainer {
  const mode =
    section?.maintainer_mode ??
    (kind === 'tech_stack' || kind === 'recent_activity' ? 'scanner' : 'ai')
  if (mode === 'scanner') return 'auto'
  if (mode === 'human') return 'human'
  return 'coauthored'
}

// DocMetaStrip is the per-note header that explains what the note is, who
// maintains it, and when it was last touched — so the page is self-describing
// instead of a bare tab label. Custom sections describe themselves via the
// blueprint's own description text.
function DocMetaStrip({
  kind,
  doc,
  section,
}: {
  kind: DocKind
  doc?: ProjectDoc
  section?: BlueprintSection
}) {
  const { t } = useTranslation()
  const maintainer = maintainerOf(kind, section)
  const purposeKey = CLASSIC_PURPOSE_KEY[kind]
  const purpose = purposeKey ? t(purposeKey) : (section?.description ?? '')
  return (
    <div className="bg-muted/20 mb-3 rounded-md p-3">
      <div className="mb-1 flex items-center gap-2">
        <span
          className={`rounded px-1.5 py-0.5 text-[10px] font-medium ${MAINTAINER_STYLE[maintainer]}`}
        >
          {t(`web.project.docMeta.maintainer.${maintainer}`)}
        </span>
        {doc?.updated_at && (
          <span className="text-muted-foreground text-[11px]">
            {t('web.project.editor.updatedBy')} <strong>{doc.updated_by}</strong> ·{' '}
            {new Date(doc.updated_at).toLocaleString()}
          </span>
        )}
      </div>
      {purpose && (
        <p className="text-muted-foreground text-xs leading-relaxed">{purpose}</p>
      )}
    </div>
  )
}

// ─── Overview — the project's rich, AI-maintained official doc ──

function OverviewTab({
  cwd,
  doc,
  hasPending,
  onGoToInbox,
  onSaved,
}: {
  cwd: string
  doc?: ProjectDoc
  hasPending?: boolean
  onGoToInbox?: () => void
  onSaved: () => void
}) {
  const { t } = useTranslation()
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState('')
  const locked = doc?.updated_by === 'operator'
  const content = stripSig(doc?.content ?? '')

  const save = useMutation({
    mutationFn: (updatedBy: 'operator' | 'agent') =>
      putProjectDoc({
        cwd,
        kind: 'overview',
        content: updatedBy === 'operator' ? draft : content,
        updatedBy,
      }),
    onSuccess: (_d, updatedBy) => {
      setEditing(false)
      onSaved()
      toast.success(
        updatedBy === 'operator'
          ? t('web.project.overview.saved')
          : t('web.project.overview.unlocked'),
      )
    },
    onError: (e: Error) =>
      toast.error(t('web.project.editor.saveFailedToast'), { description: e.message }),
  })

  const regen = useMutation({
    mutationFn: () => draftKB(),
    onSuccess: () => toast.success(t('web.project.overview.regenerating')),
    onError: () => toast.error(t('web.project.editor.saveFailedToast')),
  })

  return (
    <div className="space-y-3">
      <DocMetaStrip kind="overview" doc={doc} />
      <div className="flex items-center justify-between gap-2">
        <span className="text-muted-foreground flex items-center gap-1.5 text-xs">
          {locked ? (
            <>
              <Lock className="h-3 w-3" />
              {t('web.project.overview.locked')}
            </>
          ) : (
            <>
              <Check className="h-3 w-3" />
              {t('web.project.overview.aiManaged')}
            </>
          )}
        </span>
        <div className="flex gap-2">
          {locked && !editing && (
            <Button
              size="sm"
              variant="outline"
              disabled={save.isPending}
              onClick={() => save.mutate('agent')}
            >
              <Unlock className="mr-1 h-3 w-3" />
              {t('web.project.overview.unlock')}
            </Button>
          )}
          {!editing && (
            <Button
              size="sm"
              variant="outline"
              disabled={regen.isPending}
              onClick={() => regen.mutate()}
              title={t('web.project.overview.regenerateHint')}
            >
              <RefreshCw className="mr-1 h-3 w-3" />
              {t('web.project.overview.regenerate')}
            </Button>
          )}
          {!editing && (
            <Button
              size="sm"
              variant="outline"
              onClick={() => {
                setDraft(content)
                setEditing(true)
              }}
            >
              <Pencil className="mr-1 h-3 w-3" />
              {t('web.project.overview.edit')}
            </Button>
          )}
        </div>
      </div>

      {hasPending && !editing && (
        <button
          onClick={onGoToInbox}
          className="flex w-full items-center justify-between gap-2 rounded-md border border-amber-500/30 bg-amber-500/10 p-3 text-left text-xs text-amber-300 hover:bg-amber-500/15"
        >
          <span className="flex items-center gap-2">
            <Inbox className="h-3.5 w-3.5 flex-none" />
            {t('web.project.proposalBanner.text')}
          </span>
          <span className="flex items-center gap-1 font-medium">
            {t('web.project.proposalBanner.button')}
            <ArrowRight className="h-3.5 w-3.5" />
          </span>
        </button>
      )}

      {editing ? (
        <div className="space-y-2">
          <Textarea
            rows={26}
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            className="font-mono text-sm"
          />
          <p className="text-muted-foreground text-[11px]">
            {t('web.project.overview.editHint')}
          </p>
          <div className="flex justify-end gap-2">
            <Button size="sm" variant="ghost" onClick={() => setEditing(false)}>
              {t('web.project.overview.cancel')}
            </Button>
            <Button
              size="sm"
              disabled={save.isPending}
              onClick={() => save.mutate('operator')}
            >
              {save.isPending ? (
                <Loader2 className="mr-1 h-3 w-3 animate-spin" />
              ) : (
                <Save className="mr-1 h-3 w-3" />
              )}
              {t('web.project.overview.save')}
            </Button>
          </div>
        </div>
      ) : content ? (
        <div className="bg-muted/20 rounded-md p-4">
          <ReactMarkdown remarkPlugins={[remarkGfm]} components={MD}>
            {content}
          </ReactMarkdown>
        </div>
      ) : (
        <div className="text-muted-foreground space-y-2 text-sm">
          <p>{t('web.project.overview.empty')}</p>
          <Button
            size="sm"
            variant="outline"
            disabled={regen.isPending}
            onClick={() => regen.mutate()}
          >
            <RefreshCw className="mr-1 h-3 w-3" />
            {t('web.project.overview.generate')}
          </Button>
        </div>
      )}
    </div>
  )
}

// ─── Doc editor (goal / plan) ────────────────────────────────

interface DocEditorProps {
  cwd: string
  kind: DocKind
  doc?: ProjectDoc
  section?: BlueprintSection
  hasPending?: boolean
  onGoToInbox?: () => void
  onSaved: () => void
}

function DocEditor({
  cwd,
  kind,
  doc,
  section,
  hasPending,
  onGoToInbox,
  onSaved,
}: DocEditorProps) {
  const { t } = useTranslation()
  const labelFor = useDocLabel(section)
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
      <DocMetaStrip kind={kind} doc={doc} section={section} />
      {hasPending && (
        <button
          onClick={onGoToInbox}
          className="flex w-full items-center justify-between gap-2 rounded-md border border-amber-500/30 bg-amber-500/10 p-3 text-left text-xs text-amber-300 hover:bg-amber-500/15"
        >
          <span className="flex items-center gap-2">
            <Inbox className="h-3.5 w-3.5 flex-none" />
            {t('web.project.proposalBanner.text')}
          </span>
          <span className="flex items-center gap-1 font-medium">
            {t('web.project.proposalBanner.button')}
            <ArrowRight className="h-3.5 w-3.5" />
          </span>
        </button>
      )}
      <div className="flex items-center justify-end">
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
            : kind === 'plan'
              ? t('web.project.editor.planPlaceholder')
              : (section?.description ??
                t('web.project.editor.sectionPlaceholder'))
        }
        className="font-mono text-sm"
      />
    </div>
  )
}

// ─── Read-only doc (tech_stack / recent_activity) ────────────

interface ReadonlyDocTabProps {
  doc?: ProjectDoc
  kind: DocKind
  section?: BlueprintSection
}

function ReadonlyDocTab({ doc, kind, section }: ReadonlyDocTabProps) {
  const { t } = useTranslation()
  const classic = kind === 'tech_stack' || kind === 'recent_activity'
  const kindLabel = classic
    ? t(`web.project.readonly.${kind}.label`)
    : (section?.title ?? String(kind))
  const emptyHint = classic
    ? t(`web.project.readonly.${kind}.empty`)
    : t('web.project.readonly.customEmpty')
  if (!doc) {
    return (
      <div className="text-muted-foreground text-sm">
        <DocMetaStrip kind={kind} doc={undefined} section={section} />
        <p className="mb-2">
          {t('web.project.readonly.noneCaptured', { label: kindLabel })}
        </p>
        <p className="text-xs">{emptyHint}</p>
      </div>
    )
  }
  return (
    <div className="space-y-3">
      <DocMetaStrip kind={kind} doc={doc} section={section} />
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

// ─── Archived tab (read-only + restore) ──────────────────────
//
// The cleaner auto-applies its keep/stale/duplicate verdicts as
// reversible soft-archives — there is no approval queue anymore. This
// tab is where the operator sees what was auto-removed for this
// project and restores any false positive before the 30-day grace
// window hard-purges it.

function ArchivedTab({
  records,
  loading,
  onChange,
}: {
  records: MemoryRecord[]
  loading: boolean
  onChange: () => void
}) {
  const { t } = useTranslation()

  return (
    <div className="space-y-3">
      <p className="text-muted-foreground text-xs">
        {t('web.project.archived.hint')}
      </p>
      {loading ? (
        <Loader2 className="h-3 w-3 animate-spin" />
      ) : records.length === 0 ? (
        <p className="text-muted-foreground py-8 text-center text-sm">
          {t('web.project.archived.empty')}
        </p>
      ) : (
        records.map((r) => (
          <ArchivedCard key={r.id} record={r} onChange={onChange} />
        ))
      )}
    </div>
  )
}

function ArchivedCard({
  record,
  onChange,
}: {
  record: MemoryRecord
  onChange: () => void
}) {
  const { t } = useTranslation()
  const restore = useMutation({
    mutationFn: () => restoreMemory(record.id),
    onSuccess: () => {
      toast.success(t('web.project.archived.restoredToast'))
      onChange()
    },
    onError: (e: Error) => {
      toast.error(t('web.project.archived.restoreFailedToast'), {
        description: e.message,
      })
      onChange()
    },
  })

  return (
    <div className="bg-card rounded-md border p-3">
      <div className="mb-2 flex items-center gap-2">
        {record.archived_reason && (
          <Badge variant="muted">{record.archived_reason}</Badge>
        )}
        {record.archived_at && (
          <span className="text-muted-foreground text-[11px]">
            {t('web.project.archived.archivedAtPrefix')}{' '}
            {new Date(record.archived_at).toLocaleString()}
          </span>
        )}
      </div>
      <pre className="bg-muted/20 mb-3 max-h-32 overflow-auto rounded p-2 font-mono text-[11px] whitespace-pre-wrap">
        {record.text}
      </pre>
      <Button
        size="sm"
        variant="outline"
        onClick={() => restore.mutate()}
        disabled={restore.isPending}
      >
        {restore.isPending ? (
          <Loader2 className="mr-1 h-3 w-3 animate-spin" />
        ) : (
          <RotateCcw className="mr-1 h-3 w-3" />
        )}
        {t('web.project.archived.restoreButton')}
      </Button>
    </div>
  )
}

// ─── Lifecycle control (P-D) ─────────────────────────────────

// LifecycleControl shows the project's lifecycle status and lets the operator
// move it between active / paused / archived. Frozen (paused/archived)
// projects are excluded from spawn injection + cross-project distillation.
function LifecycleControl({ cwd }: { cwd: string }) {
  const { t } = useTranslation()
  const qc = useQueryClient()

  const projectsQuery = useQuery({
    queryKey: ['projects-lifecycle'],
    queryFn: () => listProjects(),
  })
  const summary = projectsQuery.data?.find((p) => p.cwd === cwd)
  const status: ProjectStatus = summary?.status ?? 'active'
  const suggestArchive = summary?.suggest_archive ?? false

  const mutation = useMutation({
    mutationFn: (next: ProjectStatus) => setProjectLifecycle(cwd, next),
    onSuccess: (_data, next) => {
      qc.invalidateQueries({ queryKey: ['projects-lifecycle'] })
      toast.success(t(`web.project.lifecycle.applied.${next}`))
    },
    onError: (e) =>
      toast.error(t('web.project.lifecycle.failedToast'), {
        description: e instanceof Error ? e.message : String(e),
      }),
  })

  const badgeVariant =
    status === 'archived' ? 'muted' : status === 'paused' ? 'warning' : 'success'

  return (
    <div className="flex items-center gap-1.5">
      <Badge
        variant={badgeVariant}
        className="text-[10px] capitalize"
        title={t('web.project.lifecycle.tooltip.badge')}
      >
        {t(`web.project.lifecycle.status.${status}`)}
      </Badge>
      {suggestArchive && status === 'active' && (
        <Badge
          variant="warning"
          className="text-[10px]"
          title={t('web.project.lifecycle.idleHint', {
            days: summary?.idle_days ?? 0,
          })}
        >
          {t('web.project.lifecycle.idleSuggest')}
        </Badge>
      )}
      {status !== 'active' && (
        <Button
          size="sm"
          variant="outline"
          className="flex-none"
          title={t('web.project.lifecycle.tooltip.activate')}
          disabled={mutation.isPending}
          onClick={() => mutation.mutate('active')}
        >
          <Play className="mr-1 h-3 w-3" />
          {t('web.project.lifecycle.activate')}
        </Button>
      )}
      {status === 'active' && (
        <Button
          size="sm"
          variant="outline"
          className="flex-none"
          title={t('web.project.lifecycle.tooltip.pause')}
          disabled={mutation.isPending}
          onClick={() => mutation.mutate('paused')}
        >
          <Pause className="mr-1 h-3 w-3" />
          {t('web.project.lifecycle.pause')}
        </Button>
      )}
      {status !== 'archived' && (
        <Button
          size="sm"
          variant="outline"
          className="flex-none"
          title={t('web.project.lifecycle.tooltip.archive')}
          disabled={mutation.isPending}
          onClick={() => mutation.mutate('archived')}
        >
          <Archive className="mr-1 h-3 w-3" />
          {t('web.project.lifecycle.archive')}
        </Button>
      )}
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
