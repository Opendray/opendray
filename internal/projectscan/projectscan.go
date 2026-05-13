// Package projectscan auto-detects a project's tech stack and key
// structure, then persists a markdown summary into project_docs
// kind='tech_stack'. M16 — closes the "new agent has to re-index
// the project from scratch" gap surfaced by operator feedback.
//
// Cheap, deterministic, no LLM: just look for well-known marker
// files (go.mod / package.json / pubspec.yaml / pyproject.toml /
// Cargo.toml / etc.) and a shallow directory walk skipping
// vendored / generated paths.
//
// Triggered at session spawn time (catalog adapter) and on a slow
// periodic refresh (default 6h) for any cwd that has at least one
// stored memory or doc. Operators can also POST /project-scan/run
// for an on-demand refresh.
package projectscan

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/projectdoc"
)

// Stack describes one detected runtime / framework.
type Stack struct {
	Name    string // friendly label, e.g. "Go", "Flutter", "Node.js"
	Version string // optional, e.g. "1.22" — empty when not detectable
	Source  string // file the detector keyed off, e.g. "go.mod"
}

// ProjectInfo is the structured scan result. Rendered to markdown
// by RenderMarkdown before being persisted.
type ProjectInfo struct {
	Cwd        string
	ScannedAt  time.Time
	GitBranch  string // current branch when cwd is a git repo, else ""
	GitHead    string // short SHA of HEAD, else ""
	Stacks     []Stack
	EntryFiles []string // notable entry points (cmd/*, main.go, etc.)
	KeyDirs    []DirEntry
}

// DirEntry is one top-level directory with a brief description if
// we can infer one (e.g. "cmd" = "entry point binaries").
type DirEntry struct {
	Name        string
	IsDir       bool
	Description string // optional, populated for well-known names
}

// Scanner runs detectors against a cwd.
type Scanner struct {
	log *slog.Logger
}

// New builds a Scanner. log is optional.
func New(log *slog.Logger) *Scanner {
	if log == nil {
		log = slog.Default()
	}
	return &Scanner{log: log.With("component", "projectscan")}
}

// Scan inspects cwd and returns a populated ProjectInfo. Errors
// from individual detectors are logged and swallowed — partial
// information is better than nothing.
func (s *Scanner) Scan(ctx context.Context, cwd string) (ProjectInfo, error) {
	info := ProjectInfo{
		Cwd:       cwd,
		ScannedAt: time.Now().UTC(),
	}
	if strings.TrimSpace(cwd) == "" {
		return info, errors.New("projectscan: empty cwd")
	}
	st, err := os.Stat(cwd)
	if err != nil {
		return info, fmt.Errorf("projectscan: stat cwd: %w", err)
	}
	if !st.IsDir() {
		return info, fmt.Errorf("projectscan: cwd is not a directory: %s", cwd)
	}

	// Detect each known stack. Order matters only for rendering;
	// detectors don't depend on each other.
	for _, d := range detectors {
		if stack, ok := d(cwd, s.log); ok {
			info.Stacks = append(info.Stacks, stack)
		}
	}
	sort.Slice(info.Stacks, func(i, j int) bool {
		return info.Stacks[i].Name < info.Stacks[j].Name
	})

	info.GitBranch, info.GitHead = readGitHead(cwd)
	info.EntryFiles = findEntryFiles(cwd)
	info.KeyDirs = listKeyDirs(cwd)
	return info, nil
}

// ── Detectors ──────────────────────────────────────────────────

type detectorFunc func(cwd string, log *slog.Logger) (Stack, bool)

// detectors is the ordered list of stack probes. Each returns
// (stack, true) when it finds a match.
var detectors = []detectorFunc{
	detectGo,
	detectNode,
	detectFlutter,
	detectPythonPyproject,
	detectPythonRequirements,
	detectRust,
	detectRuby,
	detectJavaMaven,
	detectJavaGradle,
	detectDockerCompose,
	detectPostgres,
}

