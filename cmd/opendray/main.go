package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/opendray/opendray/app"
	"github.com/opendray/opendray/gateway"
	"github.com/opendray/opendray/gateway/mcp"
	"github.com/opendray/opendray/kernel/auth"
	"github.com/opendray/opendray/kernel/config"
	"github.com/opendray/opendray/kernel/hub"
	opg "github.com/opendray/opendray/kernel/pg"
	"github.com/opendray/opendray/kernel/setup"
	"github.com/opendray/opendray/kernel/store"
	"github.com/opendray/opendray/plugin"
	"github.com/opendray/opendray/plugin/bridge"
	"github.com/opendray/opendray/plugin/commands"
	"github.com/opendray/opendray/plugin/compat"
	"github.com/opendray/opendray/plugin/contributions"
	"github.com/opendray/opendray/plugin/host"
	"github.com/opendray/opendray/plugin/install"
	"github.com/opendray/opendray/plugin/marketplace"

	"github.com/jackc/pgx/v5"
)

// version is injected at build time via -ldflags. Defaults to "dev" for
// `go run` without a release.
var (
	version  = "dev"
	buildSha = "unknown"
)

// printHelp is the output of `opendray help` / `-h`. Keep it short —
// full docs live in the README.
func printHelp() {
	fmt.Println(`OpenDray — pilot AI coding agents from your phone.

Usage:
  opendray [command]

Commands:
  (no args)   Start the server (first run triggers the setup wizard)
  setup       Interactive CLI wizard for headless / no-browser installs
  version     Print version info
  help        Show this help

Env vars:
  OPENDRAY_CONFIG        path to config.toml
  OPENDRAY_NO_BROWSER    don't auto-open the browser in first-run setup

Docs: https://github.com/Opendray/opendray`)
}

func main() {
	// Subcommand dispatch. Only one subcommand today — `opendray setup`
	// runs the interactive CLI wizard for headless installs where no
	// browser is available. Everything else falls through to normal
	// boot + setup-mode-on-demand.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "plugin":
			os.Exit(runPluginCLI(os.Args[2:]))
		case "setup":
			os.Exit(runSetupCLI())
		case "version", "-v", "--version":
			fmt.Println(version)
			return
		case "-h", "--help", "help":
			printHelp()
			return
		}
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// Outer loop: runs setup mode on a fresh install, then transitions
	// into normal mode once /api/setup/finalize writes config.toml. On a
	// fully-configured install, setup mode is skipped entirely.
	for {
		cfg, source, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "FATAL: %v\n", err)
			os.Exit(1)
		}
		logger.Info("config loaded", "source", source, "db_mode", cfg.DB.Mode,
			"complete", cfg.IsComplete())

		if !cfg.IsComplete() {
			if !runSetupMode(logger, cfg) {
				// Setup was interrupted (SIGTERM before finalize) — exit.
				return
			}
			// Setup finished — loop around to load the freshly-written
			// config and boot normally.
			continue
		}

		if err := cfg.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "FATAL: %v\n", err)
			os.Exit(1)
		}
		runNormalMode(logger, cfg)
		return
	}
}

