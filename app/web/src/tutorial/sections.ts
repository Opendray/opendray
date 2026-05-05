// Section registry for the Tutorial page.
//
// File layout:
//
//   sections/
//     01-getting-started/
//       01-welcome.md
//     02-sessions/
//       01-workbench.md
//     03-channels/
//       01-overview.md
//       02-telegram.md
//       …
//     04-providers/
//       01-providers.md
//
// Numeric prefixes set ordering at both directory (group) and file
// (section) level. Each directory becomes one TOC group; the
// directory's slug (sans prefix) is humanised for the group label.
//
// To add a new section:
//   1. Drop NN-name.md into the right group directory.
//   2. Vite's import.meta.glob (eager raw) picks it up on next reload
//      — no code changes needed.
//
// To add a new group:
//   1. Create a directory like 10-myfeature/.
//   2. Drop one or more .md files inside.

export interface TutorialSection {
	id: string
	title: string
	body: string
}

export interface TutorialGroup {
	id: string    // dir slug, used for stable React keys
	label: string // humanised group title shown in the TOC
	sections: TutorialSection[]
}

// Vite eager glob — every nested `*.md` under sections/ is loaded as
// raw text at build time so we keep one bundle and zero runtime
// fetches for tutorial content.
const rawModules = import.meta.glob('./sections/**/*.md', {
	query: '?raw',
	import: 'default',
	eager: true,
}) as Record<string, string>

interface ParsedFile {
	groupRank: number
	groupSlug: string
	sectionRank: number
	sectionSlug: string
	title: string
	body: string
}

// slugify keeps CJK characters so anchors in 中文 sections stay
// readable in the URL bar.
function slugify(s: string): string {
	return s
		.toLowerCase()
		.replace(/[^a-z0-9一-鿿㐀-䶿]+/g, '-')
		.replace(/^-+|-+$/g, '')
}

// humaniseSlug turns "channel-telegram" → "Channel Telegram" and
// "getting-started" → "Getting Started" for default group labels.
function humaniseSlug(slug: string): string {
	return slug
		.split('-')
		.filter((p) => p.length > 0)
		.map((p) => p.charAt(0).toUpperCase() + p.slice(1))
		.join(' ')
}

function splitNumericPrefix(name: string): { rank: number; rest: string } {
	const m = name.match(/^(\d+)-(.+)$/)
	if (m) return { rank: parseInt(m[1], 10), rest: m[2] }
	return { rank: 9999, rest: name }
}

function parseFile(filePath: string, body: string): ParsedFile | null {
	// filePath like "./sections/03-channels/02-telegram.md".
	const parts = filePath.replace(/^\.\/sections\//, '').split('/')
	if (parts.length < 2) return null

	const dir = parts[0].replace(/\/$/, '')
	const file = parts[parts.length - 1].replace(/\.md$/i, '')

	const { rank: groupRank, rest: groupSlug } = splitNumericPrefix(dir)
	const { rank: sectionRank, rest: sectionSlug } = splitNumericPrefix(file)

	const h1Match = body.match(/^#\s+(.+?)\s*$/m)
	const title = h1Match?.[1].trim() ?? sectionSlug
	if (!title) return null

	return {
		groupRank,
		groupSlug,
		sectionRank,
		sectionSlug,
		title,
		body: body.trim(),
	}
}

// Operator-friendly group labels. Falls back to humaniseSlug when
// the slug isn't in the map — so adding a new directory still gets
// a sensible (if blunt) label without code changes.
const GROUP_LABELS: Record<string, string> = {
	'getting-started': 'Getting Started',
	sessions: 'Sessions',
	channels: 'Channels',
	providers: 'Providers',
	integrations: 'Integrations',
	activity: 'Activity',
	notes: 'Notes',
	plugins: 'Plugins',
	settings: 'Settings',
	consuming: 'Consuming opendray',
	memory: 'Memory',
	backup: 'Backups',
	'ambient-memory': 'Memory · Ambient',
}

export const groups: TutorialGroup[] = (() => {
	const parsed = Object.entries(rawModules)
		.map(([file, body]) => parseFile(file, body))
		.filter((p): p is ParsedFile => p !== null)
		.sort((a, b) => {
			if (a.groupRank !== b.groupRank) return a.groupRank - b.groupRank
			return a.sectionRank - b.sectionRank
		})

	// Bucket by group slug, preserving insertion order (since we
	// already sorted by groupRank then sectionRank).
	const buckets: Record<string, TutorialGroup> = {}
	const groupOrder: string[] = []
	for (const p of parsed) {
		if (!buckets[p.groupSlug]) {
			buckets[p.groupSlug] = {
				id: p.groupSlug,
				label: GROUP_LABELS[p.groupSlug] ?? humaniseSlug(p.groupSlug),
				sections: [],
			}
			groupOrder.push(p.groupSlug)
		}
		buckets[p.groupSlug].sections.push({
			// Section id combines group + slug so anchors stay unique
			// even if two groups have e.g. "01-overview".
			id: slugify(`${p.groupSlug}-${p.sectionSlug}`),
			title: p.title,
			body: p.body,
		})
	}

	return groupOrder.map((slug) => buckets[slug])
})()

// sections is the flat ordered list — handy for scroll-spy + the
// content area, where rendering by group adds nothing.
export const sections: TutorialSection[] = groups.flatMap((g) => g.sections)
