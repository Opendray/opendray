package skills

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

// Handlers expose skills as a REST surface for the web Plugins page.
// Built-in skills are read-only; vault skills (which live under
// <vault>/skills/<id>/SKILL.md) are full CRUD. The split is enforced
// here, not by the loader — the UI shows source=builtin and disables
// edit/delete for those.
type Handlers struct {
	loader *Loader
	log    *slog.Logger
}

func NewHandlers(loader *Loader, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{loader: loader, log: log.With("component", "skills.http")}
}

func (h *Handlers) Mount(r chi.Router) {
	r.Route("/skills", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		// Multipart upload of a SKILL.md — derives the id from the
		// frontmatter `name:` field (slugified) rather than asking the
		// caller for one. Lets the dashboard offer drag-and-drop install
		// of a skill file without an intermediate "pick an id" step.
		r.Post("/upload", h.upload)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.get)
			r.Put("/", h.update)
			r.Delete("/", h.delete)
		})
	})
}

type skillView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
	Body        string `json:"body,omitempty"`
	// OverridesBuiltin is true when source=="vault" AND a built-in
	// with the same id exists embedded in the binary. UI uses this to
	// offer a "Reset to built-in" action that just removes the vault
	// override (loader falls back to the embedded version).
	OverridesBuiltin bool `json:"overrides_builtin,omitempty"`
	// HasBuiltin is true when source=="builtin" OR (source=="vault"
	// AND OverridesBuiltin). Lets the UI show a "Customize" affordance
	// on built-ins (which clones the body into a vault entry).
	HasBuiltin bool `json:"has_builtin,omitempty"`
}

type writeReq struct {
	ID   string `json:"id"`
	Body string `json:"body"`
}

