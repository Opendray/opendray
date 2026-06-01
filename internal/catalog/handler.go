package catalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray-v2/internal/eventbus"
	"github.com/opendray/opendray-v2/internal/integration"
)

// Scopes gating the provider *mutation* endpoints. Reads (list / get /
// update-check) stay open to any authenticated principal; writes and the
// code-running update require admin, or an integration key explicitly
// granted the scope. This keeps a plain integration key from changing
// what executes on the box (config.command override, npm install).
const (
	scopeProvidersWrite  = "providers:write"  // config / toggle
	scopeProvidersUpdate = "providers:update" // npm install -g
)

// requirePrivileged allows admins unconditionally, and integration keys
// only when they hold `scope`. Returns false (and writes the response)
// when denied. Mirrors the admin-or-scope check in integration/events.go.
func (h *Handlers) requirePrivileged(w http.ResponseWriter, r *http.Request, scope string) bool {
	p, ok := integration.CurrentPrincipal(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return false
	}
	if p.Kind == integration.KindAdmin || integration.HasScope(p.Scopes, scope) {
		return true
	}
	h.log.Warn("provider mutation denied",
		"principal", p.ID, "kind", p.Kind, "scope", scope)
	writeError(w, http.StatusForbidden,
		fmt.Errorf("requires admin or the %q scope", scope))
	return false
}

// Handlers serves the /providers REST surface. Mount under /api/v1.
type Handlers struct {
	cat    *Catalog
	prober *Prober
	bus    *eventbus.Hub
	log    *slog.Logger
	// activeCountFor returns the count of currently non-terminal
	// sessions on a provider, used to populate RuntimeInfo.ActiveSessions
	// for the upgrade UI. Optional; nil → ActiveSessions is left at 0.
	activeCountFor func(providerID string) int
}

func NewHandlers(cat *Catalog, bus *eventbus.Hub, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{cat: cat, prober: NewProber(), bus: bus, log: log.With("component", "catalog.http")}
}

// WithSessionCounter wires in a per-provider active-session counter so
// the update-check response includes how many live sessions are using
// the provider's CLI. Returns the receiver so it chains off the ctor.
func (h *Handlers) WithSessionCounter(f func(providerID string) int) *Handlers {
	h.activeCountFor = f
	return h
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
	if h.activeCountFor != nil {
		info.ActiveSessions = h.activeCountFor(id)
	}
	writeJSON(w, http.StatusOK, info)
}

// update patches the provider's CLI to the latest npm version. The
// route sits behind the same admin auth as the rest of /api/v1; the npm
// package comes from the trusted manifest (not the request body), so
// there's no arbitrary-package vector. The outcome is published to the
// audit log via the event bus.
func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	if !h.requirePrivileged(w, r, scopeProvidersUpdate) {
		return
	}
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
	switch {
	case errors.Is(err, ErrUpdatePrefixReadonly):
		// Not an error condition the operator can act on by retrying:
		// the daemon is unprivileged and the npm prefix is root-owned.
		// Report "unavailable" so the UI shows guidance, not a failure.
		res.Available = false
		res.Reason = "In-app updates aren't available here — the npm global prefix isn't writable by the opendray service. " +
			"Run `sudo bash scripts/enable-cli-updates.sh` on the host to set up an opendray-owned prefix and enable one-tap updates."
		h.log.Info("provider update unavailable (read-only prefix)", "provider", id)
		h.publishUpdate(id, res, false, "prefix-readonly")
		writeJSON(w, http.StatusOK, res)
		return
	case err != nil:
		h.log.Warn("provider update failed", "provider", id, "package", res.Package, "err", err)
		res.Available = true
		h.publishUpdate(id, res, false, err.Error())
		// 502: the npm install itself failed (registry error, bad package,
		// …). Surface the npm output for diagnosis.
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error":  err.Error(),
			"result": res,
		})
		return
	}
	res.Available = true
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
	if !h.requirePrivileged(w, r, scopeProvidersWrite) {
		return
	}
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
	if !h.requirePrivileged(w, r, scopeProvidersWrite) {
		return
	}
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
