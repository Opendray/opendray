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
//	POST /round-tables/{id}/reopen      → 204 (closed → active)
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
		r.Patch("/{id}", h.update)
		r.Post("/{id}/messages", h.postMessage)
		r.Post("/{id}/continue", h.continueDiscussion)
		r.Post("/{id}/summarize", h.summarize)
		r.Post("/{id}/plan/draft", h.draftPlan)
		r.Put("/{id}/plan", h.setPlan)
		r.Post("/{id}/plan/run", h.runStep)
		r.Post("/{id}/handoff", h.handoff)
		r.Post("/{id}/close", h.close)
		r.Post("/{id}/reopen", h.reopen)
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
		Topic   string `json:"topic"`
		Cwd     string `json:"cwd"`
		Seats   []Seat `json:"seats"`
		Framing string `json:"framing"`
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
	// Framing is optional at creation; set it in a follow-up so Create's
	// signature stays stable for its other callers.
	if body.Framing != "" {
		if err := h.store.SetFraming(r.Context(), rt.ID, body.Framing); err == nil {
			rt.Framing = body.Framing
		}
	}
	writeJSON(w, http.StatusCreated, rt)
}

// update patches a live round table's framing and/or seats — the operator
// reassigns roles/relationships as the discussion moves to a new topic. Both
// fields optional; only the ones present in the body are changed.
func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	id := chi.URLParam(r, "id")
	var body struct {
		Framing *string `json:"framing"`
		Cwd     *string `json:"cwd"`
		Seats   []Seat  `json:"seats"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if body.Seats != nil {
		if err := h.store.UpdateSeats(r.Context(), id, body.Seats); err != nil {
			h.updateErr(w, err)
			return
		}
	}
	if body.Framing != nil {
		if err := h.store.SetFraming(r.Context(), id, *body.Framing); err != nil {
			h.updateErr(w, err)
			return
		}
	}
	// Cwd can be bound after creation so a plan drafted on a table with no
	// project can still be run (the step sessions need a shared working tree).
	if body.Cwd != nil {
		if err := h.store.SetCwd(r.Context(), id, *body.Cwd); err != nil {
			h.updateErr(w, err)
			return
		}
	}
	rt, err := h.store.Get(r.Context(), id)
	if err != nil {
		h.updateErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rt)
}

func (h *Handlers) updateErr(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
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

// continueDiscussion resumes a paused auto-discussion for another burst.
func (h *Handlers) continueDiscussion(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	if err := h.svc.Continue(r.Context(), chi.URLParam(r, "id")); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusAccepted)
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

// draftPlan asks a member to break the discussion into a role-assigned plan.
func (h *Handlers) draftPlan(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	var body struct {
		Provider string `json:"provider"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if err := h.svc.DraftPlan(r.Context(), chi.URLParam(r, "id"), body.Provider); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// setPlan replaces the plan (operator edits the drafted steps).
func (h *Handlers) setPlan(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	id := chi.URLParam(r, "id")
	var body struct {
		Steps []PlanStep `json:"steps"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := h.store.SetPlan(r.Context(), id, body.Steps); err != nil {
		h.updateErr(w, err)
		return
	}
	rt, err := h.store.Get(r.Context(), id)
	if err != nil {
		h.updateErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rt)
}

// runStep launches a real session to carry out one plan step. Returns the new
// session id so the UI can navigate to it.
func (h *Handlers) runStep(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	var body struct {
		Index     int      `json:"index"`
		Cwd       string   `json:"cwd"`
		AccountID string   `json:"account_id"`
		Args      []string `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	sid, err := h.svc.RunStep(r.Context(), chi.URLParam(r, "id"), body.Index, body.Cwd, body.AccountID, body.Args)
	if errors.Is(err, ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if errors.Is(err, ErrHandoffUnavailable) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"session_id": sid})
}

// handoff spawns a real agent session to implement the discussion. Returns
// the new session id so the UI can navigate to it.
func (h *Handlers) handoff(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	var body struct {
		Cwd       string   `json:"cwd"`
		Provider  string   `json:"provider"`
		Model     string   `json:"model"`
		AccountID string   `json:"account_id"`
		ForceNew  bool     `json:"force_new"`
		Args      []string `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	sid, err := h.svc.Handoff(r.Context(), chi.URLParam(r, "id"), body.Cwd, body.Provider, body.Model, body.AccountID, body.ForceNew, body.Args)
	if errors.Is(err, ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if errors.Is(err, ErrHandoffUnavailable) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"session_id": sid})
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

// reopen flips a closed chat back to active so the operator can resume the
// discussion — close keeps the thread, so nothing was lost, and reopening
// re-enables posting/@mentions.
func (h *Handlers) reopen(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	if err := h.store.SetStatus(r.Context(), chi.URLParam(r, "id"), StatusActive); err != nil {
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
