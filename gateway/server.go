// Package gateway provides the HTTP/WebSocket API for OpenDray.
package gateway

import (
	"bufio"
	"crypto/subtle"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"context"

	gitpkg "github.com/opendray/opendray/gateway/git"
	"github.com/opendray/opendray/gateway/mcp"
	"github.com/opendray/opendray/gateway/tasks"
	"github.com/opendray/opendray/gateway/telegram"
	"github.com/opendray/opendray/kernel/auth"
	"github.com/opendray/opendray/kernel/hub"
	"github.com/opendray/opendray/kernel/setup"
	"github.com/opendray/opendray/plugin"
	"github.com/opendray/opendray/plugin/contributions"
	"github.com/opendray/opendray/plugin/install"
)

// Server is the main HTTP server for OpenDray.
type Server struct {
	router        chi.Router
	hub           *hub.Hub
	plugins       *plugin.Runtime
	auth          *auth.Auth
	creds         *auth.CredentialStore
	adminUsername string
	adminPassword string
	logger        *slog.Logger
	tasks         *tasks.Runner
	telegram      *telegram.Manager
	mcp           *mcp.Handlers
	git           *gitpkg.Manager
	installer     *install.Installer
	contribReg    *contributions.Registry
	cmdInvoker    commandInvoker
}

// Config holds gateway configuration.
type Config struct {
	Hub           *hub.Hub
	Plugins       *plugin.Runtime
	MCP           *mcp.Runtime
	Auth          *auth.Auth
	Credentials   *auth.CredentialStore
	AdminUsername string
	AdminPassword string
	Logger        *slog.Logger
	FrontendFS    fs.FS             // embedded frontend dist (optional)
	Installer     *install.Installer         // plugin install/uninstall orchestrator (T7)
	Contributions *contributions.Registry    // workbench contribution registry (T9)
	CommandInvoker commandInvoker            // command dispatcher (T11)
}

