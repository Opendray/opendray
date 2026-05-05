package backup

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// Handlers wires Service onto the chi router. Caller mounts under
// the admin-only group — every endpoint here assumes the caller is
// already authenticated as the operator.
type Handlers struct {
	svc *Service
}

func NewHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

// Mount registers /backups, /backup-targets, /backup-schedules,
// /backup-status under r.
func (h *Handlers) Mount(r chi.Router) {
	r.Route("/backups", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Post("/restore", h.restore)
		r.Get("/{id}", h.get)
		r.Get("/{id}/download", h.download)
		r.Delete("/{id}", h.delete)
	})
	r.Route("/backup-schedules", func(r chi.Router) {
		r.Get("/", h.listSchedules)
		r.Post("/", h.createSchedule)
		r.Get("/{id}", h.getSchedule)
		r.Patch("/{id}", h.updateSchedule)
		r.Delete("/{id}", h.deleteSchedule)
	})
	r.Route("/backup-targets", func(r chi.Router) {
		r.Get("/", h.listTargets)
		r.Post("/", h.createTarget)
		r.Patch("/{id}", h.updateTarget)
		r.Delete("/{id}", h.deleteTarget)
		r.Post("/{id}/test", h.testTarget)
	})
	h.MountExports(r)
	h.MountImports(r)
	r.Get("/backup-status", h.status)
}

// list serves GET /backups. Filters: ?status=&target_id=&limit=.
func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	f := BackupListFilter{}
	if v := r.URL.Query().Get("status"); v != "" {
		f.Status = BackupStatus(v)
	}
	if v := r.URL.Query().Get("target_id"); v != "" {
		f.TargetID = v
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			f.Limit = n
		}
	}
	list, err := h.svc.ListBackups(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if list == nil {
		list = []Backup{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"backups": list})
}

// get serves GET /backups/{id}.
func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	b, err := h.svc.GetBackup(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrBackupNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// create serves POST /backups. Body: {target_id, include_config}.
// Returns 202 + the freshly-inserted Backup row (status='pending').
func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TargetID      string `json:"target_id"`
		IncludeConfig bool   `json:"include_config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err))
		return
	}
	if req.TargetID == "" {
		req.TargetID = "local"
	}
	b, err := h.svc.RunBackupNow(r.Context(), RunBackupRequest{
		TargetID:      req.TargetID,
		TriggeredBy:   TriggeredAPI,
		IncludeConfig: req.IncludeConfig,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrTargetNotFound):
			writeError(w, http.StatusNotFound, err)
		case errors.Is(err, ErrFeatureDisabled):
			writeError(w, http.StatusServiceUnavailable, err)
		default:
			writeError(w, http.StatusInternalServerError, err)
		}
		return
	}
	writeJSON(w, http.StatusAccepted, b)
}

// download serves GET /backups/{id}/download — streams the
// (encrypted) bundle blob with octet-stream headers.
func (h *Handlers) download(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rc, b, err := h.svc.DownloadBackup(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, ErrBackupNotFound), errors.Is(err, ErrTargetNotFound):
			writeError(w, http.StatusNotFound, err)
		default:
			writeError(w, http.StatusInternalServerError, err)
		}
		return
	}
	defer rc.Close()

	filename := fmt.Sprintf("%s.tar.gz.enc", b.ID)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	if b.Bytes > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(b.Bytes, 10))
	}
	// Errors mid-copy can't change headers; client sees a truncated
	// download (and our stored sha256 disagrees). Acceptable for v1.
	_, _ = io.Copy(w, rc)
}

// delete serves DELETE /backups/{id}. Soft-deletes the row and
// best-effort removes the blob from its target.
func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteBackup(r.Context(), id); err != nil {
		if errors.Is(err, ErrBackupNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── schedules ────────────────────────────────────────────────────

func (h *Handlers) listSchedules(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListSchedules(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if list == nil {
		list = []Schedule{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"schedules": list})
}

func (h *Handlers) getSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sc, err := h.svc.GetSchedule(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrScheduleNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, sc)
}

func (h *Handlers) createSchedule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TargetID    string `json:"target_id"`
		IntervalSec int    `json:"interval_sec"`
		Retention   int    `json:"retention"`
		Enabled     bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err))
		return
	}
	if req.Retention == 0 {
		req.Retention = 7
	}
	sc, err := h.svc.CreateSchedule(r.Context(), CreateScheduleRequest{
		TargetID:    req.TargetID,
		IntervalSec: req.IntervalSec,
		Retention:   req.Retention,
		Enabled:     req.Enabled,
	})
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, sc)
}

func (h *Handlers) updateSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		IntervalSec *int  `json:"interval_sec,omitempty"`
		Retention   *int  `json:"retention,omitempty"`
		Enabled     *bool `json:"enabled,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err))
		return
	}
	if err := h.svc.UpdateSchedule(r.Context(), id, SchedulePatch{
		IntervalSec: req.IntervalSec,
		Retention:   req.Retention,
		Enabled:     req.Enabled,
	}); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	sc, err := h.svc.GetSchedule(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, sc)
}

