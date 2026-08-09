package notes

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Converting a nested vault to the flat layout is a rename of every
// document, and renames are where a doc library quietly breaks: a
// `[[wiki link]]` written against `projects/foo/bar` points at nothing
// once `projects/` is gone.
//
// So this does not shell out to `mv`. It drives the same Move() the
// rename UI uses, one document at a time, which rewrites the links
// that referenced each file as it goes. Slower than moving directories
// wholesale, and correct.
//
// Nothing is deleted and nothing is overwritten. A target that already
// exists is reported and skipped, leaving both copies for the operator
// to reconcile — a migration that resolves conflicts on its own is a
// migration that loses work.

// FlattenMove is one document the migration would move, or did.
type FlattenMove struct {
	From string `json:"from"`
	To   string `json:"to"`
	// LinksRewritten counts `[[wiki link]]` occurrences repointed at
	// the new path. Zero on a dry run — nothing has been read yet.
	LinksRewritten int `json:"links_rewritten"`
}

// FlattenSkip is a document the migration refused to touch, with the
// reason stated in the operator's terms rather than as an error code.
type FlattenSkip struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// FlattenResult is the plan (dry run) or the outcome (apply).
type FlattenResult struct {
	Moves []FlattenMove `json:"moves"`
	Skips []FlattenSkip `json:"skips"`
	// MappingsRewritten counts per-cwd project overrides repointed at
	// their new location. An override left pointing into a `projects/`
	// directory that no longer exists would silently start writing a
	// project's documents to a fresh, empty folder.
	MappingsRewritten int `json:"mappings_rewritten"`
	// DryRun is true when nothing was written.
	DryRun bool `json:"dry_run"`
	// LayoutRecorded reports that the new layout reached the config
	// file. False after an apply means the vault is flat on disk while
	// the gateway will come back believing it is nested — surface it.
	LayoutRecorded bool `json:"layout_recorded"`
	// Warning explains a partial success in the operator's terms. The
	// documents moved; something after them did not.
	Warning string `json:"warning,omitempty"`
}

// Flattenable reports whether this vault has documents the flat layout
// would file differently. It exists so the UI can OFFER the conversion:
// a migration that only ships as a CLI command is one that only the
// people who wrote it ever run, which would leave every existing vault
// on the old shape forever without its owner ever learning there was a
// choice.
//
// Deliberately cheap — this is called whenever someone opens the doc
// library. It answers "is there anything to talk about", not "what
// exactly would move"; the dry run answers that, and only when asked.
func (v *Vault) Flattenable() bool {
	if v.Layout() != LayoutNested {
		return false
	}
	projects := strings.Trim(v.projectsPrefix, "/")
	personal := strings.Trim(v.personalPrefix, "/")
	return dirHasChildren(filepath.Join(v.root, projects)) ||
		dirHasChildren(filepath.Join(v.root, personal))
}

// Flatten converts a nested vault (`projects/<name>/…` plus
// `personal/<name>.md`) to the flat layout (`<name>/…` with
// `<name>/personal.md` inside it).
//
// It must be run against a vault constructed with LayoutNested — the
// source paths are the nested ones.
//
// On success it flips its own layout and asks the caller's hook to
// persist it. An earlier version left recording to the gateway's
// startup detection, on the reasoning that a vault with no `projects/`
// directory would be detected as flat. That was wrong, and it shipped:
// detection runs ONLY when the setting is empty, so a vault already
// recorded as nested stayed nested after being converted. Projects kept
// resolving to `projects/<name>` and personal notes kept landing in
// `personal/` — against directories the migration had just emptied.
func (v *Vault) Flatten(ctx context.Context, dryRun bool) (FlattenResult, error) {
	if v.Layout() != LayoutNested {
		return FlattenResult{}, errors.New(
			"vault is already flat: nothing to convert")
	}
	res := FlattenResult{DryRun: dryRun}

	plan, skips, err := v.flattenPlan()
	if err != nil {
		return FlattenResult{}, err
	}
	res.Skips = skips
	if dryRun {
		res.Moves = plan
		return res, nil
	}

	for _, m := range plan {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		out, err := v.Move(ctx, m.From, m.To)
		if err != nil {
			// One unmovable document must not abandon the rest
			// half-migrated with no account of what happened.
			res.Skips = append(res.Skips, FlattenSkip{
				Path:   m.From,
				Reason: err.Error(),
			})
			continue
		}
		m.LinksRewritten = out.LinksRewritten
		res.Moves = append(res.Moves, m)
	}

	n, err := v.rewriteProjectMappings()
	if err != nil {
		return res, fmt.Errorf("rewrite project overrides: %w", err)
	}
	res.MappingsRewritten = n

	// Directories the migration emptied are not documents and nothing
	// lists them usefully — leaving `projects/` and `personal/` behind
	// as empty folders would show the old shape in a vault that no
	// longer has it.
	v.pruneEmptied()

	// Paths derived from here on must follow the new shape, including
	// on this very request. Waiting for a restart would have the
	// gateway answering with directories it just emptied.
	v.setLayout(LayoutFlat)
	if v.onLayoutChange != nil {
		if err := v.onLayoutChange(LayoutFlat); err != nil {
			res.Warning = "documents were moved, but the new layout could " +
				"not be saved to the config file (" + err.Error() +
				"). Set [vault] layout = \"flat\" by hand, or the gateway " +
				"will look for the old directories after a restart."
			return res, nil
		}
	}
	res.LayoutRecorded = true
	return res, nil
}

