// SetupHandlers wires the three always-mounted endpoints that let
// the admin UI / mobile app drive the backup-feature lifecycle:
//
//	GET  /backup-status         feature state + key file location
//	POST /backup-setup          generate or paste a passphrase, write the key file
//	POST /backup-setup/disable  remove the key file
//
// These three routes are mounted under the admin auth group
// regardless of whether the backup data handlers (the ones in
// Handlers.Mount) are wired up — that's how a user without backups
// enabled can use the UI to turn the feature on. The status route
// in particular is no longer a 404-vs-200 channel; it always
// responds 200 with a JSON describing exactly which side of "off"
// or "on" the server is on.
//
// The actual flip from "off" to "on" requires an opendray restart:
// the cipher needs the passphrase at NewService() time and the
// service is created during boot. The setup endpoint writes the
// key file and returns requires_restart=true so the UI can show a
// "please restart" screen rather than pretend the feature is
// instantly live.

package backup

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// SetupHandlers — see file header. Always-mountable.
//
// Carries a concrete *Service rather than an interface so callers
// can pass an untyped nil (e.g. `NewSetupHandlers(nil, ...)`)
// without tripping the classic interface-holding-typed-nil gotcha
// — `h.svc != nil` checks below mean exactly what they look like.
type SetupHandlers struct {
	// svc is non-nil iff the feature is currently active in this
	// process. When nil, only Setup + status (with enabled=false)
	// work; the live data routes (list/run/etc.) aren't wired.
	svc *Service
	// bootSource records how (or whether) the passphrase was
	// loaded at app start. The UI uses this to choose between
	// "Setup" (no source) and "Already configured via env / file"
	// flows — the latter has limited disable-via-UI capability.
	bootSource KeySource
}

// NewSetupHandlers constructs SetupHandlers. Pass nil svc when the
// backup feature failed to load (no passphrase). Pass bootSource
// from the LoadPassphrase call in app startup so the UI can
// distinguish env-var vs file-loaded deployments.
func NewSetupHandlers(svc *Service, bootSource KeySource) *SetupHandlers {
	return &SetupHandlers{svc: svc, bootSource: bootSource}
}

// Mount the three always-on endpoints under r. The caller is
// responsible for putting this inside the admin-auth group — none
// of these routes are safe for public access.
func (h *SetupHandlers) Mount(r chi.Router) {
	r.Get("/backup-status", h.status)
	r.Post("/backup-setup", h.setup)
	r.Post("/backup-setup/disable", h.disable)
}

