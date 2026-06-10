// ProjectWorkspace — a cwd picker that opens ProjectScreen for one project.
// Reused by two routes (Experience Flywheel deconflation):
//   /notes          → variant="notes"  (the project's official doc)
//   /memory/project → variant="memory" (the project's memory hygiene)
// The picker navigates within its own route so the two never bleed together.

import { useState } from 'react'
import { useSearch, useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { AlertCircle, Folder, FolderSearch } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { ProjectScreen } from '@/components/project/ProjectScreen'
import { FileBrowserDialog } from '@/components/sessions/FileBrowserDialog'
import { listProjects } from '@/lib/projectDocs'

type RoutePath = '/cortex/project' | '/cortex/memory/project'

function ProjectWorkspace({
  variant,
  routePath,
}: {
  variant: 'notes' | 'memory'
  routePath: RoutePath
}) {
  const { t } = useTranslation()
  const search = useSearch({ strict: false }) as { cwd?: string }
  const navigate = useNavigate()
  const [picker, setPicker] = useState('')
  const [browserOpen, setBrowserOpen] = useState(false)

  // Every project opendray knows about (project_docs ∪ session_logs), not just
  // the ones with episodic memory — so the picker isn't limited to the cache.
  const projectsQuery = useQuery({
    queryKey: ['known-projects'],
    queryFn: () => listProjects(),
    staleTime: 30_000,
  })
  const knownCwds = (projectsQuery.data ?? []).map((p) => p.cwd)

  const open = (cwd: string) => navigate({ to: routePath, search: { cwd } })

  if (search.cwd) return <ProjectScreen cwd={search.cwd} variant={variant} />

  return (
    <div className="mx-auto max-w-2xl space-y-4 p-6">
      <h1 className="text-xl font-semibold">{t('web.project.picker.title')}</h1>
      <p className="text-muted-foreground text-sm">
        {t('web.project.picker.subtitle')}
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
        <Button disabled={!picker.trim()} onClick={() => open(picker.trim())}>
          {t('web.project.picker.open')}
        </Button>
      </div>
      <FileBrowserDialog
        open={browserOpen}
        onOpenChange={setBrowserOpen}
        initialPath={picker.trim() || undefined}
        onSelect={(path) => {
          setPicker(path)
          open(path)
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
                onClick={() => open(cwd)}
                title={orphan ? t('web.project.picker.orphanTooltip') : undefined}
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

// NotesPage — the Cortex project workspace (flywheel rung ②: the
// project's official doc, blueprint-shaped).
export function NotesPage() {
  return <ProjectWorkspace variant="notes" routePath="/cortex/project" />
}

// ProjectMemoryPage — the project's memory hygiene (flywheel rung ①, per cwd).
export function ProjectPage() {
  return <ProjectWorkspace variant="memory" routePath="/cortex/memory/project" />
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
