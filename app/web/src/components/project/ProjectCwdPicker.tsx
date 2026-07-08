// ProjectCwdPicker — a reusable working-directory picker: a path input,
// a file-browser button, and a list of known projects. Shared by the
// Cortex project workspace (/cortex/project) and the Database tool
// (/database) so both offer the same "pick a project" affordance.

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { AlertCircle, Folder, FolderSearch } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { FileBrowserDialog } from '@/components/sessions/FileBrowserDialog'
import { listProjects } from '@/lib/projectDocs'

interface ProjectCwdPickerProps {
  onSelect: (cwd: string) => void
  title?: string
  subtitle?: string
}

export function ProjectCwdPicker({
  onSelect,
  title,
  subtitle,
}: ProjectCwdPickerProps) {
  const { t } = useTranslation()
  const [picker, setPicker] = useState('')
  const [browserOpen, setBrowserOpen] = useState(false)

  // Every project opendray knows about (project_docs ∪ session_logs), not
  // just the ones with episodic memory — so the picker isn't limited to
  // the cache.
  const projectsQuery = useQuery({
    queryKey: ['known-projects'],
    queryFn: () => listProjects(),
    staleTime: 30_000,
  })
  const knownCwds = (projectsQuery.data ?? []).map((p) => p.cwd)

  return (
    <div className="mx-auto max-w-2xl space-y-4 p-6">
      <h1 className="text-xl font-semibold">
        {title ?? t('web.project.picker.title')}
      </h1>
      <p className="text-muted-foreground text-sm">
        {subtitle ?? t('web.project.picker.subtitle')}
      </p>
      <div className="flex gap-2">
        <Input
          placeholder={t('web.project.picker.pathPlaceholder')}
          value={picker}
          onChange={(e) => setPicker(e.target.value)}
          className="font-mono"
        />
        <Button
          variant="outline"
          onClick={() => setBrowserOpen(true)}
          title={t('web.project.picker.browseTooltip')}
        >
          <FolderSearch className="mr-1 size-3.5" />
          {t('web.project.picker.browse')}
        </Button>
        <Button
          disabled={!picker.trim()}
          onClick={() => onSelect(picker.trim())}
        >
          {t('web.project.picker.open')}
        </Button>
      </div>
      <FileBrowserDialog
        open={browserOpen}
        onOpenChange={setBrowserOpen}
        initialPath={picker.trim() || undefined}
        onSelect={(path) => {
          setPicker(path)
          onSelect(path)
        }}
      />
      {knownCwds.length > 0 && (
        <div className="space-y-1">
          <p className="text-muted-foreground text-xs">
            {t('web.project.picker.recentLabel')}
          </p>
          {sortProjectsValidFirst(knownCwds).map((cwd) => {
            const orphan = isLikelyOrphanScope(cwd)
            return (
              <button
                key={cwd}
                className={`hover:bg-muted/50 flex w-full items-center gap-2 rounded-md p-2 text-left ${
                  orphan ? 'opacity-60' : ''
                }`}
                onClick={() => onSelect(cwd)}
                title={
                  orphan ? t('web.project.picker.orphanTooltip') : undefined
                }
              >
                {orphan ? (
                  <AlertCircle className="h-4 w-4 flex-none text-amber-500" />
                ) : (
                  <Folder className="text-muted-foreground h-4 w-4 flex-none" />
                )}
                <span className="truncate font-mono text-xs">{cwd}</span>
                {orphan && (
                  <span className="text-muted-foreground ml-auto text-[10px]">
                    {t('web.project.picker.orphanBadge')}
                  </span>
                )}
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}

// Heuristic: a real project cwd has at least two non-empty path segments;
// one-segment scope_keys (`/Users/`) are orphan mirror-import data.
function isLikelyOrphanScope(cwd: string): boolean {
  return cwd.split('/').filter((s) => s.length > 0).length < 2
}

function sortProjectsValidFirst(cwds: string[]): string[] {
  return [...cwds].sort((a, b) => {
    const ao = isLikelyOrphanScope(a)
    const bo = isLikelyOrphanScope(b)
    if (ao && !bo) return 1
    if (!ao && bo) return -1
    return a.localeCompare(b)
  })
}
