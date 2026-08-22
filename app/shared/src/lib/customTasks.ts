import { api } from './api'

export interface CustomTask {
  id: string
  name: string
  command: string
  description?: string
  // "" = global (visible from any session). Otherwise the absolute
  // path the task is scoped to.
  cwd: string
  created_at: string
  updated_at: string
}

export interface CreateCustomTaskRequest {
  name: string
  command: string
  description?: string
  cwd?: string
}

export interface UpdateCustomTaskRequest {
  name?: string
  command?: string
  description?: string
  cwd?: string
}

// listCustomTasks: pass cwd to get globals + cwd-scoped tasks (used
// by the inspector). Pass all=true with no cwd for the management
// view in the Plugins page.
export async function listCustomTasks(opts: {
  cwd?: string
  all?: boolean
}): Promise<CustomTask[]> {
  const params = new URLSearchParams()
  if (opts.cwd) params.set('cwd', opts.cwd)
  if (opts.all) params.set('all', '1')
  const qs = params.toString()
  const url = qs ? `/api/v1/custom-tasks?${qs}` : '/api/v1/custom-tasks'
  const res = await api<{ tasks: CustomTask[] }>(url)
  return res.tasks ?? []
}

export interface CustomTaskGroup {
  // "" for the global bucket, otherwise the project's absolute cwd.
  cwd: string
  // Basename of cwd — the project name to show in the header. Empty
  // for the global bucket; callers supply their own translated label.
  label: string
  tasks: CustomTask[]
}

// projectLabel reduces an absolute cwd to the folder name operators
// actually recognise ("/Users/me/code/backend" → "backend").
export function projectLabel(cwd: string): string {
  const parts = cwd.split('/').filter(Boolean)
  return parts.length > 0 ? parts[parts.length - 1] : cwd
}

// groupCustomTasksByProject buckets tasks by cwd so the management
// views render one block per project instead of one flat list. The
// global bucket (cwd="") comes first when present; projects follow
// sorted by label, then by full path so two projects sharing a
// basename keep a stable order. Tasks inside a group are sorted by
// name. Input is never mutated.
export function groupCustomTasksByProject(
  tasks: CustomTask[],
): CustomTaskGroup[] {
  const byCwd = new Map<string, CustomTask[]>()
  for (const t of tasks) {
    const key = t.cwd ?? ''
    byCwd.set(key, [...(byCwd.get(key) ?? []), t])
  }
  const groups: CustomTaskGroup[] = [...byCwd.entries()].map(
    ([cwd, list]) => ({
      cwd,
      label: cwd ? projectLabel(cwd) : '',
      tasks: [...list].sort((a, b) =>
        a.name.toLowerCase().localeCompare(b.name.toLowerCase()),
      ),
    }),
  )
  return groups.sort((a, b) => {
    if (a.cwd === '') return -1
    if (b.cwd === '') return 1
    return (
      a.label.toLowerCase().localeCompare(b.label.toLowerCase()) ||
      a.cwd.localeCompare(b.cwd)
    )
  })
}

export async function createCustomTask(
  req: CreateCustomTaskRequest,
): Promise<CustomTask> {
  return api<CustomTask>('/api/v1/custom-tasks', { method: 'POST', body: req })
}

export async function updateCustomTask(
  id: string,
  req: UpdateCustomTaskRequest,
): Promise<CustomTask> {
  return api<CustomTask>(`/api/v1/custom-tasks/${id}`, {
    method: 'PUT',
    body: req,
  })
}

export async function deleteCustomTask(id: string): Promise<void> {
  await api(`/api/v1/custom-tasks/${id}`, { method: 'DELETE' })
}
