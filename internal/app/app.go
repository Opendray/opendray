// Package app is opendray's composition root.
//
// All subsystems (config -> store -> eventbus -> session -> gateway) are
// constructed here. Subsystem packages must not import each other through
// globals; dependencies flow only via constructor parameters wired in
// this package.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray-v2/internal/audit"
	"github.com/opendray/opendray-v2/internal/auth"
	"github.com/opendray/opendray-v2/internal/catalog"
	"github.com/opendray/opendray-v2/internal/channel"
	"github.com/opendray/opendray-v2/internal/channel/bridge" // also registers kind=bridge via init()
	"github.com/opendray/opendray-v2/internal/cliacct"
	_ "github.com/opendray/opendray-v2/internal/channel/dingtalk" // register kind=dingtalk
	_ "github.com/opendray/opendray-v2/internal/channel/discord"  // register kind=discord
	_ "github.com/opendray/opendray-v2/internal/channel/feishu"   // register kind=feishu
	_ "github.com/opendray/opendray-v2/internal/channel/slack"    // register kind=slack
	_ "github.com/opendray/opendray-v2/internal/channel/telegram" // register kind=telegram
	_ "github.com/opendray/opendray-v2/internal/channel/wechat"   // register kind=wechat (wxpusher push)
	_ "github.com/opendray/opendray-v2/internal/channel/wecom"    // register kind=wecom
	"github.com/opendray/opendray-v2/internal/config"
	"github.com/opendray/opendray-v2/internal/eventbus"
	fsapi "github.com/opendray/opendray-v2/internal/fs"
	gitapi "github.com/opendray/opendray-v2/internal/git"
	githost "github.com/opendray/opendray-v2/internal/githost"
	customtask "github.com/opendray/opendray-v2/internal/customtask"
	mcpapi "github.com/opendray/opendray-v2/internal/mcp"
	notesapi "github.com/opendray/opendray-v2/internal/notes"
	searchapi "github.com/opendray/opendray-v2/internal/search"
	"github.com/opendray/opendray-v2/internal/skills"
	vaultgit "github.com/opendray/opendray-v2/internal/vaultgit"
	"github.com/opendray/opendray-v2/internal/gateway"
	"github.com/opendray/opendray-v2/internal/integration"
	"github.com/opendray/opendray-v2/internal/session"
	"github.com/opendray/opendray-v2/internal/settings"
	"github.com/opendray/opendray-v2/internal/store"
	"github.com/opendray/opendray-v2/internal/version"
)

type App struct {
	cfg             config.Config
	log             *slog.Logger
	store           *store.Store
	bus             *eventbus.Hub
	sessions        *session.Manager
	channels        *channel.Hub
	integrations    *integration.Service
	healthCheck     *integration.HealthChecker
	audit           *audit.Sink
	intgrCallLogger *integration.CallLogger
	vaultSync       *vaultgit.Syncer
	server          *http.Server
}

