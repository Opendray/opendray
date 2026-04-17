// Package tasks discovers and executes project tasks (Makefile targets,
// package.json scripts, shell scripts) for the task-runner panel plugin.
package tasks

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Config holds settings read from the task-runner plugin config.
type Config struct {
	AllowedRoots        []string
	DefaultPath         string
	IncludeMakefile     bool
	IncludePackageJSON  bool
	IncludeShellScripts bool
	ShellTimeoutSec     int
	MaxConcurrent       int
	OutputBufferBytes   int
}

// Source identifies where a task came from.
type Source string

const (
	SourceMakefile    Source = "makefile"
	SourcePackageJSON Source = "package.json"
	SourceShellScript Source = "shell"
)

// Task is a single runnable unit discovered in a project.
type Task struct {
	ID      string   `json:"id"`      // stable hash of source+workdir+name
	Name    string   `json:"name"`    // target / script name shown to the user
	Source  Source   `json:"source"`  // makefile | package.json | shell
	Display string   `json:"display"` // command preview e.g. "make build"
	File    string   `json:"file"`    // absolute path to the source file
	Workdir string   `json:"workdir"` // directory the task is executed from
	Argv    []string `json:"argv"`    // exec.Command arguments (program, args...)
}

// Discover scans the given path for runnable tasks.
// path must be inside one of cfg.AllowedRoots; an empty path falls back to
// cfg.DefaultPath, then the first allowed root.
func Discover(cfg Config, path string) ([]Task, error) {
	abs, err := securePath(cfg, path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("tasks: stat: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("tasks: %q is not a directory", abs)
	}

	var tasks []Task
	if cfg.IncludeMakefile {
		tasks = append(tasks, discoverMakefile(abs)...)
	}
	if cfg.IncludePackageJSON {
		tasks = append(tasks, discoverPackageJSON(abs)...)
	}
	if cfg.IncludeShellScripts {
		tasks = append(tasks, discoverShellScripts(abs)...)
	}

	sort.SliceStable(tasks, func(i, j int) bool {
		if tasks[i].Source != tasks[j].Source {
			return tasks[i].Source < tasks[j].Source
		}
		return tasks[i].Name < tasks[j].Name
	})
	return tasks, nil
}

// ── Makefile ────────────────────────────────────────────────────

// matches lines like `build:` or `build: deps` but skips `.PHONY` style
// directives, variable assignments (`X := y`) and pattern rules (`%.o: %.c`).
var makeTargetRE = regexp.MustCompile(`^([A-Za-z0-9_][A-Za-z0-9_.-]*)\s*:(?:[^=]|$)`)

func discoverMakefile(workdir string) []Task {
	candidates := []string{"Makefile", "makefile", "GNUmakefile"}
	var path string
	for _, c := range candidates {
		p := filepath.Join(workdir, c)
		if _, err := os.Stat(p); err == nil {
			path = p
			break
		}
	}
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	seen := make(map[string]struct{})
	var tasks []Task
	for _, line := range strings.Split(string(data), "\n") {
		// Recipe lines start with a tab — skip.
		if strings.HasPrefix(line, "\t") {
			continue
		}
		m := makeTargetRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := m[1]
		// Skip well-known meta targets.
		if strings.HasPrefix(name, ".") {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		tasks = append(tasks, Task{
			ID:      hashID(SourceMakefile, workdir, name),
			Name:    name,
			Source:  SourceMakefile,
			Display: "make " + name,
			File:    path,
			Workdir: workdir,
			Argv:    []string{"make", name},
		})
	}
	return tasks
}

// ── package.json ────────────────────────────────────────────────

func discoverPackageJSON(workdir string) []Task {
	path := filepath.Join(workdir, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var pkg struct {
		Scripts        map[string]string `json:"scripts"`
		PackageManager string            `json:"packageManager"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}
	if len(pkg.Scripts) == 0 {
		return nil
	}
	runner := detectNodeRunner(workdir, pkg.PackageManager)

	tasks := make([]Task, 0, len(pkg.Scripts))
	for name := range pkg.Scripts {
		tasks = append(tasks, Task{
			ID:      hashID(SourcePackageJSON, workdir, name),
			Name:    name,
			Source:  SourcePackageJSON,
			Display: runner + " run " + name,
			File:    path,
			Workdir: workdir,
			Argv:    []string{runner, "run", name},
		})
	}
	return tasks
}

// detectNodeRunner picks pnpm > yarn > npm based on lockfiles, with the
// `packageManager` field as a hint when present.
func detectNodeRunner(workdir, hint string) string {
	if hint != "" {
		// hint format is e.g. "pnpm@8.6.0"
		if i := strings.Index(hint, "@"); i > 0 {
			switch hint[:i] {
			case "pnpm", "yarn", "npm":
				return hint[:i]
			}
		}
	}
	if exists(filepath.Join(workdir, "pnpm-lock.yaml")) {
		return "pnpm"
	}
	if exists(filepath.Join(workdir, "yarn.lock")) {
		return "yarn"
	}
	return "npm"
}

// ── Shell scripts ───────────────────────────────────────────────

func discoverShellScripts(workdir string) []Task {
	var tasks []Task
	tasks = append(tasks, scanShellDir(workdir, workdir)...)
	scriptsDir := filepath.Join(workdir, "scripts")
	if info, err := os.Stat(scriptsDir); err == nil && info.IsDir() {
		tasks = append(tasks, scanShellDir(workdir, scriptsDir)...)
	}
	return tasks
}

func scanShellDir(workdir, dir string) []Task {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var tasks []Task
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sh") {
			continue
		}
		full := filepath.Join(dir, e.Name())
		rel, err := filepath.Rel(workdir, full)
		if err != nil {
			rel = e.Name()
		}
		tasks = append(tasks, Task{
			ID:      hashID(SourceShellScript, workdir, rel),
			Name:    rel,
			Source:  SourceShellScript,
			Display: "bash " + rel,
			File:    full,
			Workdir: workdir,
			Argv:    []string{"bash", full},
		})
	}
	return tasks
}

// ── Security ────────────────────────────────────────────────────

func securePath(cfg Config, path string) (string, error) {
	if path == "" {
		switch {
		case cfg.DefaultPath != "":
			path = cfg.DefaultPath
		case len(cfg.AllowedRoots) > 0:
			path = cfg.AllowedRoots[0]
		default:
			return "", fmt.Errorf("tasks: no allowed roots configured")
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("tasks: invalid path: %w", err)
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		real = abs
	}
	for _, root := range cfg.AllowedRoots {
		rootAbs, _ := filepath.Abs(root)
		rootReal, err := filepath.EvalSymlinks(rootAbs)
		if err != nil {
			rootReal = rootAbs
		}
		if strings.HasPrefix(real, rootReal) {
			return real, nil
		}
	}
	return "", fmt.Errorf("tasks: path %q is outside allowed roots", path)
}

// ── Helpers ─────────────────────────────────────────────────────

func hashID(source Source, workdir, name string) string {
	sum := sha1.Sum([]byte(string(source) + "\x00" + workdir + "\x00" + name))
	return hex.EncodeToString(sum[:8])
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
