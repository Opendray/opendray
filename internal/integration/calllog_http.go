package integration

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Middleware records the request in integration_call_log AFTER the
// handler completes, but only when the request was authenticated as
// an integration (not admin). Mount AFTER CombinedMiddleware so the
// principal is available in the request context.
//
// Direction is hard-coded to "inbound" — outbound calls are logged
// explicitly by ProxyHandlers.serve so this middleware sees only the
// third-party-app-to-opendray flow.
func (cl *CallLogger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		p, ok := CurrentPrincipal(r.Context())
		if !ok || p.Kind != KindIntegration {
			return
		}
		cl.record(writeMsg{
			integrationID: p.ID,
			direction:     "inbound",
			method:        r.Method,
			path:          r.URL.Path,
			statusCode:    ww.Status(),
			durationMS:    int(time.Since(start).Milliseconds()),
			bytesWritten:  int64(ww.BytesWritten()),
			hasBytes:      true,
			requestID:     middleware.GetReqID(r.Context()),
		})
	})
}

// CallLogHandlers serves the read endpoints for integration_call_log.
// Admin-only — caller wraps with the admin middleware before mounting.
type CallLogHandlers struct {
	cl  *CallLogger
	log *slog.Logger
}

func NewCallLogHandlers(cl *CallLogger, log *slog.Logger) *CallLogHandlers {
	if log == nil {
		log = slog.Default()
	}
	return &CallLogHandlers{
		cl:  cl,
		log: log.With("component", "integration.calllog.http"),
	}
}

// Mount registers the read endpoint. We use a "_calls" suffix to
// stay namespaced under /integrations without colliding with the
// admin CRUD `/integrations/{id}` route — chi prefers static
// segments over wildcards so this resolves correctly.
//
// TODO(adr-0010): per-integration scoped endpoint
// `GET /integrations/{id}/calls` once Integrations gets a detail
// page. Internally it would call h.cl.Query with IntegrationID
// pre-populated from the URL param.
//
// TODO(adr-0010): summary endpoint
// `GET /integrations/_calls/summary?since=...` returning per-integration
// rollups (count, error_count, p95) for the deferred KPI cards.
func (h *CallLogHandlers) Mount(r chi.Router) {
	r.Get("/integrations/_calls", h.list)
}

// list serves GET /api/v1/integrations/_calls with query params:
//
//	integration_id    filter to one integration
//	direction         "inbound" | "outbound"
//	status_class      2|3|4|5 (HTTP status family)
//	since/until       RFC3339; half-open [since, until)
//	cursor            int; rows returned have id < cursor
//	limit             1..500, default 100
//
// Response: {"entries":[...], "next_cursor": "<int>" | null}
func (h *CallLogHandlers) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	opts := CallQueryOpts{
		IntegrationID: q.Get("integration_id"),
		Direction:     q.Get("direction"),
	}

	if v := q.Get("status_class"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 5 {
			writeJSONErr(w, http.StatusBadRequest, "status_class must be 1..5")
			return
		}
		opts.StatusClass = n
	}
	if v := q.Get("since"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeJSONErr(w, http.StatusBadRequest, "since: "+err.Error())
			return
		}
		opts.Since = t
	}
	if v := q.Get("until"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeJSONErr(w, http.StatusBadRequest, "until: "+err.Error())
			return
		}
		opts.Until = t
	}
	if v := q.Get("cursor"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 0 {
			writeJSONErr(w, http.StatusBadRequest, "cursor must be a positive integer")
			return
		}
		opts.Cursor = n
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			writeJSONErr(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		opts.Limit = n
	}

	entries, next, err := h.cl.Query(r.Context(), opts)
	if err != nil {
		h.log.Error("call log query failed", "err", err)
		writeJSONErr(w, http.StatusInternalServerError, "query failed")
		return
	}

	resp := map[string]any{"entries": entries}
	if next > 0 {
		resp["next_cursor"] = strconv.FormatInt(next, 10)
	} else {
		resp["next_cursor"] = nil
	}
	writeJSONOK(w, http.StatusOK, resp)
}

// Local helpers — the package's other handler.go has its own
// writeJSON/writeError that takes an `error`; these accept a string
// so we don't have to construct an error value for static messages.
func writeJSONOK(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func writeJSONErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
