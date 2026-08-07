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

// Moving a note is the operation that turns a pile of files into a
// structure someone maintains. Without it the only way to file a note
// under a folder is delete-and-recreate, which quietly breaks every
// [[wiki-link]] pointing at it — so in practice nobody reorganises,
// and the vault stays flat.
//
// The link rewrite is therefore not a nicety bolted onto the rename; it
// is the reason the rename is safe to offer at all.

// MoveResult reports what a move touched, so the UI can say "moved, and
// updated 3 links" rather than leaving the operator to wonder.
type MoveResult struct {
	From string `json:"from"`
	To   string `json:"to"`
	// RewrittenIn lists the notes whose wiki-links were updated.
	RewrittenIn []string `json:"rewritten_in"`
	// LinksRewritten counts individual link occurrences, which can
	// exceed len(RewrittenIn) when one note links several times.
	LinksRewritten int  `json:"links_rewritten"`
	Note           Note `json:"note"`
}

// Move renames/relocates a note within the vault and repoints the
// wiki-links that referenced it.
//
// Both paths go through the same jail check as every other write, must
// end in .md, and the destination must not already exist — an
// overwriting move is indistinguishable from data loss and there is no
// undo here.
func (v *Vault) Move(ctx context.Context, fromRel, toRel string) (MoveResult, error) {
	if err := requireMarkdown(fromRel); err != nil {
		return MoveResult{}, err
	}
	if err := requireMarkdown(toRel); err != nil {
		return MoveResult{}, err
	}
	fromFull, err := v.resolve(fromRel)
	if err != nil {
		return MoveResult{}, err
	}
	toFull, err := v.resolve(toRel)
	if err != nil {
		return MoveResult{}, err
	}
	if fromFull == toFull {
		return MoveResult{}, ErrInvalidPath
	}

	info, err := os.Stat(fromFull)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return MoveResult{}, ErrNotFound
		}
		return MoveResult{}, err
	}
	if info.IsDir() {
		return MoveResult{}, ErrInvalidPath
	}
	if _, err := os.Stat(toFull); err == nil {
		return MoveResult{}, ErrAlreadyExists
	} else if !errors.Is(err, os.ErrNotExist) {
		return MoveResult{}, err
	}

	if err := os.MkdirAll(filepath.Dir(toFull), 0o700); err != nil {
		return MoveResult{}, fmt.Errorf("create parent dir: %w", err)
	}
	if err := os.Rename(fromFull, toFull); err != nil {
		return MoveResult{}, fmt.Errorf("move note: %w", err)
	}

	// Rewrite links AFTER the file has moved: if the rename fails there
	// is nothing to repoint, and if the rewrite fails the note is still
	// at its new path — a stale link is recoverable, a half-moved note
	// is not. Failures are reported, never rolled back.
	res, rewriteErr := v.rewriteLinks(ctx, fromRel, toRel)
	res.From, res.To = fromRel, toRel

	body, err := os.ReadFile(toFull)
	if err == nil {
		st, _ := os.Stat(toFull)
		res.Note = Note{
			Path:     toRel,
			Title:    titleFromBody(string(body), toRel),
			Modified: st.ModTime(),
			Size:     st.Size(),
		}
	}
	return res, rewriteErr
}

