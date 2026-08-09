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
}

// Flatten converts a nested vault (`projects/<name>/…` plus
// `personal/<name>.md`) to the flat layout (`<name>/…` with
// `<name>/personal.md` inside it).
//
// It must be run against a vault constructed with LayoutNested — the
// source paths are the nested ones. The caller records the new layout
// in config afterwards; doing it here would leave a half-migrated
// vault claiming to be flat if the process died mid-run.
func (v *Vault) Flatten(ctx context.Context, dryRun bool) (FlattenResult, error) {
	if v.layout != LayoutNested {
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
	return res, nil
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
