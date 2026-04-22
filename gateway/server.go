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
	pgpkg "github.com/opendray/opendray/gateway/pg"
	"github.com/opendray/opendray/gateway/tasks"

	"github.com/opendray/opendray/gateway/telegram"
	"github.com/opendray/opendray/kernel/auth"
	"github.com/opendray/opendray/kernel/hub"
	"github.com/opendray/opendray/plugin"
	"github.com/opendray/opendray/plugin/bridge"
	"github.com/opendray/opendray/plugin/contributions"
	"github.com/opendray/opendray/plugin/host"
	"github.com/opendray/opendray/plugin/install"
	"github.com/opendray/opendray/plugin/market"
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
	pg            *pgpkg.Manager
	installer     *install.Installer
	contribReg    *contributions.Registry
	cmdInvoker    commandInvoker

	// Bridge (M2 T7): plugin WebSocket API.
	bridgeMgr       *bridge.Manager
	bridgeCfg       PluginsConfig
	bridgeNamespace *namespaceRegistry
	// bridgePluginsOverride lets tests inject a plugin-lookup function without
	// standing up a real Runtime with filesystem manifests. Production leaves
	// this nil and the handler falls back to s.plugins.Get.
	bridgePluginsOverride func(name string) (plugin.Provider, bool)

	// consentStoreOverride / consentBridgeOverride let tests inject fakes for
	// the T12 revoke endpoints without wiring embedded-pg or a real bridge
	// Manager. Production leaves both nil and the handlers resolve via
	// s.hub.DB() and s.bridgeMgr.
	consentStoreOverride  consentStore
	consentBridgeOverride consentInvalidator

	// T14 — SSE workbench stream.
	// workbenchBus is the fan-out channel for host → Flutter out-of-band
	// events (showMessage, openView, updateStatusBar, contributionsChanged).
	// Nil disables the stream endpoint (returns 503 EBUS).
	workbenchBus *WorkbenchBus

	// heartbeatInterval overrides the 20 s production heartbeat in tests.
	// Zero means use defaultHeartbeatInterval (20 s).
	heartbeatInterval time.Duration

	// Build-time identity surfaced by /api/health so the Flutter About
	// page can render a backend version. Empty strings degrade to "dev"
	// / "unknown" in the response so the UI never crashes on missing
	// fields. buildTime (UTC ISO8601 basic: 20060102T150405Z) changes on
	// every build even when version+SHA don't — use it to tell two
	// binaries from the same commit apart (e.g. "did my deploy actually
	// recompile?").
	version   string
	buildSha  string
	buildTime string

	// marketplace is the loaded catalog that backs
	// GET /api/marketplace/plugins and the marketplace:// install
	// source. Nil when no catalog dir is configured — endpoints then
	// degrade to empty lists and EBADSRC respectively. The concrete
	// implementation is market/local in M3 and market/remote once
	// M4.1 lands; the handler code only depends on the interface.
	marketplace market.Catalog

	// marketplaceSettings carries the boot-time config values
	// surfaced via GET /api/marketplace/settings so the Settings →
	// Marketplace admin subpage can show what the server is
	// actually using. Read-only in M4.1; editable via per-user
	// preferences is M4.2.
	marketplaceSettings MarketplaceSettings

	// secretAPI + hostSupervisor back the platform-managed config
	// endpoints (GET/PUT /api/plugins/{name}/config). secretAPI is
	// used for PlatformSet/Get/Delete bypassing the bridge gate;
	// hostSupervisor.Kill restarts the sidecar after a config write
	// so the new values take effect on the next invoke.
	secretAPI      *bridge.SecretAPI
	hostSupervisor *host.Supervisor

	// Test-only overrides for the config handlers. When non-nil they
	// short-circuit the resolver chain so tests can inject fakes for
	// the KV store, secret store, and sidecar killer without booting
	// embedded Postgres or a real supervisor. Production leaves
	// these nil.
	configKVTestOverride      configStore
	configSecretsTestOverride platformSecrets
	configKillerTestOverride  sidecarKiller
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

	// M2 T7 — plugin bridge WS.
	BridgeManager *bridge.Manager // shared bridge.Manager instance; nil disables
	Plugins2      PluginsConfig   // plugin-bridge-tunable knobs

	// T14 — SSE workbench stream. Nil disables the endpoint (returns 503 EBUS).
	WorkbenchBus *WorkbenchBus

	// Version / BuildSha / BuildTime are the build-stamped identifiers
	// injected into cmd/opendray/main.go via -ldflags. They flow through
	// /api/health so the Flutter About screen can show the running
	// backend's version + distinguish two binaries built from the same
	// commit (BuildTime changes on every `make release-linux`).
	// All three are optional; defaults kick in when empty.
	Version   string
	BuildSha  string
	BuildTime string

	// Marketplace is the preloaded plugin catalog. Nil disables the
	// GET /api/marketplace/plugins endpoint (returns empty list) and
	// rejects marketplace:// install sources with EBADSRC. Accepts
	// any market.Catalog implementation; main.go wires local during
	// M3 bootstrap, remote once M4.1 T1–T7 lands.
	Marketplace market.Catalog

	// MarketplaceSettings carries the read-only config snapshot
	// returned by GET /api/marketplace/settings. Kev's Settings →
	// Marketplace admin subpage reads this to show which URL +
	// mirrors + poll cadence the gateway actually booted with.
	MarketplaceSettings MarketplaceSettings

	// SecretAPI + HostSupervisor wire the platform-managed config
	// endpoints. Both nil = /api/plugins/{name}/config returns 503.
	SecretAPI      *bridge.SecretAPI
	HostSupervisor *host.Supervisor
}

