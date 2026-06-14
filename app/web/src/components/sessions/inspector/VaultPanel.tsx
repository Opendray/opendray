import { useMemo, useState } from 'react'
import { ArrowUpRight, Maximize2, NotebookPen } from 'lucide-react'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { personalNotePath } from '@/lib/notes'

import { NoteEditor } from './NoteEditor'
import { NoteEditorDialog } from './NoteEditorDialog'

interface VaultPanelProps {
  cwd: string
}

// VaultPanel — the session's window into the markdown Vault (the
// Obsidian-sync notes utility, demoted out of the core Memory →
// Notes → Knowledge triad). It surfaces this project's personal
// scratchpad inline and links out to the full Vault browser.
//
// The project's official doc / goal / plan / journal / memory hygiene
// live in Cortex, not the Vault — they're reached from the sibling
// "Cortex" inspector tab. Keeping the two lanes separate is the whole
// point: AI agents never write the Vault, and the Vault never embeds
// the Cortex workspace.
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
