package gateway

// T7 — HTTP install endpoints.
//
// Four routes registered in the protected chi group (JWT middleware already
// applied by server.go):
//
//	POST   /api/plugins/install         → pluginsInstall   (202)
//	POST   /api/plugins/install/confirm → pluginsInstallConfirm (200)
//	DELETE /api/plugins/{name}          → pluginsUninstall  (200)
//	GET    /api/plugins/{name}/audit    → pluginsAudit      (200)
//
// Request JSON is decoded with a hard length limit already applied by the
// bodySizeLimiter middleware in server.go (1 MB). Handlers use r.Context()
// throughout — no context.Background().
//
// The existing DELETE /api/providers/{name} route stays for legacy compat;
// this new DELETE /api/plugins/{name} additionally calls Installer.Uninstall
// for the full teardown (DB + FS + runtime).

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray/plugin"
	"github.com/opendray/opendray/plugin/install"
	"github.com/opendray/opendray/plugin/market"
	"github.com/opendray/opendray/plugin/market/signing"
)

// ─── Request / Response shapes ───────────────────────────────────────────────

type installRequest struct {
	Src string `json:"src"`
}

type installResponse struct {
	Token        string               `json:"token"`
	Name         string               `json:"name"`
	Version      string               `json:"version"`
	Perms        plugin.PermissionsV1 `json:"perms"`
	ExpiresAt    time.Time            `json:"expiresAt"`
	ManifestHash string               `json:"manifestHash"`
}

type confirmRequest struct {
	Token string `json:"token"`
}

type confirmResponse struct {
	Installed bool   `json:"installed"`
	Name      string `json:"name"`
}

type uninstallResponse struct {
	Status string `json:"status"` // "uninstalled"
	Name   string `json:"name"`
}

type auditEntryDTO struct {
	Ts         time.Time `json:"ts"`
	Ns         string    `json:"ns"`
	Method     string    `json:"method"`
	Caps       []string  `json:"caps"`
	Result     string    `json:"result"`
	DurationMs int       `json:"durationMs"`
	ArgsHash   string    `json:"argsHash"`
	Message    string    `json:"message,omitempty"`
}

// ─── JSON helpers (mirroring respondJSON / respondError in server.go) ────────
//
// The gateway package already defines respondJSON and respondError in server.go.
// We add writeJSONError as a structured-code variant used only by this file so
// callers can set both a machine-readable "code" and a human "msg" field
// without duplicating the pattern across every handler.

// writeJSONError writes {"code":"<code>","msg":"<msg>"} with the given status.
func writeJSONError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
		"code": code,
		"msg":  msg,
	})
}

// ─── Handlers ────────────────────────────────────────────────────────────────

