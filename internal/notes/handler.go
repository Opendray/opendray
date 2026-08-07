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
		r.Post("/move", h.move)
		r.Get("/templates", h.templates)
		r.Post("/new", h.newFromTemplate)
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

// templates lists the shapes a new note can start from. Rendering is
// server-side (see templates.go), so this is the clients' only source
// of truth for what exists.
func (h *Handlers) templates(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"templates": h.v.Templates()})
}

// newFromTemplate creates a note from a template. Distinct from /write
// because it refuses to overwrite — "new" that silently replaces an
// existing doc loses work to a mistyped filename.
func (h *Handlers) newFromTemplate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path     string `json:"path"`
		Template string `json:"template"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	path := strings.TrimSpace(req.Path)
	if path == "" {
		writeError(w, http.StatusBadRequest, errors.New("path is required"))
		return
	}
	n, err := h.v.NewFromTemplate(path, strings.TrimSpace(req.Template))
	if err != nil {
		respond(w, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

// move renames/relocates a note and repoints the wiki-links that
// referenced it. Reports which notes were rewritten so the UI can say
// so — a rename that silently edits other files would be worse than
// one that doesn't rename at all.
func (h *Handlers) move(w http.ResponseWriter, r *http.Request) {
	var req struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	from := strings.TrimSpace(req.From)
	to := strings.TrimSpace(req.To)
	if from == "" || to == "" {
		writeError(w, http.StatusBadRequest, errors.New("from and to are required"))
		return
	}
	res, err := h.v.Move(r.Context(), from, to)
	if err != nil {
		// The note may have moved even when the link rewrite failed;
		// saying "move failed" then would send the operator looking for
		// a file that is no longer where they left it.
		if res.To != "" {
			writeJSON(w, http.StatusOK, map[string]any{
				"moved": res, "warning": err.Error(),
			})
			return
		}
		respond(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
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
