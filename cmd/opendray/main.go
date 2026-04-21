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
	"os/signal"
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
	"github.com/opendray/opendray/kernel/store"
	"github.com/opendray/opendray/plugin"
	"github.com/opendray/opendray/plugin/bridge"
	"github.com/opendray/opendray/plugin/commands"
	"github.com/opendray/opendray/plugin/compat"
	"github.com/opendray/opendray/plugin/contributions"
	"github.com/opendray/opendray/plugin/host"
	"github.com/opendray/opendray/plugin/install"
	"github.com/opendray/opendray/plugin/market"
	"github.com/opendray/opendray/plugin/market/actions"
	marketlocal "github.com/opendray/opendray/plugin/market/local"
	marketremote "github.com/opendray/opendray/plugin/market/remote"
	"github.com/opendray/opendray/plugin/market/revocation"

	"github.com/jackc/pgx/v5"
)

// version / buildSha / buildTime are injected at build time via -ldflags
// (see Makefile). Defaults land for `go run` so the binary still boots
// cleanly without a release.
//
// buildTime is the UTC clock at build time formatted as 20060102T150405Z.
// Pair it with `version` when you suspect a deploy didn't actually
// recompile — two builds from the same commit still have different
// buildTime stamps.
var (
	version   = "dev"
	buildSha  = "unknown"
	buildTime = "unknown"
)

