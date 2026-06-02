import { useEffect, useMemo, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Plus,
  Loader2,
  Search,
  Hash,
  Calendar,
  NotebookPen,
  FolderOpen,
  PanelRightClose,
  PanelRightOpen,
} from 'lucide-react'
import { format } from 'date-fns'
import { toast } from 'sonner'
import { Trans, useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'
import {
  listNotes,
  notesInfo,
  notesTags,
  writeNote,
} from '@/lib/notes'
import type { Note } from '@/lib/notes'
import { extractOutline } from '@/lib/outline'
import { vaultStatus } from '@/lib/vaultgit'
import { NoteEditor } from '@/components/sessions/inspector/NoteEditor'
import {
  NotesTreeView,
  type NotesTreeViewHandle,
} from '@/components/notes/NotesTreeView'
import {
  OutlineHeader,
  OutlineSidebar,
} from '@/components/notes/OutlineSidebar'
import {
  VaultSyncBadge,
  VaultSyncDialog,
} from '@/components/notes/VaultSyncDialog'

// localStorage keys persist the last selection / left-pane mode so
// reopening the page lands you back where you left off.
const LS_LAST_PATH = 'opendray.notes.lastPath'
const LS_LEFT_MODE = 'opendray.notes.leftMode'
const LS_OUTLINE_OPEN = 'opendray.notes.outlineOpen'

type LeftMode = 'tree' | 'tags'

// NotesPage is the top-level vault browser/editor. Two-pane layout:
// left = navigation (tree or tag picker, with title/path filter at
// top), right = NoteEditor for the selected note. Independent of any
// session — works even when no sessions are running.
export function NotesPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const { data: info } = useQuery({
    queryKey: ['notes-info'],
    queryFn: notesInfo,
    staleTime: 60_000,
  })
  const { data: notes, isLoading } = useQuery({
    queryKey: ['notes-list'],
    queryFn: () => listNotes(),
    staleTime: 5_000,
    refetchInterval: 8_000,
  })

  const [leftMode, setLeftMode] = useState<LeftMode>(() => {
    if (typeof window === 'undefined') return 'tree'
    return (localStorage.getItem(LS_LEFT_MODE) as LeftMode) || 'tree'
  })
  useEffect(() => {
    if (typeof window !== 'undefined') {
      localStorage.setItem(LS_LEFT_MODE, leftMode)
    }
  }, [leftMode])

  const [filter, setFilter] = useState('')
  const [activeTag, setActiveTag] = useState<string | null>(null)

  const [outlineOpen, setOutlineOpen] = useState<boolean>(() => {
    if (typeof window === 'undefined') return true
    const stored = localStorage.getItem(LS_OUTLINE_OPEN)
    return stored == null ? true : stored === '1'
  })

  // Vault git status for the header badge — slow poll is fine, the
  // dialog has its own faster poll while open.
  const { data: vault } = useQuery({
    queryKey: ['vault-status'],
    queryFn: vaultStatus,
    refetchInterval: 15_000,
    staleTime: 5_000,
  })
  const [syncOpen, setSyncOpen] = useState(false)
  useEffect(() => {
    if (typeof window !== 'undefined') {
      localStorage.setItem(LS_OUTLINE_OPEN, outlineOpen ? '1' : '0')
    }
  }, [outlineOpen])

  const [selected, setSelected] = useState<string | null>(() => {
    if (typeof window === 'undefined') return null
    return localStorage.getItem(LS_LAST_PATH)
  })
  useEffect(() => {
    if (typeof window === 'undefined') return
    if (selected) localStorage.setItem(LS_LAST_PATH, selected)
    else localStorage.removeItem(LS_LAST_PATH)
  }, [selected])

  // If the persisted selection no longer exists in the vault, drop it.
  useEffect(() => {
    if (!selected || !notes) return
    if (!notes.find((n) => n.path === selected)) {
      setSelected(null)
    }
  }, [selected, notes])

  // Live body of the currently-selected note. Streamed from NoteEditor
  // via onBodyChange so the outline reflects unsaved edits without
  // waiting for the 800ms debounced write.
  const [liveBody, setLiveBody] = useState('')
  useEffect(() => setLiveBody(''), [selected])
  const outline = useMemo(() => extractOutline(liveBody), [liveBody])

  // Scroll container of the editor's preview pane — wired through to
  // OutlineSidebar so it can highlight the currently visible heading
  // and scroll on click.
  const previewScrollRef = useRef<HTMLDivElement | null>(null)

  // Tree handle for the toolbar's Expand all / Collapse all buttons.
  const treeRef = useRef<NotesTreeViewHandle | null>(null)

  const { data: tagsData } = useQuery({
    queryKey: ['notes-tags', null],
    queryFn: () => notesTags(),
    staleTime: 30_000,
    enabled: leftMode === 'tags',
  })

  // Filter notes by title/path query and by active tag (when set).
  const visibleNotes = useMemo<Note[]>(() => {
    if (!notes) return []
    let out = notes
    if (activeTag && tagsData) {
      const tag = tagsData.find((x) => x.tag === activeTag)
      if (tag?.notes) {
        const set = new Set(tag.notes)
        out = out.filter((n) => set.has(n.path))
      }
    }
    const q = filter.trim().toLowerCase()
    if (q) {
      out = out.filter(
        (n) =>
          n.path.toLowerCase().includes(q) ||
          n.title.toLowerCase().includes(q),
      )
    }
    return out
  }, [notes, filter, activeTag, tagsData])

  const create = useMutation({
    mutationFn: async (path: string) => {
      const body = `# ${pathTitle(path)}\n\n`
      await writeNote(path, body)
      return path
    },
    onSuccess: (path) => {
      qc.invalidateQueries({ queryKey: ['notes-list'] })
      setSelected(path)
      toast.success(t('web.notes.newNote.createdToast'), { description: path })
    },
    onError: (err: Error) =>
      toast.error(t('web.notes.newNote.createFailedToast'), {
        description: err.message,
      }),
  })

  const handleNewNote = () => {
    const today = format(new Date(), 'yyyy-MM-dd')
    const defaultPath = t('web.notes.newNote.defaultPath', { date: today })
    const input = prompt(t('web.notes.newNote.prompt'), defaultPath)
    if (!input) return
    const cleaned = input
      .trim()
      .replace(/^\/+/, '')
      .split('/')
      .filter((seg) => seg !== '' && seg !== '..' && seg !== '.')
      .join('/')
    if (!cleaned.toLowerCase().endsWith('.md')) {
      toast.error(t('web.notes.newNote.errorMustEndMd'))
      return
    }
    create.mutate(cleaned)
  }

  const handleNewDaily = () => {
    const today = format(new Date(), 'yyyy-MM-dd')
    const path = `daily/${today}.md`
    if (notes?.find((n) => n.path === path)) {
      setSelected(path)
      return
    }
    const body = `---\ndate: ${today}\ntype: daily\n---\n\n# ${format(new Date(), 'EEEE, MMMM d, yyyy')}\n\n## What I'm doing\n\n## What I learned\n\n## TODO\n\n`
    writeNote(path, body)
      .then(() => {
        qc.invalidateQueries({ queryKey: ['notes-list'] })
        setSelected(path)
      })
      .catch((err) =>
        toast.error(t('web.notes.newNote.createFailedToast'), {
          description: (err as Error).message,
        }),
      )
  }

  return (
    <div className="h-full flex flex-col">
      <header className="h-12 border-b border-border flex items-center px-3 gap-2 shrink-0">
        <NotebookPen className="size-4 text-muted-foreground" />
        <h1 className="text-[14px] font-semibold tracking-tight">
          {t('web.notes.title')}
        </h1>
        {info && (
          <span
            className="text-[10.5px] text-muted-foreground/60 font-mono truncate"
            title={info.root}
          >
            · {info.root}
          </span>
        )}
        <div className="flex-1" />
        <VaultSyncBadge status={vault} onClick={() => setSyncOpen(true)} />
        <button
          type="button"
          onClick={() => setOutlineOpen((v) => !v)}
          className={cn(
            'inline-flex items-center gap-1 text-[11px] px-2 py-1 rounded-md transition-colors',
            outlineOpen
              ? 'text-foreground bg-card'
              : 'text-muted-foreground hover:text-foreground',
          )}
          title={
            outlineOpen
              ? t('web.notes.header.hideOutline')
              : t('web.notes.header.showOutline')
          }
        >
          {outlineOpen ? (
            <PanelRightClose className="size-3" />
          ) : (
            <PanelRightOpen className="size-3" />
          )}
          {t('web.notes.header.outline')}
        </button>
        <button
          type="button"
          onClick={handleNewDaily}
          className="inline-flex items-center gap-1 text-[11px] px-2 py-1 rounded-md hover:bg-card text-muted-foreground hover:text-foreground"
          title={t('web.notes.header.todayTooltip')}
        >
          <Calendar className="size-3" />
          {t('web.notes.header.today')}
        </button>
        <button
          type="button"
          onClick={handleNewNote}
          className="inline-flex items-center gap-1 text-[11px] px-2 py-1 rounded-md bg-accent text-accent-foreground hover:bg-accent/90"
        >
          <Plus className="size-3" />
          {t('web.notes.header.new')}
        </button>
      </header>

      <div className="flex-1 flex min-h-0">
        {/* Left pane: tree / tags + filter */}
        <aside className="w-72 shrink-0 border-r border-border flex flex-col bg-background">
          <div className="px-2 py-2 border-b border-border shrink-0 flex flex-col gap-1.5">
            <div className="flex gap-0.5">
              <button
                type="button"
                onClick={() => setLeftMode('tree')}
                className={cn(
                  'flex-1 inline-flex items-center justify-center gap-1 text-[11px] px-2 py-1 rounded-md transition-colors',
                  leftMode === 'tree'
                    ? 'bg-card border border-border text-foreground'
                    : 'text-muted-foreground hover:text-foreground',
                )}
              >
                <FolderOpen className="size-3" />
                {t('web.notes.left.tree')}
              </button>
              <button
                type="button"
                onClick={() => setLeftMode('tags')}
                className={cn(
                  'flex-1 inline-flex items-center justify-center gap-1 text-[11px] px-2 py-1 rounded-md transition-colors',
                  leftMode === 'tags'
                    ? 'bg-card border border-border text-foreground'
                    : 'text-muted-foreground hover:text-foreground',
                )}
              >
                <Hash className="size-3" />
                {t('web.notes.left.tags')}
              </button>
            </div>
            <div className="relative">
              <Search className="absolute left-2 top-1/2 -translate-y-1/2 size-3 text-muted-foreground/60" />
              <input
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
                placeholder={
                  leftMode === 'tree'
                    ? t('web.notes.left.filterNotes')
                    : t('web.notes.left.filterTags')
                }
                className={cn(
                  'w-full h-7 pl-7 pr-2 text-[11.5px] rounded-md',
                  'border border-border bg-input/40 text-foreground',
                  'placeholder:text-muted-foreground/60',
                  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
                )}
              />
            </div>
            {activeTag && (
              <div className="flex items-center gap-1 text-[10.5px]">
                <span className="text-muted-foreground/70">
                  {t('web.notes.left.filteredBy')}
                </span>
                <button
                  type="button"
                  onClick={() => setActiveTag(null)}
                  className="inline-flex items-center gap-0.5 px-1.5 py-px rounded bg-amber-500/15 border border-amber-500/30 text-amber-300 hover:bg-amber-500/25"
                  title={t('web.notes.left.clearTagTooltip')}
                >
                  <Hash className="size-2.5" />
                  {activeTag}
                  <span className="ml-0.5 opacity-70">×</span>
                </button>
              </div>
            )}
            {leftMode === 'tree' && (
              <div className="flex items-center gap-2 text-[10px] text-muted-foreground/70">
                <button
                  type="button"
                  onClick={() => treeRef.current?.expandAll()}
                  className="hover:text-foreground"
                  title={t('web.notes.left.expandAllTooltip')}
                >
                  {t('web.notes.left.expandAll')}
                </button>
                <span className="opacity-40">·</span>
                <button
                  type="button"
                  onClick={() => treeRef.current?.collapseAll()}
                  className="hover:text-foreground"
                  title={t('web.notes.left.collapseAllTooltip')}
                >
                  {t('web.notes.left.collapseAll')}
                </button>
              </div>
            )}
          </div>

          <div className="flex-1 overflow-y-auto px-1 py-2 min-h-0">
            {isLoading ? (
              <div className="flex items-center gap-2 px-2 py-3 text-[12px] text-muted-foreground">
                <Loader2 className="size-3 animate-spin" />
                {t('web.notes.left.loading')}
              </div>
            ) : leftMode === 'tree' ? (
              <NotesTreeView
                ref={treeRef}
                notes={visibleNotes}
                selected={selected}
                onSelect={setSelected}
              />
            ) : (
              <TagsList
                tags={tagsData ?? []}
                filter={filter}
                active={activeTag}
                onSelect={(tag) => {
                  setActiveTag(tag)
                  setLeftMode('tree')
                }}
              />
            )}
          </div>

          <div className="px-3 py-2 border-t border-border text-[10px] text-muted-foreground/60 font-mono">
            {t('web.notes.left.footer', {
              visible: visibleNotes.length,
              total: notes?.length ?? 0,
            })}
          </div>
        </aside>

        {/* Center pane: editor */}
        <main className="flex-1 flex flex-col min-w-0 bg-background">
          {selected ? (
            <div className="flex-1 flex flex-col min-h-0 px-4 py-3 gap-2">
              <div className="flex items-baseline gap-2 shrink-0">
                <h2 className="text-[14px] font-semibold truncate">
                  {pathTitle(selected)}
                </h2>
                <span
                  className="text-[11px] text-muted-foreground/70 font-mono truncate"
                  title={selected}
                >
                  {selected}
                </span>
              </div>
              <NoteEditor
                key={selected}
                path={selected}
                initialMode="preview"
                fillParent
                showBacklinks
                onOpenLink={(p) => setSelected(p)}
                onBodyChange={setLiveBody}
                previewScrollRef={previewScrollRef}
              />
            </div>
          ) : (
            <EmptyState onNew={handleNewNote} onDaily={handleNewDaily} />
          )}
        </main>

        {/* Right pane: outline (toggleable). Only renders when a note
            is open — empty state doesn't need an outline. */}
        {outlineOpen && selected && (
          <aside className="w-60 shrink-0 border-l border-border flex flex-col bg-background">
            <OutlineHeader count={outline.length} />
            <div className="flex-1 overflow-y-auto min-h-0">
              <OutlineSidebar
                headings={outline}
                editorScrollEl={previewScrollRef.current}
                onJump={(slug) => {
                  const root = previewScrollRef.current
                  if (!root) return
                  const el = root.querySelector<HTMLElement>(
                    `[data-outline-id="${cssEscape(slug)}"]`,
                  )
                  el?.scrollIntoView({ behavior: 'smooth', block: 'start' })
                }}
              />
            </div>
          </aside>
        )}
      </div>

      <VaultSyncDialog open={syncOpen} onOpenChange={setSyncOpen} />
    </div>
  )
}