// runSetupMode boots the minimal setup-only gateway until finalize is
// called or SIGTERM arrives. Returns true when finalize succeeded (so
// main can loop into normal mode), false on signal-driven shutdown.
func runSetupMode(logger *slog.Logger, cfg config.Config) bool {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mgr, err := setup.New(func() {
		// onFinish: /api/setup/finalize has written config. Cancel the
		// listen loop so the main loop reloads config and enters normal
		// mode. We deliberately delay a beat so the 200 reply flushes.
		go func() {
			time.Sleep(250 * time.Millisecond)
			cancel()
		}()
	})
	if err != nil {
		logger.Error("setup: init manager", "error", err)
		os.Exit(1)
	}

	// Persist the token so the Flutter wizard can fetch it over same-origin
	// when the user opens /setup without the ?token= query. Mode 0600 —
	// it's a short-lived shared secret.
	tokenPath, err := writeBootstrapToken(mgr.BootstrapToken())
	if err != nil {
		logger.Warn("setup: could not persist token file; wizard needs ?token= in URL", "error", err)
	}

	listen := cfg.Server.ListenAddr
	if listen == "" {
		listen = "127.0.0.1:8640"
	}
	setupURL := fmt.Sprintf("http://%s/setup?token=%s",
		displayAddr(listen), mgr.BootstrapToken())
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "╭───────────────────────────────────────────────────────────────╮")
	fmt.Fprintln(os.Stderr, "│                                                               │")
	fmt.Fprintln(os.Stderr, "│   🚀  OpenDray — first-run setup                              │")
	fmt.Fprintln(os.Stderr, "│                                                               │")
	fmt.Fprintln(os.Stderr, "│   Your browser should open automatically. If not, visit:      │")
	fmt.Fprintln(os.Stderr, "│                                                               │")
	fmt.Fprintf (os.Stderr, "│   %s\n", padRight(setupURL, 60))
	fmt.Fprintln(os.Stderr, "│                                                               │")
	fmt.Fprintln(os.Stderr, "│   Headless server (no browser)?                               │")
	fmt.Fprintln(os.Stderr, "│   Stop this (Ctrl-C) and run: opendray setup                  │")
	if tokenPath != "" {
		fmt.Fprintln(os.Stderr, "│                                                               │")
		fmt.Fprintf (os.Stderr, "│   Token: %s\n", padRight(tokenPath, 55))
	}
	fmt.Fprintln(os.Stderr, "│                                                               │")
	fmt.Fprintln(os.Stderr, "╰───────────────────────────────────────────────────────────────╯")
	fmt.Fprintln(os.Stderr, "")

	// Fire-and-forget browser launch. Suppressed when OPENDRAY_NO_BROWSER
	// is set (for CI / headless server deploys).
	if os.Getenv("OPENDRAY_NO_BROWSER") == "" {
		// Give the server ~250ms to start listening before we open the
		// URL, otherwise the browser races the TCP listener on slow boxes.
		go func() {
			time.Sleep(250 * time.Millisecond)
			_ = openBrowser(setupURL)
		}()
	}

	// Frontend FS — setup mode still serves the Flutter web wizard from
	// the embedded dist. If the binary was built without dist (dev mode),
	// a bare HTML stub would be nicer; left for Phase 4.
	var frontendFS fs.FS
	if distFS, err := fs.Sub(app.DistFS, "build/web"); err == nil {
		if entries, err := fs.ReadDir(distFS, "."); err == nil && len(entries) > 0 {
			frontendFS = distFS
		}
	}

	handler := gateway.NewSetup(mgr, frontendFS, logger)
	server := &http.Server{Addr: listen, Handler: handler}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		logger.Info("setup server listening", "addr", listen)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		// finalize() invoked our cancel.
		logger.Info("setup complete — transitioning to normal mode")
	case sig := <-sigCh:
		logger.Info("setup mode: signal received; exiting", "signal", sig)
		cancel()
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			logger.Error("setup server error", "error", err)
			os.Exit(1)
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)

	// Clean up the one-shot bootstrap token file — it's no longer valid
	// once the manager has flipped inactive.
	if tokenPath != "" {
		_ = os.Remove(tokenPath)
	}

	return !mgr.Active()
}

