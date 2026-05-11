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
	"context"
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
// Carries a *LiveBackup whose Service() pointer flips at runtime
// in response to /backup-setup and /backup-setup/disable. The
// status endpoint reads live state via that pointer; setup/disable
// arm/disarm it directly.
type SetupHandlers struct {
	live *LiveBackup
	// bootSource records how (or whether) the passphrase was
	// loaded at app start. The UI uses this to choose between
	// "Setup" (no source) and "Already configured via env / file"
	// flows — the env case has limited disable-via-UI capability
	// (we can't unset an env var in our parent process).
	bootSource KeySource
}

// NewSetupHandlers constructs SetupHandlers. live must be non-nil
// (it's the single shared lifecycle handle the rest of the app
// uses too). bootSource comes from the LoadPassphrase call in app
// startup so the UI can distinguish env-var vs file-loaded
// deployments.
func NewSetupHandlers(live *LiveBackup, bootSource KeySource) *SetupHandlers {
	return &SetupHandlers{live: live, bootSource: bootSource}
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
	// Re-check disk state on every call so we don't lie about
	// "configured" right after a /backup-setup/disable.
	currentLoad, _ := LoadPassphrase()

	svc := h.live.Service()
	resp := map[string]any{
		"enabled":            svc != nil,
		"configured":         currentLoad.Passphrase != "",
		"configured_via":     string(currentLoad.Source),
		"can_disable_via_ui": currentLoad.Source == KeySourceFile,
		// requires_restart now only fires for the env-var edge
		// case: an operator set OPENDRAY_BACKUP_KEY but hasn't
		// restarted the process yet. The file path is handled
		// in-process by Arm/Disarm so it doesn't trigger this.
		"requires_restart": svc == nil && currentLoad.Source == KeySourceEnv,
		"key_file_path":    keyPath,
	}

	if svc != nil {
		pgVer, err := svc.PGVersion(r.Context())
		resp["ok"] = err == nil
		resp["key_fingerprint"] = svc.CipherFingerprint()
		resp["pg_dump_version"] = pgVer
		resp["pg_restore_version"] = svc.PgRestoreVersion(r.Context())
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

	// Hot-arm: build a fresh Service + scheduler in-process and
	// activate the data routes. r.Context() is request-scoped, so
	// we pass context.Background() for the scheduler — it needs
	// to outlive this request. (The scheduler ends when the
	// LiveBackup gets Disarmed or the process exits.)
	if err := h.live.Arm(context.Background(), passphrase); err != nil {
		// Roll back the keyfile so the next /backup-status report
		// doesn't show "configured" while the feature is actually
		// in a broken half-state. The operator should see a clean
		// error and try again.
		_ = RemoveKeyFile()
		writeError(w, http.StatusInternalServerError,
			fmt.Errorf("arm backup feature: %w", err))
		return
	}

	resp := map[string]any{
		"ok":               true,
		"key_file_path":    path,
		"enabled":          true,
		"requires_restart": false,
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
	// Hot-disarm: stops the scheduler goroutine and flips the
	// Service pointer to nil. Subsequent requests to /backups etc.
	// get 503 via requireArmed middleware.
	h.live.Disarm()
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":               true,
		"enabled":          false,
		"requires_restart": false,
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
