// Package notes implements the file-system-backed notes vault that
// powers the Inspector's Notes tab and is exposed to AI agents via
// the `opendray notes` CLI subcommand.
//
// Storage model: a configurable root (default ~/.opendray/vault/notes)
// holding a tree of .md files. Conventional subdirectories (daily/,
// projects/, library/) are auto-created on first write but the layout
// is otherwise free-form — users can drop their existing Obsidian
// vault here and it'll work without changes.
//
// All operations go through Vault, which enforces a strict path jail
// against the resolved root. The HTTP layer and CLI both wrap Vault.
package notes

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound       = errors.New("note not found")
	ErrPathEscape     = errors.New("path escapes the vault root")
	ErrInvalidPath    = errors.New("invalid path")
	ErrNotMarkdown    = errors.New("path must end in .md")
	ErrAlreadyExists  = errors.New("note already exists")
)

// Note is the lightweight metadata view used in list / search results.
type Note struct {
	// Path is vault-relative, forward-slash separated, e.g. "projects/foo.md".
	Path     string    `json:"path"`
	Title    string    `json:"title"`
	Modified time.Time `json:"modified"`
	Size     int64     `json:"size"`
}

// FullNote is what /read returns: metadata plus the body bytes.
type FullNote struct {
	Note
	Body string `json:"body"`
}

// Vault is the path-jailed gateway to the notes filesystem. Construct
// once per process; all operations are concurrency-safe by virtue of
// being independent FS calls (no shared in-memory state in this phase
// — index lands in Phase 4).
type Vault struct {
	root            string // canonical absolute path to the notes root
	personalPrefix  string // default subfolder for personal scratchpads
	projectsPrefix  string // default subfolder for AI project docs

	projectMapMu sync.Mutex // guards reads/writes of projectMapFilename
}

// Options configure the per-vault defaults that drive path derivation
// (PersonalPath, ProjectDir). Empty fields fall back to opendray's
// classic conventions (`personal`, `projects`).
type Options struct {
	PersonalPrefix string
	ProjectsPrefix string
}

// New resolves and creates the notes root directory. Pass the
// directory you want notes to live IN — caller (app.go) is responsible
// for picking it from config (`<root>/notes` for the default opendray
// layout, or a user-specified path that points at an existing
// Obsidian vault, etc.). The directory is created with mode 0o700
// when missing so first-run is zero-config.
func New(notesRoot string, opts Options) (*Vault, error) {
	if notesRoot == "" {
		return nil, fmt.Errorf("notes root is empty")
	}
	notesRoot, err := expand(notesRoot)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(notesRoot, 0o700); err != nil {
		return nil, fmt.Errorf("create notes root: %w", err)
	}
	abs, err := filepath.Abs(notesRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve notes root: %w", err)
	}
	// Resolve symlinks once so the jail check below stays consistent
	// even if the user pointed `root` at a symlinked path.
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	personal := strings.Trim(opts.PersonalPrefix, "/")
	if personal == "" {
		personal = "personal"
	}
	projects := strings.Trim(opts.ProjectsPrefix, "/")
	if projects == "" {
		projects = "projects"
	}
	return &Vault{
		root:           filepath.Clean(abs),
		personalPrefix: personal,
		projectsPrefix: projects,
	}, nil
}

// PersonalPrefix / ProjectsPrefix expose the configured defaults so
// callers (handlers, CLI) can surface them in /info responses or the
// "default location" hint of the per-cwd override UI.
func (v *Vault) PersonalPrefix() string { return v.personalPrefix }
func (v *Vault) ProjectsPrefix() string { return v.projectsPrefix }

// Root returns the canonical absolute path to the notes directory.
// Useful for surfacing in /api/v1/notes/info and the CLI describe.
func (v *Vault) Root() string { return v.root }