// runNormalMode is the original boot path, now cleanly factored out of
// main so the setup→normal transition is just a function call.
func runNormalMode(logger *slog.Logger, cfg config.Config) {
	// Security posture: same rules as before, read from the merged config.
	if cfg.Auth.JWTSecret == "" && !isLoopback(cfg.Server.ListenAddr) {
		fmt.Fprintln(os.Stderr, "FATAL: auth.jwt_secret (or env JWT_SECRET) is required when binding to a non-loopback address.")
		os.Exit(1)
	}
	if cfg.Auth.JWTSecret != "" && cfg.Auth.AdminBootstrapPassword == "" {
		logger.Warn("no ADMIN_PASSWORD set; login will rely on admin_auth DB row")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Database — embedded vs external converge on a single store.Config.
	var storeCfg store.Config
	var embeddedPG *opg.Embedded
	switch cfg.DB.Mode {
	case "embedded":
		pg, err := opg.Start(ctx, opg.Config{
			DataDir:  cfg.DB.Embedded.DataDir,
			CacheDir: cfg.DB.Embedded.CacheDir,
			Port:     cfg.DB.Embedded.Port,
			Version:  cfg.DB.Embedded.Version,
			Password: cfg.DB.Embedded.Password,
			Logger:   logger,
		})
		if err != nil {
			logger.Error("embedded postgres failed to start", "error", err)
			os.Exit(1)
		}
		embeddedPG = pg
		defer embeddedPG.Stop()
		if cfg.DB.Embedded.Password == "" {
			cfg.DB.Embedded.Password = pg.Password()
			if err := config.Save(cfg); err != nil {
				logger.Warn("failed to persist embedded PG password to config", "error", err)
			}
		}
		storeCfg = store.Config{
			Host:     pg.Host(),
			Port:     strconv.Itoa(pg.Port()),
			User:     pg.UserName(),
			Password: pg.Password(),
			DBName:   pg.DBName(),
		}
	case "external":
		storeCfg = store.Config{
			Host:     cfg.DB.External.Host,
			Port:     strconv.Itoa(cfg.DB.External.Port),
			User:     cfg.DB.External.User,
			Password: cfg.DB.External.Password,
			DBName:   cfg.DB.External.Name,
		}
	default:
		fmt.Fprintf(os.Stderr, "FATAL: unknown db.mode %q\n", cfg.DB.Mode)
		os.Exit(1)
	}

	db, err := store.New(ctx, storeCfg)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Migrate(ctx); err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}
	logger.Info("database connected and migrated")

	// Auth
	var (
		jwtAuth   *auth.Auth
		credStore *auth.CredentialStore
	)
	if cfg.Auth.JWTSecret != "" {
		jwtAuth = auth.New(cfg.Auth.JWTSecret, 7*24*time.Hour)
		credStore = auth.NewCredentialStore(db.Pool)
		logger.Info("JWT authentication enabled")
	}

	// If the wizard staged a bootstrap password, plant it in admin_auth
	// now that the DB is ready. Safe-idempotent: the store.Save upsert
	// only writes when the row is absent, so a second launch after a UI
	// password change never clobbers user edits.
	if credStore != nil && cfg.Auth.AdminBootstrapPassword != "" {
		if existing, err := credStore.Load(ctx); err == nil && existing == nil {
			user := cfg.Auth.AdminBootstrapUsername
			if user == "" {
				user = "admin"
			}
			if err := credStore.Save(ctx, user, cfg.Auth.AdminBootstrapPassword); err != nil {
				logger.Warn("failed to plant bootstrap admin credentials", "error", err)
			} else {
				logger.Info("bootstrap admin credentials written", "username", user)
			}
		}
	}

	// Provider Runtime + plugin platform (M1)
	hookBus := plugin.NewHookBus(logger)
	contribRegistry := contributions.NewRegistry()
	providerRuntime := plugin.NewRuntime(db, hookBus, cfg.Plugins.Dir, logger,
		plugin.WithContributions(contribRegistry),
		plugin.WithSynthesizer(synthesizeContributes))

	if err := providerRuntime.LoadAll(ctx); err != nil {
		logger.Warn("provider loading had errors", "error", err)
	}
	providerRuntime.StartHealthCheck(ctx, 60*time.Second)
	logger.Info("providers loaded", "count", len(providerRuntime.List()))

	// Plugin platform wiring (T5 gate, T6 installer, T10 dispatcher). The
	// gate reads consents and writes audit; adapters thread store.DB
	// through bridge's deliberately-store-agnostic interfaces.
	pluginGate := bridge.NewGate(&dbConsentReader{db: db}, &dbAuditSink{db: db}, logger)
	installer := install.NewInstaller(cfg.PluginsDataDir, db, providerRuntime, pluginGate, logger)
	installer.AllowLocal = cfg.AllowLocalPlugins

	// Marketplace catalog — loaded once at boot. Dir defaults to
	// $REPO/plugins/marketplace when unset; a missing catalog.json
	// leaves the Hub empty rather than failing boot.
	marketplaceDir := cfg.MarketplaceDir
	marketplaceCatalog, merr := marketplace.Load(marketplaceDir)
	if merr != nil {
		logger.Warn("marketplace: catalog load failed; Hub will be empty",
			"dir", marketplaceDir, "err", merr)
		marketplaceCatalog, _ = marketplace.Load("") // guaranteed nil-safe empty catalog
	} else {
		logger.Info("marketplace: catalog loaded",
			"dir", marketplaceDir, "entries", len(marketplaceCatalog.List()))
	}
	// hostSupervisor is constructed below once the namespace APIs exist;
	// declared here so the dispatcher's HostCaller closure can reference
	// it by name. The closure is safe as long as a sidecar isn't invoked
	// before hostSupervisor is assigned (startup order guarantees this).
	var hostSupervisor *host.Supervisor
	dispatcher, err := commands.NewDispatcher(commands.Config{
		Registry: contribRegistry, Gate: pluginGate, Log: logger,
		Host: commands.HostCallerFunc(func(ctx context.Context, plugin, method string, params json.RawMessage) (json.RawMessage, error) {
			if hostSupervisor == nil {
				return nil, fmt.Errorf("host: supervisor not initialised")
			}
			sc, serr := hostSupervisor.Ensure(ctx, plugin)
			if serr != nil {
				return nil, serr
			}
			return sc.Call(ctx, method, params)
		}),
	})
	if err != nil {
		logger.Error("plugin command dispatcher init failed", "error", err)
		return
	}
	invoker := &dispatcherAdapter{d: dispatcher}

	// M2 bridge wiring (T7 manager, T9 workbench, T10 storage, T11 events,
	// T14 SSE bus). WorkbenchBus is the host→Flutter fan-out; it doubles
	// as ShowMessage/OpenView/StatusBar sink for the workbench namespace.
	bridgeMgr := bridge.NewManager(logger)
	workbenchBus := gateway.NewWorkbenchBus(logger)
	installer.OnContributionsChanged = workbenchBus.PublishContributionsChanged
	workbenchAPI := bridge.NewWorkbenchAPI(bridge.WorkbenchConfig{
		Message:   workbenchBus,
		OpenView:  workbenchBus,
		StatusBar: workbenchBus,
		Command:   invoker,
	})
	storageAPI := bridge.NewStorageAPI(db, pluginGate)
	hookAdapter := newHookBusAdapter(hookBus, logger)
	eventsAPI := bridge.NewEventsAPI(hookAdapter, pluginGate)

	// M3 — privileged namespaces + host sidecar supervisor.
	pathVars := &gateway.PathVarResolver{
		DataDir:   cfg.PluginsDataDir,
		Providers: providerRuntime,
	}
	fsAPI := bridge.NewFSAPI(bridge.FSConfig{
		Gate: pluginGate, Resolver: pathVars, Log: logger,
	})
	execAPI := bridge.NewExecAPI(bridge.ExecConfig{
		Gate: pluginGate, Resolver: pathVars, Log: logger,
	})
	httpAPI := bridge.NewHTTPAPI(bridge.HTTPConfig{
		Gate: pluginGate, Log: logger,
	})
	kekProvider := auth.NewKEKProviderFromAdminAuth(credStore)
	secretAPI := bridge.NewSecretAPI(bridge.SecretConfig{
		Store: &secretStoreAdapter{db: db},
		Gate:  pluginGate,
		KEK:   kekProvider,
		Log:   logger,
	})

	// Supervisor handler factory — each sidecar gets an RPCHandler
	// bound to its plugin name so sidecar→host JSON-RPC calls route
	// through the same bridge.Namespace surface webview plugins use.
	// Namespaces adapter-wraps each *bridge.*API's Dispatch(ctx,
	// plugin, method, args, envID, conn) for the stdio-only sidecar
	// path. envID="" + conn=nil — neither is read in the non-stream
	// dispatch path of M3 namespaces.
	nsMap := map[string]host.NSDispatcher{
		"fs":        host.NamespaceAdapter{Inner: fsAPI.Dispatch},
		"exec":      host.NamespaceAdapter{Inner: execAPI.Dispatch},
		"http":      host.NamespaceAdapter{Inner: httpAPI.Dispatch},
		"secret":    host.NamespaceAdapter{Inner: secretAPI.Dispatch},
		"workbench": host.NamespaceAdapter{Inner: workbenchAPI.Dispatch},
		"storage":   host.NamespaceAdapter{Inner: storageAPI.Dispatch},
		"events":    host.NamespaceAdapter{Inner: eventsAPI.Dispatch},
	}

	hostSupervisor = host.NewSupervisor(host.Config{
		DataDir:   cfg.PluginsDataDir,
		Providers: providerRuntime,
		State:     &hostStateAdapter{db: db, log: logger},
		Log:       logger,
		PluginVersion: func(name string) string {
			if p, ok := providerRuntime.Get(name); ok {
				return p.Version
			}
			return ""
		},
		HandlerFactory: func(pluginName string) host.RPCHandler {
			h, err := host.NewHostRPCHandler(host.HostRPCConfig{
				Plugin:     pluginName,
				Namespaces: nsMap,
			})
			if err != nil {
				logger.Error("host: handler factory", "plugin", pluginName, "err", err)
				return nil
			}
			return h
		},
	})

	// Session Hub
	idleThreshold := 8 * time.Second
	if cfg.Plugins.IdleThresholdSeconds > 0 {
		idleThreshold = time.Duration(cfg.Plugins.IdleThresholdSeconds) * time.Second
	}

	mcpRuntime := mcp.New(mcp.Config{DB: db, Logger: logger})

	sessionHub := hub.New(hub.Config{
		DB:            db,
		IdleThreshold: idleThreshold,
		Logger:        logger,
		Resolver:      &providerResolver{rt: providerRuntime},
		Events:        hookBus,
		Injector:      &mcpInjector{rt: mcpRuntime},
	})
	sessionHub.RecoverOnStartup(ctx, cfg.Plugins.AutoResume)
	sessionHub.StartHealthCheck(ctx, 60*time.Second)

	// Frontend
	var frontendFS fs.FS
	if distFS, err := fs.Sub(app.DistFS, "build/web"); err == nil {
		if entries, err := fs.ReadDir(distFS, "."); err == nil && len(entries) > 0 {
			frontendFS = distFS
			logger.Info("serving embedded frontend")
		}
	}

	gw := gateway.New(gateway.Config{
		Hub: sessionHub, Plugins: providerRuntime,
		MCP:           mcpRuntime,
		Auth:          jwtAuth,
		Credentials:   credStore,
		AdminUsername: cfg.Auth.AdminBootstrapUsername,
		AdminPassword: cfg.Auth.AdminBootstrapPassword,
		Logger:        logger, FrontendFS: frontendFS,
		// Plugin platform (M1)
		Installer:      installer,
		Contributions:  contribRegistry,
		CommandInvoker: invoker,
		// M2 bridge
		BridgeManager: bridgeMgr,
		WorkbenchBus:  workbenchBus,
		// Build identity surfaced via /api/health for the Flutter About page.
		Version:  version,
		BuildSha: buildSha,
		// Marketplace catalog backing /api/marketplace/plugins + marketplace://.
		Marketplace: marketplaceCatalog,
	})
	gw.RegisterNamespace("workbench", workbenchAPI)
	gw.RegisterNamespace("storage", storageAPI)
	gw.RegisterNamespace("events", eventsAPI)
	gw.RegisterNamespace("fs", fsAPI)
	gw.RegisterNamespace("exec", execAPI)
	gw.RegisterNamespace("http", httpAPI)
	gw.RegisterNamespace("secret", secretAPI)

	server := &http.Server{Addr: cfg.Server.ListenAddr, Handler: gw.Handler()}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("OpenDray starting", "addr", cfg.Server.ListenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	sig := <-sigCh
	logger.Info("shutting down", "signal", sig)
	cancel()
	sessionHub.StopAll()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	// Stop host sidecars before the HTTP server — they may depend on
	// the gateway's HTTP routes during their own shutdown requests.
	_ = hostSupervisor.Stop(shutdownCtx)
	server.Shutdown(shutdownCtx)
}

// writeBootstrapToken persists the token to ~/.opendray/setup-token so the
// Flutter wizard can fetch it same-origin if the user opens /setup without
// ?token= in the URL. Returns the path on success.
func writeBootstrapToken(token string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := home + "/.opendray"
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := dir + "/setup-token"
	if err := os.WriteFile(path, []byte(token), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// displayAddr converts a listen addr like ":8640" into something
// click-usable in the stderr hint ("127.0.0.1:8640"). Non-wildcard
// addrs pass through unchanged.
func displayAddr(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "127.0.0.1" + addr
	}
	return addr
}

// openBrowser fires the OS-native "open a URL" handler. Best-effort:
// failures are silent — the stderr banner already tells the user how to
// proceed if no browser opens.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default: // linux, freebsd, openbsd, etc.
		// Try xdg-open first, fall back to common alternatives.
		for _, candidate := range []string{"xdg-open", "sensible-browser", "gnome-open", "kfmclient"} {
			if _, err := exec.LookPath(candidate); err == nil {
				cmd = exec.Command(candidate, url)
				break
			}
		}
		if cmd == nil {
			return fmt.Errorf("no known browser launcher found")
		}
	}
	return cmd.Start()
}

