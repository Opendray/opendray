// Package files provides secure local filesystem browsing for panel plugins.
package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BrowserConfig holds settings read from plugin config.
type BrowserConfig struct {
	AllowedRoots []string // directories that can be browsed
	ShowHidden   bool
	MaxFileSize  int64 // bytes
	DefaultPath  string
}

// FileEntry represents a file or directory.
type FileEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"` // absolute path
	Type    string `json:"type"` // "file" or "dir"
	Size    int64  `json:"size,omitempty"`
	IsGit   bool   `json:"isGit,omitempty"`   // directory contains .git
	ModTime int64  `json:"modTime,omitempty"` // unix timestamp
	Ext     string `json:"ext,omitempty"`     // file extension
}

// FileContent represents a file's content.
type FileContent struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Content  string `json:"content"`
	Size     int64  `json:"size"`
	Ext      string `json:"ext"`
	Language string `json:"language"` // inferred language for syntax highlighting
	Binary   bool   `json:"binary"`
}

// ListDir lists files/directories at the given path.
func ListDir(cfg BrowserConfig, path string) ([]FileEntry, error) {
	absPath, err := securePath(cfg, path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("files: read dir: %w", err)
	}

	var result []FileEntry
	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden files unless configured
		if !cfg.ShowHidden && strings.HasPrefix(name, ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		fullPath := filepath.Join(absPath, name)
		fe := FileEntry{
			Name:    name,
			Path:    fullPath,
			Size:    info.Size(),
			ModTime: info.ModTime().Unix(),
		}

		if entry.IsDir() {
			fe.Type = "dir"
			// Check for .git
			if _, err := os.Stat(filepath.Join(fullPath, ".git")); err == nil {
				fe.IsGit = true
			}
		} else {
			fe.Type = "file"
			fe.Ext = strings.TrimPrefix(filepath.Ext(name), ".")
		}

		result = append(result, fe)
	}
	return result, nil
}

// ReadFile reads a file's content with security and size checks.
func ReadFile(cfg BrowserConfig, path string) (FileContent, error) {
	absPath, err := securePath(cfg, path)
	if err != nil {
		return FileContent{}, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return FileContent{}, fmt.Errorf("files: stat: %w", err)
	}
	if info.IsDir() {
		return FileContent{}, fmt.Errorf("files: is a directory")
	}

	name := filepath.Base(absPath)
	ext := strings.TrimPrefix(filepath.Ext(name), ".")
	fc := FileContent{
		Name:     name,
		Path:     absPath,
		Size:     info.Size(),
		Ext:      ext,
		Language: inferLanguage(ext),
	}

	// Check binary by extension
	if isBinaryExt(ext) {
		fc.Binary = true
		fc.Content = fmt.Sprintf("[Binary file: %s, %d bytes]", name, info.Size())
		return fc, nil
	}

	// Size check
	if info.Size() > cfg.MaxFileSize {
		fc.Content = fmt.Sprintf("[File too large: %d bytes, limit: %d bytes]", info.Size(), cfg.MaxFileSize)
		return fc, nil
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return FileContent{}, fmt.Errorf("files: read: %w", err)
	}

	fc.Content = string(data)
	return fc, nil
}

// Search searches for files matching a query under the allowed roots.
func Search(cfg BrowserConfig, basePath, query string) ([]FileEntry, error) {
	searchRoot := cfg.DefaultPath
	if basePath != "" {
		resolved, err := securePath(cfg, basePath)
		if err != nil {
			return nil, err
		}
		searchRoot = resolved
	}
	if searchRoot == "" && len(cfg.AllowedRoots) > 0 {
		searchRoot = cfg.AllowedRoots[0]
	}

	queryLower := strings.ToLower(query)
	var results []FileEntry

	filepath.WalkDir(searchRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if len(results) >= 50 {
			return filepath.SkipAll
		}

		name := d.Name()

		// Skip hidden dirs entirely
		if d.IsDir() && strings.HasPrefix(name, ".") {
			return filepath.SkipDir
		}
		// Skip node_modules, build dirs
		if d.IsDir() && (name == "node_modules" || name == "build" || name == "__pycache__") {
			return filepath.SkipDir
		}

		if !d.IsDir() && strings.Contains(strings.ToLower(name), queryLower) {
			info, _ := d.Info()
			size := int64(0)
			modTime := int64(0)
			if info != nil {
				size = info.Size()
				modTime = info.ModTime().Unix()
			}
			results = append(results, FileEntry{
				Name:    name,
				Path:    path,
				Type:    "file",
				Size:    size,
				ModTime: modTime,
				Ext:     strings.TrimPrefix(filepath.Ext(name), "."),
			})
		}
		return nil
	})

	return results, nil
}

