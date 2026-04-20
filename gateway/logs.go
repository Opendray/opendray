package gateway

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"github.com/opendray/opendray/gateway/logs"
	"github.com/opendray/opendray/plugin"
)

// getLogsConfig resolves the plugin config for the log-viewer plugin.
// Pulls through s.effectiveConfig so values written through the v1
// Configure form (plugin_kv.__config.*) are seen even when the legacy
// plugins.config JSONB column is stale.
func (s *Server) getLogsConfig(ctx context.Context, pluginName string) (logs.Config, error) {
	info := s.plugins.ListInfo()
	for _, pi := range info {
		if pi.Provider.Name != pluginName {
			continue
		}
		if pi.Provider.Type != plugin.ProviderTypePanel || !pi.Enabled {
			break
		}
		cfg := s.effectiveConfig(ctx, pluginName, pi.Config)
		cleanRoots := splitCSV(stringVal(cfg, "allowedRoots", ""))
		exts := splitCSV(stringVal(cfg, "extensions", ".log,.txt,.out,.err"))

		// backlogBytes is expressed in KB in the manifest — convert.
		// intVal accepts both numeric types (legacy JSONB) and JSON
		// strings (v1 Configure overlay), so one helper handles both.
		backlogBytes := int64(intVal(cfg, "backlogBytes", 64)) * 1024
		showHidden := boolVal(cfg, "showHidden", false)
		return logs.Config{
			AllowedRoots: cleanRoots,
			Extensions:   exts,
			BacklogBytes: backlogBytes,
			ShowHidden:   showHidden,
		}, nil
	}
	return logs.Config{}, fmt.Errorf("log-viewer plugin %q not found or not enabled", pluginName)
}

// logsList returns the directory listing for the Log Viewer plugin.
// GET /api/logs/{plugin}/list?path=...
func (s *Server) logsList(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getLogsConfig(r.Context(), chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	if len(cfg.AllowedRoots) == 0 {
		respondError(w, http.StatusBadRequest, "log-viewer not configured: set allowedRoots in Providers page")
		return
	}
	entries, err := logs.List(cfg, r.URL.Query().Get("path"))
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if entries == nil {
		entries = []logs.FileEntry{}
	}
	respondJSON(w, http.StatusOK, entries)
}

// logsTailWS upgrades to a WebSocket and streams the file at ?path= with
// optional ?grep= regex filter. Closing the socket (or sending any message)
// cancels the tail.
func (s *Server) logsTailWS(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getLogsConfig(r.Context(), chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	path := r.URL.Query().Get("path")
	if path == "" {
		respondError(w, http.StatusBadRequest, "path is required")
		return
	}
	grep := r.URL.Query().Get("grep")
	var re *regexp.Regexp
	if grep != "" {
		re, err = regexp.Compile(grep)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid grep regex: "+err.Error())
			return
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close() //nolint:errcheck

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Any inbound message from the client (or a close) cancels the tail.
	go func() {
		for {
			if _, _, rerr := conn.ReadMessage(); rerr != nil {
				cancel()
				return
			}
		}
	}()

	writer := &lineFilterWriter{
		grep: re,
		send: func(line string) error {
			return conn.WriteMessage(websocket.TextMessage, []byte(line))
		},
		onError: cancel,
	}
	if err := logs.Tail(ctx, cfg, path, writer); err != nil {
		_ = conn.WriteMessage(websocket.TextMessage,
			[]byte("\n--- error: "+err.Error()+" ---"))
	}
}

// lineFilterWriter buffers incoming raw bytes into complete lines, optionally
// filters them with a regex, and forwards each matching line via `send`.
// Incomplete trailing bytes are held until the next write completes the line.
type lineFilterWriter struct {
	mu      sync.Mutex
	buf     strings.Builder
	grep    *regexp.Regexp
	send    func(string) error
	onError func()
}

func (w *lineFilterWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf.Write(p)
	s := w.buf.String()
	nl := strings.LastIndex(s, "\n")
	if nl < 0 {
		return len(p), nil
	}
	complete := s[:nl]
	rest := s[nl+1:]
	w.buf.Reset()
	w.buf.WriteString(rest)
	for _, line := range strings.Split(complete, "\n") {
		if w.grep != nil && !w.grep.MatchString(line) {
			continue
		}
		if err := w.send(line); err != nil {
			if w.onError != nil {
				w.onError()
			}
			return len(p), err
		}
	}
	return len(p), nil
}

// ── Local helpers ───────────────────────────────────────────────

func splitCSV(s string) []string {
	out := make([]string, 0, 4)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