// padRight right-pads a string with spaces up to n runes. Used to align
// the ASCII border in the setup banner.
func padRight(s string, n int) string {
	diff := n - len(s)
	if diff <= 0 {
		return s + " │"
	}
	return s + strings.Repeat(" ", diff) + "│"
}

// providerResolver adapts plugin.Runtime to hub.ProviderResolver.
type providerResolver struct {
	rt *plugin.Runtime
}

func (pr *providerResolver) ResolveCLI(name string) (hub.ResolvedCLI, bool) {
	resolved, ok := pr.rt.ResolveCLI(name)
	if !ok {
		return hub.ResolvedCLI{}, false
	}
	return hub.ResolvedCLI{
		Command: resolved.Command,
		Args:    resolved.Args,
		Env:     resolved.Env,
	}, true
}

// mcpInjector adapts mcp.Runtime to hub.SessionInjector.
type mcpInjector struct {
	rt *mcp.Runtime
}

func (m *mcpInjector) RenderFor(ctx context.Context, sessionID, agent string) (hub.Injection, error) {
	inj, err := m.rt.RenderFor(ctx, sessionID, agent)
	if err != nil {
		return hub.Injection{}, err
	}
	return hub.Injection{Args: inj.Args, Env: inj.Env}, nil
}

func (m *mcpInjector) Cleanup(sessionID string) { m.rt.Cleanup(sessionID) }

