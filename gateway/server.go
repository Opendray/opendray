// Package gateway provides the HTTP/WebSocket API for NTC.
package gateway

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"context"

	gitpkg "github.com/linivek/ntc/gateway/git"
	"github.com/linivek/ntc/gateway/mcp"
	"github.com/linivek/ntc/gateway/tasks"
	"github.com/linivek/ntc/gateway/telegram"
	"github.com/linivek/ntc/kernel/auth"
	"github.com/linivek/ntc/kernel/hub"
	"github.com/linivek/ntc/plugin"
)

// Server is the main HTTP server for NTC.
type Server struct {
	router   chi.Router
	hub      *hub.Hub
	plugins  *plugin.Runtime
	auth     *auth.Auth
	logger   *slog.Logger
	tasks    *tasks.Runner
	telegram *telegram.Manager
	mcp      *mcp.Handlers
	git      *gitpkg.Manager
}

// Config holds gateway configuration.
type Config struct {
	Hub        *hub.Hub
	Plugins    *plugin.Runtime
	MCP        *mcp.Runtime
	Auth       *auth.Auth
	Logger     *slog.Logger
	FrontendFS fs.FS // embedded frontend dist (optional)
}

// New creates a gateway server with all routes configured.
func New(cfg Config) *Server {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	s := &Server{
		hub:     cfg.Hub,
		plugins: cfg.Plugins,
		auth:    cfg.Auth,
		logger:  cfg.Logger,
		tasks:   tasks.NewRunner(),
		git:     gitpkg.NewManager(),
	}
	if cfg.MCP != nil {
		s.mcp = mcp.NewHandlers(cfg.MCP)
	}
	// Telegram bridge — watches the "telegram" plugin and starts/stops
	// the bot to match. Safe to construct even if the plugin is disabled.
	s.telegram = telegram.NewManager(cfg.Plugins, cfg.Hub, cfg.Plugins.HookBus(), cfg.Logger)
	s.telegram.Start(context.Background())

	r := chi.NewRouter()

	// Global middleware
	r.Use(corsMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(loggingMiddleware(cfg.Logger))

	// Public routes
	r.Get("/api/health", s.health)
	r.Post("/api/auth/login", s.login)
	r.Get("/terminal.html", s.serveTerminalHTML)

	// Protected routes
	r.Group(func(r chi.Router) {
		if cfg.Auth != nil {
			r.Use(cfg.Auth.Middleware)
		}

		// Session management
		r.Get("/api/sessions", s.listSessions)
		r.Post("/api/sessions", s.createSession)
		r.Get("/api/sessions/{id}", s.getSession)
		r.Delete("/api/sessions/{id}", s.deleteSession)
		r.Post("/api/sessions/{id}/start", s.startSession)
		r.Post("/api/sessions/{id}/stop", s.stopSession)
		r.Post("/api/sessions/{id}/input", s.sendInput)
		r.Post("/api/sessions/{id}/resize", s.resizeSession)
		r.Post("/api/sessions/{id}/image", s.sessionAttachImage)
		r.Get("/api/sessions/{id}/ws", s.handleWebSocket)

		// Provider management
		r.Get("/api/providers", s.listProviders)
		r.Post("/api/providers", s.registerProvider)
		r.Get("/api/providers/{name}", s.getProvider)
		r.Patch("/api/providers/{name}/toggle", s.toggleProvider)
		r.Put("/api/providers/{name}/config", s.updateProviderConfig)
		r.Delete("/api/providers/{name}", s.deleteProvider)
		r.Get("/api/providers/{name}/models", s.detectModels)

		// Preview — URL discovery from terminal buffers
			r.Get("/api/preview/discover", s.previewDiscover)

			// Simulator — screenshot capture + touch/key input
			r.Get("/api/simulator/screenshot", s.simulatorScreenshot)
			r.Post("/api/simulator/input", s.simulatorInput)
			r.Get("/api/simulator/stream/ws", s.simulatorStreamWS)

			// Docs browsing (panel plugins like obsidian-reader)
		r.Get("/api/docs/{plugin}/tree", s.docsTree)
		r.Get("/api/docs/{plugin}/file", s.docsFile)
		r.Get("/api/docs/{plugin}/search", s.docsSearch)

		// File browser (panel plugins like file-browser)
		r.Get("/api/files/{plugin}/tree", s.filesTree)
		r.Get("/api/files/{plugin}/file", s.filesFile)
		r.Get("/api/files/{plugin}/search", s.filesSearch)
		r.Post("/api/files/{plugin}/mkdir", s.filesMkdir)

		// Log viewer (panel plugins, tail-follow files with grep)
		r.Get("/api/logs/{plugin}/list",    s.logsList)
		r.Get("/api/logs/{plugin}/tail/ws", s.logsTailWS)

		// Claude multi-account — one row per OAuth token the host tool
		// (`claude-acc`) manages. Sessions bind to an account via
		// claude_account_id; the token file is read only at spawn time.
		r.Get("/api/claude-accounts",                  s.listClaudeAccounts)
		r.Post("/api/claude-accounts",                 s.createClaudeAccount)
		r.Post("/api/claude-accounts/import-local",    s.importLocalClaudeAccounts)
		r.Get("/api/claude-accounts/{id}",             s.getClaudeAccount)
		r.Put("/api/claude-accounts/{id}",             s.updateClaudeAccount)
		r.Patch("/api/claude-accounts/{id}/toggle",    s.toggleClaudeAccount)
		r.Put("/api/claude-accounts/{id}/token",       s.setClaudeAccountToken)
		r.Delete("/api/claude-accounts/{id}",          s.deleteClaudeAccount)
		r.Post("/api/sessions/{id}/switch-account",    s.switchSessionAccount)

		// LLM Providers — address book of OpenAI-compatible model
		// endpoints (Mac Ollama, LM Studio, Groq, Gemini, custom).
		// Sessions bind to a provider via llm_provider_id; at spawn
		// the hub injects OPENAI_BASE_URL / OPENAI_API_KEY into the
		// agent CLI (OpenCode, crush, …). The /models probe hits
		// upstream /v1/models so the UI can offer a dropdown.
		r.Get("/api/llm-providers",                 s.listLLMProviders)
		r.Post("/api/llm-providers",                s.createLLMProvider)
		r.Get("/api/llm-providers/{id}",            s.getLLMProvider)
		r.Put("/api/llm-providers/{id}",            s.updateLLMProvider)
		r.Patch("/api/llm-providers/{id}/toggle",   s.toggleLLMProvider)
		r.Delete("/api/llm-providers/{id}",         s.deleteLLMProvider)
		r.Get("/api/llm-providers/{id}/models",     s.probeLLMProviderModels)

		// MCP server management — configs injected as temp files
		// into claude/codex sessions at spawn, never touching ~/.claude*.
		if s.mcp != nil {
			r.Get("/api/mcp/servers",           s.mcp.List)
			r.Post("/api/mcp/servers",          s.mcp.Create)
			r.Get("/api/mcp/servers/{id}",      s.mcp.Get)
			r.Put("/api/mcp/servers/{id}",      s.mcp.Update)
			r.Patch("/api/mcp/servers/{id}/toggle", s.mcp.Toggle)
			r.Delete("/api/mcp/servers/{id}",   s.mcp.Delete)
			r.Get("/api/mcp/agents",            s.mcp.Agents)
		}

		// Telegram bridge — admin endpoints (the bot itself talks
		// directly to api.telegram.org, no inbound webhook for M1/M2)
		r.Get("/api/telegram/status",  s.telegramStatus)
		r.Post("/api/telegram/test",   s.telegramTest)
		r.Get("/api/telegram/links",   s.telegramLinks)
		r.Post("/api/telegram/unlink", s.telegramUnlink)

		// Task runner (panel plugins, Makefile / package.json / shell scripts)
		r.Get("/api/tasks/{plugin}/list", s.tasksList)
		r.Post("/api/tasks/{plugin}/run", s.tasksRun)
		r.Get("/api/tasks/{plugin}/runs", s.tasksRuns)
		r.Get("/api/tasks/{plugin}/run/{runId}", s.tasksRunGet)
		r.Post("/api/tasks/{plugin}/run/{runId}/stop", s.tasksRunStop)
		r.Get("/api/tasks/{plugin}/run/{runId}/ws", s.tasksRunWS)

		// Git panel — per-repo status, diff, log, branches, commit; plus
		// a per-session baseline so the UI can show only what changed
		// during the current session (SnapshotHEAD → SessionDiff).
		r.Get("/api/git/{plugin}/status",   s.gitStatus)
		r.Get("/api/git/{plugin}/diff",     s.gitDiff)
		r.Get("/api/git/{plugin}/log",      s.gitLog)
		r.Get("/api/git/{plugin}/branches", s.gitBranches)
		r.Post("/api/git/{plugin}/stage",   s.gitStage)
		r.Post("/api/git/{plugin}/unstage", s.gitUnstage)
		r.Post("/api/git/{plugin}/discard", s.gitDiscard)
		r.Post("/api/git/{plugin}/commit",  s.gitCommit)
		r.Post("/api/git/{plugin}/session/snapshot", s.gitSessionSnapshot)
		r.Get("/api/git/{plugin}/session/diff",      s.gitSessionDiff)

		// Database browsing (panel plugins, PostgreSQL read-only)
		r.Get("/api/database/{plugin}/databases", s.dbDatabases)
		r.Get("/api/database/{plugin}/schemas", s.dbSchemas)
		r.Get("/api/database/{plugin}/tables", s.dbTables)
		r.Get("/api/database/{plugin}/columns", s.dbColumns)
		r.Get("/api/database/{plugin}/preview", s.dbPreview)
		r.Post("/api/database/{plugin}/query", s.dbQuery)

		// Hook subscriptions
		r.Get("/api/hooks", s.listHooks)
	})

	// SPA frontend (serve embedded dist or fallback)
	if cfg.FrontendFS != nil {
		fileServer := http.FileServer(http.FS(cfg.FrontendFS))
		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			// Try serving the file directly
			if !strings.HasPrefix(r.URL.Path, "/api/") {
				// Check if file exists
				if f, err := cfg.FrontendFS.Open(strings.TrimPrefix(r.URL.Path, "/")); err == nil {
					f.Close()
					fileServer.ServeHTTP(w, r)
					return
				}
				// SPA fallback: serve index.html for all non-file routes
				r.URL.Path = "/"
				fileServer.ServeHTTP(w, r)
				return
			}
			respondError(w, http.StatusNotFound, "not found")
		})
	}

	s.router = r
	return s
}

// Handler returns the HTTP handler.
func (s *Server) Handler() http.Handler {
	return s.router
}

// ── Terminal HTML (xterm.js for web) ────────────────────────────

func (s *Server) serveTerminalHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(terminalHTML))
}

// ── Health ──────────────────────────────────────────────────────

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"service":  "ntc",
		"sessions": s.hub.RunningCount(),
		"plugins":  len(s.plugins.List()),
	})
}

// ── Auth ────────────────────────────────────────────────────────

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil {
		respondJSON(w, http.StatusOK, map[string]string{"token": "no-auth-configured"})
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}

	// For now, accept configured credentials
	// TODO: proper user management
	token, err := s.auth.Issue(req.Username)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"token": token})
}

// ── Hooks ───────────────────────────────────────────────────────

func (s *Server) listHooks(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, s.plugins.HookBus().ListSubscriptions())
}

// ── Helpers ─────────────────────────────────────────────────────

func respondJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, map[string]string{"error": msg})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &statusWriter{ResponseWriter: w, status: 200}
			next.ServeHTTP(ww, r)
			logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.status,
				"duration", time.Since(start).String(),
			)
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

// Hijack implements http.Hijacker for WebSocket upgrade support.
func (sw *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := sw.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}
