package projectdoc

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// Handlers exposes project goal / plan / journal over HTTP. Mounted
// under the dual-auth group (admin OR integration token) so the
// auto-attached opendray-memory MCP subprocess can call these
// endpoints with its integration bearer alongside web/mobile
// operators.
//
// Routes (all under the gateway's /api/v1 prefix):
//
//	GET    /project-docs?cwd=…              → {docs: [Doc,…]}
//	GET    /project-docs/{kind}?cwd=…       → Doc | 404
//	PUT    /project-docs/{kind}             → Doc            body: {cwd, content, updated_by?}
//	POST   /project-doc-proposals           → Proposal       body: {cwd, kind, proposed_content, reason, session_id?}
//	GET    /project-doc-proposals/pending   → {proposals}    ?cwd=… (omit for all)
//	POST   /project-doc-proposals/{id}/approve → Doc
//	POST   /project-doc-proposals/{id}/reject  → 204
//	GET    /session-logs?cwd=…&n=…          → {logs: [LogEntry,…]}
//	POST   /session-logs                    → LogEntry       body: {cwd, kind?, session_id?, title?, content, updated_by?}
//	DELETE /session-logs/{id}               → 204
type Handlers struct {
	svc *Service
	log *slog.Logger
}

// NewHandlers wires HTTP routes against svc. svc must be non-nil —
// unlike memory the projectdoc subsystem has no "disabled" mode; the
// tables ship with every install via migration 0025.
func NewHandlers(svc *Service, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{svc: svc, log: log.With("component", "projectdoc.http")}
}

// Mount registers routes on r. r should already have auth middleware
// applied (typically the dual-auth admin+integration group).
func (h *Handlers) Mount(r chi.Router) {
	r.Route("/project-docs", func(r chi.Router) {
		r.Get("/", h.listDocs)
		r.Get("/{kind}", h.getDoc)
		r.Put("/{kind}", h.putDoc)
	})
	r.Route("/project-doc-proposals", func(r chi.Router) {
		r.Get("/pending", h.listPending)
		r.Post("/", h.propose)
		r.Post("/{id}/approve", h.approve)
		r.Post("/{id}/reject", h.reject)
	})
	r.Route("/session-logs", func(r chi.Router) {
		r.Get("/", h.listLogs)
		r.Post("/", h.appendLog)
		r.Delete("/{id}", h.deleteLog)
	})
}

// ─── project_docs ─────────────────────────────────────────────────

func (h *Handlers) listDocs(w http.ResponseWriter, r *http.Request) {
	cwd := r.URL.Query().Get("cwd")
	docs, err := h.svc.ListDocsForCwd(r.Context(), cwd)
	if err != nil {
		h.respondErr(w, err)
		return
	}
	if docs == nil {
		docs = []Doc{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"docs": docs})
}

func (h *Handlers) getDoc(w http.ResponseWriter, r *http.Request) {
	kind := Kind(chi.URLParam(r, "kind"))
	cwd := r.URL.Query().Get("cwd")
	doc, err := h.svc.GetDoc(r.Context(), cwd, kind)
	if errors.Is(err, ErrNotFound) {
		// Treat absence as an empty doc so the UI doesn't need a
		// separate "is there a doc?" probe. id stays blank so callers
		// can tell synthesised-empty from a stored row.
		writeJSON(w, http.StatusOK, Doc{Cwd: cwd, Kind: kind, UpdatedBy: AuthorOperator})
		return
	}
	if err != nil {
		h.respondErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

// putDoc is the operator-side direct write. Agent writes go through
// the propose / approve path; this handler defaults updated_by to
// 'operator' but accepts 'agent' too so admin-side scripted imports
// can still tag rows correctly.
func (h *Handlers) putDoc(w http.ResponseWriter, r *http.Request) {
	kind := Kind(chi.URLParam(r, "kind"))
	var body struct {
		Cwd       string `json:"cwd"`
		Content   string `json:"content"`
		UpdatedBy Author `json:"updated_by,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	author := body.UpdatedBy
	if author == "" {
		author = AuthorOperator
	}
	doc, err := h.svc.PutDoc(r.Context(), body.Cwd, kind, body.Content, author)
	if err != nil {
		h.respondErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

// ─── proposals ────────────────────────────────────────────────────

func (h *Handlers) propose(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Cwd             string `json:"cwd"`
		Kind            Kind   `json:"kind"`
		ProposedContent string `json:"proposed_content"`
		Reason          string `json:"reason"`
		SessionID       string `json:"session_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	p, err := h.svc.ProposeDoc(r.Context(), body.Cwd, body.Kind, body.ProposedContent, body.Reason, body.SessionID)
	if err != nil {
		h.respondErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *Handlers) listPending(w http.ResponseWriter, r *http.Request) {
	cwd := r.URL.Query().Get("cwd") // empty = global inbox
	props, err := h.svc.ListPendingProposals(r.Context(), cwd)
	if err != nil {
		h.respondErr(w, err)
		return
	}
	if props == nil {
		props = []Proposal{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"proposals": props})
}

func (h *Handlers) approve(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	doc, err := h.svc.ApproveProposal(r.Context(), id)
	if err != nil {
		h.respondErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *Handlers) reject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.RejectProposal(r.Context(), id); err != nil {
		h.respondErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── session_logs ─────────────────────────────────────────────────

func (h *Handlers) listLogs(w http.ResponseWriter, r *http.Request) {
	cwd := r.URL.Query().Get("cwd")
	limit := 50
	if v := r.URL.Query().Get("n"); v != "" {
		if x, err := strconv.Atoi(v); err == nil && x > 0 {
			limit = x
		}
	}
	logs, err := h.svc.ListLogs(r.Context(), cwd, limit)
	if err != nil {
		h.respondErr(w, err)
		return
	}
	if logs == nil {
		logs = []LogEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"logs": logs})
}

func (h *Handlers) appendLog(w http.ResponseWriter, r *http.Request) {
	var body LogEntry
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	entry, err := h.svc.AppendLog(r.Context(), body)
	if err != nil {
		h.respondErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, entry)
}

func (h *Handlers) deleteLog(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteLog(r.Context(), id); err != nil {
		h.respondErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// respondErr maps Service sentinel errors to HTTP codes. Anything
// outside the known set becomes a 500 with the message preserved
// (callers are admins / agents on a private network).
func (h *Handlers) respondErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, ErrAlreadyDecided):
		// 409 — request can't be reapplied to current state.
		writeError(w, http.StatusConflict, err)
	case errors.Is(err, ErrInvalidKind),
		errors.Is(err, ErrInvalidLogKind),
		errors.Is(err, ErrEmptyCwd):
		writeError(w, http.StatusBadRequest, err)
	default:
		h.log.Warn("projectdoc handler error", "err", err)
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