// ── Plugin-platform adapters ───────────────────────────────────────────
//
// bridge.Gate is intentionally store-agnostic (it takes small interfaces
// so tests and mocks don't need a real DB). These adapters thread the
// production store.DB through bridge's interfaces + translate between
// the two AuditEntry shapes.

type dbConsentReader struct{ db *store.DB }

func (r *dbConsentReader) Load(ctx context.Context, pluginName string) ([]byte, bool, error) {
	c, err := r.db.GetConsent(ctx, pluginName)
	if errors.Is(err, store.ErrConsentNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return c.PermsJSON, true, nil
}

type dbAuditSink struct{ db *store.DB }

func (s *dbAuditSink) Append(ctx context.Context, ev bridge.AuditEvent) error {
	return s.db.AppendAudit(ctx, store.AuditEntry{
		PluginName: ev.PluginName, Ns: ev.Ns, Method: ev.Method,
		Caps: ev.Caps, Result: ev.Result, DurationMs: ev.DurationMs,
		ArgsHash: ev.ArgsHash, Message: ev.Message,
	})
}

// synthesizeContributes adapts compat.Synthesize to plugin.SynthesizerFn —
// compat returns a full Provider overlay but the runtime only cares about
// the ContributesV1 block at load time. Keeping the adapter here avoids
// compat.Synthesize leaking into plugin.Runtime's surface.
func synthesizeContributes(p plugin.Provider) plugin.ContributesV1 {
	out := compat.Synthesize(p)
	if out.Contributes == nil {
		return plugin.ContributesV1{}
	}
	return *out.Contributes
}

// secretStoreAdapter wraps *store.DB to satisfy bridge.SecretStore.
// The only real job is to translate pgx.ErrNoRows (what the store
// layer returns from GetWrappedDEK) into bridge.WrappedDEKNotFound
// (the sentinel the bridge layer checks via errors.Is). Every other
// method passes through unchanged. Keeping this adapter here — not
// in kernel/store — preserves the bridge's rule of not importing
// pgx.
type secretStoreAdapter struct{ db *store.DB }

func (a *secretStoreAdapter) SecretGet(ctx context.Context, plugin, key string) ([]byte, []byte, bool, error) {
	return a.db.SecretGet(ctx, plugin, key)
}
func (a *secretStoreAdapter) SecretSet(ctx context.Context, plugin, key string, ciphertext, nonce []byte) error {
	return a.db.SecretSet(ctx, plugin, key, ciphertext, nonce)
}
func (a *secretStoreAdapter) SecretDelete(ctx context.Context, plugin, key string) error {
	return a.db.SecretDelete(ctx, plugin, key)
}
func (a *secretStoreAdapter) SecretList(ctx context.Context, plugin string) ([]string, error) {
	return a.db.SecretList(ctx, plugin)
}
func (a *secretStoreAdapter) EnsureKEKRow(ctx context.Context, plugin string, wrapped []byte, kid string) error {
	return a.db.EnsureKEKRow(ctx, plugin, wrapped, kid)
}
func (a *secretStoreAdapter) GetWrappedDEK(ctx context.Context, plugin string) ([]byte, string, error) {
	wrapped, kid, err := a.db.GetWrappedDEK(ctx, plugin)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", bridge.WrappedDEKNotFound
	}
	return wrapped, kid, err
}