function cssEscape(s: string): string {
  if (typeof CSS !== 'undefined' && typeof CSS.escape === 'function') {
    return CSS.escape(s)
  }
  return s.replace(/[^a-zA-Z0-9_-]/g, (c) => `\\${c}`)
}

function TagsList({
  tags,
  filter,
  active,
  onSelect,
}: {
  tags: { tag: string; count: number }[]
  filter: string
  active: string | null
  onSelect: (tag: string) => void
}) {
  const { t } = useTranslation()
  const q = filter.trim().toLowerCase()
  const visible = q
    ? tags.filter((tag) => tag.tag.toLowerCase().includes(q))
    : tags
  if (visible.length === 0) {
    return (
      <div className="px-2 py-3 text-[11px] text-muted-foreground/60">
        {tags.length === 0
          ? t('web.notes.tags.emptyVault')
          : t('web.notes.tags.noMatches', { query: filter })}
      </div>
    )
  }
  return (
    <div className="flex flex-col">
      {visible.map((tag) => (
        <button
          key={tag.tag}
          type="button"
          onClick={() => onSelect(tag.tag)}
          className={cn(
            'flex items-center gap-1.5 py-0.5 pr-1 pl-1 rounded-sm text-left',
            'hover:bg-card',
            active === tag.tag
              ? 'bg-card text-foreground'
              : 'text-muted-foreground/90',
          )}
        >
          <Hash className="size-3 shrink-0 opacity-60" />
          <span className="truncate text-[12px]">{tag.tag}</span>
          <span className="ml-auto text-[10px] text-muted-foreground/60 font-mono">
            {tag.count}
          </span>
        </button>
      ))}
    </div>
  )
}

