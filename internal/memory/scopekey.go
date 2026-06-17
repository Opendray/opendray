package memory

import "strings"

// IntegrationScopeKeyPrefix marks a project-scope memory partition that
// belongs to a single third-party integration instead of a real working
// directory. Captured facts from integration-origin sessions are routed
// here (see capture.scopeKeyForRule) so they never share the operator's
// cwd-scoped project partition — this is the third-party memory
// isolation zone. The "integration:" prefix can't collide with an
// absolute-path cwd or the "operator" global key, so the zone is both
// isolated and identifiable (e.g. via ListScopeKeys / the admin memory
// list).
const IntegrationScopeKeyPrefix = "integration:"

// IntegrationScopeKey returns the project-scope scope_key for an
// integration's isolated memory zone. Single source of truth for the
// on-disk key format.
func IntegrationScopeKey(integrationID string) string {
	return IntegrationScopeKeyPrefix + integrationID
}

// IsIntegrationScopeKey reports whether a project-scope scope_key names a
// third-party integration's isolated zone rather than a real cwd.
//
// Callers that interpret a project scope_key as a filesystem path (the
// git-activity scanner, the .claude/projects file mirror) or that distil
// operator-facing knowledge from project memory MUST skip these keys, so
// the zone stays isolated from the operator and path-assuming work never
// runs against a non-path key. Lifecycle passes that operate purely on
// memory rows (the cleaner) and surfaces meant to expose the zone (the
// admin memory list) should NOT skip them.
func IsIntegrationScopeKey(scopeKey string) bool {
	return strings.HasPrefix(scopeKey, IntegrationScopeKeyPrefix)
}