// hostStateAdapter implements host.StateWriter with best-effort
// writes to plugin_host_state (migration 015). Failures are logged
// at warn; the supervisor never fails a start on State unavailability.
type hostStateAdapter struct {
	db  *store.DB
	log *slog.Logger
}

func (a *hostStateAdapter) RecordStarted(ctx context.Context, plugin string) error {
	_, err := a.db.Pool.Exec(ctx,
		`INSERT INTO plugin_host_state (plugin_name, last_started_at, restart_count)
		 VALUES ($1, now(), 0)
		 ON CONFLICT (plugin_name) DO UPDATE
		   SET last_started_at = now(),
		       restart_count = plugin_host_state.restart_count + 1,
		       last_error = NULL`,
		plugin,
	)
	if err != nil {
		a.log.Warn("host: record started failed", "plugin", plugin, "err", err)
	}
	return err
}

func (a *hostStateAdapter) RecordExited(ctx context.Context, plugin string, exitCode int, lastErr string) error {
	_, err := a.db.Pool.Exec(ctx,
		`UPDATE plugin_host_state
		 SET last_exit_code = $2, last_error = $3
		 WHERE plugin_name = $1`,
		plugin, exitCode, lastErr,
	)
	if err != nil {
		a.log.Warn("host: record exited failed", "plugin", plugin, "err", err)
	}
	return err
}

