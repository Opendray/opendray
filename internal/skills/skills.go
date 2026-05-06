// Package skills implements the agent-skills runtime: the unit a user
// (or upstream) bundles together as "instructions + optional tools"
// for an AI agent to lazy-load on demand.
//
// Storage model — same philosophy as the notes vault:
//
//	<vault>/skills/<id>/SKILL.md     # user / shared skills (filesystem)
//	//go:embed builtin/<id>/SKILL.md # built-in skills shipped in the binary
//
// Vault entries override built-ins of the same id, so users can customise
// any shipped skill by dropping a same-named directory in their vault.
//
// At session spawn time the catalog adapter calls Inject to materialise
// enabled skills under the per-session scratch dir; downstream agents
// (Claude Code's native Agent Skills, Codex/Gemini via instructions.md)
// then discover and lazy-load them.
package skills

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed builtin
var builtinFS embed.FS

// Skill is the in-memory view of one skill folder.
type Skill struct {
	ID          string
	Name        string
	Description string // one-line summary, used for the Tier 1 index
	Body        string // full SKILL.md including frontmatter
	Source      string // "builtin" or "vault"
	// SourcePath is the fs path the skill was loaded from when source
	// is "vault" — empty for built-ins.
	SourcePath string
}

// Loader walks built-ins + the vault and returns a deduped list keyed
// by id. Vault wins on conflict so users can override shipped skills
// without editing the binary.
type Loader struct {
	vaultRoot string // absolute path to <vault>/skills (may not exist)
}

// NewLoader points the loader at a directory that contains one
// subfolder per vault skill (e.g. `<dir>/notes-keeper/SKILL.md`).
// Caller (app.go) is responsible for resolving from VaultConfig —
// no path magic happens here. Empty `dir` disables vault overlays
// entirely; only built-ins are returned in that case.
func NewLoader(dir string) *Loader {
	return &Loader{vaultRoot: dir}
}

// VaultRoot returns the path Loader looks at for user skills (may be
// empty / nonexistent).
func (l *Loader) VaultRoot() string { return l.vaultRoot }

// BuiltinIDs returns the set of skill ids that are embedded in the
// binary. Used by handlers to flag vault skills that shadow a built-in
// (so the UI can offer a "Reset to built-in" action).
func (l *Loader) BuiltinIDs() (map[string]bool, error) {
	bi, err := loadBuiltins()
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(bi))
	for _, s := range bi {
		out[s.ID] = true
	}
	return out, nil
}

// List returns all skills (built-in + vault) deduped by id (vault wins).
// Sorted alphabetically by id for stable output in CLI / UI.
func (l *Loader) List() ([]Skill, error) {
	out := map[string]Skill{}

	// Built-ins first, vault overlays.
	bi, err := loadBuiltins()
	if err != nil {
		return nil, err
	}
	for _, s := range bi {
		out[s.ID] = s
	}
	if l.vaultRoot != "" {
		v, err := loadVault(l.vaultRoot)
		if err != nil {
			return nil, err
		}
		for _, s := range v {
			out[s.ID] = s
		}
	}

	skills := make([]Skill, 0, len(out))
	for _, s := range out {
		skills = append(skills, s)
	}
	sort.Slice(skills, func(i, j int) bool { return skills[i].ID < skills[j].ID })
	return skills, nil
}

// Get returns the skill by id, vault overriding built-in.
func (l *Loader) Get(id string) (Skill, error) {
	if l.vaultRoot != "" {
		s, err := loadOne(os.DirFS(l.vaultRoot), id, "vault")
		if err == nil {
			s.SourcePath = filepath.Join(l.vaultRoot, id)
			return s, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return Skill{}, err
		}
	}
	sub, err := fs.Sub(builtinFS, "builtin")
	if err != nil {
		return Skill{}, err
	}
	return loadOne(sub, id, "builtin")
}

