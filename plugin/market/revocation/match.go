package revocation

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// defaultPublisher fills in when a revocation Entry uses the
// legacy bare-name shape or when the installed plugin has no
// registered publisher. Kept consistent with market/remote.
const defaultPublisher = "opendray-examples"

// Matches reports whether the entry applies to an installed
// plugin identified by (publisher, name, version).
//
// Matching rules:
//   - Entry.Name may be "publisher/name" or bare "name" (legacy).
//     Bare names match installed.Publisher=="opendray-examples"
//     only.
//   - Publisher + name must match exactly.
//   - Entry.Versions is a semver range (Masterminds grammar).
//     "*" matches every version. Missing/empty versions is treated
//     as "*" for operator convenience.
//   - A version that fails semver.NewVersion (e.g. "dev-build")
//     matches only when Entry.Versions is explicitly "*".
//
// Matches returns an error only when the entry itself is
// malformed (bad Name or unparseable Versions range). A mismatch
// returns (false, nil); a malformed entry returns (false, err)
// so the poller can log + skip without aborting the whole sweep.
func (e Entry) Matches(installedPublisher, installedName, installedVersion string) (bool, error) {
	entryPub, entryName, err := parseEntryName(e.Name)
	if err != nil {
		return false, err
	}
	if installedPublisher == "" {
		installedPublisher = defaultPublisher
	}
	if entryPub != installedPublisher || entryName != installedName {
		return false, nil
	}

	rangeExpr := strings.TrimSpace(e.Versions)
	if rangeExpr == "" || rangeExpr == "*" {
		return true, nil
	}
	constraint, err := semver.NewConstraint(rangeExpr)
	if err != nil {
		return false, fmt.Errorf("revocation: versions %q: %w", rangeExpr, err)
	}
	v, err := semver.NewVersion(installedVersion)
	if err != nil {
		// Non-semver installed version. Only a wildcard range
		// would have matched; since we're here the range isn't "*".
		return false, nil
	}
	return constraint.Check(v), nil
}

// parseEntryName splits "pub/name" (or bare "name") into
// (publisher, name). Rejects shapes with too many slashes or
// empty components so a typo in revocations.json can't
// accidentally match every plugin in a publisher.
func parseEntryName(raw string) (publisher, name string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", fmt.Errorf("revocation: empty Name")
	}
	if strings.ContainsAny(raw, `\@ `) {
		return "", "", fmt.Errorf("revocation: Name contains unsafe char: %q", raw)
	}
	parts := strings.Split(raw, "/")
	switch len(parts) {
	case 1:
		return defaultPublisher, parts[0], nil
	case 2:
		if parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("revocation: Name empty segment: %q", raw)
		}
		return parts[0], parts[1], nil
	default:
		return "", "", fmt.Errorf("revocation: Name has too many slashes: %q", raw)
	}
}

// NormalisedAction returns Action verbatim when it's a known
// value; unknown or empty values map to ActionWarn so a future
// schema extension never silently goes ignored.
func (e Entry) NormalisedAction() string {
	switch e.Action {
	case ActionUninstall, ActionDisable, ActionWarn:
		return e.Action
	default:
		return ActionWarn
	}
}
