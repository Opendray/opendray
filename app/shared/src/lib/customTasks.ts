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
