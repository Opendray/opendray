package gateway

// Built-in plugins HTTP surface — drives Settings → Built-in Plugins.
//
//	GET  /api/plugins/builtins                 → list bundled manifests + state
//	POST /api/plugins/builtins/{name}/restore  → un-tombstone + re-seed
//
// Lives separate from the install handlers because the lifecycle is
// different: built-ins never went through download/verify/consent, so
// "restore" is really "undo Uninstall" — clear the tombstone and call
// the runtime seeder that boot-time LoadAll uses. No consent dialog,
// no permissions round-trip (the manifest's own permissions
// declaration is re-seeded as-is and was already agreed to the first
// time the user ran OpenDray).

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray/plugin"
)

// builtinsListResponse is the wire shape for GET /api/plugins/builtins.
// Each entry carries the full Provider so the UI can render the card
// using the same fields it already knows from /api/providers, plus a
// State so it knows whether to show Enable / Restore / nothing.
type builtinsListResponse struct {
	Builtins []plugin.BuiltinInfo `json:"builtins"`
}

// builtinRestoreResponse echoes the reinstated Provider and its new
// state so the client can update its local cache without re-fetching
// the full list. State is always "installed" on success.
type builtinRestoreResponse struct {
	Provider plugin.Provider     `json:"provider"`
	State    plugin.BuiltinState `json:"state"`
}

// pluginsBuiltinsList handles GET /api/plugins/builtins.
//
// Returns every manifest bundled in the binary (plugins/builtin/*) with
// its current state — installed / disabled / uninstalled. Doesn't
// filter: the Settings page wants the full list so users can see what
// ships with OpenDray even when everything is already installed.
func (s *Server) pluginsBuiltinsList(w http.ResponseWriter, r *http.Request) {
	items := s.plugins.ListBuiltins()
	if items == nil {
		items = []plugin.BuiltinInfo{}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(builtinsListResponse{Builtins: items})
}

// pluginsBuiltinRestore handles POST /api/plugins/builtins/{name}/restore.
//
// Removes any tombstone + re-seeds the manifest from embed.FS. Fires
// the workbench contributionsChanged event on success so the Flutter
// shell refreshes its cached /api/workbench/contributions.
//
// Error mapping:
//
//	ErrNotBuiltin        → 404 ENOTBUILTIN
//	ErrAlreadyInstalled  → 409 EALREADYINSTALLED
//	anything else        → 500 ERESTOREFAIL
func (s *Server) pluginsBuiltinRestore(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	prov, err := s.plugins.RestoreBuiltin(r.Context(), name)
	if err != nil {
		switch {
		case errors.Is(err, plugin.ErrNotBuiltin):
			writeJSONError(w, http.StatusNotFound, "ENOTBUILTIN", err.Error())
		case errors.Is(err, plugin.ErrAlreadyInstalled):
			writeJSONError(w, http.StatusConflict, "EALREADYINSTALLED", err.Error())
		default:
			writeJSONError(w, http.StatusInternalServerError, "ERESTOREFAIL", err.Error())
		}
		return
	}

	// Notify SSE subscribers so Plugins page + workbench slots refresh
	// without the user having to pull-to-reload. Fire after the DB seed
	// so clients that race in see the new row.
	if s.workbenchBus != nil {
		s.workbenchBus.PublishContributionsChanged()
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(builtinRestoreResponse{
		Provider: prov,
		State:    plugin.BuiltinInstalled,
	})
}
