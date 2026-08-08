package vaultgit

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// The Vault repo is the operator's documents. Everything else that
// happens to sit in the working tree — Finder droppings, Obsidian's
// per-machine workspace state, and (on a pre-split install) opendray's
// own skills/ and mcp/ roots — has no business being pushed to their
// remote. On the machine that prompted this, `skills/secretary/
// SKILL.md` and a `.claude/` memory tree had already been committed to
// a private docs repo, purely because they shared a directory.
//
// So opendray maintains a delimited block in .gitignore rather than
// writing the file once at init and never looking again. The block is
// rewritten on every commit, which means:
//
//   - existing repos get the rules, not just newly-initialised ones
//   - the set tracks the resolved layout, so an install that later
//     moves skills/ out stops carrying a stale rule
//   - everything outside the markers is the operator's, untouched
//
// Ignoring cannot untrack what is already committed; that needs
// `git rm --cached` and is the operator's call, not something to do to
// someone's repo behind their back.

const (
	ignoreBegin = "# >>> opendray managed — edits inside this block are overwritten >>>"
	ignoreEnd   = "# <<< opendray managed <<<"
)

// baseIgnores are unconditional: none of these are documents.
var baseIgnores = []string{
	".DS_Store",
	".cache/",
	".trash/",
	".obsidian/workspace.json",
	".obsidian/workspace-mobile.json",
}

// ensureManagedIgnore rewrites opendray's block in <root>/.gitignore.
// nested names the machinery directories that resolve inside the repo,
// as absolute paths; they are recorded repo-relative.
func ensureManagedIgnore(root string, nested []string) error {
	rules := append([]string{}, baseIgnores...)
	for _, dir := range nested {
		rel, err := filepath.Rel(root, dir)
		if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
			continue
		}
		rules = append(rules, filepath.ToSlash(rel)+"/")
	}

	path := filepath.Join(root, ".gitignore")
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read .gitignore: %w", err)
	}

	want := renderBlock(rules)
	next := replaceBlock(string(existing), want)
	if next == string(existing) {
		return nil
	}
	if err := os.WriteFile(path, []byte(next), 0o600); err != nil {
		return fmt.Errorf("write .gitignore: %w", err)
	}
	return nil
}

func renderBlock(rules []string) string {
	var b strings.Builder
	b.WriteString(ignoreBegin)
	b.WriteString("\n")
	for _, r := range rules {
		b.WriteString(r)
		b.WriteString("\n")
	}
	b.WriteString(ignoreEnd)
	b.WriteString("\n")
	return b.String()
}

// replaceBlock swaps the managed block for want, appending it when the
// file has none. Content outside the markers is preserved byte for
// byte — it is the operator's file, we are a guest in it.
func replaceBlock(existing, want string) string {
	if existing == "" {
		return want
	}
	var before, after []string
	state := 0 // 0 = before block, 1 = inside, 2 = after
	found := false
	sc := bufio.NewScanner(strings.NewReader(existing))
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		switch state {
		case 0:
			if strings.TrimSpace(line) == ignoreBegin {
				state, found = 1, true
				continue
			}
			before = append(before, line)
		case 1:
			if strings.TrimSpace(line) == ignoreEnd {
				state = 2
			}
			continue
		default:
			after = append(after, line)
		}
	}
	// An unterminated block (someone deleted the end marker) still ends
	// the scan in state 1; treating it as consumed is right — the
	// alternative is nesting a fresh block inside a broken one.
	if !found {
		out := strings.TrimRight(existing, "\n")
		if out != "" {
			out += "\n\n"
		}
		return out + want
	}

	var b strings.Builder
	if len(before) > 0 {
		b.WriteString(strings.Join(before, "\n"))
		b.WriteString("\n")
	}
	b.WriteString(want)
	if len(after) > 0 {
		b.WriteString(strings.Join(after, "\n"))
		b.WriteString("\n")
	}
	return b.String()
}