// pruneEmptied removes the nested layout's directories once nothing is
// left in them. Best-effort by design: a directory the operator still
// keeps something in is one os.Remove refuses to touch, which is
// exactly the behaviour wanted here.
func (v *Vault) pruneEmptied() {
	projects := strings.Trim(v.projectsPrefix, "/")
	personal := strings.Trim(v.personalPrefix, "/")
	for _, base := range []string{projects, personal} {
		if base == "" {
			continue
		}
		dir := filepath.Join(v.root, base)
		// Children first: `projects/foo/features` has to go before
		// `projects/foo`, and that before `projects`.
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				removeEmptyTree(filepath.Join(dir, e.Name()))
			}
		}
		removeEmptyTree(dir)
	}
}

func removeEmptyTree(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			removeEmptyTree(filepath.Join(dir, e.Name()))
		}
	}
	// Fails harmlessly when anything is left, which is the guard.
	_ = os.Remove(dir)
}

// flattenPlan lists the moves without performing any.
func (v *Vault) flattenPlan() ([]FlattenMove, []FlattenSkip, error) {
	var moves []FlattenMove
	var skips []FlattenSkip

	projects := strings.Trim(v.projectsPrefix, "/")
	personal := strings.Trim(v.personalPrefix, "/")

	err := filepath.WalkDir(v.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(v.root, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && rel != "." {
				return filepath.SkipDir
			}
			return nil
		}
		if !IsDocument(d.Name()) {
			return nil
		}

		to, reason := v.flatTarget(rel, projects, personal)
		switch {
		case reason != "":
			skips = append(skips, FlattenSkip{Path: rel, Reason: reason})
		case to == "":
			// Already outside the two nested trees — daily notes,
			// templates, anything the operator filed themselves. The
			// flat layout does not claim those.
		default:
			moves = append(moves, FlattenMove{From: rel, To: to})
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	sort.Slice(moves, func(i, j int) bool { return moves[i].From < moves[j].From })
	sort.Slice(skips, func(i, j int) bool { return skips[i].Path < skips[j].Path })
	return moves, skips, nil
}

// flatTarget maps one nested path to its flat destination. An empty
// target with an empty reason means "leave this file where it is".
func (v *Vault) flatTarget(rel, projects, personal string) (to, reason string) {
	switch {
	case projects != "" && strings.HasPrefix(rel, projects+"/"):
		rest := strings.TrimPrefix(rel, projects+"/")
		name, tail, ok := strings.Cut(rest, "/")
		if !ok {
			// A loose file directly under projects/ has no project to
			// belong to. Moving it to the vault root would put a stray
			// document beside the project directories.
			return "", "sits directly in " + projects +
				"/ and belongs to no project"
		}
		to = flatRoot(name) + "/" + tail

	case personal != "" && strings.HasPrefix(rel, personal+"/"):
		rest := strings.TrimPrefix(rel, personal+"/")
		if strings.Contains(rest, "/") {
			// personal/<name>.md is the convention; a nested tree in
			// there is the operator's own filing and has no obvious
			// project to move into.
			return "", "is filed under " + personal +
				"/ in a shape opendray did not create"
		}
		name := TrimDocExt(rest)
		to = flatRoot(name) + "/personal" + filepath.Ext(rest)

	default:
		return "", ""
	}

	if _, err := os.Stat(filepath.Join(v.root, filepath.FromSlash(to))); err == nil {
		return "", to + " already exists"
	}
	return to, ""
}

// rewriteProjectMappings repoints per-cwd overrides at the flat layout.
func (v *Vault) rewriteProjectMappings() (int, error) {
	v.projectMapMu.Lock()
	defer v.projectMapMu.Unlock()

	m, err := v.readProjectMapLocked()
	if err != nil {
		return 0, err
	}
	projects := strings.Trim(v.projectsPrefix, "/")
	if projects == "" || len(m) == 0 {
		return 0, nil
	}
	changed := 0
	for cwd, p := range m {
		trimmed := strings.Trim(p, "/")
		if !strings.HasPrefix(trimmed, projects+"/") {
			continue
		}
		rest := strings.TrimPrefix(trimmed, projects+"/")
		name, tail, ok := strings.Cut(rest, "/")
		next := flatRoot(name)
		if ok {
			next += "/" + tail
		}
		m[cwd] = next
		changed++
	}
	if changed == 0 {
		return 0, nil
	}
	if err := v.writeProjectMapLocked(m); err != nil {
		return 0, err
	}
	return changed, nil
}
