import { api, APIError } from './api'

export interface Note {
  path: string
  title: string
  modified: string
  size: number
}

export interface FullNote extends Note {
  body: string
}

export interface VaultInfo {
  root: string
  personal_prefix?: string
  projects_prefix?: string
}

export async function notesInfo(): Promise<VaultInfo> {
  return api<VaultInfo>('/api/v1/notes/info')
}

// Per-cwd project mapping override.
export interface ProjectMappingResolved {
  cwd: string
  path: string         // resolved path (override OR default)
  default_path: string // what auto-derivation would produce
  custom: boolean      // true when path != default_path
}

export interface ProjectMapping {
  cwd: string
  path: string
}

export async function notesProjectMapping(
  cwd: string,
): Promise<ProjectMappingResolved> {
  return api<ProjectMappingResolved>(
    `/api/v1/notes/project-mapping?cwd=${encodeURIComponent(cwd)}`,
  )
}

export async function setNotesProjectMapping(
  cwd: string,
  path: string,
): Promise<void> {
  await api('/api/v1/notes/project-mapping', {
    method: 'PUT',
    body: { cwd, path },
  })
}

export async function listNotesProjectMappings(): Promise<ProjectMapping[]> {
  const res = await api<{ mappings: ProjectMapping[] }>(
    '/api/v1/notes/project-mappings',
  )
  return res.mappings ?? []
}

export async function listNotes(prefix?: string): Promise<Note[]> {
  const qs = prefix
    ? `?prefix=${encodeURIComponent(prefix)}`
    : ''
  const res = await api<{ notes: Note[] }>(`/api/v1/notes/list${qs}`)
  return res.notes ?? []
}

// readNote returns null when the note doesn't exist (404), so callers
// can use this to probe for "is there a project note for this cwd?"
// without throwing.
export async function readNote(path: string): Promise<FullNote | null> {
  try {
    return await api<FullNote>(
      `/api/v1/notes/read?path=${encodeURIComponent(path)}`,
    )
  } catch (e) {
    if (e instanceof APIError && e.status === 404) return null
    throw e
  }
}

export async function writeNote(path: string, body: string): Promise<Note> {
  return api<Note>('/api/v1/notes/write', {
    method: 'PUT',
    body: { path, body },
  })
}

export async function appendNote(path: string, body: string): Promise<Note> {
  return api<Note>('/api/v1/notes/append', {
    method: 'POST',
    body: { path, body },
  })
}

export async function deleteNote(path: string): Promise<void> {
  await api(`/api/v1/notes/delete?path=${encodeURIComponent(path)}`, {
    method: 'DELETE',
  })
}

export interface Backlink {
  path: string
  title: string
  modified: string
  lines: string[]
}

export async function notesBacklinks(path: string): Promise<Backlink[]> {
  const res = await api<{ links: Backlink[] }>(
    `/api/v1/notes/backlinks?path=${encodeURIComponent(path)}`,
  )
  return res.links ?? []
}

export interface TagCount {
  tag: string
  count: number
  notes?: string[]
}

export async function notesTags(prefix?: string): Promise<TagCount[]> {
  const qs = prefix ? `?prefix=${encodeURIComponent(prefix)}` : ''
  const res = await api<{ tags: TagCount[] }>(`/api/v1/notes/tags${qs}`)
  return res.tags ?? []
}

// projectNoteDir is the directory containing AI-written project docs
// (architecture, plan, decisions, …). The Notes panel lists every
// .md file in here as a "project doc"; AI agents are the primary
// writers via `opendray notes write projects/<basename>/<file>.md`.
export function projectNoteDir(cwd: string): string {
  return `projects/${cwdSlug(cwd)}`
}

// projectNotePath is the conventional default project note (README).
// Kept for the AI helper `opendray notes project <basename>` and for
// any callers that want a single canonical entry-point file.
export function projectNotePath(cwd: string): string {
  return projectNoteDir(cwd) + '/README.md'
}

// personalNotePath is the user's personal scratchpad for this project.
// One file per cwd basename, edited inline in the Notes panel. AI
// agents do NOT write here — the convention keeps human and agent
// authoring lanes clean.
export function personalNotePath(cwd: string): string {
  return `personal/${cwdSlug(cwd)}.md`
}

function cwdSlug(cwd: string): string {
  const segments = cwd.split('/').filter(Boolean)
  const base = segments[segments.length - 1] || 'untitled'
  const clean = base.replace(/[^A-Za-z0-9_.\-]/g, '-')
  return clean || 'untitled'
}