// New creates a gateway server with all routes configured.
func New(cfg Config) *Server {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	s := &Server{
		hub:           cfg.Hub,
		plugins:       cfg.Plugins,
		auth:          cfg.Auth,
		creds:         cfg.Credentials,
		adminUsername: cfg.AdminUsername,
		adminPassword: cfg.AdminPassword,
		logger:        cfg.Logger,
		tasks:         tasks.NewRunner(),
		git:           gitpkg.NewManager(),
		installer:     cfg.Installer,
		contribReg:    cfg.Contributions,
		cmdInvoker:    cfg.CommandInvoker,
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
	r.Use(bodySizeLimiter(1 << 20)) // 1 MB cap on request bodies

	// Rate limiter: 10 req/min for session mutate, 60 req/min for reads.
	sessionRL := newRateLimiter(10, time.Minute)
	readRL := newRateLimiter(60, time.Minute)

	// Public routes
	r.Get("/api/health", s.health)
	r.Get("/api/auth/status", s.authStatus)
	r.Post("/api/auth/login", s.login)
	r.Get("/terminal.html", s.serveTerminalHTML)

	// Protected routes
	r.Group(func(r chi.Router) {
		if cfg.Auth != nil {
			r.Use(cfg.Auth.Middleware)
		}

		// Admin credential management — under the authenticated group so
		// only a signed-in caller can rotate the password.
		r.Post("/api/auth/change-credentials", s.changeCredentials)

		// Session management
		r.With(readRL.middleware).Get("/api/sessions", s.listSessions)
		r.With(sessionRL.middleware).Post("/api/sessions", s.createSession)
		r.With(readRL.middleware).Get("/api/sessions/{id}", s.getSession)
		r.With(sessionRL.middleware).Delete("/api/sessions/{id}", s.deleteSession)
		r.With(sessionRL.middleware).Post("/api/sessions/{id}/start", s.startSession)
		r.With(sessionRL.middleware).Post("/api/sessions/{id}/stop", s.stopSession)
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

		// Plugin install / uninstall / audit (T7).
		// DELETE /api/providers/{name} stays for legacy compat (see api.go).
		// These new routes provide the full install lifecycle via Installer.
		r.Post("/api/plugins/install", s.pluginsInstall)
		r.Post("/api/plugins/install/confirm", s.pluginsInstallConfirm)
		r.Delete("/api/plugins/{name}", s.pluginsUninstall)
		r.Get("/api/plugins/{name}/audit", s.pluginsAudit)

		// Workbench contributions — flat view of all installed plugin contribution
		// points (commands, statusBar, keybindings, menus). Pure read; no DB (T9).
		r.Get("/api/workbench/contributions", s.workbenchContributions)

		// Command invoke — dispatches a named command on a named plugin (T11).
		r.Post("/api/plugins/{name}/commands/{id}/invoke", s.commandInvoke)
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

// NewSetup returns a bare-bones http.Handler for first-run setup mode.
// It exposes only /api/setup/* (gated by the bootstrap token) and the
// embedded frontend; every other /api/* path returns 503 so the UI
// knows setup is still pending.
//
// The handler has no DB, hub, or auth wired — main.go boots those only
// after setup completes.
func NewSetup(mgr *setup.Manager, frontendFS fs.FS, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	h := newSetupHandlers(mgr)

	r := chi.NewRouter()
	r.Use(corsMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(loggingMiddleware(logger))
	r.Use(bodySizeLimiter(1 << 20))

	// Always-public — the wizard needs these before it has a token.
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]any{
			"status":  "setup",
			"service": "opendray",
		})
	})
	// Status is public too — lets the client render the correct step
	// without authenticating.
	r.Get("/api/setup/status", h.status)
	// Token endpoint — loopback-only; handler enforces that.
	r.Get("/api/setup/token", h.loopbackToken)

	// Token-gated setup endpoints. Scope is strictly OpenDray's own
	// first-run config — DB choice, admin credentials, JWT. Installing
	// agent CLIs (claude/codex/gemini) is out of scope; users handle
	// their own package installs.
	r.Post("/api/setup/db/test", h.tokenGate(h.dbTest))
	r.Post("/api/setup/db/commit", h.tokenGate(h.dbCommit))
	r.Post("/api/setup/admin", h.tokenGate(h.adminSet))
	r.Post("/api/setup/jwt", h.tokenGate(h.jwtSet))
	r.Post("/api/setup/finalize", h.tokenGate(h.finalize))

	// Catch-all /api/* → 503 so callers realise setup is in progress.
	r.Route("/api", func(r chi.Router) {
		r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
			// Only fire for API paths not matched above.
			respondError(w, http.StatusServiceUnavailable, setupStatusString())
		})
	})

	// Static frontend (Flutter web). Same SPA fallback as the full server
	// so refreshes on /setup, /login, etc. land on index.html.
	if frontendFS != nil {
		fileServer := http.FileServer(http.FS(frontendFS))
		r.NotFound(func(w http.ResponseWriter, rq *http.Request) {
			if strings.HasPrefix(rq.URL.Path, "/api/") {
				respondError(w, http.StatusNotFound, "not found")
				return
			}
			if f, err := frontendFS.Open(strings.TrimPrefix(rq.URL.Path, "/")); err == nil {
				f.Close()
				fileServer.ServeHTTP(w, rq)
				return
			}
			rq.URL.Path = "/"
			fileServer.ServeHTTP(w, rq)
		})
	}

	return r
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
		"service":  "opendray",
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

	if !s.verifyCredentials(r.Context(), req.Username, req.Password) {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := s.auth.Issue(req.Username)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"token": token})
}

