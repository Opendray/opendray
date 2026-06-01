import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  FileText,
  Loader2,
  Plus,
  NotebookPen,
  FileCode,
  Sparkles,
  Search,
  Settings2,
  Maximize2,
} from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { toast } from 'sonner'

import { cn } from '@/lib/utils'
import {
  listNotes,
  notesProjectMapping,
  personalNotePath,
  setNotesProjectMapping,
  writeNote,
} from '@/lib/notes'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

import { VaultFolderPicker } from '@/components/notes/VaultFolderPicker'

import { NoteEditor } from './NoteEditor'
import { NoteEditorDialog } from './NoteEditorDialog'

interface NotesPanelProps {
  cwd: string
}

// NotesPanel splits two distinct authoring lanes:
//
//   "My notes"   → personal/<basename>.md
//                 inline editor, human-authored scratchpad
//
//   "Project docs" → projects/<basename>/*.md
//                    list view; AI agents write these via
//                    `opendray notes write projects/<basename>/<file>.md`.
//                    Click a row to open in a wide modal for reading
//                    (preview-first) or editing.
//
// Same vault, separate folder roots — no risk of agent writes
// clobbering the user's personal scratchpad.
export function NotesPanel({ cwd }: NotesPanelProps) {
  const personalPath = useMemo(() => personalNotePath(cwd), [cwd])
  const cwdBase = useMemo(() => cwdBasename(cwd), [cwd])

  // Per-cwd project mapping comes from the backend so it picks up
  // both the configured prefix and any user-pinned override stored
  // in `<vault>/.opendray-projects.json`.
  const { data: mapping } = useQuery({
    queryKey: ['notes-project-mapping', cwd],
    queryFn: () => notesProjectMapping(cwd),
    staleTime: 30_000,
  })
  const docsPrefix = useMemo(
    () => (mapping?.path ? mapping.path + '/' : ''),
    [mapping?.path],
  )

  // Single dialog instance shared by both sections — clicking a wiki-
  // link in My notes or a project doc row pops the same modal.
  const [opening, setOpening] = useState<string | null>(null)

  return (
    <div className="flex flex-col gap-5">
      <PersonalSection
        path={personalPath}
        basename={cwdBase}
        onOpenLink={(p) => setOpening(p)}
        onExpand={() => setOpening(personalPath)}
      />
      {docsPrefix && (
        <ProjectDocsSection
          cwd={cwd}
          prefix={docsPrefix}
          basename={cwdBase}
          mapping={mapping ?? null}
          setOpening={setOpening}
        />
      )}
      {/* Single dialog instance shared by both sections — opens any
          note the user clicks on (project doc row, wiki-link in My
          notes, or the Expand button on the personal scratchpad). */}
      <NoteEditorDialog
        path={opening}
        open={opening != null}
        onOpenChange={(v) => !v && setOpening(null)}
        onDeleted={() => setOpening(null)}
      />
    </div>
  )
}

function PersonalSection({
  path,
  basename,
  onOpenLink,
  onExpand,
}: {
  path: string
  basename: string
  onOpenLink: (path: string) => void
  onExpand: () => void
}) {
  return (
    <section className="flex flex-col gap-2">
      <SectionHeader
        icon={<NotebookPen className="size-3 text-muted-foreground" />}
        title="My notes"
        subtitle={path}
        hint="Personal scratchpad — auto-saves as you type. AI agents do not write here. Use [[wiki-links]] to reference project docs."
        action={
          <button
            type="button"
            onClick={onExpand}
            className="inline-flex items-center gap-1 text-[11px] text-muted-foreground hover:text-foreground"
            title="Open in full-screen editor (preview, backlinks, wider canvas)"
          >
            <Maximize2 className="size-3" />
            Expand
          </button>
        }
      />
      <NoteEditor
        path={path}
        initialMode="source"
        minHeight={220}
        onOpenLink={onOpenLink}
        placeholder={`# ${basename}\n\nThis is your personal scratchpad for ${basename}.\nAuto-saves to ${path}.\n\n## TODO\n- [ ] ...\n`}
      />
    </section>
  )
}