// loadBuiltins reads every embedded skill folder under builtin/.
func loadBuiltins() ([]Skill, error) {
	sub, err := fs.Sub(builtinFS, "builtin")
	if err != nil {
		return nil, err
	}
	entries, err := fs.ReadDir(sub, ".")
	if err != nil {
		return nil, err
	}
	out := []Skill{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		s, err := loadOne(sub, e.Name(), "builtin")
		if err != nil {
			continue // skip malformed built-ins rather than fail-hard
		}
		out = append(out, s)
	}
	return out, nil
}

// loadVault reads SKILL.md from each immediate child directory of the
// vault skills root.
func loadVault(root string) ([]Skill, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	out := []Skill{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		s, err := loadOne(os.DirFS(root), e.Name(), "vault")
		if err != nil {
			continue
		}
		s.SourcePath = filepath.Join(root, e.Name())
		out = append(out, s)
	}
	return out, nil
}

// loadOne reads <fsys>/<id>/SKILL.md and parses its frontmatter.
func loadOne(fsys fs.FS, id, source string) (Skill, error) {
	data, err := fs.ReadFile(fsys, filepath.ToSlash(filepath.Join(id, "SKILL.md")))
	if err != nil {
		return Skill{}, err
	}
	body := string(data)
	name, desc := parseFrontmatter(body)
	if name == "" {
		name = id
	}
	return Skill{
		ID:          id,
		Name:        name,
		Description: desc,
		Body:        body,
		Source:      source,
	}, nil
}

// parseFrontmatter pulls just the `name` and `description` keys out of
// a leading `---` … `---` block. Tolerant of minor formatting; we
// don't pull in a real YAML dep for two fields.
func parseFrontmatter(body string) (name, description string) {
	if !strings.HasPrefix(body, "---") {
		return "", ""
	}
	end := strings.Index(body[3:], "---")
	if end < 0 {
		return "", ""
	}
	header := body[3 : 3+end]
	for _, raw := range strings.Split(header, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		switch k {
		case "name":
			name = v
		case "description":
			description = v
		}
	}
	return name, description
}

// IndexPrompt returns the Tier 1 system-prompt block listing every
// available skill. Designed to be passed via `claude --append-system-
// prompt <text>` (or codex/gemini equivalents) — costs ~30 tokens per
// skill so it's safe to ship in every session. The agent loads full
// instructions on demand by invoking `opendray skill describe <id>`
// through its Bash tool.
func IndexPrompt(skills []Skill) string {
	if len(skills) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("# opendray skills available\n\n")
	b.WriteString("Each one is a self-contained capability you can invoke when relevant. ")
	b.WriteString("To get full instructions for a skill, run `opendray skill describe <id>` ")
	b.WriteString("via your Bash tool — it returns the SKILL.md verbatim. ")
	b.WriteString("Activate a skill only when its description matches what the user just asked for.\n\n")
	for _, s := range skills {
		desc := s.Description
		if desc == "" {
			desc = "(no description)"
		}
		fmt.Fprintf(&b, "- **%s** — %s\n", s.ID, desc)
	}
	return b.String()
}

