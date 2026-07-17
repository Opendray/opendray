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
	proposer   *BlueprintProposer  // nil when no worker registry
	docsSvc    *projectdoc.Service // blueprint apply
	curation   *CurationService    // nil → conversation routes 503
	convStore  *ConversationStore
	settings   *SettingsStore // nil → settings routes 503
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

// WithBlueprintProposer wires the AI blueprint proposer (Phase 3).
// Nil → the propose route 503s.
func (h *Handlers) WithBlueprintProposer(p *BlueprintProposer) *Handlers {
	h.proposer = p
	return h
}

// WithDocs wires the projectdoc service for blueprint apply.
func (h *Handlers) WithDocs(docs *projectdoc.Service) *Handlers {
	h.docsSvc = docs
	return h
}

// WithCuration wires the conversation channel (Phase 4).
func (h *Handlers) WithCuration(svc *CurationService, store *ConversationStore) *Handlers {
	h.curation = svc
	h.convStore = store
	return h
}

// WithSettings wires the Cortex runtime settings store.
func (h *Handlers) WithSettings(s *SettingsStore) *Handlers {
	h.settings = s
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
		// Blueprint (Phase 3): AI-propose + operator-accept apply.
		// Per-section CRUD lives on the re-mounted projectdoc routes
		// (/cortex/project-docs/blueprint*).
		r.Post("/blueprint/propose", h.proposeBlueprint)
		r.Put("/blueprint", h.applyBlueprint)
		// Runtime settings (spawn injection mode, …).
		h.mountSettings(r)
		// Curation conversations (Phase 4).
		h.mountConversations(r)
		// KB Librarian — launch a cross-page knowledge-base admin session.
		r.Post("/librarian", h.launchLibrarian)
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

// proposeBlueprint runs the AI blueprint proposer for ?cwd=. Nothing
// is persisted — the response is shown to the operator, who applies
// it (possibly edited) via PUT /cortex/blueprint.
func (h *Handlers) proposeBlueprint(w http.ResponseWriter, r *http.Request) {
	if h.proposer == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "blueprint proposer not configured"})
		return
	}
	cwd := r.URL.Query().Get("cwd")
	prop, err := h.proposer.Propose(r.Context(), cwd)
	if err != nil {
		h.log.Error("blueprint propose failed", "cwd", cwd, "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, prop)
}

// applyBlueprint replaces a project's blueprint with the given section
// set: every listed section is upserted, every existing section absent
// from the list is removed (overview excepted — it is reserved).
// Section docs of removed sections are kept (re-adding a slug
// resurrects its content).
func (h *Handlers) applyBlueprint(w http.ResponseWriter, r *http.Request) {
	if h.docsSvc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "docs service not wired"})
		return
	}
	var body struct {
		Cwd      string               `json:"cwd"`
		Sections []projectdoc.Section `json:"sections"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if body.Cwd == "" || len(body.Sections) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cwd and sections are required"})
		return
	}
	keep := make(map[string]bool, len(body.Sections)+1)
	keep[projectdoc.SlugOverview] = true
	for _, sec := range body.Sections {
		sec.Cwd = body.Cwd
		if _, err := h.docsSvc.PutSection(r.Context(), sec); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		keep[sec.Slug] = true
	}
	existing, err := h.docsSvc.ListSections(r.Context(), body.Cwd)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	for _, sec := range existing {
		if !keep[sec.Slug] {
			if err := h.docsSvc.DeleteSection(r.Context(), body.Cwd, sec.Slug); err != nil {
				h.log.Warn("blueprint apply: delete section failed", "cwd", body.Cwd, "slug", sec.Slug, "err", err)
			}
		}
	}
	sections, err := h.docsSvc.ListSections(r.Context(), body.Cwd)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sections": sections})
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
