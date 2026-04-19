package gateway

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray/kernel/store"
	"github.com/opendray/opendray/plugin"
)

// PATCH /api/plugins/{name}/consents — M3 T20 granular consent merge.
//
// Body is a partial PermissionsV1 JSON object. Every field is optional.
// The handler:
//   1. Reads the raw body + unmarshals into map[string]json.RawMessage
//      so it can tell which keys the client actually sent (vs. zero-
//      valued fields from a full PermissionsV1 decode).
//   2. Loads the current perms_json from plugin_consents.
//   3. Replaces each touched cap's stored value with the body's value.
//      Absent caps are left untouched.
//   4. Persists the merged perms via UpdateConsentPerms.
//   5. Fires bridgeMgr.InvalidateConsent for every touched cap so any
//      active bridge WS subscription terminates with EPERM within the
//      existing 200 ms SLO (matches M2 DELETE behaviour).
//   6. Returns the new perms as JSON.
//
// Design: shallow field-level merge. To change one glob inside
// fs.read the client sends the full desired list for fs. Simpler
// contract, same shape the granular Flutter UI builds (T21).

// pluginsConsentsPatch handles PATCH /api/plugins/{name}/consents.
func (s *Server) pluginsConsentsPatch(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		respondConsentError(w, http.StatusBadRequest, "EINVAL", "plugin name required")
		return
	}

	db := s.consentDB()
	if db == nil {
		respondConsentError(w, http.StatusServiceUnavailable, "ECONSENT", "consent store not wired")
		return
	}

	// Read raw body once. bodySizeLimiter middleware already capped
	// this at 1 MiB, so unbounded reads are not a concern here.
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		respondConsentError(w, http.StatusBadRequest, "EINVAL",
			fmt.Sprintf("read body: %v", err))
		return
	}
	if len(raw) == 0 {
		respondConsentError(w, http.StatusBadRequest, "EINVAL", "empty body")
		return
	}

	// Decode twice: once as a map to discover touched keys, once as
	// PermissionsV1 for typed values.
	var touched map[string]json.RawMessage
	if err := json.Unmarshal(raw, &touched); err != nil {
		respondConsentError(w, http.StatusBadRequest, "EINVAL",
			fmt.Sprintf("bad request body: %v", err))
		return
	}
	var patch plugin.PermissionsV1
	if err := json.Unmarshal(raw, &patch); err != nil {
		respondConsentError(w, http.StatusBadRequest, "EINVAL",
			fmt.Sprintf("bad PermissionsV1 shape: %v", err))
		return
	}

	// Reject unknown keys so typos don't silently no-op — matches
	// the existing DELETE endpoint's allowedCaps check.
	for k := range touched {
		if _, ok := allowedCaps[k]; !ok {
			respondConsentError(w, http.StatusBadRequest, "EINVAL",
				fmt.Sprintf("unknown capability %q", k))
			return
		}
	}

	// Load the existing consent row.
	current, err := db.GetConsent(r.Context(), name)
	if errors.Is(err, store.ErrConsentNotFound) {
		respondConsentError(w, http.StatusNotFound, "ENOENT",
			fmt.Sprintf("no consent row for plugin %q", name))
		return
	}
	if err != nil {
		s.logger.Error("consent: load", "plugin", name, "err", err)
		respondConsentError(w, http.StatusInternalServerError, "EINTERNAL",
			"failed to load consent")
		return
	}

	// Decode stored perms.
	var next plugin.PermissionsV1
	if len(current.PermsJSON) > 0 {
		if uerr := json.Unmarshal(current.PermsJSON, &next); uerr != nil {
			s.logger.Error("consent: parse stored", "plugin", name, "err", uerr)
			respondConsentError(w, http.StatusInternalServerError, "ERESERVE",
				"stored consent is corrupt")
			return
		}
	}

	// Apply each touched field.
	for field := range touched {
		applyPermsField(&next, &patch, field)
	}

	// Persist.
	nextJSON, err := json.Marshal(next)
	if err != nil {
		s.logger.Error("consent: marshal", "plugin", name, "err", err)
		respondConsentError(w, http.StatusInternalServerError, "EINTERNAL",
			"failed to marshal perms")
		return
	}
	if err := db.UpdateConsentPerms(r.Context(), name, nextJSON); err != nil {
		if errors.Is(err, store.ErrConsentNotFound) {
			respondConsentError(w, http.StatusNotFound, "ENOENT",
				"consent row disappeared mid-request")
			return
		}
		s.logger.Error("consent: update", "plugin", name, "err", err)
		respondConsentError(w, http.StatusInternalServerError, "EINTERNAL",
			"failed to persist perms")
		return
	}

	// Invalidate every touched cap. Always fire — even when the new
	// grant still includes the cap, the old subscription's cached
	// grant may match differently now (e.g. a removed glob). Simpler
	// to reconnect than to diff.
	if bridge := s.consentBridge(); bridge != nil {
		for field := range touched {
			bridge.InvalidateConsent(name, field)
		}
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"name":  name,
		"perms": next,
	})
}

// applyPermsField copies one field from src into dst. Kept as a
// switch because PermissionsV1 has a fixed, security-reviewed set of
// fields — reflection would add complexity without saving lines.
func applyPermsField(dst, src *plugin.PermissionsV1, field string) {
	switch field {
	case "fs":
		dst.Fs = src.Fs
	case "exec":
		dst.Exec = src.Exec
	case "http":
		dst.HTTP = src.HTTP
	case "session":
		dst.Session = src.Session
	case "storage":
		dst.Storage = src.Storage
	case "secret":
		dst.Secret = src.Secret
	case "clipboard":
		dst.Clipboard = src.Clipboard
	case "telegram":
		dst.Telegram = src.Telegram
	case "git":
		dst.Git = src.Git
	case "llm":
		dst.LLM = src.LLM
	case "events":
		dst.Events = src.Events
	}
}

// respondConsentError is a local wrapper that matches the error shape
// the other plugins_consents.go handlers use: {"code","error"}.
func respondConsentError(w http.ResponseWriter, status int, code, msg string) {
	respondJSON(w, status, map[string]string{"code": code, "error": msg})
}
