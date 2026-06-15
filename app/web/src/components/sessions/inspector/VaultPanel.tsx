import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  ArrowUpRight,
  FileText,
  Loader2,
  Maximize2,
  NotebookPen,
  Plus,
  Sparkles,
  SlidersHorizontal,
  X,
} from 'lucide-react'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

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
  listNotes,
  notesProjectMapping,
  personalNotePath,
  setNotesProjectMapping,
  writeNote,
  type Note,
} from '@/lib/notes'
import { VaultFolderPicker } from '@/components/notes/VaultFolderPicker'

import { NoteEditor } from './NoteEditor'
import { NoteEditorDialog } from './NoteEditorDialog'

interface VaultPanelProps {
  cwd: string
}

// VaultPanel — the session's window into the markdown Vault (the
// Obsidian-sync notes utility, demoted out of the core Memory →
// Notes → Knowledge triad). Two authoring lanes against the same vault:
//
//   "My notes"     → personal/<slug>.md — the human's scratchpad. AI
//                    agents never write here. Inline editor.
//   "Project docs" → the project's bound vault folder (default
//                    projects/<slug>, or a pinned override). Agent-
//                    authored .md files; the operator can re-bind the
//                    folder so a project that keeps its notes elsewhere
//                    in the vault stays connected.
//
// The project's official doc / goal / plan / journal / memory hygiene
// live in Cortex (the sibling tab), not the Vault.
export function VaultPanel({ cwd }: VaultPanelProps) {
  const { t } = useTranslation()
  const personalPath = useMemo(() => personalNotePath(cwd), [cwd])
  const cwdBase = useMemo(() => cwdBasename(cwd), [cwd])
  const [opening, setOpening] = useState<string | null>(null)

  return (
    <div className="flex flex-col gap-5">
      <PersonalSection
        path={personalPath}
        basename={cwdBase}
        openVaultLabel={t('web.sessions.inspector.vaultPanel.open')}
        onOpenLink={(p) => setOpening(p)}
        onExpand={() => setOpening(personalPath)}
      />
      <ProjectDocsSection cwd={cwd} onOpenDoc={(p) => setOpening(p)} />
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
  openVaultLabel,
  onOpenLink,
  onExpand,
}: {
  path: string
  basename: string
  openVaultLabel: string
  onOpenLink: (path: string) => void
  onExpand: () => void
}) {
  return (
    <section className="flex flex-col gap-2">
      <SectionHeader
        icon={<NotebookPen className="size-3 text-muted-foreground" />}
        title="My notes"
        subtitle={path}
        hint="Personal scratchpad — auto-saves as you type. AI agents do not write here. Use [[wiki-links]] to reference vault notes."
        action={
          <div className="flex items-center gap-3">
            <Link
              to="/vault"
              className="inline-flex items-center gap-1 text-[11px] text-muted-foreground hover:text-foreground"
              title="Open the full Vault browser (tree, tags, sync)"
            >
              <ArrowUpRight className="size-3" />
              {openVaultLabel}
            </Link>
            <button
              type="button"
              onClick={onExpand}
              className="inline-flex items-center gap-1 text-[11px] text-muted-foreground hover:text-foreground"
              title="Open in full-screen editor (preview, backlinks, wider canvas)"
            >
              <Maximize2 className="size-3" />
              Expand
            </button>
          </div>
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

// ProjectDocsSection — the project's vault-folder binding. Mirrors the
// mobile NotesTab project-docs lane. The bound folder is whatever the
// backend resolves for this cwd (notesProjectMapping): the auto-derived
// projects/<slug> by default, or a pinned override. mapping.path is
// already vault-relative — used as the listing prefix directly.
function ProjectDocsSection({
  cwd,
  onOpenDoc,
}: {
  cwd: string
  onOpenDoc: (path: string) => void
}) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [mappingOpen, setMappingOpen] = useState(false)
  const [creating, setCreating] = useState(false)
  const [newName, setNewName] = useState('')

  const mappingQ = useQuery({
    queryKey: ['notes-project-mapping', cwd],
    queryFn: () => notesProjectMapping(cwd),
    enabled: !!cwd,
    staleTime: 30_000,
  })
  const prefix = (mappingQ.data?.path ?? '').replace(/^\/+|\/+$/g, '')

  const docsQ = useQuery({
    queryKey: ['notes-list', prefix],
    queryFn: () => listNotes(prefix),
    enabled: !!prefix,
    staleTime: 30_000,
  })
  const docs = useMemo(
    () =>
      (docsQ.data ?? [])
        // Match folder CHILDREN only — a bare startsWith(prefix) would leak a
        // sibling like projects/app-old/ into a project bound to projects/app.
        .filter((n) =>
          prefix ? n.path.startsWith(`${prefix}/`) || n.path === prefix : true,
        )
        .sort((a, b) => b.modified.localeCompare(a.modified)),
    [docsQ.data, prefix],
  )

  const create = useMutation({
    mutationFn: (filename: string) => {
      const name = sanitizeFilename(filename)
      const dir = prefix.endsWith('/') ? prefix : `${prefix}/`
      const path = `${dir}${name}`
      return writeNote(path, `# ${stripExt(name)}\n\n`).then(() => path)
    },
    onSuccess: (path) => {
      setNewName('')
      setCreating(false)
      qc.invalidateQueries({ queryKey: ['notes-list', prefix] })
      qc.invalidateQueries({ queryKey: ['notes-list'] })
      onOpenDoc(path)
    },
    onError: (e: Error) =>
      toast.error(t('web.sessions.inspector.vaultPanel.createFailed'), {
        description: e.message,
      }),
  })

  const custom = mappingQ.data?.custom ?? false

  return (
    <section className="flex flex-col gap-2">
      <SectionHeader
        icon={<Sparkles className="size-3 text-muted-foreground" />}
        title={t('web.sessions.inspector.vaultPanel.projectDocs')}
        subtitle={prefix ? `${prefix}/` : undefined}
        hint={
          custom
            ? t('web.sessions.inspector.vaultPanel.pinnedHint')
            : t('web.sessions.inspector.vaultPanel.projectDocsHint')
        }
        action={
          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={() => setMappingOpen(true)}
              className="inline-flex items-center gap-1 text-[11px] text-muted-foreground hover:text-foreground"
              title={t('web.sessions.inspector.vaultPanel.changeLocation')}
            >
              <SlidersHorizontal className="size-3" />
              {t('web.sessions.inspector.vaultPanel.bind')}
            </button>
            <button
              type="button"
              onClick={() => {
                setCreating((v) => !v)
                if (creating) setNewName('')
              }}
              className="inline-flex items-center gap-1 text-[11px] text-muted-foreground hover:text-foreground"
              title={t('web.sessions.inspector.vaultPanel.newDoc')}
            >
              {creating ? (
                <X className="size-3" />
              ) : (
                <Plus className="size-3" />
              )}
              {creating
                ? t('web.sessions.inspector.vaultPanel.cancel')
                : t('web.sessions.inspector.vaultPanel.newDoc')}
            </button>
          </div>
        }
      />

      {creating && (
        <div className="flex gap-1.5">
          <Input
            value={newName}
            autoFocus
            onChange={(e) => setNewName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && newName.trim()) create.mutate(newName.trim())
            }}
            placeholder={t('web.sessions.inspector.vaultPanel.filenamePlaceholder')}
            className="h-7 font-mono text-[11px]"
          />
          <Button
            size="sm"
            className="h-7"
            disabled={!newName.trim() || create.isPending}
            onClick={() => create.mutate(newName.trim())}
          >
            {create.isPending ? (
              <Loader2 className="size-3 animate-spin" />
            ) : (
              t('web.sessions.inspector.vaultPanel.create')
            )}
          </Button>
        </div>
      )}

      {docsQ.isLoading || mappingQ.isLoading ? (
        <div className="flex items-center gap-2 px-1 py-2 text-[11px] text-muted-foreground">
          <Loader2 className="size-3 animate-spin" />…
        </div>
      ) : docs.length === 0 ? (
        <p className="px-1 py-1 text-[11px] text-muted-foreground/70">
          {t('web.sessions.inspector.vaultPanel.noDocs')}
        </p>
      ) : (
        <div className="flex flex-col">
          {docs.map((d) => (
            <DocRow key={d.path} note={d} prefix={prefix} onOpen={() => onOpenDoc(d.path)} />
          ))}
        </div>
      )}

      {mappingOpen && (
        <MappingDialog
          cwd={cwd}
          onClose={() => setMappingOpen(false)}
          currentPath={mappingQ.data?.path ?? ''}
          defaultPath={mappingQ.data?.default_path ?? ''}
          onSaved={() => {
            qc.invalidateQueries({ queryKey: ['notes-project-mapping', cwd] })
            qc.invalidateQueries({ queryKey: ['notes-list'] })
          }}
        />
      )}
    </section>
  )
}

