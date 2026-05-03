package notes

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// projectMapFilename is the JSON sidecar at the vault root that
// stores per-cwd project-folder overrides. Designed to live next to
// the user's notes (so it git-syncs with the vault) and stay small —
// just `{cwd: "vault-relative path"}`.
const projectMapFilename = ".opendray-projects.json"

// ProjectMapping records that sessions running in `Cwd` should treat
// `Path` (vault-relative, no trailing slash) as their project doc
// directory instead of the auto-derived `<projects_prefix>/<basename>`.
type ProjectMapping struct {
	Cwd  string `json:"cwd"`
	Path string `json:"path"`
}

// ResolvedProjectDir returns the project directory for sessions
// running in `cwd`. Looks up the override first; falls back to the
// auto-derived `<projects_prefix>/<basename>` when no mapping exists.
func (v *Vault) ResolvedProjectDir(cwd string) string {
	if cwd != "" {
		if p, ok := v.lookupProjectMapping(cwd); ok && p != "" {
			return strings.Trim(p, "/")
		}
	}
	base := basenameOf(cwd)
	return v.ProjectDir(base)
}

// ListProjectMappings returns every override currently saved. Used by
// the UI so users can see/manage what's configured.
func (v *Vault) ListProjectMappings() ([]ProjectMapping, error) {
	m, err := v.readProjectMap()
	if err != nil {
		return nil, err
	}
	out := make([]ProjectMapping, 0, len(m))
	for cwd, p := range m {
		out = append(out, ProjectMapping{Cwd: cwd, Path: p})
	}
	return out, nil
}

// SetProjectMapping pins (or, with empty path, clears) the override
// for one cwd. Path validation: must be vault-relative, no leading
// slash, no `..` segments — same jail as the rest of the notes API.
func (v *Vault) SetProjectMapping(cwd, path string) error {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return errors.New("cwd is required")
	}
	if !filepath.IsAbs(cwd) {
		return errors.New("cwd must be an absolute path")
	}
	path = strings.Trim(strings.TrimSpace(path), "/")
	if path != "" {
		// Validate: must be a clean vault-relative path with no `..`.
		clean := filepath.Clean(path)
		if clean == "." || strings.HasPrefix(clean, "..") || strings.Contains(clean, "..") {
			return ErrPathEscape
		}
		// Also ensure the resolved path stays under the vault root.
		// resolve() requires the basename's dir; reuse the same checks.
		if _, err := v.resolve(clean + "/.placeholder"); err != nil {
			return ErrPathEscape
		}
	}
	v.projectMapMu.Lock()
	defer v.projectMapMu.Unlock()
	m, err := v.readProjectMapLocked()
	if err != nil {
		return err
	}
	if path == "" {
		delete(m, cwd)
	} else {
		m[cwd] = path
	}
	return v.writeProjectMapLocked(m)
}

// readProjectMap is the unlocked variant — used by lookups (which
// take the lock themselves) and by callers already inside a critical
// section. We keep it cheap (file is tiny) — no in-memory cache.
func (v *Vault) readProjectMap() (map[string]string, error) {
	v.projectMapMu.Lock()
	defer v.projectMapMu.Unlock()
	return v.readProjectMapLocked()
}

func (v *Vault) readProjectMapLocked() (map[string]string, error) {
	path := filepath.Join(v.root, projectMapFilename)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("read project map: %w", err)
	}
	out := map[string]string{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parse project map: %w", err)
	}
	return out, nil
}

func (v *Vault) writeProjectMapLocked(m map[string]string) error {
	path := filepath.Join(v.root, projectMapFilename)
	if len(m) == 0 {
		// Empty map → remove the file entirely so nothing pollutes
		// the vault when the user clears all overrides.
		err := os.Remove(path)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("clear project map: %w", err)
		}
		return nil
	}
	body, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal project map: %w", err)
	}
	body = append(body, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o600); err != nil {
		return fmt.Errorf("write project map: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("commit project map: %w", err)
	}
	return nil
}

// lookupProjectMapping is the read path for ResolvedProjectDir. Cheap
// (file <100KB always) so we don't bother caching — every call hits
// disk, simplifies invalidation across processes (CLI vs gateway).
func (v *Vault) lookupProjectMapping(cwd string) (string, bool) {
	m, err := v.readProjectMap()
	if err != nil {
		return "", false
	}
	p, ok := m[cwd]
	return p, ok
}

// basenameOf extracts the trailing path segment of a cwd, dropping
// any trailing slashes. Mirrors filepath.Base but tolerates non-OS
// separators in the input.
func basenameOf(cwd string) string {
	cwd = strings.TrimRight(cwd, "/")
	if cwd == "" {
		return "untitled"
	}
	if i := strings.LastIndex(cwd, "/"); i >= 0 {
		return cwd[i+1:]
	}
	return cwd
}
