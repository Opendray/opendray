package gateway

// T12 — Consent revoke endpoints + hot-revoke broadcast hook.
//
// Three routes (registered in the protected chi group by server.go):
//
//	GET    /api/plugins/{name}/consents         → pluginsConsentsGet
//	DELETE /api/plugins/{name}/consents/{cap}   → pluginsConsentsRevokeCap
//	DELETE /api/plugins/{name}/consents         → pluginsConsentsRevokeAll
//
// Design notes
// ─────────────
//  1. Revoke-cap semantics (T12 §spec): load the current perms JSON, zero the
//     field that matches the cap (see zeroCapability), marshal back, persist
//     via UpdateConsentPerms, then call bridgeMgr.InvalidateConsent(name, cap)
//     SYNCHRONOUSLY so in-flight WS subs terminate before the HTTP response
//     is flushed. The 200 ms hot-revoke SLO (tested in M2-PLAN §T12) hinges
//     on this synchronous path — tests assert the call count before reading
//     the response body.
//
//  2. Revoke-all: DeleteConsent is idempotent per M1. We still call
//     InvalidateConsent once per capability key that was previously granted
//     so each active WS sub receives its terminal EPERM envelope. For a
//     missing row we return 200 with no broadcasts (nothing to tear down).
//
//  3. Store + bridge access uses the consentStore / consentInvalidator
//     interfaces so tests can supply fakes. Production code wires
//     s.hub.DB() and s.bridgeMgr.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray/kernel/store"
	"github.com/opendray/opendray/plugin"
)

// ─── Interfaces (test seams) ─────────────────────────────────────────────────

// consentStore is the narrow subset of kernel/store.DB the consent handlers
// need. Keeping this local lets tests inject an in-memory fake without booting
// embedded Postgres.
type consentStore interface {
	GetConsent(ctx context.Context, name string) (store.PluginConsent, error)
	UpdateConsentPerms(ctx context.Context, name string, perms json.RawMessage) error
	DeleteConsent(ctx context.Context, name string) error
}

// consentInvalidator is the minimum bridge surface the consent handlers need.
// *bridge.Manager satisfies this directly.
type consentInvalidator interface {
	InvalidateConsent(plugin string, cap string)
}

// ─── Capability allowlist ───────────────────────────────────────────────────

// allowedCaps enumerates every capability key recognised by the revoke-cap
// endpoint. Keys match the PermissionsV1 field names (lowercased) and the
// bridge namespaces they gate. Any cap outside this set is rejected with 400
// EINVAL — this prevents typos from silently creating orphaned revocations.
var allowedCaps = map[string]struct{}{
	"fs":        {},
	"exec":      {},
	"http":      {},
	"session":   {},
	"storage":   {},
	"secret":    {},
	"clipboard": {},
	"telegram":  {},
	"git":       {},
	"llm":       {},
	"events":    {},
}

// zeroCapability blanks the PermissionsV1 field associated with cap. Returns
// (false) when cap is outside the allowlist so the caller can surface EINVAL
// without having to re-check the allowlist map. The p pointer is mutated in
// place — callers that need to preserve the original should copy first.
//
// Field → zero-value mapping is enumerated explicitly here because
// PermissionsV1's polymorphic fields (Fs/Exec/HTTP) carry json.RawMessage,
// while others carry typed bool/string/[]string. A generic reflection pass
// would work too but this form is trivial to audit in a security context.
func zeroCapability(p *plugin.PermissionsV1, cap string) bool {
	if p == nil {
		return false
	}
	switch cap {
	case "fs":
		p.Fs = nil
	case "exec":
		p.Exec = nil
	case "http":
		p.HTTP = nil
	case "session":
		p.Session = ""
	case "storage":
		p.Storage = false
	case "secret":
		p.Secret = false
	case "clipboard":
		p.Clipboard = ""
	case "telegram":
		p.Telegram = false
	case "git":
		p.Git = ""
	case "llm":
		p.LLM = false
	case "events":
		p.Events = nil
	default:
		return false
	}
	return true
}

