// Map opendray provider ids to BrandIcon keys.
//
// Provider manifests still expose an `icon` (emoji) for backward
// compatibility, but the admin UI prefers an official mark when one
// is registered for the provider id. Add new providers here when
// they ship.

const MAP: Record<string, string> = {
	claude: 'claude',
	codex: 'openai', // Codex CLI is OpenAI-branded
	gemini: 'gemini',
	shell: 'shell',
}

export function providerIconKey(providerId: string | undefined): string | undefined {
	if (!providerId) return undefined
	return MAP[providerId.toLowerCase()]
}

// Map a provider id to a single fallback letter for the avatar
// disc when no brand icon exists. The first letter of the
// display name is the natural choice; provide overrides where
// the bundled letter would be misleading.
export function providerFallbackLetter(displayName: string | undefined): string {
	const trimmed = (displayName ?? '').trim()
	if (!trimmed) return '?'
	return trimmed.charAt(0)
}
