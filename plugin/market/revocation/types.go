// Package revocation implements the marketplace kill-switch
// mechanism. Entries in revocations.json name a plugin + version
// range + action (uninstall / disable / warn); the client polls
// the file periodically and applies matching actions to installed
// plugins.
//
// Wire contract: schemas/revocations.schema.json in the
// opendray-marketplace repo.
//
// Security model (see 09-marketplace.md §Kill-switch):
//   - Revocation is advisory. Airgapped installs won't auto-act
//     but will show the warning on next network contact.
//   - We do not phone home. Clients pull the file on a timer;
//     the server never pushes.
//   - A revoked plugin stays on the list forever — publisher
//     keys can't retroactively "un-revoke" a prior entry.
//
// The revocation list is fetched through market.Catalog, so the
// same mirror fallback + cache + signature-optional infrastructure
// covers it. Match logic uses Masterminds/semver for ranges.
package revocation

// Entry is one row of revocations.json — a plugin (or range of
// versions of a plugin) plus the action the client should take.
type Entry struct {
	// Name identifies the plugin in "publisher/name" form. The
	// legacy bare-name shape ("plugin") is accepted too and
	// matched against installed plugins whose publisher is the
	// M3 back-compat default ("opendray-examples").
	Name string `json:"name"`

	// Versions is a semver range accepted by Masterminds/semver.
	// Common shapes: "<=1.2.3", ">=2.0.0", "1.2.3" (exact), "*"
	// (any).
	Versions string `json:"versions"`

	// Reason is the operator-supplied short explanation. Shown to
	// the user in the revocation banner.
	Reason string `json:"reason"`

	// RecordedAt is the ISO timestamp the PR merged. Used only for
	// audit log display; match logic ignores it.
	RecordedAt string `json:"recordedAt"`

	// Action is one of:
	//   "uninstall" — auto-remove the plugin with a banner.
	//   "disable"   — flip enabled=false, keep files, show banner.
	//   "warn"      — show a red banner; plugin keeps working.
	//
	// Unknown actions are treated as "warn" so a future schema
	// extension never silently goes ignored.
	Action string `json:"action"`
}

// Response is the top-level shape of revocations.json.
type Response struct {
	Version int     `json:"version"`
	Entries []Entry `json:"entries"`
}

// Action constants. Using typed strings lets callers switch with
// exhaustiveness checking when desired.
const (
	ActionUninstall = "uninstall"
	ActionDisable   = "disable"
	ActionWarn      = "warn"
)
