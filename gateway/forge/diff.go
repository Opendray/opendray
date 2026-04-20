package forge

import (
	"strings"
)

// parseUnifiedDiff splits a concatenated unified-diff blob (what
// Gitea and GitHub return for `.diff` / `Accept: v3.diff`) into
// per-file [DiffFile] entries. The Patch field on each entry holds
// the diff for that file alone, reconstructed by re-prefixing the
// "diff --git" header so the fragment is a valid unified diff on
// its own.
//
// Only shape we care about per file:
//   diff --git a/<old> b/<new>     ← start sentinel
//   [index / similarity / rename / new file / deleted file lines]
//   --- a/<old>  (or /dev/null)
//   +++ b/<new>  (or /dev/null)
//   @@ ...                         ← hunks
//
// We don't try to parse binary diffs — those come through with
// "Binary files ... differ" and a zero hunk count, which is fine.
func parseUnifiedDiff(blob string) []DiffFile {
	if strings.TrimSpace(blob) == "" {
		return nil
	}
	lines := strings.Split(blob, "\n")
	var out []DiffFile
	var cur *DiffFile
	var buf strings.Builder

	flush := func() {
		if cur == nil {
			return
		}
		patch := buf.String()
		cur.Patch = patch
		cur.Additions, cur.Deletions = countHunkLines(patch)
		out = append(out, *cur)
		cur = nil
		buf.Reset()
	}

	for i, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			flush()
			f := parseDiffGitHeader(line)
			cur = &f
			buf.WriteString(line)
			if i < len(lines)-1 {
				buf.WriteByte('\n')
			}
			continue
		}
		if cur == nil {
			// Stray bytes before the first "diff --git" (signed
			// patches, mailmap headers, etc). Drop them — the
			// parser only exposes per-file diffs, not commit
			// metadata.
			continue
		}
		// Promote file metadata onto the current entry so callers
		// that care about "new/deleted/renamed" don't have to
		// re-parse the patch.
		switch {
		case strings.HasPrefix(line, "new file mode"):
			cur.Status = "added"
		case strings.HasPrefix(line, "deleted file mode"):
			cur.Status = "deleted"
		case strings.HasPrefix(line, "rename from "):
			cur.OldPath = strings.TrimPrefix(line, "rename from ")
			cur.Status = "renamed"
		case strings.HasPrefix(line, "rename to "):
			cur.Path = strings.TrimPrefix(line, "rename to ")
		case strings.HasPrefix(line, "--- a/") && cur.OldPath == "" && cur.Status != "added":
			cur.OldPath = strings.TrimPrefix(line, "--- a/")
		}
		buf.WriteString(line)
		if i < len(lines)-1 {
			buf.WriteByte('\n')
		}
	}
	flush()

	// Normalise: if OldPath equals Path, treat as "not a rename"
	// and drop OldPath so the UI doesn't render a redundant "from"
	// label.
	for i := range out {
		if out[i].OldPath == out[i].Path {
			out[i].OldPath = ""
		}
		if out[i].Status == "" {
			out[i].Status = "modified"
		}
	}
	return out
}

// parseDiffGitHeader extracts the new-path from a `diff --git
// a/<old> b/<new>` line. Paths containing spaces aren't handled —
// git quotes those with "diff --git \"a/foo bar\" \"b/foo bar\"",
// which we treat as opaque and stash the raw tail into Path.
func parseDiffGitHeader(line string) DiffFile {
	// Strip the "diff --git " prefix; remainder is "a/X b/Y" or
	// quoted-path variants.
	tail := strings.TrimPrefix(line, "diff --git ")
	// Split on " b/" — this is robust for the unquoted common case
	// and degrades gracefully for quoted paths (we end up with the
	// full tail in Path, which is still unambiguous).
	if idx := strings.Index(tail, " b/"); idx > 0 {
		oldSide := tail[:idx]
		newSide := tail[idx+len(" b/"):]
		return DiffFile{
			OldPath: strings.TrimPrefix(oldSide, "a/"),
			Path:    newSide,
		}
	}
	return DiffFile{Path: tail}
}

// countHunkLines walks a per-file unified diff and counts +/-
// lines, ignoring the +++/--- header pair. Used both by the
// parser (for Gitea/GitHub blobs) and by the GitLab adapter
// (which gets per-file diffs but no pre-computed line counts).
func countHunkLines(patch string) (adds, dels int) {
	inHunk := false
	for _, line := range strings.Split(patch, "\n") {
		switch {
		case strings.HasPrefix(line, "@@"):
			inHunk = true
			continue
		case !inHunk:
			continue
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			// header leaked inside — shouldn't happen, but skip
			continue
		case strings.HasPrefix(line, "+"):
			adds++
		case strings.HasPrefix(line, "-"):
			dels++
		}
	}
	return adds, dels
}