// permsGrantedCaps inspects a PermissionsV1 and returns the list of cap keys
// that were granted (non-zero). Used by revoke-all to broadcast one
// InvalidateConsent per live capability so every WS sub gets its EPERM.
func permsGrantedCaps(p plugin.PermissionsV1) []string {
	out := make([]string, 0, len(allowedCaps))
	if len(p.Fs) > 0 {
		out = append(out, "fs")
	}
	if len(p.Exec) > 0 {
		out = append(out, "exec")
	}
	if len(p.HTTP) > 0 {
		out = append(out, "http")
	}
	if p.Session != "" {
		out = append(out, "session")
	}
	if p.Storage {
		out = append(out, "storage")
	}
	if p.Secret {
		out = append(out, "secret")
	}
	if p.Clipboard != "" {
		out = append(out, "clipboard")
	}
	if p.Telegram {
		out = append(out, "telegram")
	}
	if p.Git != "" {
		out = append(out, "git")
	}
	if p.LLM {
		out = append(out, "llm")
	}
	if len(p.Events) > 0 {
		out = append(out, "events")
	}
	return out
}

// ─── Server resolvers ───────────────────────────────────────────────────────

// consentDB returns the consentStore the handler should use — test override
// first, production fallback via s.hub.DB().
func (s *Server) consentDB() consentStore {
	if s.consentStoreOverride != nil {
		return s.consentStoreOverride
	}
	if s.hub == nil {
		return nil
	}
	return s.hub.DB()
}

// consentBridge returns the consentInvalidator the handler should use.
func (s *Server) consentBridge() consentInvalidator {
	if s.consentBridgeOverride != nil {
		return s.consentBridgeOverride
	}
	if s.bridgeMgr == nil {
		return nil
	}
	return s.bridgeMgr
}

// consentManifestPerms returns the install-time PermissionsV1 declared by the
// plugin's manifest, or nil when the plugin isn't registered (or declared no
// perms at all). The GET handler surfaces this so the UI can offer a
// re-grant toggle on a previously-revoked cap without a reinstall.
//
// Lookup precedence matches bridgePluginExists: test override first, then the
// real *plugin.Runtime. A nil runtime (test-only) yields (nil, false).
func (s *Server) consentManifestPerms(name string) (*plugin.PermissionsV1, bool) {
	if s.bridgePluginsOverride != nil {
		p, ok := s.bridgePluginsOverride(name)
		if !ok {
			return nil, false
		}
		return p.Permissions, true
	}
	if s.plugins == nil {
		return nil, false
	}
	p, ok := s.plugins.Get(name)
	if !ok {
		return nil, false
	}
	return p.Permissions, true
}

// ─── Handlers ────────────────────────────────────────────────────────────────

// pluginsConsentsGet handles GET /api/plugins/{name}/consents.
//
// Returns the current perms JSON plus timestamps. 404 ENOCONSENT when no row
// exists.
func (s *Server) pluginsConsentsGet(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	db := s.consentDB()
	if db == nil {
		writeJSONError(w, http.StatusInternalServerError, "ECONFIG",
			"consent store not configured")
		return
	}

	row, err := db.GetConsent(r.Context(), name)
	if err != nil {
		if errors.Is(err, store.ErrConsentNotFound) {
			writeJSONError(w, http.StatusNotFound, "ENOCONSENT",
				"no consent row for plugin "+name)
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "ERESERVE", err.Error())
		return
	}

	// Surface the manifest-declared permissions so the UI can offer a
	// re-grant toggle on revoked caps without a reinstall. Absent when
	// the plugin isn't registered (e.g. uninstalled while the consent
	// row still lingers) or the manifest declared no permissions block.
	var manifestPerms *plugin.PermissionsV1
	if mp, ok := s.consentManifestPerms(name); ok {
		manifestPerms = mp
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"perms":         row.PermsJSON,
		"manifestPerms": manifestPerms,
		"grantedAt":     row.GrantedAt,
		"updatedAt":     row.UpdatedAt,
	})
}