// resolve takes a vault-relative path (forward slashes, no leading /)
// and returns the absolute filesystem path after the jail check. Any
// attempt to escape via `..` or absolute paths is rejected.
func (v *Vault) resolve(rel string) (string, error) {
	if rel == "" {
		return "", ErrInvalidPath
	}
	if filepath.IsAbs(rel) {
		return "", ErrInvalidPath
	}
	clean := filepath.Clean(rel)
	if clean == "." || strings.HasPrefix(clean, "..") || strings.Contains(clean, "..") {
		return "", ErrPathEscape
	}
	full := filepath.Join(v.root, clean)
	// Belt-and-suspenders: even after Join + Clean, verify the result
	// is still under root. Catches edge cases like rel containing
	// embedded NULs or weird unicode.
	relAfter, err := filepath.Rel(v.root, full)
	if err != nil || strings.HasPrefix(relAfter, "..") || relAfter == ".." {
		return "", ErrPathEscape
	}
	return full, nil
}

// requireMarkdown enforces the .md extension on writeable paths so
// the vault doesn't accumulate random files via the API. Reads are
// less restrictive — anything inside the vault is fair game.
func requireMarkdown(rel string) error {
	if !strings.HasSuffix(strings.ToLower(rel), ".md") {
		return ErrNotMarkdown
	}
	return nil
}

// List returns every .md file under the notes root, optionally
// filtered to a path prefix (e.g. "projects/" returns only project
// notes). Sorted by modified time descending so recent work surfaces
// first in pickers / autocompletes.
func (v *Vault) List(prefix string) ([]Note, error) {
	out := []Note{}
	prefix = strings.TrimPrefix(prefix, "/")
	err := filepath.WalkDir(v.root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			// A single permission error shouldn't kill the whole walk.
			if errors.Is(walkErr, os.ErrPermission) {
				return nil
			}
			return walkErr
		}
		if d.IsDir() {
			// Skip hidden subdirs (e.g. .git, .obsidian) — they're not
			// part of the notes themselves.
			if path != v.root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		rel, err := filepath.Rel(v.root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if prefix != "" && !strings.HasPrefix(rel, prefix) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		out = append(out, Note{
			Path:     rel,
			Title:    titleFromPath(rel),
			Modified: info.ModTime(),
			Size:     info.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Modified.After(out[j].Modified)
	})
	return out, nil
}

// Read returns the full markdown body. Returns ErrNotFound if the
// path is missing rather than a generic os error so callers can
// branch on that case cleanly.
func (v *Vault) Read(rel string) (FullNote, error) {
	full, err := v.resolve(rel)
	if err != nil {
		return FullNote{}, err
	}
	info, err := os.Stat(full)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return FullNote{}, ErrNotFound
		}
		return FullNote{}, err
	}
	if info.IsDir() {
		return FullNote{}, ErrInvalidPath
	}
	body, err := os.ReadFile(full)
	if err != nil {
		return FullNote{}, err
	}
	bodyStr := string(body)
	return FullNote{
		Note: Note{
			Path:     rel,
			Title:    titleFromBody(bodyStr, rel),
			Modified: info.ModTime(),
			Size:     info.Size(),
		},
		Body: bodyStr,
	}, nil
}

// Write replaces the file at rel with body. Creates parent
// directories as needed. mode=0o600 because notes can hold private
// content and the vault is single-user.
func (v *Vault) Write(rel string, body string) (Note, error) {
	if err := requireMarkdown(rel); err != nil {
		return Note{}, err
	}
	full, err := v.resolve(rel)
	if err != nil {
		return Note{}, err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
		return Note{}, fmt.Errorf("create parent dir: %w", err)
	}
	if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
		return Note{}, fmt.Errorf("write note: %w", err)
	}
	info, err := os.Stat(full)
	if err != nil {
		return Note{}, err
	}
	return Note{
		Path:     rel,
		Title:    titleFromBody(body, rel),
		Modified: info.ModTime(),
		Size:     info.Size(),
	}, nil
}

// Append concatenates body to an existing note (creates if missing).
// A leading newline is inserted when the existing file doesn't end in
// one, so daily-log style appends always start on their own line.
func (v *Vault) Append(rel string, body string) (Note, error) {
	if err := requireMarkdown(rel); err != nil {
		return Note{}, err
	}
	full, err := v.resolve(rel)
	if err != nil {
		return Note{}, err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
		return Note{}, fmt.Errorf("create parent dir: %w", err)
	}
	existing, err := os.ReadFile(full)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Note{}, err
	}
	var combined []byte
	if len(existing) == 0 {
		combined = []byte(body)
	} else {
		combined = existing
		if combined[len(combined)-1] != '\n' {
			combined = append(combined, '\n')
		}
		combined = append(combined, body...)
	}
	if err := os.WriteFile(full, combined, 0o600); err != nil {
		return Note{}, err
	}
	info, _ := os.Stat(full)
	return Note{
		Path:     rel,
		Title:    titleFromBody(string(combined), rel),
		Modified: info.ModTime(),
		Size:     info.Size(),
	}, nil
}

