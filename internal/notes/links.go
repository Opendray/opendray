package notes

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Hard caps so a sloppy regex / huge vault never blocks the gateway.
const (
	scanTimeout    = 6 * time.Second
	scanMaxFileLen = 1 << 20 // 1 MiB per .md
	maxBacklinks   = 200
	maxTags        = 500
	contextChars   = 80 // length of snippet around a backlink hit
)

// Backlink is one note that references the target via a wiki-link.
type Backlink struct {
	Path     string   `json:"path"`
	Title    string   `json:"title"`
	Modified string   `json:"modified"`
	Lines    []string `json:"lines"` // matching line snippets
}

// TagCount is one tag and the count of notes mentioning it.
type TagCount struct {
	Tag   string   `json:"tag"`
	Count int      `json:"count"`
	Notes []string `json:"notes,omitempty"`
}

// Backlinks scans the vault for notes that reference `targetRel`
// (vault-relative .md path) via a wiki-link. Recognised forms:
//
//	[[basename]]              — without .md, without folder prefix
//	[[basename|alias]]        — pipe-aliased
//	[[projects/foo/spec]]     — full vault path without .md
//
// Code blocks (fenced ```) and inline code (`...`) are stripped before
// matching so wiki-link syntax discussed in code samples doesn't show
// up as a real backlink.
func (v *Vault) Backlinks(ctx context.Context, targetRel string) ([]Backlink, error) {
	full, err := v.resolve(targetRel)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(full); err != nil {
		return nil, ErrNotFound
	}
	target := strings.TrimSuffix(targetRel, ".md")
	base := filepath.Base(target)

	// Build a single regex matching any of the recognised reference
	// shapes. \b doesn't work around `[[` so we anchor manually.
	pat, err := regexp.Compile(
		`\[\[(?:` +
			regexp.QuoteMeta(target) + `|` + regexp.QuoteMeta(base) +
			`)(?:\|[^\]]*)?\]\]`,
	)
	if err != nil {
		return nil, err
	}

	out := []Backlink{}
	deadline := time.Now().Add(scanTimeout)
	err = filepath.WalkDir(v.root, func(path string, d fs.DirEntry, walkErr error) error {
		if time.Now().After(deadline) {
			return errors.New("backlink scan timed out")
		}
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if path != v.root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		// Skip the target itself — a note's own [[basename]] mention
		// inside its body isn't a "backlink".
		if path == full {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > scanMaxFileLen {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		stripped := stripCodeRegions(string(body))
		matches := findMatchLines(stripped, pat)
		if len(matches) == 0 {
			return nil
		}
		rel, _ := filepath.Rel(v.root, path)
		out = append(out, Backlink{
			Path:     filepath.ToSlash(rel),
			Title:    titleFromBody(string(body), filepath.ToSlash(rel)),
			Modified: info.ModTime().Format(time.RFC3339),
			Lines:    matches,
		})
		if len(out) >= maxBacklinks {
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return out, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// Tags walks the vault and aggregates all #tag mentions and frontmatter
// tag arrays. `prefix` filters to notes whose path starts with the
// prefix (e.g. "projects/" → only project docs).
func (v *Vault) Tags(ctx context.Context, prefix string) ([]TagCount, error) {
	prefix = strings.TrimPrefix(prefix, "/")
	counts := map[string]map[string]struct{}{} // tag → set of paths
	tagPat := regexp.MustCompile(`(?:^|[^A-Za-z0-9_-])#([A-Za-z][A-Za-z0-9_/-]{0,40})`)

	deadline := time.Now().Add(scanTimeout)
	err := filepath.WalkDir(v.root, func(path string, d fs.DirEntry, walkErr error) error {
		if time.Now().After(deadline) {
			return errors.New("tag scan timed out")
		}
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if path != v.root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		rel, _ := filepath.Rel(v.root, path)
		relSlash := filepath.ToSlash(rel)
		if prefix != "" && !strings.HasPrefix(relSlash, prefix) {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > scanMaxFileLen {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		bodyStr := string(body)
		// Frontmatter `tags: [a, b]` / `tags:\n  - a` — both forms.
		for _, t := range frontmatterTags(bodyStr) {
			addTag(counts, t, relSlash)
		}
		// Inline #tag mentions, excluding code regions to avoid
		// picking up #include / # comments / shebangs.
		for _, m := range tagPat.FindAllStringSubmatch(stripCodeRegions(bodyStr), -1) {
			tag := strings.TrimSuffix(m[1], "/")
			if tag == "" {
				return nil
			}
			addTag(counts, tag, relSlash)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	out := make([]TagCount, 0, len(counts))
	for tag, notes := range counts {
		paths := make([]string, 0, len(notes))
		for p := range notes {
			paths = append(paths, p)
		}
		sort.Strings(paths)
		out = append(out, TagCount{Tag: tag, Count: len(paths), Notes: paths})
		if len(out) >= maxTags {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Tag < out[j].Tag
	})
	return out, nil
}

func addTag(m map[string]map[string]struct{}, tag, path string) {
	set, ok := m[tag]
	if !ok {
		set = map[string]struct{}{}
		m[tag] = set
	}
	set[path] = struct{}{}
}

// findMatchLines returns the lines containing any regex match, with a
// little surrounding context. Cap at 5 hits per file to keep payloads
// small — the panel renders these as previews, not authoritative diff.
func findMatchLines(body string, pat *regexp.Regexp) []string {
	out := []string{}
	for _, line := range strings.Split(body, "\n") {
		if !pat.MatchString(line) {
			continue
		}
		l := strings.TrimSpace(line)
		if len(l) > contextChars*2 {
			// Pull a window around the first match for readability.
			loc := pat.FindStringIndex(l)
			if loc != nil {
				start := max(0, loc[0]-contextChars)
				end := min(len(l), loc[1]+contextChars)
				prefix := ""
				suffix := ""
				if start > 0 {
					prefix = "…"
				}
				if end < len(l) {
					suffix = "…"
				}
				l = prefix + l[start:end] + suffix
			}
		}
		out = append(out, l)
		if len(out) >= 5 {
			break
		}
	}
	return out
}

// frontmatterTags pulls `tags:` from a `---` block. Tolerant of both
// inline (`tags: [a, b]`) and block (`tags:\n  - a\n  - b`) forms.
func frontmatterTags(body string) []string {
	if !strings.HasPrefix(body, "---") {
		return nil
	}
	end := strings.Index(body[3:], "---")
	if end < 0 {
		return nil
	}
	header := body[3 : 3+end]
	out := []string{}
	lines := strings.Split(header, "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "tags:") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, "tags:"))
		// Inline: tags: [a, b, c]
		if strings.HasPrefix(rest, "[") && strings.HasSuffix(rest, "]") {
			inner := strings.Trim(rest, "[]")
			for _, t := range strings.Split(inner, ",") {
				t = strings.Trim(strings.TrimSpace(t), `"'`)
				if t != "" {
					out = append(out, t)
				}
			}
			continue
		}
		// Block: tags:\n  - a\n  - b
		for j := i + 1; j < len(lines); j++ {
			next := lines[j]
			t := strings.TrimSpace(next)
			if !strings.HasPrefix(t, "- ") {
				break
			}
			t = strings.Trim(strings.TrimSpace(strings.TrimPrefix(t, "- ")), `"'`)
			if t != "" {
				out = append(out, t)
			}
		}
	}
	return out
}

// stripCodeRegions removes fenced (```…```) and inline (`…`) code so
// wiki-link / tag syntax discussed in code samples doesn't trigger
// false matches. Cheap line-by-line state machine, no parser.
func stripCodeRegions(body string) string {
	var b strings.Builder
	inFence := false
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "```") {
			inFence = !inFence
			b.WriteByte('\n')
			continue
		}
		if inFence {
			b.WriteByte('\n')
			continue
		}
		// Strip backtick spans on this line.
		stripped := stripInlineCode(line)
		b.WriteString(stripped)
		b.WriteByte('\n')
	}
	return b.String()
}

func stripInlineCode(line string) string {
	var b strings.Builder
	in := false
	for _, r := range line {
		if r == '`' {
			in = !in
			b.WriteByte(' ')
			continue
		}
		if in {
			b.WriteByte(' ')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// fmtErrf is unused right now but kept for parity with neighbour files.
var _ = fmt.Errorf