// MirrorClaudeConfig builds a per-session CLAUDE_CONFIG_DIR at destDir
// by symlinking every top-level entry of srcDir (the user's account
// config dir) into destDir, then layering opendray-managed skills as
// REAL directories under destDir/skills/. The agent sees a unified
// config — its own settings, projects history, credentials, AND our
// injected skills — but nothing is written back to srcDir.
//
// When the session ends and the manager removes destDir, every
// opendray skill goes with it. The user's account dir stays untouched.
//
// Symlink rules:
//   - srcDir/<entry>     → destDir/<entry>      (any non-skills entry)
//   - srcDir/skills/     → real dir at destDir/skills/
//   - srcDir/skills/<id> → destDir/skills/<id> (preserves user skills)
//   - opendray skill   → real dir at destDir/skills/<id>/SKILL.md
//   - opendray-injected skills shadow same-id entries from srcDir
//     (so user can override an opendray skill by creating one with
//     matching id in their account dir, but only if opendray's loader
//     decides to skip it — currently we always overwrite)
//
// srcDir == "" creates a fresh destDir (no mirroring) and only writes
// opendray skills + the Tier 1 index — used for the no-account path.
func MirrorClaudeConfig(srcDir, destDir string, ourSkills []Skill) error {
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}

	// Mirror everything except `skills` — that one we hand-build below
	// so we can layer in our own without touching the user's dir.
	if srcDir != "" {
		entries, err := os.ReadDir(srcDir)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("read account dir: %w", err)
		}
		for _, e := range entries {
			if e.Name() == "skills" {
				continue
			}
			src := filepath.Join(srcDir, e.Name())
			dst := filepath.Join(destDir, e.Name())
			if err := os.Symlink(src, dst); err != nil && !errors.Is(err, os.ErrExist) {
				return fmt.Errorf("symlink %s: %w", e.Name(), err)
			}
		}
	}

	// Build the skills dir: real, with one symlink per existing user
	// skill, then real dirs for our injected skills (shadowing if id
	// collides — opendray skills win for that session only).
	skillsDir := filepath.Join(destDir, "skills")
	if err := os.MkdirAll(skillsDir, 0o700); err != nil {
		return fmt.Errorf("create skills dir: %w", err)
	}

	occupied := map[string]bool{}
	for _, s := range ourSkills {
		occupied[s.ID] = true
	}
	if srcDir != "" {
		userSkillsSrc := filepath.Join(srcDir, "skills")
		userEntries, err := os.ReadDir(userSkillsSrc)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("read user skills: %w", err)
		}
		for _, e := range userEntries {
			if !e.IsDir() {
				continue
			}
			if occupied[e.Name()] {
				continue // we'll write our own at this id
			}
			src := filepath.Join(userSkillsSrc, e.Name())
			dst := filepath.Join(skillsDir, e.Name())
			if err := os.Symlink(src, dst); err != nil && !errors.Is(err, os.ErrExist) {
				return fmt.Errorf("symlink user skill %s: %w", e.Name(), err)
			}
		}
	}

	for _, s := range ourSkills {
		dir := filepath.Join(skillsDir, s.ID)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create skill dir %s: %w", s.ID, err)
		}
		path := filepath.Join(dir, "SKILL.md")
		if err := os.WriteFile(path, []byte(s.Body), 0o600); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}

	return nil
}

// Inject is the variant for fresh per-session config dirs (no existing
// account state to mirror). Writes opendray skills + a Tier 1 index
// CLAUDE.md describing what's available — used for codex / gemini and
// for the no-account claude path. NOT used when an account is bound;
// MirrorClaudeConfig handles that case.
func Inject(skills []Skill, destDir string) error {
	if err := MirrorClaudeConfig("", destDir, skills); err != nil {
		return err
	}
	if len(skills) == 0 {
		return nil
	}
	if err := os.WriteFile(
		filepath.Join(destDir, "CLAUDE.md"),
		[]byte(buildIndex(skills)),
		0o600,
	); err != nil {
		return fmt.Errorf("write CLAUDE.md index: %w", err)
	}
	return nil
}

func buildIndex(skills []Skill) string {
	var b strings.Builder
	b.WriteString("# Available skills\n\n")
	b.WriteString("This session has the following opendray skills available. Each one is a\n")
	b.WriteString("self-contained capability you can invoke when needed. Read the full instructions\n")
	b.WriteString("for a skill with `opendray skill describe <id>` (or use the built-in Skill\n")
	b.WriteString("tool if your runtime supports it — SKILL.md files are present under\n")
	b.WriteString("`$CLAUDE_CONFIG_DIR/skills/<id>/SKILL.md`).\n\n")
	for _, s := range skills {
		desc := s.Description
		if desc == "" {
			desc = "(no description)"
		}
		fmt.Fprintf(&b, "- **%s** — %s\n", s.ID, desc)
	}
	return b.String()
}
