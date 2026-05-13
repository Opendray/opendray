// /memory/project route — wraps ProjectScreen with a cwd picker
// when no cwd is passed via search param. Mirrors the mobile
// Project screen's "pick a project" prompt + autoload behavior.

import { useEffect, useState } from 'react'
import { useSearch, useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Folder } from 'lucide-react'

import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { ProjectScreen } from '@/components/project/ProjectScreen'
import { listScopeKeys } from '@/lib/memory'

export function ProjectPage() {
  const search = useSearch({ strict: false }) as { cwd?: string }
  const navigate = useNavigate()
  const [picker, setPicker] = useState('')

  const projectsQuery = useQuery({
    queryKey: ['memory-project-scope-keys'],
    queryFn: () => listScopeKeys('project'),
    staleTime: 30_000,
  })

  // Auto-pick the most recent project if none specified and there
  // are projects with stored memory.
  useEffect(() => {
    if (search.cwd) return
    if (!projectsQuery.data || projectsQuery.data.length === 0) return
    void navigate({
      to: '/memory/project',
      search: { cwd: projectsQuery.data[0] },
    })
  }, [search.cwd, projectsQuery.data, navigate])

  if (!search.cwd) {
    return (
      <div className="mx-auto max-w-2xl space-y-4 p-6">
        <h1 className="text-xl font-semibold">Pick a project</h1>
        <p className="text-muted-foreground text-sm">
          Project memory is scoped by working directory. Pick one to manage its
          goal, plan, journal, and cleanup queue.
        </p>
        <div className="flex gap-2">
          <Input
            placeholder="/path/to/your/project"
            value={picker}
            onChange={(e) => setPicker(e.target.value)}
            className="font-mono"
          />
          <Button
            disabled={!picker.trim()}
            onClick={() =>
              navigate({
                to: '/memory/project',
                search: { cwd: picker.trim() },
              })
            }
          >
            Open
          </Button>
        </div>
        {projectsQuery.data && projectsQuery.data.length > 0 && (
          <div className="space-y-1">
            <p className="text-muted-foreground text-xs">
              Recent projects (from stored memory):
            </p>
            {projectsQuery.data.slice(0, 12).map((cwd) => (
              <button
                key={cwd}
                className="hover:bg-muted/50 flex w-full items-center gap-2 rounded-md p-2 text-left"
                onClick={() =>
                  navigate({
                    to: '/memory/project',
                    search: { cwd },
                  })
                }
              >
                <Folder className="text-muted-foreground h-4 w-4 flex-none" />
                <span className="truncate font-mono text-xs">{cwd}</span>
              </button>
            ))}
          </div>
        )}
      </div>
    )
  }

  return <ProjectScreen cwd={search.cwd} />
}