// rewriteLinks repoints every wiki-link that referenced fromRel at
// toRel. It mirrors Backlinks' matching rules, because a link the
// backlinks panel shows and the rename doesn't fix is exactly the
// surprise this feature exists to avoid.
func (v *Vault) rewriteLinks(ctx context.Context, fromRel, toRel string) (MoveResult, error) {
	out := MoveResult{RewrittenIn: []string{}}

	fromTarget := strings.TrimSuffix(fromRel, ".md")
	fromBase := filepath.Base(fromTarget)
	toTarget := strings.TrimSuffix(toRel, ".md")
	toBase := filepath.Base(toTarget)

	// Capture the alias so [[old|display text]] keeps its display text.
	pat, err := regexp.Compile(
		`\[\[(` + regexp.QuoteMeta(fromTarget) + `|` + regexp.QuoteMeta(fromBase) +
			`)(\|[^\]]*)?\]\]`,
	)
	if err != nil {
		return out, err
	}

	deadline := time.Now().Add(scanTimeout)
	walkErr := filepath.WalkDir(v.root, func(path string, d fs.DirEntry, we error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if time.Now().After(deadline) {
			return errors.New("link rewrite timed out")
		}
		if we != nil {
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
		st, err := d.Info()
		if err != nil || st.Size() > scanMaxFileLen {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		body := string(raw)
		if !pat.MatchString(body) {
			return nil
		}

		// Code regions are excluded from backlinks, so they must be
		// excluded here too — rewriting a wiki-link inside a fenced
		// example would silently edit someone's documentation of the
		// syntax itself.
		count := 0
		updated := replaceOutsideCode(body, pat, func(m []string) string {
			count++
			// Preserve the reference's shape: a link written with the
			// full path keeps a full path, a bare basename stays bare.
			ref := toBase
			if m[1] == fromTarget && strings.Contains(fromTarget, "/") {
				ref = toTarget
			}
			return "[[" + ref + m[2] + "]]"
		})
		if count == 0 {
			return nil
		}
		if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
			return nil
		}
		rel, _ := filepath.Rel(v.root, path)
		out.RewrittenIn = append(out.RewrittenIn, filepath.ToSlash(rel))
		out.LinksRewritten += count
		return nil
	})
	sort.Strings(out.RewrittenIn)
	return out, walkErr
}

// maskCodeRegions blanks out fenced and inline code, byte for byte, so
// offsets in the result map exactly onto the original.
//
// stripCodeRegions (used by Backlinks) can't be reused here: it
// collapses each fenced line to a single "\n" and rewrites inline-code
// runes one byte at a time, so its output is a different length than
// its input. That is harmless when you only want to know *whether* a
// line matched, and fatal when you intend to splice a replacement in at
// a matched offset. Masking in place keeps every position true.
func maskCodeRegions(body string) string {
	out := []byte(body)
	blank := func(from, to int) {
		for i := from; i < to; i++ {
			out[i] = ' '
		}
	}
	inFence := false
	for i := 0; i < len(out); {
		j := i
		for j < len(out) && out[j] != '\n' {
			j++
		}
		line := strings.TrimSpace(string(out[i:j]))
		switch {
		case strings.HasPrefix(line, "```"):
			inFence = !inFence
			blank(i, j)
		case inFence:
			blank(i, j)
		default:
			// Inline spans: mask the backticks and everything between.
			inCode := false
			for k := i; k < j; k++ {
				if out[k] == '`' {
					inCode = !inCode
					out[k] = ' '
					continue
				}
				if inCode {
					out[k] = ' '
				}
			}
		}
		i = j + 1
	}
	return string(out)
}

// replaceOutsideCode applies repl to every match of pat that falls
// outside a code region.
func replaceOutsideCode(body string, pat *regexp.Regexp, repl func([]string) string) string {
	masked := maskCodeRegions(body)
	if len(masked) != len(body) {
		// Can't happen — masking is in-place — but splicing at a wrong
		// offset would corrupt someone's notes, so verify rather than
		// assume.
		return body
	}
	var b strings.Builder
	last := 0
	for _, loc := range pat.FindAllStringSubmatchIndex(masked, -1) {
		groups := make([]string, 0, len(loc)/2)
		for i := 0; i < len(loc); i += 2 {
			if loc[i] < 0 {
				groups = append(groups, "")
				continue
			}
			groups = append(groups, body[loc[i]:loc[i+1]])
		}
		b.WriteString(body[last:loc[0]])
		b.WriteString(repl(groups))
		last = loc[1]
	}
	b.WriteString(body[last:])
	return b.String()
}