// MakeDir creates a new directory inside an allowed root.
// parent is the directory to create `name` inside; name must not contain
// path separators. Returns the absolute path of the created directory.
func MakeDir(cfg BrowserConfig, parent, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("files: folder name is required")
	}
	if strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("files: folder name cannot contain path separators")
	}
	if name == "." || name == ".." {
		return "", fmt.Errorf("files: invalid folder name")
	}

	parentAbs, err := securePath(cfg, parent)
	if err != nil {
		return "", err
	}
	target := filepath.Join(parentAbs, name)
	// Re-verify the target is still inside an allowed root (defence in depth).
	if _, err := securePath(cfg, target); err != nil {
		return "", err
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return "", fmt.Errorf("files: mkdir: %w", err)
	}
	return target, nil
}

// ── Security ────────────────────────────────────────────────────

// securePath validates and resolves a path, ensuring it's under an allowed root.
func securePath(cfg BrowserConfig, path string) (string, error) {
	if path == "" {
		if cfg.DefaultPath != "" {
			path = cfg.DefaultPath
		} else if len(cfg.AllowedRoots) > 0 {
			path = cfg.AllowedRoots[0]
		} else {
			return "", fmt.Errorf("files: no allowed roots configured")
		}
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("files: invalid path: %w", err)
	}

	// Resolve symlinks
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// Path might not exist yet, use absPath
		realPath = absPath
	}

	// Check against allowed roots
	for _, root := range cfg.AllowedRoots {
		rootAbs, _ := filepath.Abs(root)
		rootReal, err := filepath.EvalSymlinks(rootAbs)
		if err != nil {
			rootReal = rootAbs
		}
		if strings.HasPrefix(realPath, rootReal) {
			return realPath, nil
		}
	}

	return "", fmt.Errorf("files: path %q is outside allowed roots", path)
}

// ── Helpers ─────────────────────────────────────────────────────

var langMap = map[string]string{
	"go": "go", "dart": "dart", "js": "javascript", "ts": "typescript",
	"jsx": "javascript", "tsx": "typescript", "py": "python", "rb": "ruby",
	"rs": "rust", "java": "java", "kt": "kotlin", "swift": "swift",
	"c": "c", "cpp": "cpp", "h": "c", "hpp": "cpp",
	"md": "markdown", "json": "json", "yaml": "yaml", "yml": "yaml",
	"toml": "toml", "xml": "xml", "html": "html", "css": "css",
	"scss": "scss", "sql": "sql", "sh": "bash", "zsh": "bash",
	"bash": "bash", "dockerfile": "dockerfile", "makefile": "makefile",
	"vue": "vue", "svelte": "svelte", "graphql": "graphql",
	"proto": "protobuf", "tf": "hcl", "mod": "go",
}

func inferLanguage(ext string) string {
	if lang, ok := langMap[strings.ToLower(ext)]; ok {
		return lang
	}
	return "text"
}

var binaryExts = map[string]bool{
	"png": true, "jpg": true, "jpeg": true, "gif": true, "webp": true, "ico": true,
	"pdf": true, "zip": true, "gz": true, "tar": true, "bz2": true,
	"exe": true, "dll": true, "so": true, "dylib": true,
	"woff": true, "woff2": true, "ttf": true, "otf": true, "eot": true,
	"mp3": true, "mp4": true, "avi": true, "mov": true, "wav": true,
	"apk": true, "ipa": true, "dmg": true,
}

func isBinaryExt(ext string) bool {
	return binaryExts[strings.ToLower(ext)]
}