// New wires the runtime dependencies but does not start any goroutines.
// Caller is responsible for calling Run or Close.
func New(ctx context.Context, cfg config.Config) (*App, error) {
	log, logRing, err := newLogger(cfg.Log)
	if err != nil {
		return nil, err
	}
	st, err := store.Open(ctx, cfg.Database.URL)
	if err != nil {
		return nil, err
	}

	bus := eventbus.New(log)

	authSvc := auth.New(cfg.Admin, bus, log)
	authHandlers := auth.NewHandlers(authSvc, log)

	cat, err := catalog.New(st.Pool(), log)
	if err != nil {
		st.Close()
		return nil, err
	}
	if err := cat.Sync(ctx); err != nil {
		st.Close()
		return nil, err
	}
	catalogHandlers := catalog.NewHandlers(cat, log)

	var cliacctOpts []cliacct.Option
	if d := strings.TrimSpace(cfg.Providers.Claude.AccountsDir); d != "" {
		cliacctOpts = append(cliacctOpts, cliacct.WithAccountsDir(expandPath(d)))
	}
	cliacctSvc := cliacct.NewService(st.Pool(), bus, log, cliacctOpts...)
	cliacctHandlers := cliacct.NewHandlers(cliacctSvc, log)

	// Vault + skills are needed by the SessionProvider so spawn-time
	// injection has them available. Constructed here (before the
	// session manager) so the manager's first Resolve call sees them.
	notesRoot, skillsRoot, gitRoot := resolveVaultPaths(cfg.Vault)
	vault, err := notesapi.New(notesRoot, notesapi.Options{
		PersonalPrefix: cfg.Vault.PersonalPrefix,
		ProjectsPrefix: cfg.Vault.ProjectsPrefix,
	})
	if err != nil {
		st.Close()
		return nil, fmt.Errorf("init notes vault: %w", err)
	}
	log.Info("notes vault ready", "root", vault.Root())
	notesHandlers := notesapi.NewHandlers(vault, log)
	skillsLoader := skills.NewLoader(skillsRoot)
	if list, _ := skillsLoader.List(); len(list) > 0 {
		log.Info("agent skills loaded", "count", len(list),
			"vault", skillsLoader.VaultRoot())
	}

	mcpRoot, secretsFile := resolveMCPPaths(cfg.MCP, notesRoot, skillsRoot)
	mcpLoader := mcpapi.NewLoader(mcpRoot)
	if list, _ := mcpLoader.List(); len(list) > 0 {
		log.Info("mcp registry loaded", "count", len(list),
			"vault", mcpLoader.VaultRoot(), "secrets", secretsFile)
	}

	var sessionOpts []session.ManagerOption
	if d := cfg.Session.Threshold(); d > 0 {
		sessionOpts = append(sessionOpts, session.WithIdleThreshold(d))
	}
	if d := cfg.Session.Interval(); d > 0 {
		sessionOpts = append(sessionOpts, session.WithIdleInterval(d))
	}
	sessionOpts = append(sessionOpts,
		session.WithClaudeHistoryConfig(resolveClaudeHistoryConfig(cfg.Providers.Claude)),
		session.WithCodexHistoryConfig(resolveCodexHistoryConfig(cfg.Providers.Codex)),
		session.WithGeminiHistoryConfig(resolveGeminiHistoryConfig(cfg.Providers.Gemini)),
	)
	sessionMgr := session.NewManager(
		st.Pool(),
		bus,
		catalog.NewSessionProvider(cat, cliacctSvc, skillsLoader, mcpLoader, secretsFile, log),
		log,
		sessionOpts...,
	)
	// Best-effort reconcile of leftover rows from a prior gateway
	// process — their PTYs are gone, so flip them to 'ended' so the
	// web UI can stop the WS reconnect loop and show EndedSessionView.
	if err := sessionMgr.ReconcileStartup(ctx); err != nil {
		log.Warn("session reconcile on startup failed", "err", err)
	}
	sessionHandlers := session.NewHandlers(sessionMgr, log)

	channelHub := channel.NewHub(st.Pool(), bus, log)
	// Plain-text inbound from a channel (e.g. a Telegram reply that
	// is not a slash command) gets forwarded to the last session that
	// notified that channel — letting the operator drive a running
	// CLI from chat without opening the web UI.
	channelHub.SetSessionInput(sessionMgr)
	channelHandlers := channel.NewHandlers(channelHub, log)
	bridgeHandlers := bridge.NewHandlers(bridge.DefaultBroker(), log)

	intgrSvc := integration.NewService(st.Pool(), bus, log)
	intgrHandlers := integration.NewHandlers(intgrSvc, log)
	intgrCallLogger := integration.NewCallLogger(st.Pool(), log)
	intgrCallLogHandlers := integration.NewCallLogHandlers(intgrCallLogger, log)
	proxyHandlers := integration.NewProxyHandlers(intgrSvc, intgrCallLogger, log)
	eventsHandler := integration.NewEventsHandler(bus, log)
	healthCheck := integration.NewHealthChecker(intgrSvc, bus, log)

	auditSink := audit.NewSink(st.Pool(), bus, log)
	auditSvc := audit.NewService(st.Pool())
	auditHandlers := audit.NewHandlers(auditSvc, log)
	fsHandlers := fsapi.NewHandlers(log)
	gitHandlers := gitapi.NewHandlers(log)
	gitHostSvc := githost.NewService(st.Pool(), log)
	gitHostHandlers := githost.NewHandlers(gitHostSvc, log)
	customTaskSvc := customtask.NewService(st.Pool(), log)
	customTaskHandlers := customtask.NewHandlers(customTaskSvc, log)
	searchHandlers := searchapi.NewHandlers(log)
	skillsHandlers := skills.NewHandlers(skillsLoader, log)
	mcpHandlers := mcpapi.NewHandlers(mcpLoader, secretsFile, log)
	// Vault git ops are scoped to whatever the user told us is the
	// repo root (defaults: notes root if user pinned `notes` directly,
	// otherwise the parent that contains both notes/ and skills/).
	// The githost service is passed so private-repo HTTPS push/pull
	// picks up tokens stored in Plugins → Git hosts.
	vaultGitHandlers, err := vaultgit.NewHandlers(gitRoot, gitHostSvc, log)
	if err != nil {
		st.Close()
		return nil, fmt.Errorf("init vault git handlers: %w", err)
	}
	log.Info("vault git ready", "root", gitRoot)
	vaultSyncer := vaultgit.NewSyncer(st.Pool(), bus, vaultGitHandlers, log)

	// Settings: read/write the same config.toml the gateway booted
	// from. Empty FilePath (env-only mode) disables the API — Get
	// returns "no config path" so the UI shows a read-only banner.
	settingsSvc := settings.NewService(cfg.FilePath, log)
	settingsHandlers := settings.NewHandler(settingsSvc, logRing, log)

	gw := gateway.NewServer(gateway.Deps{
		Logger:    log,
		DB:        st,
		Version:   version.Current(),
		StartedAt: time.Now(),
		V1Routes: func(r chi.Router) {
			// Public: only login + bridge adapter WS endpoint
			// (token-authenticated via the register frame) +
			// per-channel webhook routes (feishu/dingtalk/wecom
			// receive events from the platform; channel impls verify
			// the request themselves).
			authHandlers.MountPublic(r)
			bridgeHandlers.Mount(r)
			channelHandlers.MountPublic(r)

			// Admin-only: integration CRUD + reverse proxy.
			r.Group(func(r chi.Router) {
				r.Use(authSvc.Middleware)
				intgrHandlers.MountAdmin(r)
				proxyHandlers.Mount(r)
				fsHandlers.Mount(r)
				gitHandlers.Mount(r)
				gitHostHandlers.Mount(r)
				customTaskHandlers.Mount(r)
				searchHandlers.Mount(r)
				notesHandlers.Mount(r)
				skillsHandlers.Mount(r)
				mcpHandlers.Mount(r)
				vaultGitHandlers.Mount(r)
				vaultSyncer.Mount(r)
				auditHandlers.Mount(r)
				intgrCallLogHandlers.Mount(r)
				settingsHandlers.Mount(r)
			})

			// Dual-auth (admin OR integration API key): all business
			// endpoints. ADR 0006 §1 + ADR 0009 (events WS extended
			// to admin so the web Activity viewer rides the same
			// admin token). The integration call logger middleware
			// runs after auth so it can attribute requests to the
			// integration principal (admin requests are skipped
			// inside the middleware).
			r.Group(func(r chi.Router) {
				r.Use(integration.CombinedMiddleware(authSvc, intgrSvc))
				r.Use(intgrCallLogger.Middleware)
				authHandlers.MountProtected(r)
				sessionHandlers.Mount(r)
				catalogHandlers.Mount(r)
				cliacctHandlers.Mount(r)
				channelHandlers.Mount(r)
				r.Get("/integrations/_events", eventsHandler.Serve)
			})
		},
	})

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           gw.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	return &App{
		cfg:             cfg,
		log:             log,
		store:           st,
		bus:             bus,
		sessions:        sessionMgr,
		channels:        channelHub,
		integrations:    intgrSvc,
		healthCheck:     healthCheck,
		audit:           auditSink,
		intgrCallLogger: intgrCallLogger,
		vaultSync:       vaultSyncer,
		server:          srv,
	}, nil
}