func detectGo(cwd string, log *slog.Logger) (Stack, bool) {
	path := filepath.Join(cwd, "go.mod")
	f, err := os.Open(path)
	if err != nil {
		return Stack{}, false
	}
	defer f.Close()
	var version string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "go ") && version == "" {
			version = strings.TrimPrefix(line, "go ")
		}
	}
	return Stack{Name: "Go", Version: version, Source: "go.mod"}, true
}

func detectNode(cwd string, log *slog.Logger) (Stack, bool) {
	path := filepath.Join(cwd, "package.json")
	if _, err := os.Stat(path); err != nil {
		// also check workspace-style: app/web/package.json etc.
		return Stack{}, false
	}
	// Don't parse the JSON — just note the file exists. A separate
	// pnpm/yarn lockfile detector below could refine.
	return Stack{Name: "Node.js", Version: "", Source: "package.json"}, true
}

func detectFlutter(cwd string, log *slog.Logger) (Stack, bool) {
	// Flutter projects can be nested (e.g. app/mobile/pubspec.yaml
	// in a monorepo). Walk one level down for pubspec.yaml.
	candidates := []string{
		filepath.Join(cwd, "pubspec.yaml"),
		filepath.Join(cwd, "app", "mobile", "pubspec.yaml"),
		filepath.Join(cwd, "mobile", "pubspec.yaml"),
	}
	for _, path := range candidates {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		hasFlutter := false
		var sdk string
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "flutter:") || strings.Contains(line, "flutter:") {
				hasFlutter = true
			}
			if strings.HasPrefix(line, "sdk:") && sdk == "" {
				sdk = strings.TrimSpace(strings.TrimPrefix(line, "sdk:"))
			}
		}
		if hasFlutter {
			rel, _ := filepath.Rel(cwd, path)
			return Stack{Name: "Flutter", Version: sdk, Source: rel}, true
		}
	}
	return Stack{}, false
}

func detectPythonPyproject(cwd string, log *slog.Logger) (Stack, bool) {
	if _, err := os.Stat(filepath.Join(cwd, "pyproject.toml")); err != nil {
		return Stack{}, false
	}
	return Stack{Name: "Python", Version: "", Source: "pyproject.toml"}, true
}

func detectPythonRequirements(cwd string, log *slog.Logger) (Stack, bool) {
	if _, err := os.Stat(filepath.Join(cwd, "requirements.txt")); err != nil {
		return Stack{}, false
	}
	return Stack{Name: "Python", Version: "", Source: "requirements.txt"}, true
}

func detectRust(cwd string, log *slog.Logger) (Stack, bool) {
	if _, err := os.Stat(filepath.Join(cwd, "Cargo.toml")); err != nil {
		return Stack{}, false
	}
	return Stack{Name: "Rust", Version: "", Source: "Cargo.toml"}, true
}

func detectRuby(cwd string, log *slog.Logger) (Stack, bool) {
	if _, err := os.Stat(filepath.Join(cwd, "Gemfile")); err != nil {
		return Stack{}, false
	}
	return Stack{Name: "Ruby", Version: "", Source: "Gemfile"}, true
}

func detectJavaMaven(cwd string, log *slog.Logger) (Stack, bool) {
	if _, err := os.Stat(filepath.Join(cwd, "pom.xml")); err != nil {
		return Stack{}, false
	}
	return Stack{Name: "Java (Maven)", Version: "", Source: "pom.xml"}, true
}

func detectJavaGradle(cwd string, log *slog.Logger) (Stack, bool) {
	for _, name := range []string{"build.gradle", "build.gradle.kts"} {
		if _, err := os.Stat(filepath.Join(cwd, name)); err == nil {
			return Stack{Name: "Java (Gradle)", Version: "", Source: name}, true
		}
	}
	return Stack{}, false
}

func detectDockerCompose(cwd string, log *slog.Logger) (Stack, bool) {
	for _, name := range []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"} {
		if _, err := os.Stat(filepath.Join(cwd, name)); err == nil {
			return Stack{Name: "Docker Compose", Version: "", Source: name}, true
		}
	}
	return Stack{}, false
}

