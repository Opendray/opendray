// /memory/project route — wraps ProjectScreen with a cwd picker
// when no cwd is passed via search param.
//
// Earlier auto-picked the first project from listScopeKeys when
// no cwd was set, but that order is alphabetical (not most-recent)
// and could land on truncated scope_keys like `/Users/` left over
// from old mirror imports. Now: always show the picker, mark
// orphan-looking scope_keys as such, sort real projects first.

import { useState } from 'react'
import { useSearch, useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { AlertCircle, Folder, FolderSearch } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { ProjectScreen } from '@/components/project/ProjectScreen'
import { FileBrowserDialog } from '@/components/sessions/FileBrowserDialog'
import { listScopeKeys } from '@/lib/memory'

export function ProjectPage() {
  const { t } = useTranslation()
  const search = useSearch({ strict: false }) as { cwd?: string }
  const navigate = useNavigate()
  const [picker, setPicker] = useState('')
  const [browserOpen, setBrowserOpen] = useState(false)

  const projectsQuery = useQuery({
    queryKey: ['memory-project-scope-keys'],
    queryFn: () => listScopeKeys('project'),
    staleTime: 30_000,
  })

  if (!search.cwd) {
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
          <Button
            disabled={!picker.trim()}
            onClick={() =>
              navigate({
                to: '/memory/project',
                search: { cwd: picker.trim() },
              })
            }
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
            navigate({
              to: '/memory/project',
              search: { cwd: path },
            })
          }}
        />
        {projectsQuery.data && projectsQuery.data.length > 0 && (
          <div className="space-y-1">
            <p className="text-muted-foreground text-xs">
              {t('web.project.picker.recentLabel')}
            </p>
            {sortProjectsValidFirst(projectsQuery.data).map((cwd) => {
              const orphan = isLikelyOrphanScope(cwd)
              return (
                <button
                  key={cwd}
                  className={`hover:bg-muted/50 flex w-full items-center gap-2 rounded-md p-2 text-left ${
                    orphan ? 'opacity-60' : ''
                  }`}
                  onClick={() =>
                    navigate({
                      to: '/memory/project',
                      search: { cwd },
                    })
                  }
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

  return <ProjectScreen cwd={search.cwd} />
}

// Heuristic: a real opendray project cwd has at least two non-empty
// path segments (`/tmp/foo`, `/home/alice/projects/my-app`).
// One-segment scope_keys like `/Users/` are bug data from old
// mirror imports that truncated the source path; they shouldn't
// be presented as live project navigation targets.
function isLikelyOrphanScope(cwd: string): boolean {
  const parts = cwd.split('/').filter((s) => s.length > 0)
  return parts.length < 2
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
