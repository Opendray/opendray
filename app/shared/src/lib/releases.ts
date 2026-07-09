// Release "what's new" feed for the admin sidebar Updates drawer.
//
// Source of truth order:
//   1. GitHub Releases (latest) — freshest publish date + notes URL
//   2. CHANGELOG.md on main — when the release body is a stub that
//      only points at the changelog (common for this project)
//
// Read state is local-first (`opendray:lastReadRelease` in localStorage).
// A later PR can sync that key per-user via the backend.

const REPO = 'Opendray/opendray'
const RELEASES_LATEST = `https://api.github.com/repos/${REPO}/releases/latest`
const CHANGELOG_RAW = `https://raw.githubusercontent.com/${REPO}/main/CHANGELOG.md`
const CHANGELOG_HTML = `https://github.com/${REPO}/blob/main/CHANGELOG.md`

/** localStorage key for the last release the operator marked as read. */
export const LAST_READ_RELEASE_KEY = 'opendray:lastReadRelease'

export interface ReleaseInfo {
  /** Tag as published, e.g. "v2.11.2". */
  tag: string
  /** Normalised without leading "v", e.g. "2.11.2". */
  version: string
  name: string
  /** ISO publish date when known. */
  publishedAt?: string
  /** GitHub release page (or changelog section fallback). */
  htmlUrl: string
  /** 3–5 short highlight lines for the drawer. */
  highlights: string[]
  /** Where the highlights were parsed from. */
  source: 'github-release' | 'changelog'
}

type GhRelease = {
  tag_name?: string
  name?: string
  body?: string | null
  published_at?: string
  html_url?: string
}

export function normalizeReleaseVersion(v: string | undefined | null): string {
  if (!v) return ''
  let s = v.trim()
  if (s.toLowerCase().startsWith('v')) s = s.slice(1)
  const plus = s.indexOf('+')
  if (plus >= 0) s = s.slice(0, plus)
  return s
}

export function formatReleaseTag(version: string): string {
  const n = normalizeReleaseVersion(version)
  return n ? `v${n}` : ''
}

export function getLastReadRelease(): string | null {
  if (typeof localStorage === 'undefined') return null
  try {
    const raw = localStorage.getItem(LAST_READ_RELEASE_KEY)
    return raw ? normalizeReleaseVersion(raw) : null
  } catch {
    return null
  }
}

export function setLastReadRelease(version: string): void {
  if (typeof localStorage === 'undefined') return
  const n = normalizeReleaseVersion(version)
  if (!n) return
  try {
    localStorage.setItem(LAST_READ_RELEASE_KEY, formatReleaseTag(n))
  } catch {
    /* private mode / quota — ignore */
  }
}

/** True when the latest release differs from the last one the operator read. */
export function isReleaseUnread(latestVersion: string | undefined | null): boolean {
  const latest = normalizeReleaseVersion(latestVersion)
  if (!latest) return false
  const read = getLastReadRelease()
  if (!read) return true
  return read !== latest
}

/**
 * Pull short highlight lines from markdown (release body or CHANGELOG section).
 * Prefers the bold title in `- **Title.** rest` bullets (CHANGELOG style);
 * falls back to the first line of a plain list item. Continuation lines
 * under a bullet are ignored — the drawer wants scannable titles, not
 * full paragraphs.
 */