// printHelp is the output of `opendray help` / `-h`. Grouped by
// domain (server / service / plugins / misc) so it stays scannable
// as more subcommands land.
func printHelp() {
	fmt.Println(`OpenDray — pilot AI coding agents from anywhere.

Usage:
  opendray [command] [flags]

Server lifecycle
  (no args)             Start the server. Requires a completed setup.
                        Refuses to start without config — run setup first.
  setup                 Interactive terminal wizard (database, listen
                        address, admin account, JWT). --yes + flags
                        for scripted installs (CI / cloud-init).
  uninstall             Stop, remove data + config, delete this binary.
                        --yes / --dry-run / --keep-data

Background service
  service install       Install as a systemd (Linux) or launchd (macOS)
                        service that auto-starts on boot. Needs sudo.
                        --user / --binary / --force / --dry-run
  service uninstall     Remove the service definition + stop it.
  service start         Start the service now.
  service stop          Stop the running service.
  service restart       Stop then start.
  service status        Show what the service is doing right now.
  service logs          Tail service logs. -f / --follow (default on),
                        -n / --lines N.
  service help          Full service-command reference.

Plugins
  plugin                Plugin lifecycle subcommands — see:
                          opendray plugin help

Diagnostics
  version               Print version / buildSha / buildTime.
  help                  Show this help.

Environment
  OPENDRAY_CONFIG       Override the config.toml path.

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
		case "uninstall":
			os.Exit(runUninstallCLI())
		case "service":
			os.Exit(runServiceCLI(os.Args[2:]))
		case "version", "-v", "--version":
			// One line per field so `opendray version | tr ' ' '\n'` or
			// `grep` can pick them out easily. Deploy verification
			// scripts compare buildTime between runs to prove the
			// binary was actually recompiled.
			fmt.Printf("version:   %s\n", version)
			fmt.Printf("buildSha:  %s\n", buildSha)
			fmt.Printf("buildTime: %s\n", buildTime)
			return
		case "-h", "--help", "help":
			printHelp()
			return
		}
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// Load config and demand it be complete. The browser-based setup
	// wizard was removed — the ONLY supported first-run path is now
	// `opendray setup` in a terminal (see cmd/opendray/setup_cli.go).
	//
	// If the config is missing or incomplete, we exit with a clear
	// stderr message rather than starting some half-mode that confuses
	// the user. No automatic fallback, no auto-launch of the wizard —
	// a bare `opendray` invocation either boots the full server or
	// tells the user to run `opendray setup` first.
	cfg, source, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: %v\n", err)
		os.Exit(1)
	}
	logger.Info("config loaded", "source", source, "db_mode", cfg.DB.Mode,
		"complete", cfg.IsComplete())

	if !cfg.IsComplete() {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "✗ OpenDray is not configured.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  Run the setup wizard first:")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "      opendray setup")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  For scripted installs:")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "      opendray setup --yes --db=bundled \\")
		fmt.Fprintln(os.Stderr, "          --admin-user=admin --admin-password-file=/path/to/pw")
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: %v\n", err)
		os.Exit(1)
	}

	// Fail fast on the root + embedded-PG combo. PostgreSQL's initdb
	// rejects uid 0 and downstream errors are cryptic — surface the
	// same friendly message the setup wizard gives before we even try
	// to boot the DB.
	if cfg.DB.Mode == "embedded" && os.Geteuid() == 0 {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "✗ Bundled PostgreSQL cannot run as root.")
		fmt.Fprintln(os.Stderr, "  PostgreSQL's initdb refuses uid 0 for security reasons.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  Fix — create an unprivileged user and re-run as that user:")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "      useradd -r -m -s /bin/bash -d /home/opendray opendray")
		fmt.Fprintln(os.Stderr, "      su - opendray")
		fmt.Fprintln(os.Stderr, "      opendray setup")
		fmt.Fprintln(os.Stderr, "      opendray")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  Or reconfigure to use an external PostgreSQL:")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "      opendray setup")
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)
	}

	runNormalMode(logger, cfg)
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

	// Marketplace catalog. Picks local-disk or remote-HTTPS based
	// on config — remote wins when both are set.
	//
	//   MarketplaceURL set → market/remote hits the registry URL
	//                        (+ mirrors, + in-memory TTL cache,
	//                        + signature verification at install).
	//   MarketplaceDir set → market/local reads catalog.json off disk.
	//                        M3 behaviour; preserved for airgapped
	//                        deployments and for syz's mock registry.
	//   neither set         → empty local catalog; Hub shows nothing.
	marketplaceCatalog := buildMarketplaceCatalog(cfg, logger)
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

	// LLM Endpoints — platform capability exposing the kernel's
	// llm_providers table to any agent plugin that declares
	// `"permissions": {"llm": true}`. Metadata-only; API keys stay in
	// the kernel (bridge never surfaces them). Writes still flow
	// through /api/llm-providers HTTP endpoints — the bridge surface
	// is intentionally read-only.
	llmAPI := bridge.NewLLMAPI(bridge.LLMConfig{
		Source: &llmEndpointSourceAdapter{db: db},
		Gate:   pluginGate,
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
		"llm":       host.NamespaceAdapter{Inner: llmAPI.Dispatch},
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

	// CLI config overlay — mirrors gateway.effectiveConfig but lives
	// here because plugin.Runtime.ResolveCLI runs before the gateway
	// Server exists (session recovery on startup). Without this,
	// values saved via PUT /api/plugins/{name}/config land in
	// plugin_kv / plugin_secret but never reach session spawn — so
	// Claude's bypassPermissions, Gemini's yolo, etc. silently drop.
	// The shared store key prefix is "__config." (see
	// gateway/plugins_config.go). Kept as a literal rather than a
	// cross-package import to keep the plugin runtime store-agnostic.
	providerRuntime.SetConfigOverlay(func(ctx context.Context, pluginName string, base plugin.ProviderConfig) plugin.ProviderConfig {
		merged := make(plugin.ProviderConfig, len(base)+8)
		for k, v := range base {
			merged[k] = v
		}
		prov, ok := providerRuntime.Get(pluginName)
		if !ok {
			return merged
		}
		const cfgPrefix = "__config."
		for _, f := range prov.ConfigSchema {
			storeKey := cfgPrefix + f.Key
			if f.Type == "secret" {
				val, found, err := secretAPI.PlatformGet(ctx, pluginName, storeKey)
				if err != nil || !found {
					continue
				}
				merged[f.Key] = val
				continue
			}
			raw, found, err := db.KVGet(ctx, pluginName, storeKey)
			if err != nil || !found {
				continue
			}
			var str string
			if json.Unmarshal(raw, &str) == nil {
				merged[f.Key] = str
			} else {
				merged[f.Key] = string(raw)
			}
		}
		return merged
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
		Version:   version,
		BuildSha:  buildSha,
		BuildTime: buildTime,
		// Marketplace catalog backing /api/marketplace/plugins + marketplace://.
		Marketplace:         marketplaceCatalog,
		MarketplaceSettings: buildMarketplaceSettings(cfg),
		// User-config endpoints — SecretAPI for encrypted fields,
		// hostSupervisor.Kill for post-write sidecar restart.
		SecretAPI:      secretAPI,
		HostSupervisor: hostSupervisor,
	})
	gw.RegisterNamespace("workbench", workbenchAPI)
	gw.RegisterNamespace("storage", storageAPI)
	gw.RegisterNamespace("events", eventsAPI)
	gw.RegisterNamespace("fs", fsAPI)
	gw.RegisterNamespace("exec", execAPI)
	gw.RegisterNamespace("http", httpAPI)
	gw.RegisterNamespace("secret", secretAPI)
	gw.RegisterNamespace("llm", llmAPI)

	// Marketplace revocation poller. Only starts when the
	// marketplace has something to fetch revocations from —
	// empty-catalog deployments skip it to avoid endless "fetch
	// revocations.json: not found" log spam.
	if cfg.MarketplaceURL != "" || cfg.MarketplaceDir != "" {
		startRevocationPoller(
			ctx,
			logger,
			marketplaceCatalog,
			installer,
			providerRuntime,
			workbenchBus,
			cfg.RevocationPollHours,
		)
	}

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

// buildMarketplaceCatalog picks the concrete market.Catalog
// backend based on config. Remote wins when both RegistryURL and
// MarketplaceDir are set. A construction failure falls back to
// an empty local catalog so the server always boots — missing
// marketplace is not a hard error.
func buildMarketplaceCatalog(cfg config.Config, logger *slog.Logger) market.Catalog {
	if cfg.MarketplaceURL != "" {
		remote, err := marketremote.New(marketremote.Config{
			RegistryURL: cfg.MarketplaceURL,
			Mirrors:     cfg.MarketplaceMirrors,
		})
		if err != nil {
			logger.Warn("marketplace: remote init failed; Hub will be empty",
				"url", cfg.MarketplaceURL, "err", err)
			return emptyCatalog()
		}
		logger.Info("marketplace: remote catalog",
			"url", cfg.MarketplaceURL,
			"mirrors", len(cfg.MarketplaceMirrors))
		return remote
	}

	dir := cfg.MarketplaceDir
	local, err := marketlocal.Load(dir)
	if err != nil {
		logger.Warn("marketplace: local load failed; Hub will be empty",
			"dir", dir, "err", err)
		return emptyCatalog()
	}
	entries, _ := local.List(context.Background())
	logger.Info("marketplace: local catalog", "dir", dir, "entries", len(entries))
	return local
}

// buildMarketplaceSettings returns the read-only snapshot surfaced
// via GET /api/marketplace/settings. Decision logic mirrors
// buildMarketplaceCatalog so the two can't drift — whichever
// backend the gateway actually uses is the one Settings reports.
func buildMarketplaceSettings(cfg config.Config) gateway.MarketplaceSettings {
	source := "empty"
	switch {
	case cfg.MarketplaceURL != "":
		source = "remote"
	case cfg.MarketplaceDir != "":
		source = "local"
	}
	pollHours := cfg.RevocationPollHours
	if pollHours == 0 && source != "empty" {
		pollHours = 6 // defaultPollInterval / time.Hour
	}
	if pollHours < 1 {
		pollHours = 1
	}
	if pollHours > 168 {
		pollHours = 168
	}
	return gateway.MarketplaceSettings{
		Source:            source,
		RegistryURL:       cfg.MarketplaceURL,
		RegistryDir:       cfg.MarketplaceDir,
		Mirrors:           append([]string(nil), cfg.MarketplaceMirrors...),
		PollHours:         pollHours,
		AllowLocalPlugins: cfg.AllowLocalPlugins,
	}
}

func emptyCatalog() market.Catalog {
	// Load("") constructs a nil-safe empty local catalog; guaranteed
	// not to error per market/local.Load's contract.
	c, _ := marketlocal.Load("")
	return c
}

// startRevocationPoller wires the market kill-switch. Install
// Handler's Uninstall method + Runtime.SetEnabled feed into
// actions.Handler; a WorkbenchBus.Publish closure routes banners
// to every connected Flutter client. The poller runs for the
// lifetime of runNormalMode's context.
func startRevocationPoller(
	ctx context.Context,
	logger *slog.Logger,
	cat market.Catalog,
	installer *install.Installer,
	runtime *plugin.Runtime,
	bus *gateway.WorkbenchBus,
	pollHours int,
) {
	// Filter installed plugins to v1 only. Legacy compat-synthesized
	// plugins are baked into the binary and can't be uninstalled;
	// applying a revocation action to them would just churn logs.
	installedSnapshot := func() []revocation.InstalledPlugin {
		out := make([]revocation.InstalledPlugin, 0)
		for _, pi := range runtime.ListInfo() {
			if !pi.Provider.IsV1() {
				continue
			}
			out = append(out, revocation.InstalledPlugin{
				Publisher: pi.Provider.Publisher,
				Name:      pi.Provider.Name,
				Version:   pi.Provider.Version,
			})
		}
		return out
	}

	notify := func(kind, pluginName, reason string) {
		// Emit TWO events — one in the existing showMessage shape
		// so the current Flutter snackbar renders immediately
		// (payload.text + payload.kind=="error"/"info"), and a
		// dedicated "revocation" kind carrying the full record
		// for the future persistent banner UI. Clients that only
		// know showMessage degrade gracefully; the revocation
		// kind is purely additive.
		var (
			msgKind = "error"
			verb    = "revoked"
		)
		if kind == revocation.ActionWarn {
			msgKind = "warn"
			verb = "flagged"
		}
		text := fmt.Sprintf("Plugin %s %s: %s", pluginName, verb, reason)

		showMsg, _ := json.Marshal(map[string]any{
			"text": text,
			"kind": msgKind,
		})
		revocationEvt, _ := json.Marshal(map[string]any{
			"plugin": pluginName,
			"action": kind,
			"reason": reason,
			"text":   text,
		})

		if bus == nil {
			return
		}
		bus.Publish(gateway.WorkbenchEvent{
			Kind:    "showMessage",
			Plugin:  "",
			Payload: showMsg,
		})
		bus.Publish(gateway.WorkbenchEvent{
			Kind:    "revocation",
			Plugin:  pluginName,
			Payload: revocationEvt,
		})
	}

	handler, err := actions.New(actions.Config{
		Uninstall:  installer.Uninstall,
		SetEnabled: runtime.SetEnabled,
		Notify:     notify,
		Logger:     logger.With("mod", "revocation"),
	})
	if err != nil {
		logger.Error("revocation: handler init failed", "err", err)
		return
	}

	interval := time.Duration(pollHours) * time.Hour
	poller, err := revocation.New(revocation.Config{
		Catalog:   cat,
		Interval:  interval,
		Installed: installedSnapshot,
		OnAction:  handler.Dispatch,
		Logger:    logger.With("mod", "revocation"),
	})
	if err != nil {
		logger.Error("revocation: poller init failed", "err", err)
		return
	}
	go poller.Run(ctx)
	logger.Info("revocation: poller started", "interval", interval)
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

// llmEndpointSourceAdapter wraps *store.DB to satisfy
// bridge.LLMSource. The adapter projects the kernel's LLMProvider
// rows into the bridge-facing LLMEndpoint shape (metadata only —
// APIKeyEnv is dropped so plugins can't even see the env-var name).
type llmEndpointSourceAdapter struct{ db *store.DB }

func (a *llmEndpointSourceAdapter) ListLLMEndpoints(ctx context.Context) ([]bridge.LLMEndpoint, error) {
	rows, err := a.db.ListLLMProviders(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]bridge.LLMEndpoint, 0, len(rows))
	for _, r := range rows {
		out = append(out, bridge.LLMEndpoint{
			ID:           r.ID,
			Name:         r.Name,
			DisplayName:  r.DisplayName,
			ProviderType: r.ProviderType,
			BaseURL:      r.BaseURL,
			Description:  r.Description,
			Enabled:      r.Enabled,
		})
	}
	return out, nil
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