// Delete removes a note. Refuses to delete directories — use the
// dedicated rmdir flow if/when needed.
func (v *Vault) Delete(rel string) error {
	full, err := v.resolve(rel)
	if err != nil {
		return err
	}
	info, err := os.Stat(full)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrNotFound
		}
		return err
	}
	if info.IsDir() {
		return ErrInvalidPath
	}
	return os.Remove(full)
}

// titleFromBody pulls the first H1 from the body (ignoring frontmatter)
// and falls back to the basename without extension. Lets the picker
// show "Quarterly Plan" instead of "2026-q2-plan.md".
func titleFromBody(body, rel string) string {
	lines := strings.SplitN(body, "\n", 200)
	inFrontmatter := false
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if i == 0 && t == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter {
			if t == "---" {
				inFrontmatter = false
			}
			continue
		}
		if strings.HasPrefix(t, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(t, "# "))
		}
	}
	return titleFromPath(rel)
}

func titleFromPath(rel string) string {
	base := filepath.Base(rel)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// expand resolves ~/ and ${HOME} prefixes against the calling user's
// home dir. Falls back to leaving the path unchanged when the user
// info isn't resolvable (rare but possible in container environments).
func expand(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", errors.New("empty path")
	}
	if strings.HasPrefix(p, "~/") || p == "~" {
		home, err := userHome()
		if err == nil {
			if p == "~" {
				return home, nil
			}
			return filepath.Join(home, p[2:]), nil
		}
	}
	if !filepath.IsAbs(p) {
		abs, err := filepath.Abs(p)
		if err == nil {
			return abs, nil
		}
	}
	return p, nil
}

func userHome() (string, error) {
	if u, err := user.Current(); err == nil && u.HomeDir != "" {
		return u.HomeDir, nil
	}
	return os.UserHomeDir()
}

// DailyPath returns the conventional path for today's daily note.
// Used by the CLI's `notes daily` shortcut and the future Inspector
// "Today" button. Daily notes live at daily/YYYY-MM-DD.md regardless
// of the personal/projects prefix config — daily is its own concept.
func DailyPath(t time.Time) string {
	return fmt.Sprintf("daily/%s.md", t.Format("2006-01-02"))
}

// PersonalPath returns the personal scratchpad path for `basename`,
// using the vault's configured prefix. Sanitises basename so weird
// cwds still produce sane file names.
func (v *Vault) PersonalPath(basename string) string {
	return v.personalPrefix + "/" + sanitiseBasename(basename) + ".md"
}

// ProjectDir returns the directory for a project's notes using the
// vault's configured prefix. Per-cwd overrides (from the project
// mapping store) take precedence — call ResolvedProjectDir(cwd) for
// that behaviour. This is the "default location" used as a fallback.
func (v *Vault) ProjectDir(basename string) string {
	return v.projectsPrefix + "/" + sanitiseBasename(basename)
}

// ProjectPath returns the conventional README inside ProjectDir.
func (v *Vault) ProjectPath(basename string) string {
	return v.ProjectDir(basename) + "/README.md"
}

// Backwards-compat package-level shims used by `cmd/opendray notes`.
// These hardcode the original "personal"/"projects" prefixes — the
// CLI doesn't have access to the configured Vault when it runs.
func ProjectPath(basename string) string {
	return ProjectDir(basename) + "/README.md"
}

func ProjectDir(basename string) string {
	return "projects/" + sanitiseBasename(basename)
}

func sanitiseBasename(basename string) string {
	clean := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_' || r == '.':
			return r
		default:
			return '-'
		}
	}, basename)
	if clean == "" {
		clean = "untitled"
	}
	return clean
}
