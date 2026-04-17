package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"github.com/linivek/ntc/gateway/tasks"
	"github.com/linivek/ntc/plugin"
)

// getTasksConfig resolves a task-runner panel plugin's config into a
// tasks.Config. The plugin must be enabled and have allowedRoots set.
func (s *Server) getTasksConfig(pluginName string) (tasks.Config, error) {
	info := s.plugins.ListInfo()
	for _, pi := range info {
		if pi.Provider.Name != pluginName {
			continue
		}
		if pi.Provider.Type != plugin.ProviderTypePanel || !pi.Enabled {
			return tasks.Config{}, fmt.Errorf("task-runner plugin %q not enabled", pluginName)
		}
		cfg := pi.Config

		var roots []string
		for _, r := range strings.Split(stringVal(cfg, "allowedRoots", ""), ",") {
			if r = strings.TrimSpace(r); r != "" {
				roots = append(roots, r)
			}
		}

		return tasks.Config{
			AllowedRoots:        roots,
			DefaultPath:         stringVal(cfg, "defaultPath", ""),
			IncludeMakefile:     boolVal(cfg, "includeMakefile", true),
			IncludePackageJSON:  boolVal(cfg, "includePackageJson", true),
			IncludeShellScripts: boolVal(cfg, "includeShellScripts", true),
			ShellTimeoutSec:     intVal(cfg, "shellTimeoutSec", 600),
			MaxConcurrent:       intVal(cfg, "maxConcurrent", 4),
			OutputBufferBytes:   intVal(cfg, "outputBufferKB", 256) * 1024,
		}, nil
	}
	return tasks.Config{}, fmt.Errorf("task-runner plugin %q not found", pluginName)
}

func boolVal(m map[string]any, key string, fallback bool) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return fallback
}

// tasksList: GET /api/tasks/{plugin}/list?path=
func (s *Server) tasksList(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getTasksConfig(chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	if len(cfg.AllowedRoots) == 0 {
		respondError(w, http.StatusBadRequest, "plugin not configured: set allowedRoots in Providers page")
		return
	}
	out, err := tasks.Discover(cfg, r.URL.Query().Get("path"))
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if out == nil {
		out = []tasks.Task{}
	}
	respondJSON(w, http.StatusOK, out)
}

// tasksRun: POST /api/tasks/{plugin}/run  body: { taskId, path }
// path is the directory to discover the task from (must match the listing call).
func (s *Server) tasksRun(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getTasksConfig(chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	var req struct {
		TaskID string `json:"taskId"`
		Path   string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.TaskID == "" {
		respondError(w, http.StatusBadRequest, "taskId is required")
		return
	}
	candidates, err := tasks.Discover(cfg, req.Path)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	var match *tasks.Task
	for i := range candidates {
		if candidates[i].ID == req.TaskID {
			match = &candidates[i]
			break
		}
	}
	if match == nil {
		respondError(w, http.StatusNotFound, "task not found at given path")
		return
	}
	run, err := s.tasks.Start(cfg, *match)
	if err != nil {
		respondError(w, http.StatusConflict, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, run)
}

// tasksRuns: GET /api/tasks/{plugin}/runs
func (s *Server) tasksRuns(w http.ResponseWriter, r *http.Request) {
	if _, err := s.getTasksConfig(chi.URLParam(r, "plugin")); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, s.tasks.List())
}

// tasksRunGet: GET /api/tasks/{plugin}/run/{runId}
func (s *Server) tasksRunGet(w http.ResponseWriter, r *http.Request) {
	if _, err := s.getTasksConfig(chi.URLParam(r, "plugin")); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	run, ok := s.tasks.Get(chi.URLParam(r, "runId"))
	if !ok {
		respondError(w, http.StatusNotFound, "run not found")
		return
	}
	respondJSON(w, http.StatusOK, run)
}

// tasksRunStop: POST /api/tasks/{plugin}/run/{runId}/stop
func (s *Server) tasksRunStop(w http.ResponseWriter, r *http.Request) {
	if _, err := s.getTasksConfig(chi.URLParam(r, "plugin")); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	if err := s.tasks.Stop(chi.URLParam(r, "runId")); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "stopping"})
}

// tasksRunWS: GET /api/tasks/{plugin}/run/{runId}/ws
// Streams the historical snapshot followed by live output. Closes when the run
// finishes; clients should then GET the run to read the final status/exit code.
func (s *Server) tasksRunWS(w http.ResponseWriter, r *http.Request) {
	if _, err := s.getTasksConfig(chi.URLParam(r, "plugin")); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	runID := chi.URLParam(r, "runId")
	snap, ch, done, unsubscribe, err := s.tasks.Subscribe(runID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	defer unsubscribe()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("tasks ws: upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	if len(snap) > 0 {
		if err := conn.WriteMessage(websocket.BinaryMessage, snap); err != nil {
			return
		}
	}

	// Ping keepalive
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
					return
				}
			}
		}
	}()

	for {
		select {
		case <-done:
			// Drain any remaining buffered chunks before closing.
			for {
				select {
				case chunk, ok := <-ch:
					if !ok {
						sendTaskExit(conn, s.tasks, runID)
						return
					}
					if err := conn.WriteMessage(websocket.BinaryMessage, chunk); err != nil {
						return
					}
				default:
					sendTaskExit(conn, s.tasks, runID)
					return
				}
			}
		case chunk, ok := <-ch:
			if !ok {
				sendTaskExit(conn, s.tasks, runID)
				return
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, chunk); err != nil {
				return
			}
		}
	}
}

func sendTaskExit(conn *websocket.Conn, runner *tasks.Runner, runID string) {
	run, ok := runner.Get(runID)
	if !ok {
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"type":     "exit",
		"status":   run.Status,
		"exitCode": run.ExitCode,
		"error":    run.Error,
	})
	conn.WriteMessage(websocket.TextMessage, payload)
}