function DocRow({
  note,
  prefix,
  onOpen,
}: {
  note: Note
  prefix: string
  onOpen: () => void
}) {
  const rel = prefix && note.path.startsWith(`${prefix}/`)
    ? note.path.slice(prefix.length + 1)
    : note.path
  return (
    <button
      type="button"
      onClick={onOpen}
      className="group flex items-start gap-2 rounded-md border border-transparent px-2 py-1.5 text-left hover:bg-card hover:border-border/60"
      title={note.path}
    >
      <FileText className="text-muted-foreground/60 group-hover:text-foreground mt-0.5 size-3 shrink-0" />
      <div className="flex min-w-0 flex-1 flex-col">
        <span className="truncate text-[12px] font-medium">
          {note.title || rel}
        </span>
        <span className="text-muted-foreground/70 truncate font-mono text-[10px]">
          {rel}
        </span>
      </div>
    </button>
  )
}

// MappingDialog — bind/re-point this project's vault folder. A vault
// folder combobox (VaultFolderPicker) seeded with the current path;
// empty clears the override back to the auto-derived default.
// Mounted only while open (the caller gates it), so useState seeds the
// field from the current mapping on every open — no re-seed effect needed.
function MappingDialog({
  cwd,
  onClose,
  currentPath,
  defaultPath,
  onSaved,
}: {
  cwd: string
  onClose: () => void
  currentPath: string
  defaultPath: string
  onSaved: () => void
}) {
  const { t } = useTranslation()
  const [draft, setDraft] = useState(currentPath)

  const save = useMutation({
    mutationFn: (path: string) => setNotesProjectMapping(cwd, path.trim()),
    onSuccess: (_d, path) => {
      onClose()
      onSaved()
      toast.success(
        path.trim()
          ? t('web.sessions.inspector.vaultPanel.boundToast')
          : t('web.sessions.inspector.vaultPanel.clearedToast'),
      )
    },
    onError: (e: Error) =>
      toast.error(t('web.sessions.inspector.vaultPanel.saveFailed'), {
        description: e.message,
      }),
  })

  return (
    <Dialog open onOpenChange={(v) => !v && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {t('web.sessions.inspector.vaultPanel.mappingTitle')}
          </DialogTitle>
          <DialogDescription>
            {t('web.sessions.inspector.vaultPanel.mappingHelp')}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-3">
          <div>
            <div className="text-muted-foreground mb-1 text-[10px] uppercase tracking-wider">
              {t('web.sessions.inspector.vaultPanel.sessionCwd')}
            </div>
            <div className="bg-muted/30 rounded px-2 py-1 font-mono text-[11px] break-all">
              {cwd}
            </div>
          </div>
          <div>
            <div className="text-muted-foreground mb-1 text-[10px] uppercase tracking-wider">
              {t('web.sessions.inspector.vaultPanel.folderLabel')}
            </div>
            <VaultFolderPicker
              value={draft}
              onChange={setDraft}
              placeholder={defaultPath}
            />
            <p className="text-muted-foreground mt-1 text-[10.5px]">
              {t('web.sessions.inspector.vaultPanel.mappingStoredHint')}
            </p>
          </div>
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={onClose}>
            {t('web.sessions.inspector.vaultPanel.cancel')}
          </Button>
          <Button disabled={save.isPending} onClick={() => save.mutate(draft)}>
            {save.isPending ? (
              <Loader2 className="mr-1 size-3 animate-spin" />
            ) : null}
            {draft.trim()
              ? t('web.sessions.inspector.vaultPanel.save')
              : t('web.sessions.inspector.vaultPanel.clearOverride')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
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

function sanitizeFilename(input: string): string {
  const cleaned = input.trim().replace(/[^A-Za-z0-9_.\- ]/g, '-').replace(/\s+/g, '-')
  const safe = cleaned.replace(/^\.+/, '') || 'untitled'
  return safe.toLowerCase().endsWith('.md') ? safe : `${safe}.md`
}

function stripExt(name: string): string {
  return name.toLowerCase().endsWith('.md') ? name.slice(0, -3) : name
}