func detectPostgres(cwd string, log *slog.Logger) (Stack, bool) {
	// Heuristic: if there's an internal/store/migrations dir with
	// .sql files, it's probably a Postgres project.
	migrations := filepath.Join(cwd, "internal", "store", "migrations")
	entries, err := os.ReadDir(migrations)
	if err != nil {
		return Stack{}, false
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") {
			return Stack{Name: "PostgreSQL", Version: "", Source: "internal/store/migrations/*.sql"}, true
		}
	}
	return Stack{}, false
}

// ── Git head ──────────────────────────────────────────────────

// readGitHead returns (branch, shortSHA) when cwd is a git repo.
// Reads .git/HEAD and the referenced ref file directly — no
// shelling out to git.
func readGitHead(cwd string) (string, string) {
	headFile := filepath.Join(cwd, ".git", "HEAD")
	data, err := os.ReadFile(headFile)
	if err != nil {
		return "", ""
	}
	head := strings.TrimSpace(string(data))
	if strings.HasPrefix(head, "ref: ") {
		ref := strings.TrimPrefix(head, "ref: ")
		branch := strings.TrimPrefix(ref, "refs/heads/")
		sha, _ := os.ReadFile(filepath.Join(cwd, ".git", ref))
		shaStr := strings.TrimSpace(string(sha))
		if len(shaStr) > 7 {
			shaStr = shaStr[:7]
		}
		return branch, shaStr
	}
	// Detached HEAD
	if len(head) > 7 {
		return "(detached)", head[:7]
	}
	return "(detached)", head
}

// ── Entry files + key dirs ────────────────────────────────────

// findEntryFiles enumerates well-known entry points at common
// locations. We don't recurse far — operators want a glance, not a
// full inventory.
func findEntryFiles(cwd string) []string {
	candidates := []string{
		"main.go",
		"cmd",
		"index.js",
		"index.ts",
		"main.py",
		"app.py",
		"src/main.rs",
		"lib/main.dart",
		"app/mobile/lib/main.dart",
	}
	var found []string
	for _, c := range candidates {
		full := filepath.Join(cwd, c)
		if info, err := os.Stat(full); err == nil {
			if info.IsDir() {
				// For cmd/*, list children one level deep.
				children, _ := os.ReadDir(full)
				for _, ch := range children {
					if ch.IsDir() {
						found = append(found, filepath.Join(c, ch.Name()))
					}
				}
			} else {
				found = append(found, c)
			}
		}
	}
	return found
}

// keyDirNames maps well-known top-level directory names to short
// descriptions. Skipped names are returned as nil entries.
var keyDirNames = map[string]string{
	"cmd":      "entry point binaries",
	"internal": "private Go packages",
	"pkg":      "public Go packages",
	"app":      "application code (mobile / web)",
	"src":      "source code",
	"lib":      "library / Dart source",
	"api":      "API contracts / OpenAPI",
	"docs":     "documentation",
	"scripts":  "build / deploy scripts",
	"tests":    "tests",
	"test":     "tests",
	"web":      "web frontend",
	"mobile":   "mobile app",
	"backend":  "backend services",
	"frontend": "frontend code",
}

// skipDirs are never listed.
var skipDirs = map[string]bool{
	".git":          true,
	".idea":         true,
	".vscode":       true,
	"node_modules":  true,
	"vendor":        true,
	".dart_tool":    true,
	"build":         true,
	"dist":          true,
	"target":        true,
	"__pycache__":   true,
	".pytest_cache": true,
	".venv":         true,
	"venv":          true,
	".next":         true,
	".nuxt":         true,
	".opendray":     true,
}

