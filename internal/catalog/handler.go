package catalog

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

// Handlers serves the /providers REST surface. Mount under /api/v1.
type Handlers struct {
	cat    *Catalog
	prober *Prober
	bus    *eventbus.Hub
	log    *slog.Logger
}

func NewHandlers(cat *Catalog, bus *eventbus.Hub, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{cat: cat, prober: NewProber(), bus: bus, log: log.With("component", "catalog.http")}
}

func (h *Handlers) Mount(r chi.Router) {
	r.Route("/providers", func(r chi.Router) {
		r.Get("/", h.list)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.get)
			r.Patch("/config", h.updateConfig)
			r.Patch("/toggle", h.toggle)
			// Network npm lookup → its own endpoint so the list stays fast.
			r.Get("/update-check", h.updateCheck)
			// Admin-only action: patch the CLI to the latest npm version.
			r.Post("/update", h.update)
		})
	})
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	ps, err := h.cat.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// Enrich with the cheap, locally-probed install state + real version
	// (cached). The npm "update available" check is the separate
	// /update-check endpoint to keep this response snappy.
	for i := range ps {
		info := h.prober.Installed(r.Context(), ps[i].Manifest)
		ps[i].Runtime = &info
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": ps})
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := h.cat.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	info := h.prober.Installed(r.Context(), p.Manifest)
	p.Runtime = &info
	writeJSON(w, http.StatusOK, p)
}

// updateCheck probes the installed version AND the latest npm version,
// reporting whether an update is available. Separate from list because
// it makes a network call (cached for an hour).
func (h *Handlers) updateCheck(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := h.cat.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	info := h.prober.CheckUpdate(r.Context(), p.Manifest)
	writeJSON(w, http.StatusOK, info)
}

// update patches the provider's CLI to the latest npm version. The
// route sits behind the same admin auth as the rest of /api/v1; the npm
// package comes from the trusted manifest (not the request body), so
// there's no arbitrary-package vector. The outcome is published to the
// audit log via the event bus.
func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := h.cat.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	res, err := h.prober.Update(r.Context(), p.Manifest)
	if err != nil {
		h.log.Warn("provider update failed", "provider", id, "package", res.Package, "err", err)
		h.publishUpdate(id, res, false, err.Error())
		// 502: the npm install itself failed (e.g. non-writable prefix),
		// not a bad request. Surface the npm output for diagnosis.
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error":  err.Error(),
			"result": res,
		})
		return
	}
	h.log.Info("provider updated", "provider", id, "package", res.Package,
		"before", res.BeforeVersion, "after", res.AfterVersion, "changed", res.Changed)
	h.publishUpdate(id, res, true, "")
	writeJSON(w, http.StatusOK, res)
}

// publishUpdate emits a provider.updated event for the audit sink.
func (h *Handlers) publishUpdate(id string, res UpdateResult, ok bool, errMsg string) {
	if h.bus == nil {
		return
	}
	h.bus.Publish(eventbus.Event{
		Topic: "provider.updated",
		Data: map[string]any{
			"provider_id": id,
			"package":     res.Package,
			"before":      res.BeforeVersion,
			"after":       res.AfterVersion,
			"changed":     res.Changed,
			"ok":          ok,
			"error":       errMsg,
		},
	})
}

func (h *Handlers) updateConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var cfg map[string]any
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.cat.UpdateConfig(r.Context(), id, cfg); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	p, _ := h.cat.Get(r.Context(), id)
	writeJSON(w, http.StatusOK, p)
}

func (h *Handlers) toggle(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.cat.Toggle(r.Context(), id, req.Enabled); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	p, _ := h.cat.Get(r.Context(), id)
	writeJSON(w, http.StatusOK, p)
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
