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

/** How the vault files projects. Mirrors notes.Layout in Go. */
export type VaultLayout = 'flat' | 'nested'

export interface VaultInfo {
  root: string
  /**
   * "flat" files each project at the vault root under its own name,
   * with its personal notes inside it; "nested" uses the prefixes
   * below. Absent on a gateway older than this field — treat that as
   * nested, which is what those gateways do.
   */
  layout?: VaultLayout
  /**
   * True when this vault still files projects under `projects/` AND has
   * something the conversion would move. Drives the offer to convert:
   * a migration nobody is told about is one nobody runs.
   */
  flattenable?: boolean
  /** Nested layout only. */
  personal_prefix?: string
  /** Nested layout only. */
  projects_prefix?: string
}

export interface FlattenMove {
  from: string
  to: string
  /** `[[wiki links]]` repointed at the new path. Zero on a dry run. */
  links_rewritten: number
}

export interface FlattenSkip {
  path: string
  /** Stated in plain terms — surface it, don't summarise it away. */
  reason: string
}

export interface FlattenResult {
  moves: FlattenMove[] | null
  skips: FlattenSkip[] | null
  mappings_rewritten: number
  dry_run: boolean
}

/**
 * Preview or perform the nested → flat conversion.
 *
 * `apply` defaults to false and must be passed explicitly to move
 * anything: this renames every project document in the vault, so
 * looking and rewriting must not be one field apart.
 */
export async function flattenVault(apply = false): Promise<FlattenResult> {
  return api<FlattenResult>('/api/v1/notes/flatten', {
    method: 'POST',
    body: { apply },
  })
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
  /**
   * Where this cwd's personal scratchpad belongs. Served rather than
   * derived here because it depends on the vault's layout: the flat
   * layout keeps it inside the project directory (so a per-cwd override
   * moves it too), while the nested layout files it in its own tree.
   * Use personalNotePath() only as a fallback before this arrives.
   */
  personal_path?: string
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

export interface NoteTemplate {
  id: string
  name: string
  /** "builtin" or "vault" — the latter is one the operator authored. */
  source: string
  body: string
}

export async function listNoteTemplates(): Promise<NoteTemplate[]> {
  const res = await api<{ templates: NoteTemplate[] }>('/api/v1/notes/templates')
  return res.templates ?? []
}

/**
 * Create a note from a template. Distinct from writeNote: this refuses
 * to overwrite, and the placeholders (title, date…) are rendered
 * server-side so web and mobile can't drift on what a new doc looks
 * like.
 */
export async function newNoteFromTemplate(
  path: string,
  template: string,
): Promise<Note> {
  return api<Note>('/api/v1/notes/new', {
    method: 'POST',
    body: { path, template },
  })
}

/**
 * Filenames treated as a folder's index page, in preference order.
 * Resolved client-side: the caller already holds the folder's listing,
 * so asking the gateway would re-derive what it can already see.
 */
export const INDEX_NAMES = [
  'README.md',
  'index.md',
  '_index.md',
  'index.html',
  'index.htm',
]

/**
 * What kind of document a vault path holds. Mirrors notes.KindOf in
 * internal/notes/doc.go — keep the extension lists in step.
 *
 * Clients derive this from the PATH rather than asking the gateway, and
 * that is deliberate: it means HTML documents never need an endpoint
 * that serves them with `Content-Type: text/html`. Such an endpoint
 * would run the document's scripts on opendray's own origin, so a doc
 * pulled from a git remote could take over the admin session. Rendering
 * happens from an in-memory string in a sandboxed frame instead.
 */
export type DocKind = 'markdown' | 'html' | 'unknown'

const DOC_KINDS: Record<string, DocKind> = {
  md: 'markdown',
  markdown: 'markdown',
  html: 'html',
  htm: 'html',
}

export function docKind(path: string): DocKind {
  const base = path.split('/').pop() ?? ''
  const dot = base.lastIndexOf('.')
  // A leading dot is a dotfile, not an extension — `.md` is not a
  // markdown document with an empty name.
  if (dot <= 0) return 'unknown'
  return DOC_KINDS[base.slice(dot + 1).toLowerCase()] ?? 'unknown'
}

/** Extensions the vault accepts, for "new document" pickers. */
export const DOC_EXTENSIONS = ['.md', '.html'] as const

/** Find `dir`'s index note among paths, or undefined. Paths and dir are
 * relative to the same root. */
export function folderIndexPath(
  dir: string,
  paths: string[],
): string | undefined {
  for (const name of INDEX_NAMES) {
    const candidate = dir ? `${dir}/${name}` : name
    if (paths.includes(candidate)) return candidate
  }
  return undefined
}

export interface MoveResult {
  from: string
  to: string
  /** Notes whose wiki-links were repointed at the new path. */
  rewritten_in: string[]
  /** Individual link occurrences updated; ≥ rewritten_in.length. */
  links_rewritten: number
  note: Note
}

/**
 * Move or rename a note, repointing the [[wiki-links]] that referenced
 * it. The backend may report a warning when the file moved but the link
 * rewrite didn't finish — the move still happened, so surface it rather
 * than treating it as a failure.
 */
export async function moveNote(
  from: string,
  to: string,
): Promise<MoveResult & { warning?: string }> {
  const res = await api<MoveResult | { moved: MoveResult; warning: string }>(
    '/api/v1/notes/move',
    { method: 'POST', body: { from, to } },
  )
  if ('moved' in res) return { ...res.moved, warning: res.warning }
  return res
}

/**
 * Clean a user-typed note path, per segment.
 *
 * Slashes are meaningful — they are how someone files a doc under
 * `features/`. The old single-regex sanitiser replaced every character
 * outside `[A-Za-z0-9_.- ]` with a dash, which silently turned
 * `features/canvas.md` into `features-canvas.md`, making it impossible
 * to create a folder from the UI at all. Each segment is cleaned on its
 * own instead, and empty / dot-only segments are dropped so `..` can
 * never survive into the request.
 */
export function sanitizeNotePath(input: string, defaultExt = '.md'): string {
  const segments = input
    .trim()
    .split('/')
    .map((s) =>
      s
        .trim()
        .replace(/[^A-Za-z0-9_.\- ]/g, '-')
        .replace(/\s+/g, '-')
        .replace(/^\.+/, ''),
    )
    .filter(Boolean)
  if (segments.length === 0) return `untitled${defaultExt}`
  const last = segments[segments.length - 1]
  // Only append an extension when the name doesn't already carry a
  // recognised one, so `guide.html` stays HTML instead of becoming
  // `guide.html.md`.
  segments[segments.length - 1] =
    docKind(last) === 'unknown' ? `${last}${defaultExt}` : last
  return segments.join('/')
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
// One file per cwd, edited inline in the Notes panel. AI agents do NOT
// write here — the convention keeps human and agent authoring lanes
// clean, and under the flat layout that rule is the ONLY thing keeping
// them apart, since the file now sits inside the project's directory.
//
// This returns the NESTED path and is a fallback only: ask the gateway
// via notesProjectMapping(cwd).personal_path, which knows the layout.
export function personalNotePath(cwd: string): string {
  return `personal/${cwdSlug(cwd)}.md`
}

function cwdSlug(cwd: string): string {
  const segments = cwd.split('/').filter(Boolean)
  const base = segments[segments.length - 1] || 'untitled'
  const clean = base.replace(/[^A-Za-z0-9_.\-]/g, '-')
  return clean || 'untitled'
}
