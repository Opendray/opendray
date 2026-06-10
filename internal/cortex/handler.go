package cortex

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray-v2/internal/knowledge"
	"github.com/opendray/opendray-v2/internal/memory"
	"github.com/opendray/opendray-v2/internal/projectdoc"
)

// Handlers exposes the unified Cortex namespace. It owns only the
// cross-layer endpoints (status; quarantine, blueprint, and
// conversations in later phases) and re-mounts the existing layer
// handler sets under /cortex so web/mobile can consume one coherent
// namespace while the legacy mounts (/project-docs, /memory,
// /knowledge) stay untouched for integrations and the not-yet-migrated
// mobile app.
//
// Routes (all under the gateway's /api/v1 prefix, dual-auth group):
//
//	GET /cortex/status                → Status (flywheel aggregation)
//	    /cortex/project-docs*         → same handlers as /project-docs*
//	    /cortex/project-doc-proposals* /cortex/session-logs*
//	    /cortex/memory*               → same handlers as /memory*
//	    /cortex/knowledge*            → same handlers as /knowledge*
type Handlers struct {
	svc        *Service
	docs       *projectdoc.Handlers
	mem        *memory.Handlers
	know       *knowledge.Handlers // nil when the knowledge layer is disabled
	quarantine QuarantineSource    // nil when memory is disabled
	log        *slog.Logger
}

// NewHandlers wires the Cortex HTTP surface. svc and docs must be
// non-nil; know may be nil (knowledge disabled).
func NewHandlers(svc *Service, docs *projectdoc.Handlers, mem *memory.Handlers, know *knowledge.Handlers, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{
		svc:  svc,
		docs: docs,
		mem:  mem,
		know: know,
		log:  log.With("component", "cortex.http"),
	}
}

// WithQuarantine wires the quarantine review queue (Phase 2). q may be
// nil when memory is disabled — the routes then 503.
func (h *Handlers) WithQuarantine(q QuarantineSource) *Handlers {
	h.quarantine = q
	return h
}

// Mount registers the /cortex namespace on r. r should already have
// the dual-auth (admin OR integration) middleware applied — the
// re-mounted layer handlers enforce their own per-route scopes
// exactly as they do on the legacy mounts.
func (h *Handlers) Mount(r chi.Router) {
	r.Route("/cortex", func(r chi.Router) {
		r.Get("/status", h.status)
		// Quarantine routes register before the re-mounted memory
		// handlers so /memory/quarantine wins over memory's /{id}.
		h.mountQuarantine(r)
		h.docs.Mount(r)
		h.mem.Mount(r)
		if h.know != nil {
			h.know.Mount(r)
		}
	})
}

func (h *Handlers) status(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.Status(r.Context())
	if err != nil {
		h.log.Error("status failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cortex status failed"})
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
