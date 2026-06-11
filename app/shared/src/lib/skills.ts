import { api } from './api'

export interface Skill {
  id: string
  name: string
  description: string
  // "builtin" → embedded in the binary. Editable via Customize → save
  //             writes a vault override at the same id.
  // "vault"   → lives in <vault>/skills/<id>/SKILL.md, fully editable.
  source: 'builtin' | 'vault'
  body?: string
  // True when source=='vault' AND a built-in with the same id exists
  // (this row shadows the embedded skill). UI shows a "Reset" action
  // that deletes the vault entry to fall back to the built-in.
  overrides_builtin?: boolean
  // True when an embedded built-in with the same id exists. UI uses
  // this to offer "Customize" on built-in rows.
  has_builtin?: boolean
}

export async function listSkills(): Promise<Skill[]> {
  const res = await api<{ skills: Skill[] }>('/api/v1/skills')
  return res.skills ?? []
}

export async function getSkill(id: string): Promise<Skill> {
  return api<Skill>(`/api/v1/skills/${id}`)
}

export async function createSkill(id: string, body: string): Promise<Skill> {
  return api<Skill>('/api/v1/skills', { method: 'POST', body: { id, body } })
}

export async function updateSkill(id: string, body: string): Promise<Skill> {
  return api<Skill>(`/api/v1/skills/${id}`, {
    method: 'PUT',
    body: { id, body },
  })
}

export async function deleteSkill(id: string): Promise<void> {
  await api(`/api/v1/skills/${id}`, { method: 'DELETE' })
}

/**
 * Upload a SKILL.md to install it as a vault skill. The server derives
 * the id from the frontmatter `name:` field (slugified) — no separate
 * id prompt. Used by the drag-and-drop affordance on the Plugins page.
 *
 * Rejects on:
 *   - missing/blank `name:` in the frontmatter
 *   - id collision with an existing vault skill (409 — delete first)
 *   - empty file or >4 MB body
 */
export async function uploadSkill(file: File): Promise<Skill> {
  const fd = new FormData()
  fd.append('file', file)
  return api<Skill>('/api/v1/skills/upload', { method: 'POST', body: fd })
}