// Migrate applies pending DB migrations and returns. Used by `opendray migrate`.
func (a *App) Migrate(ctx context.Context) error {
	return a.store.Migrate(ctx, a.log)
}

// Run starts the HTTP server, channel hub, and audit sink, then blocks
// until ctx is cancelled. Graceful shutdown order:
//
//	HTTP server -> session manager -> channel hub -> audit sink -> event bus -> store
func (a *App) Run(ctx context.Context) error {
	a.log.Info("opendray starting",
		"listen", a.cfg.Listen,
		"version", version.Version,
		"commit", version.Commit)

	if err := a.channels.Start(ctx); err != nil {
		a.log.Error("channel hub start", "err", err)
	}

	healthDone := make(chan struct{})
	go func() {
		a.healthCheck.Run(ctx)
		close(healthDone)
	}()

	auditDone := make(chan struct{})
	go func() {
		a.audit.Run(ctx)
		close(auditDone)
	}()

	vaultSyncDone := make(chan struct{})
	go func() {
		a.vaultSync.Run(ctx)
		close(vaultSyncDone)
	}()

	errCh := make(chan error, 1)
	go func() {
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http server: %w", err)
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		a.log.Info("shutdown signal received")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		a.log.Error("http shutdown", "err", err)
	}
	if err := a.sessions.Shutdown(shutdownCtx); err != nil {
		a.log.Error("session shutdown", "err", err)
	}
	if err := a.channels.Shutdown(shutdownCtx); err != nil {
		a.log.Error("channel shutdown", "err", err)
	}
	// Drain the call log queue after the server stops accepting new
	// requests. Bounded by the queue size + per-row write timeout
	// (5s), so this returns quickly.
	a.intgrCallLogger.Close()

	select {
	case <-healthDone:
	case <-time.After(2 * time.Second):
		a.log.Warn("health checker shutdown timed out")
	}

	select {
	case <-auditDone:
	case <-time.After(5 * time.Second):
		a.log.Warn("audit shutdown timed out")
	}

	select {
	case <-vaultSyncDone:
	case <-time.After(5 * time.Second):
		a.log.Warn("vault auto-sync shutdown timed out")
	}

	a.bus.Close()
	a.store.Close()
	a.log.Info("opendray stopped")
	return nil
}

