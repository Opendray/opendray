package roundtable

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handlers exposes the Round Table REST surface — a cross-vendor AI group
// chat. Self-contained: mounts its own /round-tables group so the feature
// can be removed by deleting this package + one wiring block (ROLLBACK.md).
//
// Routes (under the gateway's /api/v1 dual-auth group):
//
//	POST /round-tables                  {topic, cwd?, seats:[{provider,model?,account_id?}]}
//	GET  /round-tables?cwd=             → {round_tables}
//	GET  /round-tables/{id}             → {round_table, messages}
//	POST /round-tables/{id}/messages    {content} → operator Message (202; @mentioned members reply async)
//	POST /round-tables/{id}/summarize   {provider?} → 202 (a member condenses the chat; lands async)
//	POST /round-tables/{id}/close       → 204
type Handlers struct {
	store *Store
	svc   *Service
	log   *slog.Logger
}

// NewHandlers wires the surface. store + svc required.
func NewHandlers(store *Store, svc *Service, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{store: store, svc: svc, log: log.With("component", "roundtable.http")}
}

// Mount registers the routes on r (already inside the dual-auth group).
func (h *Handlers) Mount(r chi.Router) {
	r.Route("/round-tables", func(r chi.Router) {
		r.Post("/", h.create)
		r.Get("/", h.list)
		// Literal /models must be registered before /{id} so chi routes it
		// to the model catalog, not the get-by-id handler.
		r.Get("/models", h.models)
		r.Get("/{id}", h.get)
		r.Post("/{id}/messages", h.postMessage)
		r.Post("/{id}/summarize", h.summarize)
		r.Post("/{id}/close", h.close)
		r.Delete("/{id}", h.remove)
	})
}

func (h *Handlers) ready(w http.ResponseWriter) bool {
	if h.store == nil || h.svc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "round table not configured"})
		return false
	}
	return true
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	var body struct {
		Topic string `json:"topic"`
		Cwd   string `json:"cwd"`
		Seats []Seat `json:"seats"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	rt, err := h.store.Create(r.Context(), body.Topic, body.Cwd, body.Seats, OriginOperator, "")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, rt)
}

// models returns the selectable model options per seat provider, so the
// create dialog can offer a dropdown instead of a hand-typed model string.
func (h *Handlers) models(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"models": ProviderModelOptions(r.Context()),
	})
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	tables, err := h.store.List(r.Context(), r.URL.Query().Get("cwd"), 0)
	if err != nil {
		h.log.Error("list round tables failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list failed"})
		return
	}
	if tables == nil {
		tables = []RoundTable{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"round_tables": tables})
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	id := chi.URLParam(r, "id")
	rt, err := h.store.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	msgs, err := h.store.Messages(r.Context(), id, 0)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if msgs == nil {
		msgs = []Message{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"round_table": rt, "messages": msgs})
}

func (h *Handlers) postMessage(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	msg, err := h.svc.PostMessage(r.Context(), chi.URLParam(r, "id"), body.Content)
	if errors.Is(err, ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, msg)
}

func (h *Handlers) summarize(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	var body struct {
		Provider string `json:"provider"`
	}
	// Body is optional — default to the first seat.
	_ = json.NewDecoder(r.Body).Decode(&body)
	if err := h.svc.Summarize(r.Context(), chi.URLParam(r, "id"), body.Provider); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handlers) close(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	if err := h.store.SetStatus(r.Context(), chi.URLParam(r, "id"), StatusClosed); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) remove(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	if err := h.store.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
