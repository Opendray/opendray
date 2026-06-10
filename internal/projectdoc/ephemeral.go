package projectdoc

import (
	"errors"
	"strings"
)

// IsEphemeralCwd reports whether a cwd is a throwaway/temp dir —
// sessions run from /tmp, /var/folders, .cache, etc. Third-party
// consumers and tests spawn sessions there constantly; those are NOT
// projects and must leave no project footprint: no journal, no doc
// blueprint, no scanner docs, no capture, no entry in the project
// list. (The knowledge anchorer has applied the same predicate to the
// graph since P-G; this exports it for the whole Notes layer.)
func IsEphemeralCwd(cwd string) bool {
	if strings.TrimSpace(cwd) == "" {
		return true
	}
	c := strings.ToLower(cwd)
	return c == "/tmp" ||
		strings.HasPrefix(c, "/tmp/") ||
		strings.HasPrefix(c, "/private/tmp") ||
		strings.HasPrefix(c, "/var/folders/") ||
		strings.HasPrefix(c, "/private/var/folders/") ||
		strings.Contains(c, "/tmp.") ||
		strings.Contains(c, "/.cache/")
}

// ErrEphemeralCwd is returned when a caller tries to create project
// footprint (docs, proposals, blueprint sections) under a temp dir.
var ErrEphemeralCwd = errors.New("projectdoc: ephemeral cwd is not a project")