func (a *App) Logger() *slog.Logger { return a.log }

// Close releases resources without waiting on the HTTP server. Use Run for
// the normal lifecycle; Close is for failure paths after New succeeded.
func (a *App) Close() {
	if a.sessions != nil {
		_ = a.sessions.Shutdown(context.Background())
	}
	if a.channels != nil {
		_ = a.channels.Shutdown(context.Background())
	}
	if a.bus != nil {
		a.bus.Close()
	}
	if a.store != nil {
		a.store.Close()
	}
}

// parentOf returns the parent directory of an absolute path. Used to
// derive the vault base from <vault>/notes — skills live next to it
// at <vault>/skills, so both share one git-able root.
func parentOf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[:i]
		}
	}
	return p
}

// resolveVaultPaths derives the three vault-related paths (notes root,
// skills root, git working tree) from VaultConfig. Each can be set
// explicitly; otherwise we fall back to the legacy layout under the
// shared root. Returns absolute paths suitable for filesystem ops.
//
// The defaults preserve opendray's original `<root>/notes` +
// `<root>/skills` layout. Users coming in with an existing Obsidian
// vault can pin `vault.notes = "~/Documents/MyVault"` (or similar)
// and opendray's notes API will read straight from that directory.
func resolveVaultPaths(c config.VaultConfig) (notes, skills, git string) {
	root := c.Root
	if root == "" {
		root = "~/.opendray/vault"
	}
	root = expandPath(root)

	notes = c.Notes
	if notes == "" {
		notes = filepath.Join(root, "notes")
	} else {
		notes = expandPath(notes)
	}

	skills = c.Skills
	if skills == "" {
		skills = filepath.Join(root, "skills")
	} else {
		skills = expandPath(skills)
	}

	git = c.GitRoot
	if git == "" {
		// Pick the most natural git working tree: if the user pinned a
		// custom notes path, that IS their vault repo (Obsidian-style);
		// otherwise the legacy combined root holds both notes + skills
		// under one repo.
		if c.Notes != "" {
			git = notes
		} else {
			git = root
		}
	} else {
		git = expandPath(git)
	}
	return notes, skills, git
}

