package notes

import (
	"os"
	"path/filepath"
	"strings"
)

// The vault is a project documentation library, and it used to file
// every project under `projects/` — a folder whose name repeats what
// the whole library already is. One level of nesting that tells the
// reader nothing, on every path, in every listing, and at the top of
// the git repository the vault syncs to.
//
// The operator's own notes were split off further still: agent-written
// docs at `projects/<name>/…` and the human scratchpad at
// `personal/<name>.md` — the same project's material in two distant
// places, sorted by *who wrote it* rather than by what it is about.
//
//	nested (original)          flat (default for new installs)
//	  projects/opendray-v2/      opendray-v2/
//	    features/                  features/
//	    issues/                    issues/
//	  personal/opendray-v2.md      personal.md
//
// Existing vaults are NOT rearranged. The layout is decided once and
// written into config.toml, because the alternative — re-deriving it
// from what happens to be on disk — is exactly how the doc root once
// moved out from under a live install: a probe that asks "does this
// directory have content?" changes its answer when someone adds
// content. See `opendray notes flatten` to convert deliberately.

// Layout decides where a project's documents live in the vault.
type Layout string

const (
	// LayoutFlat puts each project at the vault root under its own
	// name, with the operator's personal notes inside it.
	LayoutFlat Layout = "flat"
	// LayoutNested keeps the original split: `projects/<name>/` for
	// documents and `personal/<name>.md` for the operator's notes.
	LayoutNested Layout = "nested"
)

// Valid reports whether l is a layout the vault understands. The empty
// string is not valid here — it means "not decided yet", which callers
// resolve with DetectLayout before constructing a Vault.
func (l Layout) Valid() bool { return l == LayoutFlat || l == LayoutNested }

// DetectLayout infers the layout of an existing vault, for the one
// moment where it has to be guessed: an install that predates the
// setting. A vault already filing documents under `projects/` is
// nested; anything else — including an empty vault — is flat.
//
// Callers MUST persist the result. Detection is deliberately called
// once, not on every resolve.
func DetectLayout(root string) Layout {
	if dirHasChildren(filepath.Join(root, "projects")) {
		return LayoutNested
	}
	return LayoutFlat
}

func dirHasChildren(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		// A folder holding nothing but .DS_Store is empty as far as
		// anyone reading the vault is concerned.
		if e.Name() != ".DS_Store" {
			return true
		}
	}
	return false
}

// reservedRoots are top-level names the flat layout cannot hand to a
// project, because the vault already means something by them. In the
// nested layout there is nothing to collide with — `projects/daily` is
// just a project — so this applies to flat only.
var reservedRoots = map[string]bool{
	"daily": true, // daily/YYYY-MM-DD.md, a vault-wide concept
}

// reservedRoot reports whether a top-level directory name belongs to
// the vault rather than to a project. Underscore and dot prefixes are
// reserved wholesale: `_templates` already uses the first, and the
// second is how every listing recognises a hidden file.
func reservedRoot(name string) bool {
	if name == "" {
		return true
	}
	if strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".") {
		return true
	}
	return reservedRoots[strings.ToLower(name)]
}

// flatRoot returns the top-level directory for a project slug,
// stepping aside when the name is one the vault has already claimed.
// The suffix is predictable rather than clever: an operator who lands
// on it can see what happened, and a per-cwd mapping override remains
// available for anyone who wants a different name entirely.
func flatRoot(slug string) string {
	if reservedRoot(slug) {
		return slug + "-docs"
	}
	return slug
}
