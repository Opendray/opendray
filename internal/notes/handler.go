package notes

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	v   *Vault
	log *slog.Logger
}

func NewHandlers(v *Vault, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{v: v, log: log.With("component", "notes.http")}
}

func (h *Handlers) Mount(r chi.Router) {
	r.Route("/notes", func(r chi.Router) {
		r.Get("/info", h.info)
		r.Get("/list", h.list)
		r.Get("/read", h.read)
		r.Put("/write", h.write)
		r.Post("/append", h.append_)
		r.Delete("/delete", h.delete)
		r.Get("/backlinks", h.backlinks)
		r.Get("/tags", h.tags)
		r.Get("/project-mapping", h.projectMappingGet)
		r.Put("/project-mapping", h.projectMappingPut)
		r.Get("/project-mappings", h.projectMappingsList)
	})
}

type writeRequest struct {
	Path string `json:"path"`
	Body string `json:"body"`
}

func (h *Handlers) info(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"root":            h.v.Root(),
		"personal_prefix": h.v.PersonalPrefix(),
		"projects_prefix": h.v.ProjectsPrefix(),
	})
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	prefix := strings.TrimSpace(r.URL.Query().Get("prefix"))
	notes, err := h.v.List(prefix)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"notes": notes})
}

func (h *Handlers) read(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimSpace(r.URL.Query().Get("path"))
	if p == "" {
		writeError(w, http.StatusBadRequest, errors.New("path is required"))
		return
	}
	n, err := h.v.Read(p)
	if err != nil {
		respond(w, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (h *Handlers) write(w http.ResponseWriter, r *http.Request) {
	var req writeRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 4<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	n, err := h.v.Write(req.Path, req.Body)
	if err != nil {
		respond(w, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (h *Handlers) append_(w http.ResponseWriter, r *http.Request) {
	var req writeRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 4<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	n, err := h.v.Append(req.Path, req.Body)
	if err != nil {
		respond(w, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimSpace(r.URL.Query().Get("path"))
	if p == "" {
		writeError(w, http.StatusBadRequest, errors.New("path is required"))
		return
	}
	if err := h.v.Delete(p); err != nil {
		respond(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) backlinks(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimSpace(r.URL.Query().Get("path"))
	if p == "" {
		writeError(w, http.StatusBadRequest, errors.New("path is required"))
		return
	}
	links, err := h.v.Backlinks(r.Context(), p)
	if err != nil {
		respond(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"links": links})
}

func (h *Handlers) tags(w http.ResponseWriter, r *http.Request) {
	prefix := strings.TrimSpace(r.URL.Query().Get("prefix"))
	tags, err := h.v.Tags(r.Context(), prefix)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tags": tags})
}

// projectMappingGet returns the resolved project directory for a cwd
// plus the auto-derived default — UI uses this to show the user
// which path will be used and what the default would be without an
// override.
func (h *Handlers) projectMappingGet(w http.ResponseWriter, r *http.Request) {
	cwd := strings.TrimSpace(r.URL.Query().Get("cwd"))
	if cwd == "" {
		writeError(w, http.StatusBadRequest, errors.New("cwd is required"))
		return
	}
	resolved := h.v.ResolvedProjectDir(cwd)
	defaultDir := h.v.ProjectDir(filepath.Base(cwd))
	custom := resolved != defaultDir
	writeJSON(w, http.StatusOK, map[string]any{
		"cwd":          cwd,
		"path":         resolved,
		"default_path": defaultDir,
		"custom":       custom,
	})
}

type projectMappingPutReq struct {
	Cwd  string `json:"cwd"`
	Path string `json:"path"`
}

// projectMappingPut sets or clears the override for a single cwd.
// Empty path = clear (revert to default). Path is validated against
// the vault root jail.
func (h *Handlers) projectMappingPut(w http.ResponseWriter, r *http.Request) {
	var req projectMappingPutReq
	if err := json.NewDecoder(io.LimitReader(r.Body, 8<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.v.SetProjectMapping(req.Cwd, req.Path); err != nil {
		respond(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) projectMappingsList(w http.ResponseWriter, _ *http.Request) {
	items, err := h.v.ListProjectMappings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"mappings": items})
}

func respond(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, ErrPathEscape), errors.Is(err, ErrInvalidPath),
		errors.Is(err, ErrNotMarkdown), errors.Is(err, ErrAlreadyExists):
		writeError(w, http.StatusBadRequest, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
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