// pluginsConsentsRevokeCap handles DELETE /api/plugins/{name}/consents/{cap}.
//
// Flow — MUST stay synchronous through InvalidateConsent for the 200 ms SLO.
//  1. Validate cap against allowedCaps → 400 EINVAL.
//  2. Load current perms via GetConsent → 404 ENOCONSENT on missing row.
//  3. Unmarshal into PermissionsV1 → 500 ERESERVE on corrupt JSON (defence).
//  4. Zero the field matching cap, marshal back.
//  5. Persist via UpdateConsentPerms → 500 ERESERVE on failure.
//  6. bridgeMgr.InvalidateConsent(name, cap) — synchronous.
//  7. 200 {"revoked": true, "cap": "<cap>"}.
func (s *Server) pluginsConsentsRevokeCap(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	cap := chi.URLParam(r, "cap")

	if _, ok := allowedCaps[cap]; !ok {
		writeJSONError(w, http.StatusBadRequest, "EINVAL",
			"unknown capability: "+cap)
		return
	}

	db := s.consentDB()
	if db == nil {
		writeJSONError(w, http.StatusInternalServerError, "ECONFIG",
			"consent store not configured")
		return
	}

	row, err := db.GetConsent(r.Context(), name)
	if err != nil {
		if errors.Is(err, store.ErrConsentNotFound) {
			writeJSONError(w, http.StatusNotFound, "ENOCONSENT",
				"no consent row for plugin "+name)
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "ERESERVE", err.Error())
		return
	}

	var perms plugin.PermissionsV1
	if len(row.PermsJSON) > 0 {
		if err := json.Unmarshal(row.PermsJSON, &perms); err != nil {
			// Defensive: a corrupted row should never reach production,
			// but if one does we refuse to make a revocation decision
			// from unparseable state.
			writeJSONError(w, http.StatusInternalServerError, "ERESERVE",
				"corrupt perms JSON: "+err.Error())
			return
		}
	}

	if !zeroCapability(&perms, cap) {
		// Already validated above, but keep this defensive — the allowedCaps
		// map and zeroCapability switch must stay in sync.
		writeJSONError(w, http.StatusBadRequest, "EINVAL",
			"unknown capability: "+cap)
		return
	}

	newPerms, err := json.Marshal(perms)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "ERESERVE",
			"marshal perms: "+err.Error())
		return
	}

	if err := db.UpdateConsentPerms(r.Context(), name, json.RawMessage(newPerms)); err != nil {
		if errors.Is(err, store.ErrConsentNotFound) {
			// Race: row was removed between Get and Update.
			writeJSONError(w, http.StatusNotFound, "ENOCONSENT",
				"no consent row for plugin "+name)
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "ERESERVE", err.Error())
		return
	}

	// Fire hot-revoke SYNCHRONOUSLY. Manager.InvalidateConsent is
	// non-blocking (T6 contract: async WS writes, synchronous done-chan
	// closes), so this returns within a few milliseconds even with many
	// live subs. Required for the 200 ms SLO.
	if br := s.consentBridge(); br != nil {
		br.InvalidateConsent(name, cap)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"revoked": true,
		"cap":     cap,
	})
}

// pluginsConsentsRevokeAll handles DELETE /api/plugins/{name}/consents.
//
// Idempotent. Removes the consent row entirely and fires InvalidateConsent
// once per capability that was previously granted so every live WS sub gets
// its EPERM terminal envelope. A missing row returns 200 with no broadcasts.
func (s *Server) pluginsConsentsRevokeAll(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	db := s.consentDB()
	if db == nil {
		writeJSONError(w, http.StatusInternalServerError, "ECONFIG",
			"consent store not configured")
		return
	}

	// Snapshot the currently-granted caps BEFORE we delete the row, so we
	// know which capabilities need InvalidateConsent broadcasts. A missing
	// row yields an empty list and the endpoint degrades to a no-op.
	var grantedCaps []string
	if row, err := db.GetConsent(r.Context(), name); err == nil {
		var perms plugin.PermissionsV1
		if len(row.PermsJSON) > 0 {
			// A corrupt row is tolerated here: we'd rather delete it and
			// broadcast nothing than fail a revoke-all on stale data.
			_ = json.Unmarshal(row.PermsJSON, &perms)
		}
		grantedCaps = permsGrantedCaps(perms)
	} else if !errors.Is(err, store.ErrConsentNotFound) {
		writeJSONError(w, http.StatusInternalServerError, "ERESERVE", err.Error())
		return
	}

	if err := db.DeleteConsent(r.Context(), name); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "ERESERVE", err.Error())
		return
	}

	// Broadcast AFTER delete so any race between "delete succeeded" and
	// "invalidate fired" resolves in favour of deny-by-default: the row is
	// gone → Gate.Check will fail, and every live sub still gets its EPERM.
	if br := s.consentBridge(); br != nil {
		for _, cap := range grantedCaps {
			br.InvalidateConsent(name, cap)
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"revoked": "all",
		"name":    name,
	})
}
