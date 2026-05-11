package backup

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// MountImports registers the /imports endpoints. Caller mounts
// under the admin-only group.
func (h *Handlers) MountImports(r chi.Router) {
	r.Route("/imports", func(r chi.Router) {
		r.Get("/", h.listImports)
		r.Post("/", h.createImport)
		r.Get("/{id}", h.getImport)
	})
}

func (h *Handlers) listImports(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	list, err := h.live.Service().ListImports(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if list == nil {
		list = []Import{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"imports": list})
}

func (h *Handlers) getImport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	imp, err := h.live.Service().GetImport(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrImportNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, imp)
}

// createImport serves POST /imports. Multipart form:
//
//	bundle (file)         — the export zip
//	memories (bool)       — import the memories.jsonl section
//	integrations (bool)   — import the integrations.jsonl section
//	custom_tasks (bool)   — import the custom_tasks.jsonl section
//	requested_by (string) — audit username
//
// Returns the Import row (with status + counts) on success. On
// per-section failure returns 200 with the row's status='failed'
// — the row itself is the response body.
func (h *Handlers) createImport(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("multipart: %w", err))
		return
	}
	file, hdr, err := r.FormFile("bundle")
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("missing bundle file: %w", err))
		return
	}
	defer file.Close()

	imp, err := h.live.Service().ImportBundle(r.Context(), ImportRequest{
		Source:       file,
		Filename:     hdr.Filename,
		RequestedBy:  r.FormValue("requested_by"),
		Memories:     r.FormValue("memories") == "true",
		Integrations: r.FormValue("integrations") == "true",
		CustomTasks:  r.FormValue("custom_tasks") == "true",
	})
	if err != nil {
		// Hard failures (bad bundle, invalid request) → 400.
		// Soft failures (per-section partial) → 200 with the row;
		// the UI distinguishes by checking imp.status.
		switch {
		case errors.Is(err, ErrImportBadBundle):
			writeError(w, http.StatusBadRequest, err)
			return
		}
		// Persisted soft failure: return the row so the UI shows
		// counts + the section that failed.
		writeJSON(w, http.StatusOK, imp)
		return
	}
	writeJSON(w, http.StatusCreated, imp)
}