// status replies with a JSON map describing the feature's current
// state. Response shape (all keys always present):
//
//	enabled                bool     — feature is actively running in this process
//	configured             bool     — a passphrase file exists on disk OR env var is set
//	configured_via         string   — "env" | "file" | ""
//	can_disable_via_ui     bool     — false when configured_via == "env"
//	requires_restart       bool     — true when configured but !enabled (UI just wrote a file)
//	key_file_path          string   — canonical default location, always populated for UX
//
// When enabled is true the following are also populated (parity
// with the original /backup-status response so existing clients
// keep working):
//
//	ok                  bool
//	key_fingerprint     string
//	pg_dump_version     string
//	pg_restore_version  string
//	pg_dump_error       string   — only on !ok
func (h *SetupHandlers) status(w http.ResponseWriter, r *http.Request) {
	keyPath, _ := DefaultKeyFilePath()
	// Re-check disk state on every call rather than caching boot
	// state: if the operator wrote the file via /backup-setup,
	// hits status before restart, the response should reflect
	// "configured but not yet enabled" — i.e. requires_restart.
	currentLoad, _ := LoadPassphrase()

	resp := map[string]any{
		"enabled":            h.svc != nil,
		"configured":         currentLoad.Passphrase != "",
		"configured_via":     string(currentLoad.Source),
		"can_disable_via_ui": currentLoad.Source == KeySourceFile,
		"requires_restart":   h.svc == nil && currentLoad.Passphrase != "",
		"key_file_path":      keyPath,
	}

	if h.svc != nil {
		pgVer, err := h.svc.PGVersion(r.Context())
		resp["ok"] = err == nil
		resp["key_fingerprint"] = h.svc.CipherFingerprint()
		resp["pg_dump_version"] = pgVer
		resp["pg_restore_version"] = h.svc.PgRestoreVersion(r.Context())
		if err != nil {
			resp["pg_dump_error"] = err.Error()
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// setupRequest is the body for POST /backup-setup.
//
//	{"mode": "generate"}              — server picks a random 32-byte key,
//	                                    base64-encodes it, returns it once.
//	{"mode": "paste", "passphrase":"..."} — operator-supplied; we just store it.
type setupRequest struct {
	Mode       string `json:"mode"`
	Passphrase string `json:"passphrase"`
	// Overwrite forces replacing an existing key file. Reserved
	// for a future rotation flow; the default (false) refuses to
	// overwrite, which protects against the "I have encrypted
	// blobs already, don't lose my key" footgun.
	Overwrite bool `json:"overwrite"`
}

// minPassphraseLen — sized to be marginally inconvenient for
// dictionary attack while still being possible to type or paste
// from a password manager. Generated keys are longer (44 chars
// after base64-encoding 32 random bytes); only the paste-your-own
// path needs validation.
const minPassphraseLen = 20

// setup writes the key file and tells the caller to restart.
//
// Refuses when an env-var passphrase is already active — that path
// is what the operator opted into; silently overwriting it with a
// file-based one would mean the env var still wins next boot, and
// the file would be effectively dead. The UI should see
// can_disable_via_ui=false and gate the Setup button accordingly,
// but server-side check is the load-bearing guard.
func (h *SetupHandlers) setup(w http.ResponseWriter, r *http.Request) {
	if h.bootSource == KeySourceEnv {
		writeError(w, http.StatusConflict,
			errors.New("backup is already configured via OPENDRAY_BACKUP_KEY env var; unset it before configuring via UI"))
		return
	}

	var req setupRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode body: %w", err))
		return
	}

	var passphrase string
	switch strings.ToLower(strings.TrimSpace(req.Mode)) {
	case "generate":
		var b [32]byte
		if _, err := rand.Read(b[:]); err != nil {
			writeError(w, http.StatusInternalServerError,
				fmt.Errorf("read random bytes: %w", err))
			return
		}
		// URLEncoding gives base64 without `/` or `+`, safer for
		// copy-paste into shell or env files (`/` is harmless,
		// `+` confuses some prompts). Length is 44 chars including
		// the trailing `=` padding.
		passphrase = base64.URLEncoding.EncodeToString(b[:])
	case "paste":
		p := strings.TrimSpace(req.Passphrase)
		if len(p) < minPassphraseLen {
			writeError(w, http.StatusBadRequest,
				fmt.Errorf("passphrase must be at least %d characters", minPassphraseLen))
			return
		}
		passphrase = p
	default:
		writeError(w, http.StatusBadRequest,
			errors.New(`mode must be "generate" or "paste"`))
		return
	}

	path, err := WriteKeyFile(passphrase, req.Overwrite)
	if err != nil {
		// "already exists" is a 409 (Conflict), everything else
		// is a 500. We don't want to leak the passphrase back
		// to the operator from a generate flow if writing failed
		// — they'd have a key with no way to retrieve it.
		if strings.Contains(err.Error(), "already exists") {
			writeError(w, http.StatusConflict, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	resp := map[string]any{
		"ok":               true,
		"key_file_path":    path,
		"requires_restart": true,
	}
	// Only return the passphrase on `generate` — pasted ones are
	// already known to the caller. Returning it twice would give
	// the impression we're stashing it somewhere recoverable.
	if strings.EqualFold(req.Mode, "generate") {
		resp["passphrase"] = passphrase
	}
	writeJSON(w, http.StatusCreated, resp)
}

// disable removes the default key file. Does not touch env-var
// passphrases. Refuses when bootSource is env — the file isn't
// what's keeping the feature on, so removing it would be a no-op
// and we'd rather surface that clearly than pretend success.
//
// Importantly: existing encrypted backups remain on disk and are
// unreadable without the passphrase. The UI must warn about this
// before sending the request. Server-side we don't refuse — the
// operator might intentionally want to "lose" backups (e.g. test
// data) — but the warning is critical for production.
func (h *SetupHandlers) disable(w http.ResponseWriter, r *http.Request) {
	if h.bootSource == KeySourceEnv {
		writeError(w, http.StatusConflict,
			errors.New("backup is configured via OPENDRAY_BACKUP_KEY env var; UI cannot remove env vars — unset OPENDRAY_BACKUP_KEY in the parent process and restart"))
		return
	}
	if err := RemoveKeyFile(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":               true,
		"requires_restart": h.svc != nil, // only need restart if the feature is currently on
	})
}

// decodeJSON is a tiny helper to keep the request handlers tidy.
// We don't bother with a maxBytes limit — these handlers only
// accept tiny JSON bodies (mode + passphrase) and chi's default
// timeout middleware caps body read size at the gateway layer.
func decodeJSON(r io.Reader, v any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