// resolveMCPPaths picks the registry root + secrets file with the
// same precedence story as the vault paths above. Defaults:
//
//	root         = <vault root>/mcp        (next to notes/, skills/)
//	secrets_file = ~/.opendray/secrets.env (intentionally OUTSIDE the
//	               vault so a `git add .` in vault never picks it up)
//
// notesRoot / skillsRoot are passed only to derive `<vault root>` —
// we use parentOf(notes) which is the same dir all the other vault
// children live under in the default layout.
func resolveMCPPaths(c config.MCPConfig, notesRoot, skillsRoot string) (root, secrets string) {
	root = c.Root
	if root == "" {
		// Pick the same parent the vault siblings share. notes/ and
		// skills/ are always under <vault root>; using parentOf(notes)
		// works regardless of which the user pinned explicitly.
		base := parentOf(notesRoot)
		if base == "" || base == "/" {
			base = parentOf(skillsRoot)
		}
		if base == "" || base == "/" {
			base = expandPath("~/.opendray/vault")
		}
		root = filepath.Join(base, "mcp")
	} else {
		root = expandPath(root)
	}

	secrets = c.SecretsFile
	if secrets == "" {
		secrets = expandPath("~/.opendray/secrets.env")
	} else {
		secrets = expandPath(secrets)
	}
	return root, secrets
}

// resolveClaudeHistoryConfig translates the operator's
// [providers.claude] TOML section into a session-package config,
// expanding ~/ in any path. Empty fields stay empty so the
// session package falls back to its built-in HOME defaults.
func resolveClaudeHistoryConfig(c config.ClaudeProviderConfig) session.ClaudeHistoryConfig {
	out := session.ClaudeHistoryConfig{}
	if len(c.HistoryRoots) > 0 {
		out.HistoryRoots = make([]string, 0, len(c.HistoryRoots))
		for _, r := range c.HistoryRoots {
			if r = strings.TrimSpace(r); r != "" {
				out.HistoryRoots = append(out.HistoryRoots, expandPath(r))
			}
		}
	}
	if c.AccountsDir != "" {
		out.AccountsDir = expandPath(c.AccountsDir)
	}
	return out
}

// resolveCodexHistoryConfig expands ~/ in [providers.codex].
func resolveCodexHistoryConfig(c config.CodexProviderConfig) session.CodexHistoryConfig {
	out := session.CodexHistoryConfig{}
	if c.SessionsRoot != "" {
		out.SessionsRoot = expandPath(c.SessionsRoot)
	}
	return out
}

// resolveGeminiHistoryConfig expands ~/ in [providers.gemini].
func resolveGeminiHistoryConfig(c config.GeminiProviderConfig) session.GeminiHistoryConfig {
	out := session.GeminiHistoryConfig{}
	if c.TmpRoot != "" {
		out.TmpRoot = expandPath(c.TmpRoot)
	}
	if c.ProjectsFile != "" {
		out.ProjectsFile = expandPath(c.ProjectsFile)
	}
	return out
}

// expandPath resolves ~/ prefixes against the calling user's home
// dir, then makes the path absolute. Mirrors what notes.expand does
// internally — kept here so app-level path resolution doesn't reach
// into the notes package's privates.
func expandPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return p
	}
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			if p == "~" {
				p = home
			} else {
				p = filepath.Join(home, p[2:])
			}
		}
	}
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}
