package settings

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"github.com/opendray/opendray-v2/internal/applog"
	"github.com/opendray/opendray-v2/internal/config"
	"github.com/opendray/opendray-v2/internal/wsutil"
)

// Handler exposes settings endpoints. Mount under an admin-only
// route group so only operators can read/write config or restart.
type Handler struct {
	svc      *Service
	log      *slog.Logger
	ring     *applog.Buffer
	upgrader websocket.Upgrader
}

// NewHandler builds the settings handler. `ring` is the in-process
// log ring buffer; pass nil to disable the /admin/logs endpoints.
func NewHandler(svc *Service, ring *applog.Buffer, log *slog.Logger) *Handler {
	if log == nil {
		log = slog.Default()
	}
	return &Handler{
		svc:  svc,
		log:  log.With("component", "settings.http"),
		ring: ring,
		upgrader: websocket.Upgrader{
			// Admin-only WS (logs/stream). Same CSWSH mitigation as
			// session/: bearer-token in ?token=, plus Origin must be
			// same-host or a LAN private range.
			CheckOrigin: wsutil.SameOriginCheck(),
		},
	}
}

// Mount registers /admin/settings*, /admin/restart, and /admin/logs*.
func (h *Handler) Mount(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
		r.Get("/settings", h.get)
		r.Put("/settings", h.put)
		r.Get("/settings/test-path", h.testPath)
		r.Post("/restart", h.restart)
		r.Get("/logs/tail", h.logsTail)
		r.Get("/logs/stream", h.logsStream)
		r.Get("/logs/download", h.logsDownload)
	})
}

type getResponse struct {
	Config     *config.Config `json:"config"`
	ConfigPath string         `json:"config_path"`
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.Get()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, getResponse{Config: c, ConfigPath: h.svc.ConfigPath()})
}

func (h *Handler) put(w http.ResponseWriter, r *http.Request) {
	var patch config.Config
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.Update(&patch); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// testPath answers a "does this path exist + does it look right for
// `kind`" probe so the UI can give immediate feedback as the operator
// types a custom roots path. Always returns 200 with an outcome
// payload; never errors out the request.
type testPathResponse struct {
	Path       string `json:"path"`
	Exists     bool   `json:"exists"`
	IsDir      bool   `json:"is_dir"`
	ChildCount int    `json:"child_count,omitempty"`
	Note       string `json:"note,omitempty"`
}

func (h *Handler) testPath(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimSpace(r.URL.Query().Get("path"))
	if p == "" {
		writeError(w, http.StatusBadRequest, errors.New("path query param required"))
		return
	}
	resolved := expandHome(p)
	resp := testPathResponse{Path: resolved}
	info, err := os.Stat(resolved)
	if err == nil {
		resp.Exists = true
		resp.IsDir = info.IsDir()
		if info.IsDir() {
			if entries, err := os.ReadDir(resolved); err == nil {
				resp.ChildCount = len(entries)
			}
		} else {
			resp.Note = "file"
		}
	} else if errors.Is(err, os.ErrNotExist) {
		resp.Note = "not found"
	} else {
		resp.Note = err.Error()
	}
	writeJSON(w, http.StatusOK, resp)
}

// restart self-execs the binary. The HTTP response is sent + flushed
// before the exec so the client gets a clean ack; the new process
// inherits the same PID, port, and args.
//
// Caller must run opendray under a process supervisor OR via a
// wrapper that keeps go run alive — without one, an exec failure
// leaves no daemon. macOS / Linux: syscall.Exec replaces the current
// process image; Go runtime is gone, control passes to the new image.
func (h *Handler) restart(w http.ResponseWriter, r *http.Request) {
	bin, err := os.Executable()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"status":"restarting"}`))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Detach from request scope; let response drain.
	go func() {
		time.Sleep(500 * time.Millisecond)
		h.log.Info("restart requested; exec self", "bin", bin, "args", os.Args)
		// Drop the request context so the http server doesn't try to
		// cancel us mid-exec.
		_ = context.TODO()
		if err := syscall.Exec(bin, os.Args, os.Environ()); err != nil {
			h.log.Error("self-exec failed", "err", err)
		}
	}()
}

// logsTail returns up to N most recent log records as a JSON
// array (oldest first). Empty array when the ring is empty or
// disabled at startup. Default n = 200, max = 2000.
func (h *Handler) logsTail(w http.ResponseWriter, r *http.Request) {
	if h.ring == nil {
		writeJSON(w, http.StatusOK, map[string]any{"records": []applog.Record{}})
		return
	}
	n := 200
	if v := r.URL.Query().Get("n"); v != "" {
		if x, err := strconv.Atoi(v); err == nil && x > 0 {
			n = x
		}
	}
	if n > 2000 {
		n = 2000
	}
	writeJSON(w, http.StatusOK, map[string]any{"records": h.ring.Snapshot(n)})
}

// logsStream upgrades to WebSocket, replays the current ring
// (oldest → newest) once, then forwards every new record as it
// arrives. Closes when the client disconnects.
func (h *Handler) logsStream(w http.ResponseWriter, r *http.Request) {
	if h.ring == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("log ring not initialised"))
		return
	}
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Debug("log ws upgrade failed", "err", err)
		return
	}
	defer conn.Close()

	// Subscribe BEFORE replaying the snapshot so a record landing
	// between the two doesn't slip through the gap.
	ch, unsub := h.ring.Subscribe()
	defer unsub()

	// Replay the buffer.
	for _, rec := range h.ring.Snapshot(0) {
		if err := conn.WriteJSON(rec); err != nil {
			return
		}
	}

	// Forward live records until the client disconnects.
	closed := make(chan struct{})
	go func() {
		defer close(closed)
		for {
			if _, _, err := conn.NextReader(); err != nil {
				return
			}
		}
	}()
	for {
		select {
		case <-closed:
			return
		case rec, ok := <-ch:
			if !ok {
				return
			}
			if err := conn.WriteJSON(rec); err != nil {
				return
			}
		}
	}
}

// logsDownload returns the entire ring (oldest first) as plain text
// — one line per record, formatted to mirror the live tail.
// Content-Disposition: attachment so the browser saves it.
func (h *Handler) logsDownload(w http.ResponseWriter, r *http.Request) {
	if h.ring == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("log ring not initialised"))
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set(
		"Content-Disposition",
		`attachment; filename="opendray-`+time.Now().Format("20060102-150405")+`.log"`,
	)
	for _, rec := range h.ring.Snapshot(0) {
		_, _ = w.Write([]byte(rec.Text))
		_, _ = w.Write([]byte{'\n'})
	}
}

// expandHome resolves a leading ~/ to $HOME. Mirrors the resolver in
// internal/app/expandPath but kept local to avoid importing the
// whole app package from a leaf handler.
func expandHome(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return p
	}
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			if p == "~" {
				return home
			}
			return filepath.Join(home, p[2:])
		}
	}
	return p
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