// dispatcherAdapter satisfies both gateway's unexported commandInvoker
// and bridge.CommandInvoker by widening commands.Dispatcher.Invoke's
// *Result return to any.
type dispatcherAdapter struct{ d *commands.Dispatcher }

func (a *dispatcherAdapter) Invoke(ctx context.Context, pluginName, commandID string, args map[string]any) (any, error) {
	return a.d.Invoke(ctx, pluginName, commandID, args)
}

// ── HookBus ↔ bridge.HookBusLike adapter (M2 T11) ──────────────────────
//
// The production *plugin.HookBus exposes typed HookEvent callbacks via
// SubscribeLocal + DispatchOutput/DispatchSessionEvent — not the generic
// SubscribeByName / Publish surface bridge.EventsAPI consumes. This
// adapter translates:
//
//   - Legacy HookEvent types → M2 dotted names:
//     onSessionStart → "session.start"
//     onSessionStop  → "session.stop"
//     onIdle         → "session.idle"
//     onOutput       → "session.output"
//
//   - bridge.Publish calls into a private in-memory fan-out (not
//     re-dispatched to HookBus — legacy plugins filter by HookEvent.Type
//     and would drop unqualified plugin.* names).
//
// Pattern matching reuses bridge.MatchEventPattern so the semantics are
// identical to the subscribe path in EventsAPI.

