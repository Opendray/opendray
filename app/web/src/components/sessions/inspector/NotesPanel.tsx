import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  ArrowUpRight,
  FileText,
  Loader2,
  Lock,
  NotebookPen,
  Sparkles,
  Maximize2,
} from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'
import { personalNotePath } from '@/lib/notes'
import {
  listBlueprintSections,
  listProjectDocs,
  type ProjectDoc,
} from '@/lib/projectDocs'

import { NoteEditor } from './NoteEditor'
import { NoteEditorDialog } from './NoteEditorDialog'

interface NotesPanelProps {
  cwd: string
}

// NotesPanel splits two distinct lanes:
//
//   "My notes"     → personal/<basename>.md in the vault —
//                    the human's scratchpad, inline editor, AI never
//                    writes here.
//
//   "Project doc"  → the project's OFFICIAL document (Cortex Notes
//                    rung): a read-only list of the blueprint's
//                    sections with freshness + lock state, deep-
//                    linking into the Cortex project workspace. The
//                    old vault-backed "project docs" lane is gone —
//                    one official doc system, not two.
export function NotesPanel({ cwd }: NotesPanelProps) {
  const personalPath = useMemo(() => personalNotePath(cwd), [cwd])
  const cwdBase = useMemo(() => cwdBasename(cwd), [cwd])
  const [opening, setOpening] = useState<string | null>(null)

  return (
    <div className="flex flex-col gap-5">
      <PersonalSection
        path={personalPath}
        basename={cwdBase}
        onOpenLink={(p) => setOpening(p)}
        onExpand={() => setOpening(personalPath)}
      />
      <CortexDocSection cwd={cwd} />
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
        hint="Personal scratchpad — auto-saves as you type. AI agents do not write here. Use [[wiki-links]] to reference vault notes."
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

// CortexDocSection — read-only view of the project's official doc
// (blueprint sections + freshness + lock), linking into the Cortex
// project workspace where editing / curation chat lives.
function CortexDocSection({ cwd }: { cwd: string }) {
  const { t } = useTranslation()
  const sectionsQuery = useQuery({
    queryKey: ['blueprint', cwd],
    queryFn: () => listBlueprintSections(cwd),
    enabled: !!cwd,
    staleTime: 30_000,
  })
  const docsQuery = useQuery({
    queryKey: ['project-docs', cwd],
    queryFn: () => listProjectDocs(cwd),
    enabled: !!cwd,
    staleTime: 30_000,
  })
  const docsByKind = useMemo(() => {
    const map: Record<string, ProjectDoc | undefined> = {}
    for (const d of docsQuery.data ?? []) map[d.kind] = d
    return map
  }, [docsQuery.data])

  return (
    <section className="flex flex-col gap-2">
      <SectionHeader
        icon={<Sparkles className="size-3 text-muted-foreground" />}
        title={t('web.sessions.notes.cortexDoc.title')}
        hint={t('web.sessions.notes.cortexDoc.hint')}
        action={
          <Link
            to="/cortex/project"
            search={{ cwd }}
            className="inline-flex items-center gap-1 text-[11px] text-muted-foreground hover:text-foreground"
          >
            <ArrowUpRight className="size-3" />
            {t('web.sessions.notes.cortexDoc.open')}
          </Link>
        }
      />
      {sectionsQuery.isLoading ? (
        <div className="flex items-center gap-2 px-1 py-2 text-[11px] text-muted-foreground">
          <Loader2 className="size-3 animate-spin" />
          …
        </div>
      ) : (
        <div className="flex flex-col">
          {(sectionsQuery.data ?? []).map((sec) => {
            const doc = docsByKind[sec.slug]
            const locked = doc?.updated_by === 'operator'
            return (
              <Link
                key={sec.slug}
                to="/cortex/project"
                search={{ cwd }}
                className={cn(
                  'group flex items-start gap-2 rounded-md border border-transparent px-2 py-1.5 text-left',
                  'hover:bg-card hover:border-border/60',
                )}
                title={sec.description}
              >
                <FileText className="text-muted-foreground/60 group-hover:text-foreground mt-0.5 size-3 shrink-0" />
                <div className="flex min-w-0 flex-1 flex-col">
                  <span className="flex items-center gap-1.5 truncate text-[12px] font-medium">
                    {sec.title}
                    {locked && <Lock className="size-2.5 opacity-60" />}
                  </span>
                  <span className="text-muted-foreground/70 truncate font-mono text-[10px]">
                    {doc?.updated_at
                      ? `${doc.updated_by} · ${relTime(doc.updated_at)}`
                      : t('web.sessions.notes.cortexDoc.empty')}
                  </span>
                </div>
              </Link>
            )
          })}
        </div>
      )}
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

function relTime(iso: string): string {
  try {
    return formatDistanceToNow(new Date(iso), { addSuffix: true })
  } catch {
    return iso
  }
}