// PluginsConfig holds runtime-tunable knobs for the bridge handler.
type PluginsConfig struct {
	// BridgeRatePerMinute caps inbound requests per plugin connection (default 60).
	BridgeRatePerMinute int
	// BridgeReadTimeout is the idle read deadline on each bridge WS (default 60s).
	BridgeReadTimeout time.Duration
	// FrontendOrigin, when set, is an additional Origin allowed by the bridge
	// handshake (for production deployments where the UI is served from a
	// different host than the gateway).
	FrontendOrigin string
}

// New creates a gateway server with all routes configured.
func New(cfg Config) *Server {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	// Fill in bridge config defaults.
	bridgeCfg := cfg.Plugins2
	if bridgeCfg.BridgeRatePerMinute <= 0 {
		bridgeCfg.BridgeRatePerMinute = 60
	}
	if bridgeCfg.BridgeReadTimeout <= 0 {
		bridgeCfg.BridgeReadTimeout = 60 * time.Second
	}

	s := &Server{
		hub:             cfg.Hub,
		plugins:         cfg.Plugins,
		auth:            cfg.Auth,
		creds:           cfg.Credentials,
		adminUsername:   cfg.AdminUsername,
		adminPassword:   cfg.AdminPassword,
		logger:          cfg.Logger,
		tasks:           tasks.NewRunner(),
		git:             gitpkg.NewManager(),
		pg:              pgpkg.NewManager(),
		installer:       cfg.Installer,
		contribReg:      cfg.Contributions,
		cmdInvoker:      cfg.CommandInvoker,
		bridgeMgr:       cfg.BridgeManager,
		bridgeCfg:       bridgeCfg,
		bridgeNamespace: newNamespaceRegistry(),
		workbenchBus:    cfg.WorkbenchBus,
		version:         cfg.Version,
		buildSha:        cfg.BuildSha,
		buildTime:       cfg.BuildTime,
		marketplace:         cfg.Marketplace,
		marketplaceSettings: cfg.MarketplaceSettings,
		secretAPI:       cfg.SecretAPI,
		hostSupervisor:  cfg.HostSupervisor,
	}
	if cfg.MCP != nil {
		s.mcp = mcp.NewHandlers(cfg.MCP)
	}
	// Telegram bridge — watches the "telegram" plugin and starts/stops
	// the bot to match. Safe to construct even if the plugin is disabled.
	// Install the config resolver so the reconcile loop sees values
	// the user wrote through the v1 Configure form (plugin_kv.__config.*).
	s.telegram = telegram.NewManager(cfg.Plugins, cfg.Hub, cfg.Plugins.HookBus(), cfg.Logger)
	s.telegram.SetConfigResolver(s.effectiveConfig)
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

		// Source Control panel — read-only Git surface. Covers repo
		// auto-discovery across allowedRoots, user bookmarks, multi-file
		// diff, DB-backed per-session baselines, and PR review against
		// Gitea / GitHub / GitLab forge instances. Write paths
		// (stage/commit/push/merge/approve/comment) flow through the
		// Claude session, not here.
		r.Get("/api/source-control/{plugin}/repos",         s.scRepos)
		r.Post("/api/source-control/{plugin}/bookmarks",    s.scBookmarksAdd)
		r.Delete("/api/source-control/{plugin}/bookmarks",  s.scBookmarksRemove)
		r.Get("/api/source-control/{plugin}/status",        s.scStatus)
		r.Get("/api/source-control/{plugin}/log",           s.scLog)
		r.Get("/api/source-control/{plugin}/branches",      s.scBranches)
		r.Get("/api/source-control/{plugin}/diff",          s.scDiff)
		r.Post("/api/source-control/{plugin}/baseline",     s.scBaselinePut)
		r.Get("/api/source-control/{plugin}/baseline",      s.scBaselineGet)
		r.Delete("/api/source-control/{plugin}/baseline",   s.scBaselineDelete)

		// Forge instances (Phase 2.A). One source-control install may
		// track many forges; tokens are held in plugin_secret per id.
		r.Get("/api/source-control/{plugin}/forges",            s.scForgesList)
		r.Post("/api/source-control/{plugin}/forges",           s.scForgesCreate)
		r.Put("/api/source-control/{plugin}/forges/{id}",       s.scForgesUpdate)
		r.Delete("/api/source-control/{plugin}/forges/{id}",    s.scForgesDelete)
		r.Get("/api/source-control/{plugin}/forges/{id}/repos", s.scForgesRepos)

		// PR routes (Phase 2.C). `?repo=owner/name` picks the repo per
		// request, so one forge instance covers any number of repos —
		// no more "one repo locked in config" as the old git-forge had.
		r.Get("/api/source-control/{plugin}/forges/{id}/pulls",                          s.scPullsList)
		r.Get("/api/source-control/{plugin}/forges/{id}/pulls/{number}",                 s.scPullDetail)
		r.Get("/api/source-control/{plugin}/forges/{id}/pulls/{number}/diff",            s.scPullDiff)
		r.Get("/api/source-control/{plugin}/forges/{id}/pulls/{number}/comments",        s.scPullComments)
		r.Get("/api/source-control/{plugin}/forges/{id}/pulls/{number}/reviews",         s.scPullReviews)
		r.Get("/api/source-control/{plugin}/forges/{id}/pulls/{number}/review-comments", s.scPullReviewComments)
		r.Get("/api/source-control/{plugin}/forges/{id}/pulls/{number}/checks",          s.scPullChecks)

		// pg-browser panel — SQL editor + schema browser backed by pgx.
		// Read-only is enforced server-side via BEGIN READ ONLY + verb
		// guard; statement timeout + max rows gated from configSchema.
		// Write statements go through the separate /execute route so
		// the Flutter client can prompt for confirmation first.
		r.Post("/api/pg/{plugin}/query",     s.pgQuery)
		r.Post("/api/pg/{plugin}/execute",   s.pgExecute)
		r.Get("/api/pg/{plugin}/databases",  s.pgDatabases)
		r.Get("/api/pg/{plugin}/schemas",    s.pgSchemas)
		r.Get("/api/pg/{plugin}/tables",     s.pgTables)
		r.Get("/api/pg/{plugin}/columns",    s.pgColumns)


		// Hook subscriptions
		r.Get("/api/hooks", s.listHooks)

		// Plugin install / uninstall / audit (T7).
		// DELETE /api/providers/{name} stays for legacy compat (see api.go).
		// These new routes provide the full install lifecycle via Installer.
		r.Post("/api/plugins/install", s.pluginsInstall)
		r.Post("/api/plugins/install/confirm", s.pluginsInstallConfirm)
		r.Delete("/api/plugins/{name}", s.pluginsUninstall)
		r.Get("/api/plugins/{name}/audit", s.pluginsAudit)

		// Built-in plugin restore — undo an Uninstall on a bundled
		// plugin. Clears the tombstone + re-seeds from embed.FS.
		// Drives the Settings → Built-in Plugins page in the app.
		r.Get("/api/plugins/builtins", s.pluginsBuiltinsList)
		r.Post("/api/plugins/builtins/{name}/restore", s.pluginsBuiltinRestore)

		// Marketplace catalog — lists installable plugins for the Hub
		// page. Install still flows through /api/plugins/install with
		// src="marketplace://<name>".
		r.Get("/api/marketplace/plugins", s.marketplaceList)
		r.Get("/api/marketplace/settings", s.marketplaceSettingsGet)
		r.Post("/api/marketplace/refresh", s.marketplaceRefresh)

		// Consent management (T12) — read current perms + hot-revoke.
		// DELETE /consents/{cap} fires bridgeMgr.InvalidateConsent synchronously
		// so in-flight WS subs terminate within the 200 ms SLO.
		r.Get("/api/plugins/{name}/consents", s.pluginsConsentsGet)
		r.Patch("/api/plugins/{name}/consents", s.pluginsConsentsPatch)
		r.Delete("/api/plugins/{name}/consents/{cap}", s.pluginsConsentsRevokeCap)
		r.Delete("/api/plugins/{name}/consents", s.pluginsConsentsRevokeAll)

		// User-editable config (configSchema-driven form). GET returns
		// schema + masked values; PUT writes to plugin_kv + plugin_secret
		// and restarts the sidecar.
		r.Get("/api/plugins/{name}/config", s.pluginsConfigGet)
		r.Put("/api/plugins/{name}/config", s.pluginsConfigPut)

		// Plugin asset server — serves plugin ui/ bundles (T8).
		r.Get("/api/plugins/{name}/assets/*", s.pluginsAssets)

		// Plugin bridge WebSocket (T7) — per-plugin JSON envelope protocol.
		r.Get("/api/plugins/{name}/bridge/ws", s.pluginsBridgeWS)

		// Workbench contributions — flat view of all installed plugin contribution
		// points (commands, statusBar, keybindings, menus). Pure read; no DB (T9).
		r.Get("/api/workbench/contributions", s.workbenchContributions)

		// Workbench SSE stream — host → Flutter out-of-band events (T14).
		// Streams showMessage, openView, updateStatusBar, contributionsChanged.
		r.Get("/api/workbench/stream", s.workbenchStream)

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


// ── Terminal HTML (xterm.js for web) ────────────────────────────

func (s *Server) serveTerminalHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(terminalHTML))
}

// ── Health ──────────────────────────────────────────────────────

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	version := s.version
	if version == "" {
		version = "dev"
	}
	buildSha := s.buildSha
	if buildSha == "" {
		buildSha = "unknown"
	}
	buildTime := s.buildTime
	if buildTime == "" {
		buildTime = "unknown"
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"service":   "opendray",
		"version":   version,
		"buildSha":  buildSha,
		"buildTime": buildTime,
		"sessions":  s.hub.RunningCount(),
		"plugins":   len(s.plugins.List()),
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

	// M5 D3 — changing the password rotates the KEK. The rewrap walk of
	// every plugin_secret_kek row must happen in the SAME tx as the
	// admin_auth update, otherwise a crash between Save() and the
	// rewrap would strand every wrapped DEK with no recoverable key.
	if err := s.creds.RotateCredentialsAndKEK(r.Context(), newUser, req.NewPassword); err != nil {
		s.logger.Error("auth: rotate credentials+KEK", "error", err)
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
