// Package docs provides Git forge document browsing for panel plugins.
package docs

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ForgeConfig holds connection params for a Git forge, read from plugin config.
type ForgeConfig struct {
	ForgeType      string // gitea, github, gitlab
	BaseURL        string // e.g. https://gitea.com or https://api.github.com
	Repo           string // owner/repo
	Token          string // API token (optional)
	Branch         string // main
	BasePath       string // Projects/
	FileExtensions string // .md,.txt
}

// FileEntry represents a file or directory in the repo.
type FileEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"` // "file" or "dir"
	Size int64  `json:"size,omitempty"`
}

// FileContent represents a file's content.
type FileContent struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Content string `json:"content"` // raw text content
	Size    int64  `json:"size"`
}

var httpClient = &http.Client{Timeout: 15 * time.Second}

// ListDir lists files/directories at the given path.
func ListDir(cfg ForgeConfig, path string) ([]FileEntry, error) {
	bps := parseBasePaths(cfg.BasePath)
	if len(bps) > 1 {
		return listDirMulti(cfg, bps, path)
	}
	base := ""
	if len(bps) == 1 {
		base = bps[0]
	}
	return listDirSingle(cfg, base, path)
}

// listDirSingle is the core listing logic for a single base path.
func listDirSingle(cfg ForgeConfig, basePath, path string) ([]FileEntry, error) {
	fullPath := joinPath(basePath, path)
	apiURL := buildContentsURL(cfg, fullPath)

	body, err := doRequest(cfg, apiURL)
	if err != nil {
		return nil, err
	}

	// Gitea/GitHub return array for directories
	var items []struct {
		Name string `json:"name"`
		Path string `json:"path"`
		Type string `json:"type"` // "file" or "dir"
		Size int64  `json:"size"`
	}
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("docs: parse response: %w", err)
	}

	exts := parseExtensions(cfg.FileExtensions)
	var entries []FileEntry
	for _, item := range items {
		// Filter: show dirs always, files only if extension matches
		if item.Type == "dir" || matchesExtension(item.Name, exts) {
			entries = append(entries, FileEntry{
				Name: item.Name,
				Path: stripBasePath(item.Path, basePath),
				Type: item.Type,
				Size: item.Size,
			})
		}
	}
	return entries, nil
}

// listDirMulti handles multiple base paths.
// At root (path == ""), returns a virtual directory entry per base path.
// Otherwise treats path as an absolute repo path and lists it directly.
func listDirMulti(cfg ForgeConfig, basePaths []string, path string) ([]FileEntry, error) {
	if path == "" {
		var entries []FileEntry
		for _, bp := range basePaths {
			entries = append(entries, FileEntry{
				Name: pathBaseName(strings.TrimSuffix(bp, "/")),
				Path: bp,
				Type: "dir",
			})
		}
		return entries, nil
	}
	// Path is already absolute (e.g. "Projects/NTC"); no basePath prefix needed.
	return listDirSingle(cfg, "", path)
}

// ReadFile reads a file's content.
func ReadFile(cfg ForgeConfig, path string) (FileContent, error) {
	bps := parseBasePaths(cfg.BasePath)
	base := ""
	if len(bps) == 1 {
		base = bps[0]
	}
	// For multiple base paths, path is already absolute — no prefix needed.

	fullPath := joinPath(base, path)
	apiURL := buildContentsURL(cfg, fullPath)

	body, err := doRequest(cfg, apiURL)
	if err != nil {
		return FileContent{}, err
	}

	var item struct {
		Name     string `json:"name"`
		Path     string `json:"path"`
		Size     int64  `json:"size"`
		Content  string `json:"content"`  // base64 for Gitea/GitHub
		Encoding string `json:"encoding"` // "base64"
	}
	if err := json.Unmarshal(body, &item); err != nil {
		return FileContent{}, fmt.Errorf("docs: parse file: %w", err)
	}

	content := item.Content
	if item.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(content, "\n", ""))
		if err != nil {
			return FileContent{}, fmt.Errorf("docs: decode base64: %w", err)
		}
		content = string(decoded)
	}

	return FileContent{
		Name:    item.Name,
		Path:    stripBasePath(item.Path, base),
		Content: content,
		Size:    item.Size,
	}, nil
}