func (h *Handlers) deleteSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteSchedule(r.Context(), id); err != nil {
		if errors.Is(err, ErrScheduleNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── targets / status ─────────────────────────────────────────────

// listTargets serves GET /backup-targets.
func (h *Handlers) listTargets(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListTargets(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if list == nil {
		list = []TargetSpec{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"targets": list})
}

func (h *Handlers) createTarget(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID      string         `json:"id"`
		Kind    TargetKind     `json:"kind"`
		Config  map[string]any `json:"config"`
		Enabled bool           `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err))
		return
	}
	spec, err := h.svc.CreateTarget(r.Context(), CreateTargetRequest{
		ID:      req.ID,
		Kind:    req.Kind,
		Config:  req.Config,
		Enabled: req.Enabled,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrTargetUnsupported):
			writeError(w, http.StatusBadRequest, err)
		default:
			writeError(w, http.StatusBadRequest, err)
		}
		return
	}
	writeJSON(w, http.StatusCreated, spec)
}

func (h *Handlers) updateTarget(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Config  map[string]any `json:"config,omitempty"`
		Enabled *bool          `json:"enabled,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err))
		return
	}
	spec, err := h.svc.UpdateTarget(r.Context(), id, UpdateTargetRequest{
		Config:  req.Config,
		Enabled: req.Enabled,
	})
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, spec)
}

func (h *Handlers) deleteTarget(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteTarget(r.Context(), id); err != nil {
		if errors.Is(err, ErrTargetNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// testTarget runs HealthCheck against the target with a tight timeout.
// Response: {ok: bool, latency_ms: number, error?: string}.
func (h *Handlers) testTarget(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	resp := map[string]any{"ok": true}
	if err := h.svc.TestTarget(r.Context(), id); err != nil {
		resp["ok"] = false
		resp["error"] = err.Error()
	}
	writeJSON(w, http.StatusOK, resp)
}

// restore serves POST /backups/restore. Multipart form:
//
//	bundle (file)        — encrypted .tar.gz.enc
//	target_dsn (string)  — empty = opendray's own DSN (DANGER)
//	clean (bool)         — true = pg_restore --clean --if-exists
//	confirm (string)     — must equal "I understand" when target_dsn is empty
//	note (string)        — free-form audit note
//
// Returns RestoreResult on success.
func (h *Handlers) restore(w http.ResponseWriter, r *http.Request) {
	// 256 MiB cap on bundle size; opendray's own DB is much
	// smaller in practice, and pg_restore happily streams from
	// disk. Bumps as needed via env later.
	if err := r.ParseMultipartForm(256 << 20); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("multipart: %w", err))
		return
	}
	file, hdr, err := r.FormFile("bundle")
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("missing bundle file: %w", err))
		return
	}
	defer file.Close()

	targetDSN := r.FormValue("target_dsn")
	clean := r.FormValue("clean") == "true"
	confirm := r.FormValue("confirm")
	note := r.FormValue("note")

	if targetDSN == "" && confirm != "I understand" {
		writeError(w, http.StatusBadRequest,
			errors.New("restoring to opendray's own DB requires confirm=\"I understand\""))
		return
	}

	res, err := h.svc.RestoreBackup(r.Context(), RestoreRequest{
		Source:       file,
		TargetDSN:    targetDSN,
		Clean:        clean,
		OperatorNote: note,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrPgRestoreUnavailable):
			writeError(w, http.StatusServiceUnavailable, err)
		case errors.Is(err, ErrCipherWrongKey), errors.Is(err, ErrCipherFormat):
			writeError(w, http.StatusBadRequest, err)
		case errors.Is(err, ErrRestoreFingerprintMismatch),
			errors.Is(err, ErrRestoreNoDump):
			writeError(w, http.StatusBadRequest, err)
		default:
			// Mid-restore failure (pg_restore exit). Return the
			// result body so the UI can show pg_restore's tail.
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"error":  err.Error(),
				"result": res,
			})
		}
		return
	}
	_ = hdr // we don't currently expose the original filename
	writeJSON(w, http.StatusOK, res)
}

// status surfaces feature health for the UI banner: pg_dump
// resolved version, cipher fingerprint, etc.
func (h *Handlers) status(w http.ResponseWriter, r *http.Request) {
	pgVer, err := h.svc.PGVersion(r.Context())
	resp := map[string]any{
		"ok":                 err == nil,
		"key_fingerprint":    h.svc.CipherFingerprint(),
		"pg_dump_version":    pgVer,
		"pg_restore_version": h.svc.PgRestoreVersion(r.Context()),
	}
	if err != nil {
		resp["pg_dump_error"] = err.Error()
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}
