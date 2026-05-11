// BrandIcon — render an official-mark SVG for any provider or
// messaging-platform brand we integrate with.
//
// Source mix:
//   - simple-icons npm pkg: official monochrome SVG paths + brand
//     hex colour for everything that ships there (Claude, Gemini,
//     Telegram, Discord, WeChat).
//   - inline path map: for brands simple-icons does not carry
//     (OpenAI, Slack, Feishu/Lark, DingTalk, WeCom). Paths are
//     drawn to match each brand's well-known mark closely enough
//     that a glance recognises them; they are not pixel-exact
//     copies of the platform's official asset (use the brand
//     guidelines if you need that).
//
// All icons render at the brand's canonical hex colour by default.
// Pass `tone="muted"` to dim by 70% (for compact lists) or
// `color="..."` to override entirely.

import {
	siClaude,
	siGooglegemini,
	siTelegram,
	siDiscord,
	siWechat,
} from 'simple-icons'

type IconData = {
	title: string
	hex: string
	path: string
	viewBox?: string
}

const SIMPLE: Record<string, IconData> = {
	claude: { title: siClaude.title, hex: siClaude.hex, path: siClaude.path },
	gemini: {
		title: siGooglegemini.title,
		hex: siGooglegemini.hex,
		path: siGooglegemini.path,
	},
	telegram: { title: siTelegram.title, hex: siTelegram.hex, path: siTelegram.path },
	discord: { title: siDiscord.title, hex: siDiscord.hex, path: siDiscord.path },
	wechat: { title: siWechat.title, hex: siWechat.hex, path: siWechat.path },
}