// Search searches for files matching a query in the repo.
func Search(cfg ForgeConfig, query string) ([]FileEntry, error) {
	// Gitea search API: /api/v1/repos/{owner}/{repo}/contents?ref=branch
	// For simplicity, do a tree walk. For large repos, use search API.
	// Here we use the Gitea search endpoint if available.
	apiURL := fmt.Sprintf("%s/api/v1/repos/%s/git/trees/%s?recursive=true",
		cfg.BaseURL, cfg.Repo, url.PathEscape(cfg.Branch))

	body, err := doRequest(cfg, apiURL)
	if err != nil {
		return nil, err
	}

	var tree struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"` // "blob" or "tree"
			Size int64  `json:"size"`
		} `json:"tree"`
	}
	if err := json.Unmarshal(body, &tree); err != nil {
		return nil, fmt.Errorf("docs: parse tree: %w", err)
	}

	bps := parseBasePaths(cfg.BasePath)
	queryLower := strings.ToLower(query)
	exts := parseExtensions(cfg.FileExtensions)
	var results []FileEntry

	for _, item := range tree.Tree {
		if item.Type != "blob" {
			continue
		}
		if !matchesBasePaths(item.Path, bps) {
			continue
		}
		if !matchesExtension(item.Path, exts) {
			continue
		}
		if strings.Contains(strings.ToLower(item.Path), queryLower) {
			// Single basePath: return relative path (existing behaviour).
			// Multiple or no basePath: return absolute path.
			entryPath := item.Path
			if len(bps) == 1 {
				entryPath = stripBasePath(item.Path, bps[0])
			}
			results = append(results, FileEntry{
				Name: pathBaseName(item.Path),
				Path: entryPath,
				Type: "file",
				Size: item.Size,
			})
		}
		if len(results) >= 50 {
			break
		}
	}
	return results, nil
}

// ── Internal helpers ────────────────────────────────────────────

func buildContentsURL(cfg ForgeConfig, path string) string {
	escapedPath := url.PathEscape(path)
	// Fix: url.PathEscape escapes /, we need to keep them
	escapedPath = strings.ReplaceAll(escapedPath, "%2F", "/")

	switch cfg.ForgeType {
	case "github":
		return fmt.Sprintf("https://api.github.com/repos/%s/contents/%s?ref=%s",
			cfg.Repo, escapedPath, cfg.Branch)
	case "gitlab":
		encodedRepo := url.PathEscape(cfg.Repo)
		encodedPath := url.PathEscape(path)
		return fmt.Sprintf("%s/api/v4/projects/%s/repository/tree?path=%s&ref=%s",
			cfg.BaseURL, encodedRepo, encodedPath, cfg.Branch)
	default: // gitea
		return fmt.Sprintf("%s/api/v1/repos/%s/contents/%s?ref=%s",
			cfg.BaseURL, cfg.Repo, escapedPath, cfg.Branch)
	}
}

func doRequest(cfg ForgeConfig, apiURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("docs: create request: %w", err)
	}

	if cfg.Token != "" {
		switch cfg.ForgeType {
		case "github":
			req.Header.Set("Authorization", "Bearer "+cfg.Token)
		case "gitlab":
			req.Header.Set("PRIVATE-TOKEN", cfg.Token)
		default: // gitea
			req.Header.Set("Authorization", "token "+cfg.Token)
		}
	}
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("docs: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		msg := string(body)
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return nil, fmt.Errorf("docs: API %d: %s", resp.StatusCode, msg)
	}

	return io.ReadAll(resp.Body)
}

func joinPath(base, path string) string {
	base = strings.TrimSuffix(base, "/")
	path = strings.TrimPrefix(path, "/")
	if base == "" {
		return path
	}
	if path == "" {
		return base
	}
	return base + "/" + path
}

func stripBasePath(path, basePath string) string {
	basePath = strings.TrimSuffix(basePath, "/")
	if basePath == "" {
		return path
	}
	return strings.TrimPrefix(strings.TrimPrefix(path, basePath), "/")
}

// parseBasePaths splits a comma-separated basePath config value into a
// normalised slice of paths (each guaranteed to have a trailing slash).
// Returns nil when the value is empty (meaning: no filtering, show whole repo).
func parseBasePaths(basePath string) []string {
	if strings.TrimSpace(basePath) == "" {
		return nil
	}
	var paths []string
	for _, p := range strings.Split(basePath, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.HasSuffix(p, "/") {
			p += "/"
		}
		paths = append(paths, p)
	}
	return paths
}

// matchesBasePaths reports whether path is under at least one of the base paths.
// Returns true when basePaths is empty (no filtering).
func matchesBasePaths(path string, basePaths []string) bool {
	if len(basePaths) == 0 {
		return true
	}
	for _, bp := range basePaths {
		if strings.HasPrefix(path, bp) {
			return true
		}
	}
	return false
}

func parseExtensions(exts string) []string {
	if exts == "" {
		return []string{".md"}
	}
	var result []string
	for _, ext := range strings.Split(exts, ",") {
		ext = strings.TrimSpace(ext)
		if ext != "" {
			result = append(result, ext)
		}
	}
	return result
}

func matchesExtension(name string, exts []string) bool {
	nameLower := strings.ToLower(name)
	for _, ext := range exts {
		if strings.HasSuffix(nameLower, ext) {
			return true
		}
	}
	return false
}

func pathBaseName(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}