function EmptyState({
  onNew,
  onDaily,
}: {
  onNew: () => void
  onDaily: () => void
}) {
  const { t } = useTranslation()
  return (
    <div className="flex-1 flex flex-col items-center justify-center gap-3 text-center px-6">
      <NotebookPen className="size-8 text-muted-foreground/40" strokeWidth={1.5} />
      <div className="space-y-1">
        <h2 className="text-[14px] font-semibold">{t('web.notes.empty.title')}</h2>
        <p className="text-[12px] text-muted-foreground max-w-[420px]">
          <Trans
            i18nKey="web.notes.empty.hint"
            components={{ 1: <code />, 3: <code /> }}
          />
        </p>
      </div>
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={onDaily}
          className="inline-flex items-center gap-1.5 text-[12px] px-3 py-1.5 rounded-md hover:bg-card border border-border text-foreground"
        >
          <Calendar className="size-3.5" />
          {t('web.notes.empty.today')}
        </button>
        <button
          type="button"
          onClick={onNew}
          className="inline-flex items-center gap-1.5 text-[12px] px-3 py-1.5 rounded-md bg-accent text-accent-foreground hover:bg-accent/90"
        >
          <Plus className="size-3.5" />
          {t('web.notes.empty.new')}
        </button>
      </div>
    </div>
  )
}

function pathTitle(p: string): string {
  const base = p.split('/').pop() ?? p
  return base.replace(/\.md$/i, '')
}