// Inline SVGs for brands missing from simple-icons. Each path uses
// the canonical 24x24 viewBox so the component renders all icons at
// the same metric. Paths are simplified renderings of each brand's
// official mark.
const INLINE: Record<string, IconData> = {
	// OpenAI — six-fold rotational sunburst, simplified.
	openai: {
		title: 'OpenAI',
		hex: '412991',
		path:
			'M22.282 9.821a5.985 5.985 0 0 0-.516-4.91 6.046 6.046 0 0 0-6.51-2.9A6.065 6.065 0 0 0 4.981 4.18a5.985 5.985 0 0 0-3.998 2.9 6.046 6.046 0 0 0 .743 7.097 5.98 5.98 0 0 0 .51 4.911 6.051 6.051 0 0 0 6.515 2.9A5.985 5.985 0 0 0 13.26 24a6.056 6.056 0 0 0 5.772-4.206 5.99 5.99 0 0 0 3.997-2.9 6.056 6.056 0 0 0-.747-7.073zM13.26 22.43a4.476 4.476 0 0 1-2.876-1.04l.141-.081 4.779-2.758a.795.795 0 0 0 .392-.681v-6.737l2.02 1.168a.071.071 0 0 1 .038.052v5.583a4.504 4.504 0 0 1-4.494 4.494zM3.6 18.304a4.47 4.47 0 0 1-.535-3.014l.142.085 4.783 2.759a.771.771 0 0 0 .78 0l5.843-3.369v2.332a.08.08 0 0 1-.033.062L9.74 19.95a4.5 4.5 0 0 1-6.14-1.646zM2.34 7.896a4.485 4.485 0 0 1 2.366-1.973V11.6a.766.766 0 0 0 .388.676l5.815 3.355-2.02 1.168a.076.076 0 0 1-.071 0l-4.83-2.786A4.504 4.504 0 0 1 2.34 7.872zm16.597 3.855-5.833-3.387L15.119 7.2a.076.076 0 0 1 .071 0l4.83 2.791a4.494 4.494 0 0 1-.676 8.105v-5.678a.79.79 0 0 0-.407-.667zm2.01-3.023-.141-.085-4.774-2.782a.776.776 0 0 0-.785 0L9.409 9.23V6.897a.066.066 0 0 1 .028-.061l4.83-2.787a4.5 4.5 0 0 1 6.68 4.66zm-12.64 4.135-2.02-1.164a.08.08 0 0 1-.038-.057V6.075a4.5 4.5 0 0 1 7.375-3.453l-.142.08L8.704 5.46a.795.795 0 0 0-.393.681zm1.097-2.365 2.602-1.5 2.607 1.5v2.999l-2.597 1.5-2.607-1.5z',
	},

	// Slack — the iconic 4-square hash. Simplified to a single-fill
	// path that traces the two interlocking pairs in flat colour.
	slack: {
		title: 'Slack',
		hex: '4A154B',
		path:
			'M5.042 15.165a2.528 2.528 0 0 1-2.52 2.523A2.528 2.528 0 0 1 0 15.165a2.527 2.527 0 0 1 2.522-2.52h2.52v2.52zm1.27 0a2.527 2.527 0 0 1 2.521-2.52 2.527 2.527 0 0 1 2.521 2.52v6.313A2.528 2.528 0 0 1 8.833 24a2.528 2.528 0 0 1-2.521-2.522v-6.313zM8.833 5.042a2.528 2.528 0 0 1-2.521-2.52A2.528 2.528 0 0 1 8.833 0a2.528 2.528 0 0 1 2.521 2.522v2.52H8.833zm0 1.27a2.527 2.527 0 0 1 2.521 2.521 2.527 2.527 0 0 1-2.521 2.521H2.522A2.527 2.527 0 0 1 0 8.833a2.527 2.527 0 0 1 2.522-2.521h6.311zm10.122 2.521a2.528 2.528 0 0 1 2.522-2.521A2.528 2.528 0 0 1 24 8.833a2.528 2.528 0 0 1-2.523 2.521h-2.522V8.833zm-1.268 0a2.527 2.527 0 0 1-2.521 2.521 2.527 2.527 0 0 1-2.522-2.521V2.522A2.527 2.527 0 0 1 15.166 0a2.528 2.528 0 0 1 2.521 2.522v6.311zm-2.521 10.122a2.528 2.528 0 0 1 2.521 2.522A2.528 2.528 0 0 1 15.166 24a2.528 2.528 0 0 1-2.522-2.523v-2.522h2.522zm0-1.268a2.527 2.527 0 0 1-2.522-2.521 2.527 2.527 0 0 1 2.522-2.522h6.312A2.528 2.528 0 0 1 24 15.165a2.527 2.527 0 0 1-2.523 2.521h-6.311z',
	},

	// Feishu / Lark — the swallow's wing curl, simplified.
	feishu: {
		title: 'Feishu',
		hex: '00D6B9',
		path:
			'M10.97 4.21a8.79 8.79 0 0 1 8.79 8.79 8.79 8.79 0 0 1-8.79 8.79 8.79 8.79 0 0 1-8.79-8.79 8.79 8.79 0 0 1 8.79-8.79zm0 2.34a6.45 6.45 0 0 0-6.45 6.45 6.45 6.45 0 0 0 6.45 6.45 6.45 6.45 0 0 0 6.45-6.45 6.45 6.45 0 0 0-6.45-6.45zm0 1.95a4.5 4.5 0 0 1 4.5 4.5 4.5 4.5 0 0 1-4.5 4.5 4.5 4.5 0 0 1-4.5-4.5 4.5 4.5 0 0 1 4.5-4.5zm9.45-6.5l1.86 1.86c.41.41.41 1.07 0 1.48l-7.86 7.86a1.05 1.05 0 0 1-1.48 0l-1.86-1.86a1.05 1.05 0 0 1 0-1.48l7.86-7.86c.41-.41 1.07-.41 1.48 0z',
	},

	// DingTalk — the rounded square with a chat tail, simplified.
	dingtalk: {
		title: 'DingTalk',
		hex: '0089FF',
		path:
			'M12 0C5.373 0 0 5.373 0 12s5.373 12 12 12 12-5.373 12-12S18.627 0 12 0zm5.4 8.2l-2.7 6.8 1.4 0.5-3.7 5-2.4-3.5 1.7-0.4-1.4-3.6-3.5 0.9 4.2-5.7H17.4z',
	},

	// WeCom — green-blue corporate variant of WeChat. Use a
	// stylised W in a rounded square.
	wecom: {
		title: 'WeCom',
		hex: '0082EF',
		path:
			'M3.5 2h17a1.5 1.5 0 0 1 1.5 1.5v17a1.5 1.5 0 0 1-1.5 1.5h-17A1.5 1.5 0 0 1 2 20.5v-17A1.5 1.5 0 0 1 3.5 2zm2.7 6l1.5 8 2-5 2 5 1.5-8h2.4l-2.7 10h-2.5l-1.7-4.5L7 18H4.5L1.8 8h2.4zm14 0l-2.7 8h-2.4l1.6-8h2.5z',
	},

	// Generic shell mark — used for the "Shell" provider so it
	// doesn't share Claude's colour by default.
	shell: {
		title: 'Shell',
		hex: '4D4D4D',
		path:
			'M3 4h18a1 1 0 0 1 1 1v14a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V5a1 1 0 0 1 1-1zm2 4l5 4-5 4v-2.5l2-1.5-2-1.5V8z',
	},
}

