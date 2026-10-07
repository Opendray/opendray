package opencodeacct

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	svc *Service
	log *slog.Logger
}

func NewHandlers(svc *Service, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{svc: svc, log: log.With("component", "opencodeacct.http")}
}

func (h *Handlers) Mount(r chi.Router) {
	r.Route("/opencode-accounts", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Post("/import-local", h.importLocal)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.get)
			r.Put("/", h.update)
			r.Patch("/toggle", h.toggle)
			r.Delete("/", h.del)
		})
	})
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	accs, err := h.svc.List(r.Context())
	if err != nil {
		h.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"accounts": accs})
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	a, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid JSON body"))
		return
	}
	a, err := h.svc.Create(r.Context(), req)
	if err != nil {
		h.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid JSON body"))
		return
	}
	a, err := h.svc.Update(r.Context(), chi.URLParam(r, "id"), req)
	if err != nil {
		h.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *Handlers) toggle(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid JSON body"))
		return
	}
	a, err := h.svc.Update(r.Context(), chi.URLParam(r, "id"), UpdateRequest{Enabled: &body.Enabled})
	if err != nil {
		h.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *Handlers) del(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		h.respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) importLocal(w http.ResponseWriter, r *http.Request) {
	var req ImportLocalRequest
	// Body is optional.
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, errors.New("invalid JSON body"))
		return
	}
	a, err := h.svc.ImportLocal(r.Context(), req)
	if err != nil {
		h.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

// respondError maps service errors to HTTP. Error strings never contain
// credential values (validation messages name providers only).
func (h *Handlers) respondError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, ErrDuplicate), errors.Is(err, ErrDisabled):
		writeError(w, http.StatusConflict, err)
	case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrNoCredentials):
		writeError(w, http.StatusBadRequest, err)
	case errors.Is(err, ErrCipherRequired):
		writeError(w, http.StatusPreconditionFailed, err)
	default:
		h.log.Error("opencodeacct handler", "err", err)
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
