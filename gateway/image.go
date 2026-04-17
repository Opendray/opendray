package gateway

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// Maximum image upload size — 25 MB.
const maxImageBytes = 25 << 20

// sessionAttachImage stores an uploaded image and returns its absolute path.
//
// The image is written to an NTC-owned directory outside the session's working
// directory (so it never pollutes the user's project). The server does NOT
// type the path into the PTY — that is the client's decision, exposed via the
// existing /api/sessions/{id}/input endpoint, so the user can preview, copy,
// or explicitly insert the path when they are ready. This avoids corrupting
// whatever the CLI was in the middle of when the upload completed.
//
// Request:
//
//	POST /api/sessions/{id}/image
//	Content-Type: image/png | image/jpeg | image/webp | image/gif | image/heic
//	Body:         raw image bytes (≤ 25 MB)
//
// Response:
//
//	{ "path": "<abs path>", "name": "img-…png", "size": N }
func (s *Server) sessionAttachImage(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	// Validate that the session exists (no need for cwd — we use our own dir).
	if _, ok, err := s.hub.Get(r.Context(), sessionID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	} else if !ok {
		respondError(w, http.StatusNotFound, "session not found")
		return
	}

	ext := imageExtFromContentType(r.Header.Get("Content-Type"))
	r.Body = http.MaxBytesReader(w, r.Body, maxImageBytes)
	defer r.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, fmt.Errorf("read body: %w", err).Error())
		return
	}
	if len(body) == 0 {
		respondError(w, http.StatusBadRequest, "empty image body")
		return
	}

	// Use a server-owned directory outside the user's project. Persists across
	// session stop/start; cleaned up by the OS's tmp-reaper over time.
	baseDir := filepath.Join(os.TempDir(), "ntc-images", sessionID)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Errorf("mkdir: %w", err).Error())
		return
	}

	name := fmt.Sprintf("img-%d%s", time.Now().UnixNano(), ext)
	absPath := filepath.Join(baseDir, name)
	if err := os.WriteFile(absPath, body, 0o644); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Errorf("write: %w", err).Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"path": absPath,
		"name": name,
		"size": len(body),
	})
}

func imageExtFromContentType(ct string) string {
	ct = strings.ToLower(ct)
	switch {
	case strings.Contains(ct, "jpeg"), strings.Contains(ct, "jpg"):
		return ".jpg"
	case strings.Contains(ct, "webp"):
		return ".webp"
	case strings.Contains(ct, "gif"):
		return ".gif"
	case strings.Contains(ct, "heic"):
		return ".heic"
	default:
		return ".png"
	}
}