export type BrandIconKey = keyof typeof SIMPLE | keyof typeof INLINE

export interface BrandIconProps {
	iconKey?: string
	size?: number
	className?: string
	color?: string
	tone?: 'normal' | 'muted'
	title?: string
}

// Keys whose curated SVG lives at app/web/public/icons/<key>.svg
// (served at /admin/icons/<key>.svg in production). When set, BrandIcon
// prefers the on-disk asset over the inline path because operator-
// supplied SVGs are pixel-exact to the upstream brand mark, while the
// inline paths are the abbreviated-by-hand fallbacks.
const CURATED = new Set([
	'claude',
	'gemini',
	'openai',
	'shell',
	'slack',
	'dingtalk',
	'feishu',
	'wecom',
])

// Curated SVGs that ship as monochrome black (#000000). On the
// admin's dark theme these need `invert` so the glyph reads against
// the popover surface. Multi-colour curated SVGs (claude orange,
// gemini gem, slack 4-colour) carry their own ink and are excluded.
const CURATED_MONOCHROME_DARK_INVERT = new Set([
	'openai',
	'shell',
	'dingtalk',
	'feishu',
])

/**
 * Resolve an iconKey to its IconData record, regardless of source.
 * Returns null when the key is unknown so the caller can render a
 * fallback (initial letter, generic icon, etc.).
 */
function resolve(iconKey: string | undefined): IconData | null {
	if (!iconKey) return null
	const k = iconKey.toLowerCase()
	return SIMPLE[k] ?? INLINE[k] ?? null
}

/** True when the key has a curated SVG asset on disk. */
export function hasCuratedSvg(iconKey: string | undefined): boolean {
	if (!iconKey) return false
	return CURATED.has(iconKey.toLowerCase())
}

export function BrandIcon({
	iconKey,
	size = 16,
	className,
	color,
	tone = 'normal',
	title,
}: BrandIconProps) {
	const k = iconKey?.toLowerCase()

	// Curated SVG path — use the operator-supplied asset under
	// /admin/icons/<key>.svg. Skips colour/tone overrides because the
	// asset's own colours are the point of using it.
	if (k && CURATED.has(k)) {
		const invert = CURATED_MONOCHROME_DARK_INVERT.has(k)
		const opacity = tone === 'muted' ? 0.72 : 1
		const cls = [
			'inline-block object-contain',
			invert ? 'dark:invert dark:brightness-90' : '',
			className ?? '',
		]
			.filter(Boolean)
			.join(' ')
		return (
			<img
				src={`/admin/icons/${k}.svg`}
				width={size}
				height={size}
				alt={title ?? k}
				title={title}
				className={cls}
				style={{ width: size, height: size, opacity }}
			/>
		)
	}

	const data = resolve(iconKey)
	if (!data) return null

	const fill = color ?? `#${data.hex}`
	const opacity = tone === 'muted' ? 0.72 : 1
	const viewBox = data.viewBox ?? '0 0 24 24'

	return (
		<svg
			role="img"
			viewBox={viewBox}
			width={size}
			height={size}
			fill={fill}
			opacity={opacity}
			className={className}
			aria-label={title ?? data.title}
			xmlns="http://www.w3.org/2000/svg"
		>
			<title>{title ?? data.title}</title>
			<path d={data.path} />
		</svg>
	)
}

/**
 * Returns the brand hex (no leading `#`) for an iconKey, or
 * undefined when unknown. Useful for tinting backgrounds, status
 * pills, etc., to match the brand colour without rendering the
 * full icon.
 */
export function brandHex(iconKey: string | undefined): string | undefined {
	const data = resolve(iconKey)
	return data?.hex
}

/**
 * Returns true when `iconKey` resolves to a known brand. Callers
 * can use this to swap between BrandIcon and a fallback (emoji,
 * letter avatar, lucide icon).
 */
export function hasBrandIcon(iconKey: string | undefined): boolean {
	return resolve(iconKey) !== null
}