function ProjectDocsSection({
  cwd,
  prefix,
  mapping,
  setOpening,
}: {
  cwd: string
  prefix: string
  basename: string
  mapping: { path: string; default_path: string; custom: boolean } | null
  setOpening: (path: string | null) => void
}) {
  const qc = useQueryClient()
  const [creating, setCreating] = useState(false)
  const [newName, setNewName] = useState('')
  const [filter, setFilter] = useState('')
  const [editingPath, setEditingPath] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: ['notes-list', prefix],
    queryFn: () => listNotes(prefix),
    staleTime: 5_000,
    refetchInterval: 8_000, // pick up agent writes
  })

  const create = useMutation({
    mutationFn: async (name: string) => {
      const path = prefix + sanitiseFilename(name)
      const body = `# ${stripExtension(name)}\n\n`
      await writeNote(path, body)
      return path
    },
    onSuccess: (newPath) => {
      qc.invalidateQueries({ queryKey: ['notes-list', prefix] })
      setCreating(false)
      setNewName('')
      setOpening(newPath)
    },
    onError: (err: Error) =>
      toast.error('Create failed', { description: err.message }),
  })

  const docs = useMemo(() => {
    const all = data ?? []
    const q = filter.trim().toLowerCase()
    if (!q) return all
    return all.filter(
      (d) =>
        d.path.toLowerCase().includes(q) || d.title.toLowerCase().includes(q),
    )
  }, [data, filter])

  return (
    <section className="flex flex-col gap-2">
      <SectionHeader
        icon={<Sparkles className="size-3 text-muted-foreground" />}
        title="Project docs"
        subtitle={prefix.endsWith('/') ? prefix : prefix + '/'}
        hint={
          mapping?.custom
            ? `Pinned to ${prefix} (overrides the auto-derived ${mapping.default_path}/). AI agents authoring docs go here too — click ⚙ to change.`
            : 'Architecture / spec / decisions / plan / retros — typically authored by AI agents. Click ⚙ to point this section at a different vault folder if your Obsidian layout differs.'
        }
        action={
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => setEditingPath(true)}
              className="inline-flex items-center gap-1 text-[11px] text-muted-foreground hover:text-foreground"
              title="Change project docs location"
            >
              <Settings2 className="size-3" />
            </button>
            <button
              type="button"
              onClick={() => setCreating((v) => !v)}
              className="inline-flex items-center gap-1 text-[11px] text-muted-foreground hover:text-foreground"
            >
              <Plus className="size-3" />
              New doc
            </button>
          </div>
        }
      />

      {creating && (
        <form
          onSubmit={(e) => {
            e.preventDefault()
            const n = newName.trim()
            if (!n) return
            create.mutate(n)
          }}
          className="flex items-center gap-1"
        >
          <input
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder="filename (e.g. architecture or spec.md)"
            className={cn(
              'flex-1 h-7 px-2 text-[12px] font-mono rounded-md border border-border',
              'bg-input/40 text-foreground placeholder:text-muted-foreground/60',
              'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
            )}
            autoFocus
          />
          <button
            type="submit"
            disabled={create.isPending || !newName.trim()}
            className={cn(
              'h-7 px-2.5 text-[11px] rounded-md',
              'bg-accent text-accent-foreground hover:bg-accent/90',
              'disabled:opacity-50 disabled:cursor-not-allowed',
            )}
          >
            {create.isPending ? 'Creating…' : 'Create'}
          </button>
          <button
            type="button"
            onClick={() => {
              setCreating(false)
              setNewName('')
            }}
            className="h-7 px-2 text-[11px] text-muted-foreground hover:text-foreground"
          >
            Cancel
          </button>
        </form>
      )}

      {!isLoading && (data?.length ?? 0) > 0 && (
        <div className="relative">
          <Search className="absolute left-2 top-1/2 -translate-y-1/2 size-3 text-muted-foreground/60" />
          <input
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            placeholder="Filter…"
            className={cn(
              'w-full h-7 pl-7 pr-2 text-[11.5px] rounded-md border border-border',
              'bg-input/40 text-foreground placeholder:text-muted-foreground/60',
              'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
            )}
          />
        </div>
      )}

      {isLoading ? (
        <div className="flex items-center gap-2 text-[11px] text-muted-foreground py-2 px-1">
          <Loader2 className="size-3 animate-spin" />
          Loading…
        </div>
      ) : (data?.length ?? 0) === 0 ? (
        <div className="rounded-md border border-dashed border-border bg-card/30 p-3 text-[11px] text-muted-foreground">
          No project docs yet. AI agents can create them via{' '}
          <code className="text-[10.5px]">
            opendray notes write {prefix}&lt;name&gt;.md
          </code>
          , or click "New doc" above.
        </div>
      ) : docs.length === 0 ? (
        <div className="text-[11px] text-muted-foreground/60 px-1 py-2">
          No matches for "{filter}".
        </div>
      ) : (
        <div className="flex flex-col">
          {docs.map((d) => {
            const rel = d.path.slice(prefix.length) // path within the project
            return (
              <button
                key={d.path}
                type="button"
                onClick={() => setOpening(d.path)}
                className={cn(
                  'group flex items-start gap-2 px-2 py-1.5 text-left rounded-md',
                  'hover:bg-card border border-transparent hover:border-border/60',
                )}
                title={d.path}
              >
                <FileCode className="size-3 mt-0.5 text-muted-foreground/60 shrink-0 group-hover:text-foreground" />
                <div className="flex flex-col min-w-0 flex-1">
                  <span className="text-[12px] font-medium truncate">
                    {rel || d.title}
                  </span>
                  <span className="text-[10px] text-muted-foreground/70 font-mono truncate">
                    {formatBytes(d.size)} · {relTime(d.modified)}
                  </span>
                </div>
              </button>
            )
          })}
        </div>
      )}

      <ProjectMappingDialog
        open={editingPath}
        onOpenChange={setEditingPath}
        cwd={cwd}
        currentPath={mapping?.path ?? ''}
        defaultPath={mapping?.default_path ?? ''}
      />
    </section>
  )
}

