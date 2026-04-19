package main

import (
	"context"
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

	// Provider Runtime
	hookBus := plugin.NewHookBus(logger)
	providerRuntime := plugin.NewRuntime(db, hookBus, cfg.Plugins.Dir, logger)

	if err := providerRuntime.LoadAll(ctx); err != nil {
		logger.Warn("provider loading had errors", "error", err)
	}
	providerRuntime.StartHealthCheck(ctx, 60*time.Second)
	logger.Info("providers loaded", "count", len(providerRuntime.List()))

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
	})

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
