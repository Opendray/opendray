package roundtable

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handlers exposes the Round Table REST surface. Self-contained — mounts
// its own /round-tables group so the feature can be removed by deleting
// this package + one wiring block (see ROLLBACK.md).
//
// Routes (under the gateway's /api/v1 dual-auth group):
//
//	POST /round-tables                  body: {topic, cwd?, seats:[{provider,model?,account_id?}]}
//	GET  /round-tables?cwd=             → {round_tables}
//	GET  /round-tables/{id}             → {round_table, turns}
//	POST /round-tables/{id}/start       → 202 (discussion runs async; watch eventbus "roundtable.updated")
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
		r.Get("/{id}", h.get)
		r.Post("/{id}/start", h.start)
		r.Post("/{id}/close", h.close)
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
	turns, err := h.store.Turns(r.Context(), id, 0)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if turns == nil {
		turns = []Turn{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"round_table": rt, "turns": turns})
}

func (h *Handlers) start(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.svc.Start(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	rt, err := h.store.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, rt)
}

func (h *Handlers) close(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.store.SetStatus(r.Context(), id, StatusClosed, ""); err != nil {
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