export function extractHighlights(markdown: string, limit = 5): string[] {
  if (!markdown.trim()) return []
  // Drop "Announce on X" / tweet blocks that ship in many release bodies.
  const cleaned = markdown
    .replace(/##\s*Announce on X[\s\S]*$/i, '')
    .replace(/```[\s\S]*?```/g, '')

  const lines = cleaned.split(/\r?\n/)
  const out: string[] = []
  for (const line of lines) {
    // Only real list starters (not wrapped continuation indented with spaces).
    const m = line.match(/^\s*[-*+]\s+(.+)$/)
    if (!m) continue
    const item = m[1]
    const bold = item.match(/^\*\*(.+?)\*\*/)
    let text = bold ? bold[1].trim() : item.trim()
    text = text
      .replace(/\*\*/g, '')
      .replace(/`([^`]+)`/g, '$1')
      .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
      .replace(/\s+/g, ' ')
      .replace(/[.:]+$/g, '')
      .trim()
    if (!text || text.length < 4) continue
    // Cap length so the drawer stays scannable.
    if (text.length > 120) text = text.slice(0, 117).trimEnd() + '…'
    out.push(text)
    if (out.length >= limit) break
  }
  return out
}

/** Extract the section under `## [vX.Y.Z]` (or unbracketed) from Keep-a-Changelog. */
export function extractChangelogSection(changelog: string, version: string): string {
  const v = normalizeReleaseVersion(version)
  if (!v || !changelog) return ''
  // Match ## [v2.11.2], ## [2.11.2], ## v2.11.2, with optional date suffix.
  const re = new RegExp(
    `^##\\s*\\[?v?${escapeRegExp(v)}\\]?\\s*(?:—|-|–).*$`,
    'im',
  )
  const reAlt = new RegExp(`^##\\s*\\[?v?${escapeRegExp(v)}\\]?\\s*$`, 'im')
  const startMatch = changelog.match(re) ?? changelog.match(reAlt)
  if (!startMatch || startMatch.index === undefined) return ''
  const from = startMatch.index + startMatch[0].length
  const rest = changelog.slice(from)
  const next = rest.search(/^##\s+/m)
  return next >= 0 ? rest.slice(0, next) : rest
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function isStubBody(body: string): boolean {
  const t = body.trim()
  if (!t) return true
  // Typical stub: "See CHANGELOG.md for the full release notes." + announce block.
  const withoutAnnounce = t.replace(/##\s*Announce on X[\s\S]*$/i, '').trim()
  if (withoutAnnounce.length < 80) return true
  if (/see\s+\[?changelog/i.test(withoutAnnounce) && extractHighlights(withoutAnnounce).length === 0) {
    return true
  }
  return false
}

async function fetchText(url: string): Promise<string> {
  const resp = await fetch(url, {
    headers: { Accept: 'text/plain, text/markdown, */*' },
  })
  if (!resp.ok) throw new Error(`fetch ${url}: ${resp.status}`)
  return resp.text()
}

/**
 * Load the latest release highlights for the Updates drawer.
 * Prefer GitHub Releases; fall back to CHANGELOG.md for bullet text.
 */
export async function fetchLatestReleaseInfo(): Promise<ReleaseInfo> {
  const resp = await fetch(RELEASES_LATEST, {
    headers: {
      Accept: 'application/vnd.github+json',
    },
  })
  if (!resp.ok) {
    // Full fallback: no releases API — try changelog head only.
    return fetchFromChangelogOnly()
  }
  const rel = (await resp.json()) as GhRelease
  const tag = rel.tag_name?.trim() || ''
  const version = normalizeReleaseVersion(tag)
  if (!version) throw new Error('github release missing tag_name')

  const body = rel.body ?? ''
  let highlights = extractHighlights(body)
  let source: ReleaseInfo['source'] = 'github-release'

  if (isStubBody(body) || highlights.length === 0) {
    try {
      const changelog = await fetchText(CHANGELOG_RAW)
      const section = extractChangelogSection(changelog, version)
      const fromLog = extractHighlights(section)
      if (fromLog.length > 0) {
        highlights = fromLog
        source = 'changelog'
      }
    } catch {
      /* keep whatever we got from the release body */
    }
  }

  return {
    tag: tag.startsWith('v') ? tag : `v${version}`,
    version,
    name: rel.name?.trim() || tag || `v${version}`,
    publishedAt: rel.published_at,
    htmlUrl: rel.html_url || `https://github.com/${REPO}/releases/tag/v${version}`,
    highlights,
    source,
  }
}

async function fetchFromChangelogOnly(): Promise<ReleaseInfo> {
  const changelog = await fetchText(CHANGELOG_RAW)
  // First version heading after Unreleased.
  const m = changelog.match(/^##\s*\[?(v?\d+\.\d+\.\d+)\]?\s*(?:—|-|–)\s*(\d{4}-\d{2}-\d{2})?/m)
  if (!m) throw new Error('changelog: no version heading found')
  const tagRaw = m[1]
  const version = normalizeReleaseVersion(tagRaw)
  const section = extractChangelogSection(changelog, version)
  const highlights = extractHighlights(section)
  return {
    tag: formatReleaseTag(version),
    version,
    name: formatReleaseTag(version),
    publishedAt: m[2] ? `${m[2]}T00:00:00Z` : undefined,
    htmlUrl: `${CHANGELOG_HTML}#v${version.replace(/\./g, '')}`,
    highlights,
    source: 'changelog',
  }
}