// verifyCredentials checks a candidate username/password against the DB
// row if one exists, otherwise against the bootstrap env credentials.
// Always does BOTH comparisons so the branch doesn't leak through timing
// whether the username or the password was wrong.
func (s *Server) verifyCredentials(ctx context.Context, username, password string) bool {
	var (
		dbUser     string
		dbHashOK   bool
		usingStore = false
	)

	if s.creds != nil {
		c, err := s.creds.Load(ctx)
		if err != nil {
			s.logger.Error("auth: load credentials", "error", err)
			return false
		}
		if c != nil {
			usingStore = true
			dbUser = c.Username
			dbHashOK = auth.VerifyPassword(c.PasswordHash, password)
		}
	}

	if usingStore {
		userOK := subtle.ConstantTimeCompare([]byte(username), []byte(dbUser)) == 1
		return userOK && dbHashOK
	}

	// Bootstrap path — env-provided credentials. Constant-time compare to
	// keep bootstrap-mode and DB-mode behaviour similar in timing.
	userOK := subtle.ConstantTimeCompare([]byte(username), []byte(s.adminUsername)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(password), []byte(s.adminPassword)) == 1
	return userOK && passOK
}

// changeCredentials updates the admin username + password after verifying
// the current password. On success writes a bcrypt hash to the DB (so env
// values stop being used) and issues a fresh token under the new username.
func (s *Server) changeCredentials(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil || s.creds == nil {
		respondError(w, http.StatusBadRequest, "credential management not enabled")
		return
	}

	var req struct {
		CurrentPassword string `json:"currentPassword"`
		NewUsername     string `json:"newUsername"`
		NewPassword     string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}

	// Identity from the JWT middleware — trust this over anything in the
	// body, so a valid token can't be used to rename someone else.
	actingUser := r.Header.Get("X-User")
	if actingUser == "" {
		respondError(w, http.StatusUnauthorized, "missing identity")
		return
	}

	if !s.verifyCredentials(r.Context(), actingUser, req.CurrentPassword) {
		respondError(w, http.StatusUnauthorized, "current password is incorrect")
		return
	}

	newUser := strings.TrimSpace(req.NewUsername)
	if newUser == "" {
		newUser = actingUser
	}
	if len(req.NewPassword) < 8 {
		respondError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	if err := s.creds.Save(r.Context(), newUser, req.NewPassword); err != nil {
		s.logger.Error("auth: save credentials", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to save credentials")
		return
	}

	token, err := s.auth.Issue(newUser)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{
		"token":    token,
		"username": newUser,
	})
}

// authStatus is a public endpoint the client hits before showing a login
// page — it reveals whether the server has JWT enabled at all, without
// exposing any config detail.
func (s *Server) authStatus(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]bool{"authRequired": s.auth != nil})
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

// ── Rate limiter (token bucket per IP, no external deps) ───────

type ipBucket struct {
	tokens int
	last   time.Time
}

type rateLimiter struct {
	mu       sync.Mutex
	buckets  sync.Map // string -> *ipBucket
	limit    int
	window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{limit: limit, window: window}
	// Background cleanup every 5 minutes to prevent unbounded growth.
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			cutoff := time.Now().Add(-2 * window)
			rl.buckets.Range(func(key, value any) bool {
				b := value.(*ipBucket)
				if b.last.Before(cutoff) {
					rl.buckets.Delete(key)
				}
				return true
			})
		}
	}()
	return rl
}

func (rl *rateLimiter) allow(ip string) bool {
	now := time.Now()
	val, _ := rl.buckets.LoadOrStore(ip, &ipBucket{tokens: rl.limit, last: now})
	b := val.(*ipBucket)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	elapsed := now.Sub(b.last)
	refill := int(elapsed * time.Duration(rl.limit) / rl.window)
	if refill > 0 {
		b.tokens += refill
		if b.tokens > rl.limit {
			b.tokens = rl.limit
		}
		b.last = now
	}
	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}

func (rl *rateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		if ip == "" {
			ip = r.RemoteAddr
		}
		if !rl.allow(ip) {
			respondError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ── Body size limiter ──────────────────────────────────────────

func bodySizeLimiter(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost, http.MethodPut, http.MethodPatch:
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}