func (h *Handlers) list(w http.ResponseWriter, _ *http.Request) {
	all, err := h.loader.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	builtinIDs, _ := h.loader.BuiltinIDs()
	out := make([]skillView, 0, len(all))
	for _, s := range all {
		// Body is omitted from list to keep the response small; UI
		// fetches it via /skills/{id} on demand.
		v := skillView{
			ID:          s.ID,
			Name:        s.Name,
			Description: s.Description,
			Source:      s.Source,
			HasBuiltin:  builtinIDs[s.ID],
		}
		if s.Source == "vault" && builtinIDs[s.ID] {
			v.OverridesBuiltin = true
		}
		out = append(out, v)
	}
	writeJSON(w, http.StatusOK, map[string]any{"skills": out})
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	s, err := h.loader.Get(id)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	builtinIDs, _ := h.loader.BuiltinIDs()
	v := skillView{
		ID:          s.ID,
		Name:        s.Name,
		Description: s.Description,
		Source:      s.Source,
		Body:        s.Body,
		HasBuiltin:  builtinIDs[s.ID],
	}
	if s.Source == "vault" && builtinIDs[s.ID] {
		v.OverridesBuiltin = true
	}
	writeJSON(w, http.StatusOK, v)
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	var req writeReq
	if err := json.NewDecoder(io.LimitReader(r.Body, 4<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	id := strings.TrimSpace(req.ID)
	if !validID(id) {
		writeError(w, http.StatusBadRequest,
			errors.New("id must be lowercase alphanumeric / dash / underscore"))
		return
	}
	if h.loader.VaultRoot() == "" {
		writeError(w, http.StatusConflict,
			errors.New("vault root is not configured; cannot create skills"))
		return
	}
	dest := filepath.Join(h.loader.VaultRoot(), id)
	if _, err := os.Stat(dest); err == nil {
		writeError(w, http.StatusConflict,
			fmt.Errorf("skill %s already exists in vault", id))
		return
	}
	if err := os.MkdirAll(dest, 0o700); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	body := req.Body
	if strings.TrimSpace(body) == "" {
		body = defaultSkillBody(id)
	}
	if err := os.WriteFile(filepath.Join(dest, "SKILL.md"), []byte(body), 0o600); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s, err := h.loader.Get(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, skillView{
		ID: s.ID, Name: s.Name, Description: s.Description,
		Source: s.Source, Body: s.Body,
	})
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !validID(id) {
		writeError(w, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	if h.loader.VaultRoot() == "" {
		writeError(w, http.StatusConflict,
			errors.New("vault root is not configured"))
		return
	}
	var req writeReq
	if err := json.NewDecoder(io.LimitReader(r.Body, 4<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	dest := filepath.Join(h.loader.VaultRoot(), id)
	if err := os.MkdirAll(dest, 0o700); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := os.WriteFile(filepath.Join(dest, "SKILL.md"), []byte(req.Body), 0o600); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s, err := h.loader.Get(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, skillView{
		ID: s.ID, Name: s.Name, Description: s.Description,
		Source: s.Source, Body: s.Body,
	})
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !validID(id) {
		writeError(w, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	if h.loader.VaultRoot() == "" {
		writeError(w, http.StatusConflict, errors.New("vault root not configured"))
		return
	}
	dest := filepath.Join(h.loader.VaultRoot(), id)
	info, err := os.Stat(dest)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Built-in skills can't be deleted — this id only exists
			// embedded in the binary. Return 409 so the UI can surface
			// a clear message.
			writeError(w, http.StatusConflict,
				fmt.Errorf("skill %s is not in the vault (built-in or absent)", id))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !info.IsDir() {
		writeError(w, http.StatusInternalServerError,
			errors.New("vault skill path is not a directory"))
		return
	}
	if err := os.RemoveAll(dest); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// upload accepts a multipart-uploaded SKILL.md and installs it in the
// vault under <vault>/<slug-of-name>/SKILL.md. The id is derived from
// the frontmatter `name:` field (slugified to validID's alphabet) so
// the operator doesn't have to pick one — drag the .md, get a working
// skill. Collides-with-vault is a 409 (delete first); collides-with-
// builtin is allowed and just becomes an override (matches the
// existing PUT semantics).
func (h *Handlers) upload(w http.ResponseWriter, r *http.Request) {
	if h.loader.VaultRoot() == "" {
		writeError(w, http.StatusConflict,
			errors.New("vault root is not configured; cannot upload skills"))
		return
	}
	const maxUploadBytes = 4 << 20 // 4 MB — matches the JSON create cap.
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeError(w, http.StatusBadRequest,
			fmt.Errorf("read multipart form: %w", err))
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest,
			errors.New("missing 'file' field in multipart upload"))
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxUploadBytes))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	body := string(data)
	if strings.TrimSpace(body) == "" {
		writeError(w, http.StatusBadRequest, errors.New("uploaded file is empty"))
		return
	}
	name, _ := parseFrontmatter(body)
	if strings.TrimSpace(name) == "" {
		writeError(w, http.StatusBadRequest,
			errors.New("SKILL.md must declare a 'name:' field in its frontmatter"))
		return
	}
	id := slugify(name)
	if !validID(id) {
		writeError(w, http.StatusBadRequest,
			fmt.Errorf("derived id %q is invalid; rename the skill so the frontmatter name slugs to lowercase alphanumeric / dash / underscore", id))
		return
	}

	dest := filepath.Join(h.loader.VaultRoot(), id)
	if _, err := os.Stat(dest); err == nil {
		writeError(w, http.StatusConflict,
			fmt.Errorf("skill %s already exists in vault; delete it first or rename the upload", id))
		return
	}
	if err := os.MkdirAll(dest, 0o700); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := os.WriteFile(filepath.Join(dest, "SKILL.md"), []byte(body), 0o600); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s, err := h.loader.Get(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, skillView{
		ID: s.ID, Name: s.Name, Description: s.Description,
		Source: s.Source, Body: s.Body,
	})
}

// slugify converts a frontmatter name into an id that validID will
// accept: lowercase, collapse non-alphanumeric runs into single
// dashes, trim leading/trailing dashes and underscores. Preserves
// underscores so authors who want them get them.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-_")
}

func validID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	for _, r := range id {
		ok := (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') ||
			r == '-' || r == '_'
		if !ok {
			return false
		}
	}
	return true
}

func defaultSkillBody(id string) string {
	return fmt.Sprintf(`---
name: %s
description: One-line description of when to use this skill (shown in the index, ~30 tokens).
---

# %s

## When to use

(describe the trigger conditions — what user phrases / situations should activate this skill)

## Commands

(if the skill has CLI commands, list them here)

## Patterns

(common usage patterns with examples)
`, id, id)
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
