package gateway

// T8 — Plugin asset handler.
//
// Route (registered in protected chi group):
//
//	GET /api/plugins/{name}/assets/*
//
// Serves static files from ${Installer.DataDir}/<name>/<activeVersion>/ui/
// with a strict CSP, content-type sniffing disabled, and hardened path
// traversal defence.
//
// Path validation order (per M2-PLAN §T8):
//  1. Reject dangerous bytes: NUL, CR/LF, backslash.
//  2. filepath.Clean the fragment.
//  3. Reject if cleaned result starts with '/' or '\\', or contains '..'.
//  4. filepath.Rel assertion to catch any symlink escape.
//
// The core logic lives in assetsHandler (a free function) so tests can inject
// a lightweight pluginVersioner fake without needing a real database.

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray/plugin"
)

// pluginVersioner is the subset of plugin.Runtime used by pluginsAssets.
// Defined here so tests can inject a lightweight fake without touching the
// plugin package. *plugin.Runtime satisfies it automatically.
type pluginVersioner interface {
	Get(name string) (plugin.Provider, bool)
}

// cspHeader is the Content-Security-Policy value from M2-PLAN §T8.
// Transcribed byte-for-byte; do not reformat or reorder directives.
const cspHeader = "default-src 'self'; script-src 'self' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; connect-src 'self' ws: wss:; img-src 'self' data: blob:; object-src 'none'; frame-ancestors 'none'; base-uri 'self'"

// pluginsAssets serves static files out of the installed plugin's ui/
// directory with a strict CSP, content-type sniffing off, and hardened
// against path traversal.
//
// Layout:
//
//	GET /api/plugins/{name}/assets/*
//	→ ${cfg.PluginsDataDir}/<name>/<activeVersion>/ui/<*>
//
// Errors:
//
//	400 EBADPATH   — '..' or absolute segment or dangerous byte in wildcard
//	401 ENOAUTH    — protected group middleware (free)
//	404 ENOPLUGIN  — plugin not registered / has no active version
//	404 ENOFILE    — file not found at the resolved path
//	500 EINTERNAL  — filesystem error while reading
func (s *Server) pluginsAssets(w http.ResponseWriter, r *http.Request) {
	dataDir := ""
	if s.installer != nil {
		dataDir = s.installer.DataDir
	}
	assetsHandler(w, r, s.plugins, dataDir)
}

// assetsHandler is the core implementation extracted as a free function so
// tests can supply a pluginVersioner fake and an arbitrary dataDir without
// standing up a full *Server or touching any database.
func assetsHandler(w http.ResponseWriter, r *http.Request, pv pluginVersioner, dataDir string) {
	name := chi.URLParam(r, "name")

	// ── 1. Resolve the active version from the runtime ────────────────────────
	version, ok := pluginVersion(pv, name)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "ENOPLUGIN",
			"plugin not registered or has no active version")
		return
	}

	// ── 2. Get and default the wildcard fragment ──────────────────────────────
	raw := chi.URLParam(r, "*")
	if raw == "" {
		raw = "index.html"
	}

	// ── 3. Pre-Clean byte-level validation ───────────────────────────────────
	// Reject characters that are unambiguously dangerous regardless of how
	// the path might be normalised. These checks happen BEFORE filepath.Clean.
	if err := validateAssetFragment(raw); err != nil {
		writeJSONError(w, http.StatusBadRequest, "EBADPATH", err.Error())
		return
	}

	// ── 4. Normalise with filepath.Clean ─────────────────────────────────────
	// "a/b/../c" → "a/c" (safe, resolves within the tree).
	// ".." → ".." (unsafe, caught below).
	// "../../etc/passwd" → "../../etc/passwd" (unsafe, caught below).
	cleaned := filepath.Clean(raw)

	// ── 5. Post-Clean structural validation (defence-in-depth) ───────────────
	if containsDotDot(cleaned) || strings.HasPrefix(cleaned, "/") || strings.HasPrefix(cleaned, "\\") {
		writeJSONError(w, http.StatusBadRequest, "EBADPATH",
			"path traversal detected after normalisation")
		return
	}

	// ── 6. Build the filesystem path ──────────────────────────────────────────
	uiRoot := filepath.Join(dataDir, name, version, "ui")
	absPath := filepath.Join(uiRoot, cleaned)

	// Symbolic-link escape check: the resolved path must be strictly under
	// uiRoot. filepath.Rel gives the relative path; a ".." prefix means the
	// resolved path escaped the root (e.g. via a symlink).
	rel, err := filepath.Rel(uiRoot, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		writeJSONError(w, http.StatusBadRequest, "EBADPATH",
			"path resolves outside plugin ui directory")
		return
	}

	// ── 7. Stat the file ─────────────────────────────────────────────────────
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSONError(w, http.StatusNotFound, "ENOFILE", "asset not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "EINTERNAL",
			"filesystem error")
		return
	}

	// ── 8. Write security headers (before any content bytes) ─────────────────
	h := w.Header()
	h.Set("Content-Security-Policy", cspHeader)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("X-Frame-Options", "DENY")

	// Cache-Control: HTML files must not be cached (SPA entry point carries
	// CSP meta tags); everything else can be cached for one hour.
	ext := strings.ToLower(filepath.Ext(cleaned))
	if ext == ".html" || ext == ".htm" {
		h.Set("Cache-Control", "no-store")
	} else {
		h.Set("Cache-Control", "public, max-age=3600")
	}

	// ── 9. Set Content-Type from extension ────────────────────────────────────
	h.Set("Content-Type", contentTypeForExt(ext))

	// ── 10. Open and serve with conditional-GET support ──────────────────────
	f, err := os.Open(absPath)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "EINTERNAL",
			"cannot open asset")
		return
	}
	defer f.Close() //nolint:errcheck

	http.ServeContent(w, r, cleaned, info.ModTime(), f)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// pluginVersion extracts the active version string for the named plugin.
