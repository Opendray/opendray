// ProjectWorkspace — a cwd picker that opens the Cortex ProjectScreen
// for one project. Backs the /cortex/project route: pick (or browse to)
// a working directory, then render that project's unified Cortex
// workspace (doc + journal + inbox + memory hygiene).

import { useSearch, useNavigate } from '@tanstack/react-router'

import { ProjectScreen } from '@/components/project/ProjectScreen'
import { ProjectCwdPicker } from '@/components/project/ProjectCwdPicker'

function ProjectWorkspace() {
  const search = useSearch({ strict: false }) as { cwd?: string }
  const navigate = useNavigate()

  if (search.cwd) return <ProjectScreen cwd={search.cwd} />

  return (
    <ProjectCwdPicker
      onSelect={(cwd) => navigate({ to: '/cortex/project', search: { cwd } })}
    />
  )
}

// NotesPage — the Cortex project workspace, mounted at /cortex/project.
export function NotesPage() {
  return <ProjectWorkspace />
}
