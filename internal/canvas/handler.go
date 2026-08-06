package canvas

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// Handlers exposes the Canvas REST surface. Self-contained: mounts its own
// /canvas group so the feature can be removed by deleting this package + one
// wiring block.
//
// Routes (under the gateway's /api/v1 dual-auth group):
//
//	POST   /canvas                {cwd, slug?, title, kind?, html} → 201 Artifact (agent render, via canvas_render)
//	GET    /canvas?cwd=           → {artifacts:[Summary]}                    (Canvas tab selector)
//	POST   /canvas/focus          {cwd, slug, session_id?, notify?}          → 200 (which canvas we're discussing)
//	GET    /canvas/focus?cwd=     → {slug, title, kind}
//	GET    /canvas/design?cwd=    → DesignSystem                             (tokens + style notes)
//	POST   /canvas/design         {cwd, tokens, notes}                       → 200 DesignSystem
//	POST   /canvas/design/task    {session_id, cwd, task}                    → 202 (extract | showcase)
//	GET    /canvas/{id}           → Artifact (with html)                     (panel render)
//	POST   /canvas/{id}/feedback  {session_id, message?, annotations[]}      → 202 (seeded into the session)
//	DELETE /canvas/{id}           → 204
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
	return &Handlers{store: store, svc: svc, log: log.With("component", "canvas.http")}
}

// Mount registers the routes on r (already inside the dual-auth group).
func (h *Handlers) Mount(r chi.Router) {
	r.Route("/canvas", func(r chi.Router) {
		r.Post("/", h.render)
		r.Get("/", h.list)
		// Literal paths before /{id} so chi routes them here, not to get-by-id.
		r.Post("/request", h.request)
		r.Post("/focus", h.setFocus)
		r.Get("/focus", h.getFocus)
		r.Get("/design", h.getDesign)
		r.Post("/design", h.setDesign)
		r.Post("/design/task", h.designTask)
		r.Get("/{id}", h.get)
		r.Post("/{id}/feedback", h.feedback)
		r.Delete("/{id}", h.remove)
	})
}

func (h *Handlers) ready(w http.ResponseWriter) bool {
	if h.store == nil || h.svc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "canvas not configured"})
		return false
	}
	return true
}

func (h *Handlers) render(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	var body struct {
		Cwd   string `json:"cwd"`
		Slug  string `json:"slug"`
		Title string `json:"title"`
		Kind  string `json:"kind"`
		HTML  string `json:"html"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	a, err := h.svc.Render(r.Context(), body.Cwd, body.Slug, body.Title, body.Kind, body.HTML)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	cwd := strings.TrimSpace(r.URL.Query().Get("cwd"))
	if cwd == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cwd query param required"})
		return
	}
	items, err := h.store.List(r.Context(), cwd)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"artifacts": items})
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	a, err := h.store.Get(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, a)
}

// request is opendray's deterministic Canvas entry point: the operator types a
// design ask in the Canvas panel and it is seeded into the live session as a
// prompt that names canvas_render — no reliance on the agent guessing, no
// override of the agent's native tools.
func (h *Handlers) request(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	var body struct {
		SessionID string `json:"session_id"`
		Cwd       string `json:"cwd"`
		Prompt    string `json:"prompt"`
		Slug      string `json:"slug"`
		Title     string `json:"title"`
		Kind      string `json:"kind"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := h.svc.RequestDesign(r.Context(), body.SessionID, body.Cwd, body.Prompt, body.Slug, body.Title, body.Kind); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "requested"})
}

// setFocus records which canvas the operator is working on, so plain
// conversation in the session resolves to it. notify=true (an explicit switch)
// also seeds a one-line focus note into the session.
func (h *Handlers) setFocus(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	var body struct {
		Cwd       string `json:"cwd"`
		Slug      string `json:"slug"`
		SessionID string `json:"session_id"`
		Notify    bool   `json:"notify"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	a, err := h.svc.SetFocus(r.Context(), body.Cwd, body.Slug, body.SessionID, body.Notify)
	if errors.Is(err, ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"slug": a.Slug, "title": a.Title, "kind": a.Kind})
}

// getFocus returns the project's focused canvas (empty slug when none).
func (h *Handlers) getFocus(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	cwd := strings.TrimSpace(r.URL.Query().Get("cwd"))
	if cwd == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cwd query param required"})
		return
	}
	slug := h.svc.GetFocus(cwd)
	out := map[string]string{"slug": slug}
	if slug != "" {
		if a, err := h.store.GetBySlug(r.Context(), cwd, slug); err == nil {
			out["title"], out["kind"] = a.Title, a.Kind
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// getDesign returns the project's canvas design system (empty when unset).
func (h *Handlers) getDesign(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	cwd := strings.TrimSpace(r.URL.Query().Get("cwd"))
	if cwd == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cwd query param required"})
		return
	}
	d, err := h.store.GetDesign(r.Context(), cwd)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, d.View())
}

// setDesign replaces the project's canvas design system. Both the operator
// (from the panel) and an agent (via the canvas_design MCP tool, after reading
// the project's real theme) write through here.
func (h *Handlers) setDesign(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	// PreserveNotation is set by the panels, whose colour picker can only emit
	// hex; an agent writing through canvas_design omits it and keeps the
	// notation it chose. See Store.SetDesign.
	var body struct {
		DesignSystem
		PreserveNotation bool `json:"preserve_notation"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	d, err := h.store.SetDesign(r.Context(), body.DesignSystem, body.PreserveNotation)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, d.View())
}

// designTask hands the operator's one-click design-system jobs to the agent:
// read the project's real theme and record it, or draw the system as a canvas.
func (h *Handlers) designTask(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	var body struct {
		SessionID string `json:"session_id"`
		Cwd       string `json:"cwd"`
		Task      string `json:"task"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := h.svc.RequestDesignTask(r.Context(), body.SessionID, body.Cwd, body.Task); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "requested"})
}

func (h *Handlers) feedback(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	var fb Feedback
	if err := json.NewDecoder(r.Body).Decode(&fb); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	err := h.svc.SubmitFeedback(r.Context(), chi.URLParam(r, "id"), fb)
	if errors.Is(err, ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "seeded"})
}

func (h *Handlers) remove(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	err := h.store.Delete(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if err != nil {
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