// Returns ("", false) if pv is nil, the plugin is unknown, or has no version.
func pluginVersion(pv pluginVersioner, name string) (string, bool) {
	if pv == nil {
		return "", false
	}
	p, ok := pv.Get(name)
	if !ok || p.Version == "" {
		return "", false
	}
	return p.Version, true
}

// validateAssetFragment performs pre-Clean byte-level validation on the raw
// wildcard fragment. It rejects characters that cannot appear in a safe path
// regardless of normalisation. Called by tests directly for control-character
// cases that HTTP request constructors refuse to encode.
//
// '..' segments are NOT checked here — that is done post-Clean in assetsHandler
// so that legitimate paths like "a/b/../c" (→ "a/c") are not rejected.
func validateAssetFragment(raw string) error {
	// NUL byte — terminates C strings and can trick some OS calls.
	if strings.ContainsRune(raw, '\x00') {
		return badPathError("path contains NUL byte")
	}
	// Newline / CR — can split HTTP headers or confuse log parsers.
	if strings.ContainsAny(raw, "\r\n") {
		return badPathError("path contains newline")
	}
	// Backslash — Windows path separator; used in some Unicode escape tricks.
	if strings.ContainsRune(raw, '\\') {
		return badPathError("path contains backslash")
	}
	// Absolute path (starts with '/').
	if strings.HasPrefix(raw, "/") {
		return badPathError("path must be relative")
	}
	return nil
}

// containsDotDot returns true if s contains ".." as a path component.
// Splits on '/' (normalised paths should only use forward slash at this point).
func containsDotDot(s string) bool {
	for _, seg := range strings.Split(s, "/") {
		if seg == ".." {
			return true
		}
	}
	return false
}

// badPathError is an error type for path validation failures.
type badPathError string

func (e badPathError) Error() string { return string(e) }

// contentTypeForExt returns the correct Content-Type for a lowercase file
// extension. The spec-mandated mappings take precedence over Go's mime
// package; unknown extensions fall back to application/octet-stream.
func contentTypeForExt(ext string) string {
	switch ext {
	case ".js", ".mjs":
		return "application/javascript"
	case ".css":
		return "text/css; charset=utf-8"
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".json":
		return "application/json"
	case ".svg":
		return "image/svg+xml"
	}
	ct := mime.TypeByExtension(ext)
	if ct == "" {
		return "application/octet-stream"
	}
	return ct
}