// pluginsInstall handles POST /api/plugins/install.
//
// Flow:
//  1. Decode {"src":"..."}.
//  2. Missing src → 400 EINVAL.
//  3. Local scheme without OPENDRAY_ALLOW_LOCAL_PLUGINS=1 → 403 EFORBIDDEN.
//  4. ParseSource → 400 EBADSRC on unknown scheme.
//  5. Installer.Stage → 400 EBADMANIFEST | 501 ENOTIMPL | 500 ESTAGEFAIL.
//  6. 202 Accepted with installResponse.
func (s *Server) pluginsInstall(w http.ResponseWriter, r *http.Request) {
	var req installRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "EINVAL", "invalid JSON body")
		return
	}

	if req.Src == "" {
		writeJSONError(w, http.StatusBadRequest, "EINVAL", "src is required")
		return
	}

	// T25: Gate local: + bare absolute-path sources on the config-backed
	// AllowLocal flag (populated from OPENDRAY_ALLOW_LOCAL_PLUGINS by the
	// config loader). The env var is no longer read directly here — the
	// config layer owns that decision and propagates it via Installer.AllowLocal.
	// This prevents production deployments from being used to install
	// arbitrary local filesystem paths.
	if isLocalScheme(req.Src) && (s.installer == nil || !s.installer.AllowLocal) {
		writeJSONError(w, http.StatusForbidden, "EFORBIDDEN",
			"local plugin installs disabled; set OPENDRAY_ALLOW_LOCAL_PLUGINS=1")
		return
	}

	// Marketplace refs are resolved HERE — not by ParseSource — so the
	// gateway can translate them into a TrustedSource that bypasses the
	// AllowLocal gate. Client-supplied local: / absolute-path sources
	// still go through the LocalSource path and keep getting gated.
	var src install.Source
	if strings.HasPrefix(req.Src, "marketplace://") {
		if s.marketplace == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "EBADSRC",
				"marketplace catalog not configured on this server")
			return
		}
		ref, rerr := market.ParseRef(req.Src)
		if rerr != nil {
			writeJSONError(w, http.StatusBadRequest, "EBADSRC", rerr.Error())
			return
		}
		entry, rerr := s.marketplace.Resolve(r.Context(), ref)
		if errors.Is(rerr, market.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "ENOENT",
				"marketplace entry not found: "+req.Src)
			return
		}
		if rerr != nil {
			writeJSONError(w, http.StatusBadRequest, "EBADSRC", rerr.Error())
			return
		}

		// Signature policy gate. Fetch the publisher record (for
		// registered keys + trust level) and enforce. Community
		// plugins without a signature still install; official /
		// verified must verify. See signing.EnforcePolicy docstring.
		publisher, perr := s.marketplace.FetchPublisher(r.Context(), entry.Publisher)
		if perr != nil && !errors.Is(perr, market.ErrNotFound) {
			writeJSONError(w, http.StatusInternalServerError, "EMARKET",
				"fetch publisher: "+perr.Error())
			return
		}
		// Empty PublisherRecord (ErrNotFound) falls through to the
		// default-deny branch of EnforcePolicy; attackers can't
		// install a "community" plugin by deleting their record.
		if err := signing.EnforcePolicy(entry, publisher, time.Now()); err != nil {
			if errors.Is(err, signing.ErrSignatureRequired) {
				writeJSONError(w, http.StatusForbidden, "ESIGNREQ",
					"signature required for this trust level: "+err.Error())
				return
			}
			writeJSONError(w, http.StatusForbidden, "ESIGNFAIL",
				"signature verification failed: "+err.Error())
			return
		}

		bundleDir, haveLocal, berr := s.marketplace.BundlePath(r.Context(), ref)
		if berr != nil {
			writeJSONError(w, http.StatusInternalServerError, "EMARKET", berr.Error())
			return
		}
		if haveLocal {
			// Local-backed catalog (M3 mock + airgapped). The bundle
			// is already on disk; TrustedSource skips HTTPS + sha256
			// (those guarantees come from the filesystem itself).
			src = install.TrustedSource{
				Path:  bundleDir,
				Label: "marketplace://" + entry.Publisher + "/" + entry.Name + "@" + entry.Version,
			}
		} else {
			// Remote-backed entry. HTTPSSource downloads the zip,
			// verifies SHA-256, extracts to staging — all gated by
			// the same signature policy as local installs.
			if entry.ArtifactURL == "" {
				writeJSONError(w, http.StatusBadRequest, "EBADSRC",
					"marketplace entry has no artifact URL")
				return
			}
			src = install.HTTPSSource{
				URL:            entry.ArtifactURL,
				ExpectedSHA256: entry.SHA256,
			}
		}
	} else {
		var perr error
		src, perr = install.ParseSource(req.Src)
		if perr != nil {
			writeJSONError(w, http.StatusBadRequest, "EBADSRC", perr.Error())
			return
		}
	}

	pending, err := s.installer.Stage(r.Context(), src)
	if err != nil {
		switch {
		case errors.Is(err, install.ErrLocalDisabled):
			// Defence-in-depth: Stage itself returns ErrLocalDisabled when
			// AllowLocal==false. The explicit isLocalScheme check above fires
			// first for direct local: requests, but this catches any path
			// that reaches Stage with a local source despite the handler gate.
			writeJSONError(w, http.StatusForbidden, "EFORBIDDEN",
				"local plugin installs disabled; set OPENDRAY_ALLOW_LOCAL_PLUGINS=1")
		case errors.Is(err, install.ErrInvalidManifest):
			writeJSONError(w, http.StatusBadRequest, "EBADMANIFEST", err.Error())
		case errors.Is(err, install.ErrNotImplemented):
			writeJSONError(w, http.StatusNotImplemented, "ENOTIMPL", err.Error())
		default:
			writeJSONError(w, http.StatusInternalServerError, "ESTAGEFAIL", err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(installResponse{ //nolint:errcheck
		Token:        pending.Token,
		Name:         pending.Name,
		Version:      pending.Version,
		Perms:        pending.Perms,
		ExpiresAt:    pending.ExpiresAt,
		ManifestHash: pending.ManifestHash,
	})
}

// pluginsInstallConfirm handles POST /api/plugins/install/confirm.
//
// Flow:
//  1. Decode {"token":"..."}.
//  2. Missing token → 400.
//  3. PeekName to retrieve the plugin name before Confirm consumes the token.
//  4. Installer.Confirm → 410 ETOKEN on ErrTokenNotFound | 500 on other.
//  5. 200 {installed:true, name:<name>}.
func (s *Server) pluginsInstallConfirm(w http.ResponseWriter, r *http.Request) {
	var req confirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "EINVAL", "invalid JSON body")
		return
	}

	if req.Token == "" {
		writeJSONError(w, http.StatusBadRequest, "EINVAL", "token is required")
		return
	}

	// PeekName before consuming the token so we can return the name in the
	// response. Confirm's error will tell us if the token is gone/expired.
	name, _ := s.installer.PeekName(req.Token)

	if err := s.installer.Confirm(r.Context(), req.Token); err != nil {
		if errors.Is(err, install.ErrTokenNotFound) {
			writeJSONError(w, http.StatusGone, "ETOKEN", err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "ECONFIRMFAIL", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(confirmResponse{ //nolint:errcheck
		Installed: true,
		Name:      name,
	})
}

// pluginsUninstall handles DELETE /api/plugins/{name}.
//
// Installer.Uninstall is idempotent per T6 spec, so unknown plugin names
// return 200 {status:"uninstalled"} — the desired post-condition is already
// satisfied.
func (s *Server) pluginsUninstall(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if err := s.installer.Uninstall(r.Context(), name); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "EUNINSTALLFAIL", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(uninstallResponse{ //nolint:errcheck
		Status: "uninstalled",
		Name:   name,
	})
}

// pluginsAudit handles GET /api/plugins/{name}/audit?limit=N.
//
// Default limit: 100. Clamped to [1, 1000] by store.DB.TailAudit.
// The handler does its own parse-and-default so limit=0 sends 1 to the DB
// (clamped), not 0. limit=2000 sends 1000 (clamped by DB layer).
func (s *Server) pluginsAudit(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			limit = n
		}
	}

	entries, err := s.hub.DB().TailAudit(r.Context(), name, limit)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "EAUDITFAIL", err.Error())
		return
	}

	// Convert store.AuditEntry slice to []auditEntryDTO.
	// We also need the per-row timestamp from the DB. TailAudit does not
	// return the ts column because store.AuditEntry lacks a Ts field.
	// We populate Ts as time.Time{} (zero) for M1 — the DB column is
	// returned via the ORDER BY clause for correctness but we only surface
	// the zero timestamp in the DTO until AuditEntry gains a Ts field (M2).
	//
	// NOTE: store.AuditEntry does not yet carry a Ts field. We use zero-time
	// for now and document this as a known M1 limitation in the report.
	dtos := make([]auditEntryDTO, 0, len(entries))
	for _, e := range entries {
		caps := e.Caps
		if caps == nil {
			caps = []string{}
		}
		dtos = append(dtos, auditEntryDTO{
			Ts:         e.Ts,
			Ns:         e.Ns,
			Method:     e.Method,
			Caps:       caps,
			Result:     e.Result,
			DurationMs: e.DurationMs,
			ArgsHash:   e.ArgsHash,
			Message:    e.Message,
		})
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dtos) //nolint:errcheck
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

// isLocalScheme returns true when raw starts with "local:" or is a bare
// absolute path (filepath.IsAbs). These are the sources gated by
// OPENDRAY_ALLOW_LOCAL_PLUGINS.
func isLocalScheme(raw string) bool {
	if len(raw) == 0 {
		return false
	}
	// Bare absolute path (Unix "/" or Windows "C:\").
	if raw[0] == '/' || (len(raw) > 2 && raw[1] == ':') {
		return true
	}
	// Explicit local: scheme.
	const prefix = "local:"
	if len(raw) >= len(prefix) && raw[:len(prefix)] == prefix {
		return true
	}
	return false
}
