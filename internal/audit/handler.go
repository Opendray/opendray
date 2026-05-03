package audit

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// Handlers serves /api/v1/audit/* read endpoints. Admin-only —
// caller wraps with admin middleware before mounting.
type Handlers struct {
	svc *Service
	log *slog.Logger
}

func NewHandlers(svc *Service, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{svc: svc, log: log.With("component", "audit.http")}
}

func (h *Handlers) Mount(r chi.Router) {
	r.Get("/audit/log", h.list)
}

// list serves GET /api/v1/audit/log with query params:
//
//	subject_kind=session|integration|channel|admin
//	subject_id=<exact id>
//	action=session.idle  OR  action=session.* (prefix match)
//	since=<RFC3339>      until=<RFC3339>      (half-open: [since, until))
//	cursor=<int>         limit=<int, default 100, max 500>
//
// Returns {"entries": [...], "next_cursor": "<int>" | null}.
func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	opts := QueryOpts{
		SubjectKind: q.Get("subject_kind"),
		SubjectID:   q.Get("subject_id"),
		Action:      q.Get("action"),
	}

	if v := q.Get("since"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "since: "+err.Error())
			return
		}
		opts.Since = t
	}
	if v := q.Get("until"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "until: "+err.Error())
			return
		}
		opts.Until = t
	}
	if v := q.Get("cursor"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "cursor must be a positive integer")
			return
		}
		opts.Cursor = n
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		opts.Limit = n
	}

	entries, next, err := h.svc.Query(r.Context(), opts)
	if err != nil {
		h.log.Error("audit query failed", "err", err)
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}

	resp := map[string]any{"entries": entries}
	if next > 0 {
		resp["next_cursor"] = strconv.FormatInt(next, 10)
	} else {
		resp["next_cursor"] = nil
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
