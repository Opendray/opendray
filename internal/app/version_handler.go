package app

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray-v2/internal/eventbus"
	"github.com/opendray/opendray-v2/internal/integration"
	"github.com/opendray/opendray-v2/internal/selfupdate"
	"github.com/opendray/opendray-v2/internal/version"
)

// unitFile is present only when the install shipped the privileged
// self-update units (path + oneshot). Its absence means the install
// predates the feature → in-app upgrade can't be triggered, so the API
// reports capable=false and the UI falls back to a guided command.
const selfUpdatePathUnit = "/etc/systemd/system/opendray-selfupdate.path"

// versionHandlers serves the dashboard's About/version surface: a
// read-only release check and the admin-only "request a background
// upgrade" trigger. Mounted in the admin-session-only group.
type versionHandlers struct {
	dataDir string // daemon-writable dir the privileged oneshot watches
	bus     *eventbus.Hub
	log     *slog.Logger
}

func newVersionHandlers(dataDir string, bus *eventbus.Hub, log *slog.Logger) *versionHandlers {
	return &versionHandlers{dataDir: dataDir, bus: bus, log: log.With("component", "version.http")}
}

// selfUpdateStateDir is the daemon-writable directory the privileged
// self-update oneshot watches for a request file. Matches the unit's
// WorkingDirectory/ReadWritePaths; overridable for non-standard installs.
func selfUpdateStateDir() string {
	if d := strings.TrimSpace(os.Getenv("OPENDRAY_STATE_DIR")); d != "" {
		return d
	}
	return "/var/lib/opendray"
}

func (h *versionHandlers) Mount(r chi.Router) {
	r.Get("/version", h.get)
	r.Post("/version/update", h.requestUpdate)
}

// selfUpdateCapable reports whether a one-click background upgrade can be
// triggered on this host: Linux, the privileged units are installed, and
// the watched dir is writable by the daemon.
func (h *versionHandlers) selfUpdateCapable() bool {
	if runtime.GOOS != "linux" || h.dataDir == "" {
		return false
	}
	if _, err := os.Stat(selfUpdatePathUnit); err != nil {
		return false
	}
	return true
}

func (h *versionHandlers) get(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{
		"current":         selfupdate.NormalizeVersion(version.Version),
		"commit":          version.Commit,
		"updateAvailable": false,
		"selfUpdate":      h.selfUpdateCapable(),
		"pending":         selfupdate.PendingRequest(h.dataDir),
	}
	st, err := selfupdate.Check(r.Context(), version.Version)
	if err != nil {
		// Soft-fail: still report the current version + capability, just
		// no "latest" (offline / rate-limited). The UI shows current only.
		resp["checkError"] = err.Error()
		writeVersionJSON(w, http.StatusOK, resp)
		return
	}
	resp["latest"] = st.Latest
	resp["updateAvailable"] = st.Available
	resp["notesUrl"] = st.NotesURL
	writeVersionJSON(w, http.StatusOK, resp)
}

func (h *versionHandlers) requestUpdate(w http.ResponseWriter, r *http.Request) {
	if !h.selfUpdateCapable() {
		writeVersionJSON(w, http.StatusConflict, map[string]any{
			"error": "in-app upgrade isn't available on this install. " +
				"Run `opendray update` (Linux: re-run the installer first to add the self-update units), or upgrade manually.",
			"selfUpdate": false,
		})
		return
	}
	if selfupdate.PendingRequest(h.dataDir) {
		writeVersionJSON(w, http.StatusConflict, map[string]any{
			"error": "an upgrade is already in progress.", "pending": true,
		})
		return
	}

	force := r.URL.Query().Get("force") == "true" || r.URL.Query().Get("force") == "1"

	st, err := selfupdate.Check(r.Context(), version.Version)
	if err != nil {
		writeVersionJSON(w, http.StatusBadGateway, map[string]any{"error": "couldn't reach the release feed: " + err.Error()})
		return
	}
	if !st.Available && !force {
		writeVersionJSON(w, http.StatusOK, map[string]any{"error": "already on the latest release.", "current": st.Current})
		return
	}

	by := "admin"
	if p, ok := integration.CurrentPrincipal(r.Context()); ok && p.ID != "" {
		by = p.ID
	}
	req := selfupdate.Request{Version: st.Latest, RequestedBy: by, RequestedAt: time.Now().UTC(), Force: force}
	if err := selfupdate.WriteRequest(h.dataDir, req); err != nil {
		writeVersionJSON(w, http.StatusInternalServerError, map[string]any{"error": "couldn't queue the upgrade: " + err.Error()})
		return
	}
	// Privileged action — leave a loud trail in the journal + audit bus.
	h.log.Warn("self-update requested", "from", st.Current, "to", st.Latest, "force", force, "by", by)
	if h.bus != nil {
		h.bus.Publish(eventbus.Event{Topic: "selfupdate.requested", Data: map[string]any{
			"from": st.Current, "to": st.Latest, "force": force, "requestedBy": by,
		}})
	}
	// 202: the root oneshot will pick up the request, upgrade, and restart
	// the daemon — so the client should poll GET /version afterward and
	// expect the connection to drop during the restart.
	writeVersionJSON(w, http.StatusAccepted, map[string]any{
		"queued": true, "from": st.Current, "to": st.Latest, "force": force,
		"note": "Upgrading in the background; the service will restart and running sessions will reconnect.",
	})
}

func writeVersionJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
