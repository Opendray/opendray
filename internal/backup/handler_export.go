package backup

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// MountExports registers the /exports endpoints. Caller mounts under
// the admin-only group; token check on download is a second factor.
func (h *Handlers) MountExports(r chi.Router) {
	r.Route("/exports", func(r chi.Router) {
		r.Get("/", h.listExports)
		r.Post("/", h.createExport)
		r.Get("/{id}", h.getExport)
		r.Get("/{id}/download", h.downloadExport)
		r.Delete("/{id}", h.deleteExport)
	})
}

func (h *Handlers) listExports(w http.ResponseWriter, r *http.Request) {
	list, err := h.live.Service().ListExports(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if list == nil {
		list = []Export{}
	}
	// Don't echo download_token in list (operator already saw it on
	// create; future tooling can re-fetch via /exports/{id} if
	// they recorded the id).
	for i := range list {
		list[i].DownloadToken = ""
	}
	writeJSON(w, http.StatusOK, map[string]any{"exports": list})
}

func (h *Handlers) getExport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	e, err := h.live.Service().GetExport(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrExportNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// Reveal the token here (this is the admin-authed channel).
	writeJSON(w, http.StatusOK, e)
}

func (h *Handlers) createExport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RequestedBy  string                `json:"requested_by"`
		Memories     bool                  `json:"memories"`
		Integrations IntegrationExportMode `json:"integrations"`
		CustomTasks  bool                  `json:"custom_tasks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err))
		return
	}
	e, err := h.live.Service().CreateExport(r.Context(), ExportRequest{
		RequestedBy:  req.RequestedBy,
		Memories:     req.Memories,
		Integrations: req.Integrations,
		CustomTasks:  req.CustomTasks,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

// downloadExport requires both admin auth (already enforced by the
// router group) AND a valid ?token= so a leaked download URL alone
// isn't enough to download the bundle.
func (h *Handlers) downloadExport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	token := r.URL.Query().Get("token")
	if token == "" {
		writeError(w, http.StatusBadRequest, errors.New("token query param required"))
		return
	}
	rc, e, err := h.live.Service().DownloadExport(r.Context(), id, token)
	if err != nil {
		switch {
		case errors.Is(err, ErrExportNotFound), errors.Is(err, ErrInvalidDownloadToken):
			writeError(w, http.StatusNotFound, errors.New("export not found"))
		case errors.Is(err, ErrExportExpired):
			writeError(w, http.StatusGone, err)
		default:
			writeError(w, http.StatusInternalServerError, err)
		}
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+e.ID+`.zip"`)
	if e.Bytes > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(e.Bytes, 10))
	}
	_, _ = io.Copy(w, rc)
}

func (h *Handlers) deleteExport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.live.Service().DeleteExport(r.Context(), id); err != nil {
		if errors.Is(err, ErrExportNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