function SectionHeader({
  icon,
  title,
  subtitle,
  hint,
  action,
}: {
  icon: React.ReactNode
  title: string
  subtitle?: string
  hint?: string
  action?: React.ReactNode
}) {
  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center gap-1.5">
        {icon}
        <span className="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium">
          {title}
        </span>
        {subtitle && (
          <>
            <span className="text-muted-foreground/40 text-[10px]">·</span>
            <span
              className="text-[10px] text-muted-foreground/70 font-mono truncate"
              title={subtitle}
            >
              {subtitle}
            </span>
          </>
        )}
        <div className="flex-1" />
        {action}
      </div>
      {hint && (
        <p className="text-[10.5px] text-muted-foreground/70 leading-snug">
          <FileText className="size-2.5 inline-block mr-1 opacity-60" />
          {hint}
        </p>
      )}
    </div>
  )
}

function cwdBasename(cwd: string): string {
  const parts = cwd.split('/').filter(Boolean)
  return parts[parts.length - 1] || 'project'
}

function sanitiseFilename(input: string): string {
  // Strip leading slashes and drop any `..` / `.` segments so the name
  // can't escape the project folder. Split/filter/join avoids the
  // overlap bypass that a single `.replace(/\.\.\//g)` is vulnerable to
  // (e.g. `....//` collapses to `../` after one pass). Append .md if
  // missing.
  let name = input
    .trim()
    .replace(/^\/+/, '')
    .split('/')
    .filter((seg) => seg !== '' && seg !== '..' && seg !== '.')
    .join('/')
  if (!name.toLowerCase().endsWith('.md')) name = name + '.md'
  return name
}

function stripExtension(name: string): string {
  const i = name.lastIndexOf('.')
  return i > 0 ? name.slice(0, i) : name
}

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KiB`
  return `${(n / (1024 * 1024)).toFixed(2)} MiB`
}

function relTime(iso: string): string {
  try {
    return formatDistanceToNow(new Date(iso), { addSuffix: true })
  } catch {
    return iso
  }
}

// ProjectMappingDialog edits the per-cwd "where do project docs live"
// override stored at <vault>/.opendray-projects.json. Empty input
// clears the override (revert to the configured default).
function ProjectMappingDialog({
  open,
  onOpenChange,
  cwd,
  currentPath,
  defaultPath,
}: {
  open: boolean
  onOpenChange: (v: boolean) => void
  cwd: string
  currentPath: string
  defaultPath: string
}) {
  const qc = useQueryClient()
  const [path, setPath] = useState('')

  // Pre-fill with the current path each time the dialog opens.
  useMemo(() => {
    if (open) setPath(currentPath ?? '')
  }, [open, currentPath])

  const save = useMutation({
    mutationFn: () => setNotesProjectMapping(cwd, path.trim()),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['notes-project-mapping', cwd] })
      qc.invalidateQueries({ queryKey: ['notes-list'] })
      toast.success(
        path.trim() === ''
          ? 'Override cleared — using default'
          : `Project docs pinned to ${path.trim()}`,
      )
      onOpenChange(false)
    },
    onError: (e: Error) =>
      toast.error('Save failed', { description: e.message }),
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Project docs location</DialogTitle>
          <DialogDescription>
            Pin this session's cwd to a specific folder under your vault.
            Leave the field empty to revert to the default
            <code className="ml-1">{defaultPath}/</code>.
          </DialogDescription>
        </DialogHeader>
        <form
          onSubmit={(e) => {
            e.preventDefault()
            save.mutate()
          }}
          className="flex flex-col gap-3"
        >
          <div className="space-y-1.5">
            <Label
              htmlFor="cwd"
              className="text-[10.5px] text-muted-foreground/80"
            >
              Session cwd
            </Label>
            <Input
              id="cwd"
              value={cwd}
              readOnly
              className="font-mono text-[11px] bg-muted/40"
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="path">Vault-relative project docs path</Label>
            <VaultFolderPicker
              inputId="path"
              value={path}
              onChange={setPath}
              placeholder={defaultPath}
            />
            <p className="text-[10.5px] text-muted-foreground/80">
              Type to filter existing folders, ↑/↓ to pick, Enter to select,
              Tab to complete-into. Or save a non-existent path to lazy-
              create on first write. Stored in{' '}
              <code>&lt;vault&gt;/.opendray-projects.json</code> — git-syncs
              with your notes.
            </p>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => onOpenChange(false)}
              disabled={save.isPending}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              variant="accent"
              size="sm"
              disabled={save.isPending}
            >
              {save.isPending && <Loader2 className="size-3.5 animate-spin" />}
              {path.trim() === '' ? 'Clear override' : 'Save'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