type hookBusAdapter struct {
	mu   sync.Mutex
	subs []*hookBusAdapterSub
	log  *slog.Logger
}

type hookBusAdapterSub struct {
	pattern string
	handler func(name string, data any)
	active  bool
}

func newHookBusAdapter(hb *plugin.HookBus, log *slog.Logger) *hookBusAdapter {
	a := &hookBusAdapter{log: log}
	// One local listener covers every hook type the bridge surfaces today.
	hb.SubscribeLocal(
		[]string{
			plugin.HookOnSessionStart,
			plugin.HookOnSessionStop,
			plugin.HookOnIdle,
			plugin.HookOnOutput,
		},
		func(e plugin.HookEvent) {
			name := hookEventToM2Name(e.Type)
			if name == "" {
				return
			}
			a.dispatch(name, map[string]any{
				"sessionId": e.SessionID,
				"data":      e.Data,
				"timestamp": e.Timestamp,
			})
		},
	)
	return a
}

func hookEventToM2Name(t string) string {
	switch t {
	case plugin.HookOnSessionStart:
		return "session.start"
	case plugin.HookOnSessionStop:
		return "session.stop"
	case plugin.HookOnIdle:
		return "session.idle"
	case plugin.HookOnOutput:
		return "session.output"
	}
	return ""
}

func (a *hookBusAdapter) SubscribeByName(pattern string, handler func(name string, data any)) (func(), error) {
	sub := &hookBusAdapterSub{pattern: pattern, handler: handler, active: true}
	a.mu.Lock()
	a.subs = append(a.subs, sub)
	a.mu.Unlock()
	return func() {
		a.mu.Lock()
		sub.active = false
		a.mu.Unlock()
	}, nil
}

func (a *hookBusAdapter) Publish(name string, data any) {
	a.dispatch(name, data)
}

func (a *hookBusAdapter) dispatch(name string, data any) {
	a.mu.Lock()
	matched := make([]*hookBusAdapterSub, 0, len(a.subs))
	for _, s := range a.subs {
		if s.active && bridge.MatchEventPattern([]string{s.pattern}, name) {
			matched = append(matched, s)
		}
	}
	a.mu.Unlock()
	for _, s := range matched {
		s := s
		go func() {
			defer func() {
				if r := recover(); r != nil && a.log != nil {
					a.log.Warn("bridge hookBusAdapter: subscriber panicked",
						"pattern", s.pattern, "panic", r)
				}
			}()
			s.handler(name, data)
		}()
	}
}

// isLoopback returns true if the listen address binds only to a loopback
// interface. An empty host (e.g. ":8640") binds all interfaces and is NOT
// considered loopback.
func isLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