// listKeyDirs returns the top-level directories ordered alpha, with
// short descriptions when we recognise the name.
func listKeyDirs(cwd string) []DirEntry {
	entries, err := os.ReadDir(cwd)
	if err != nil {
		return nil
	}
	var out []DirEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue // hidden
		}
		if skipDirs[name] {
			continue
		}
		out = append(out, DirEntry{
			Name:        name,
			IsDir:       true,
			Description: keyDirNames[name],
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ── Render ────────────────────────────────────────────────────

// RenderMarkdown turns a ProjectInfo into a markdown body suitable
// for storing as project_docs.kind='tech_stack' or for injection
// into a system prompt.
//
// M23 — dropped the "auto-generated, do not hand-edit" header and
// the standalone Last-scanned line. The first is operator-facing
// noise (the doc is rewritten on every scan; humans wouldn't be
// editing it anyway), and the timestamp leaks into the spawn
// banner without informing agent behaviour. Scanner status is
// still queryable via the project_docs row directly.
func RenderMarkdown(info ProjectInfo) string {
	var b strings.Builder

	if len(info.Stacks) > 0 {
		b.WriteString("## Tech stack\n\n")
		for _, s := range info.Stacks {
			fmt.Fprintf(&b, "- **%s**", s.Name)
			if s.Version != "" {
				fmt.Fprintf(&b, " (%s)", s.Version)
			}
			if s.Source != "" {
				fmt.Fprintf(&b, " — detected via `%s`", s.Source)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if info.GitBranch != "" || info.GitHead != "" {
		b.WriteString("## Git\n\n")
		if info.GitBranch != "" {
			fmt.Fprintf(&b, "- Branch: `%s`\n", info.GitBranch)
		}
		if info.GitHead != "" {
			fmt.Fprintf(&b, "- HEAD: `%s`\n", info.GitHead)
		}
		b.WriteString("\n")
	}

	if len(info.EntryFiles) > 0 {
		b.WriteString("## Entry points\n\n")
		for _, f := range info.EntryFiles {
			fmt.Fprintf(&b, "- `%s`\n", f)
		}
		b.WriteString("\n")
	}

	if len(info.KeyDirs) > 0 {
		b.WriteString("## Top-level structure\n\n")
		for _, d := range info.KeyDirs {
			if d.Description != "" {
				fmt.Fprintf(&b, "- `%s/` — %s\n", d.Name, d.Description)
			} else {
				fmt.Fprintf(&b, "- `%s/`\n", d.Name)
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

// ── Service ───────────────────────────────────────────────────

// Service wraps Scanner with persistence to project_docs.
type Service struct {
	scanner *Scanner
	docs    *projectdoc.Service
	log     *slog.Logger
}

// NewService wires the scanner against an existing projectdoc.Service.
func NewService(docs *projectdoc.Service, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		scanner: New(log),
		docs:    docs,
		log:     log.With("component", "projectscan.service"),
	}
}

// RunAndReturn scans cwd and persists the rendered markdown,
// returning the resulting project_doc row. HTTP handlers call this
// when they want to echo the new doc back in the response.
func (s *Service) RunAndReturn(ctx context.Context, cwd string) (projectdoc.Doc, error) {
	info, err := s.scanner.Scan(ctx, cwd)
	if err != nil {
		return projectdoc.Doc{}, err
	}
	body := RenderMarkdown(info)
	doc, err := s.docs.PutDoc(ctx, cwd, projectdoc.KindTechStack, body, projectdoc.AuthorScanner)
	if err != nil {
		return projectdoc.Doc{}, fmt.Errorf("projectscan: persist: %w", err)
	}
	s.log.Info("projectscan.scanned",
		"cwd", cwd,
		"stacks", len(info.Stacks),
		"entry_files", len(info.EntryFiles),
		"branch", info.GitBranch,
		"head", info.GitHead,
	)
	return doc, nil
}

// Run scans + persists and returns only the error. Matches the
// catalog.ProjectScanner interface so the spawn-time hook can call
// it without unpacking the doc.
func (s *Service) Run(ctx context.Context, cwd string) error {
	_, err := s.RunAndReturn(ctx, cwd)
	return err
}

// IsStale returns true when the cwd hasn't been scanned within
// maxAge. Used by the scheduler / spawn-time hook so they don't
// re-scan a project that was just refreshed by a parallel session.
func (s *Service) IsStale(ctx context.Context, cwd string, maxAge time.Duration) bool {
	doc, err := s.docs.GetDoc(ctx, cwd, projectdoc.KindTechStack)
	if err != nil {
		// Missing doc counts as "stale" — needs a first scan.
		return true
	}
	return time.Since(doc.UpdatedAt) > maxAge
}

// errIgnore is a small helper so the build doesn't complain about
// unused fs.ErrNotExist imports in environments where we don't
// strictly need them yet. Kept for future extensions.
var _ = fs.ErrNotExist
