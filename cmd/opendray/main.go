package main

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
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
	"github.com/opendray/opendray/kernel/store"
	"github.com/opendray/opendray/plugin"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, source, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: %v\n", err)
		os.Exit(1)
	}
	logger.Info("config loaded", "source", source, "db_mode", cfg.DB.Mode)

	// Setup mode is wired in Phase 3. For now, refuse to start with an
	// incomplete config — same behaviour as today, just via the new loader.
	if !cfg.IsComplete() {
		fmt.Fprintln(os.Stderr, "FATAL: OpenDray is not configured.")
		fmt.Fprintln(os.Stderr, "  • Write ~/.opendray/config.toml (setup wizard will land in Phase 3-4), OR")
		fmt.Fprintln(os.Stderr, "  • Set env vars: DB_HOST, DB_USER, DB_PASSWORD, DB_NAME, JWT_SECRET, ADMIN_PASSWORD.")
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: %v\n", err)
		os.Exit(1)
	}

	// Security posture: same rules as before, just read from the merged config.
	if cfg.Auth.JWTSecret == "" && !isLoopback(cfg.Server.ListenAddr) {
		fmt.Fprintln(os.Stderr, "FATAL: auth.jwt_secret (or env JWT_SECRET) is required when binding to a non-loopback address. Use listen_addr 127.0.0.1:8640 for local-only development.")
		os.Exit(1)
	}
	// When auth is enabled, either DB-backed credentials or a bootstrap
	// password must exist — otherwise any body can mint tokens.
	// The DB-row check runs later (we need a live DB); here we only
	// enforce the bootstrap-env fallback.
	if cfg.Auth.JWTSecret != "" && cfg.Auth.AdminBootstrapPassword == "" {
		// Not fatal yet — setup wizard writes credentials directly to the
		// admin_auth table. Phase 3 makes this precise by checking DB.
		logger.Warn("no ADMIN_PASSWORD set; login will rely on admin_auth DB row")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Database. Two paths — embedded (child PG managed by us) vs external
	// (user-supplied). They converge on a single store.Config before we
	// connect, so everything downstream is oblivious to the choice.
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
		// First-run password generation — persist so subsequent boots reuse it.
		if cfg.DB.Embedded.Password == "" {
			cfg.DB.Embedded.Password = pg.Password()
			if err := config.Save(cfg); err != nil {
				logger.Warn("failed to persist embedded PG password to config; next boot will regenerate and init will fail",
					"error", err)
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

	// Provider Runtime (load before hub)
	hookBus := plugin.NewHookBus(logger)
	providerRuntime := plugin.NewRuntime(db, hookBus, cfg.Plugins.Dir, logger)

	if err := providerRuntime.LoadAll(ctx); err != nil {
		logger.Warn("provider loading had errors", "error", err)
	}
	providerRuntime.StartHealthCheck(ctx, 60*time.Second)
	logger.Info("providers loaded", "count", len(providerRuntime.List()))

	// Session Hub — uses provider runtime to resolve CLI specs
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

	// Frontend FS
	var frontendFS fs.FS
	if distFS, err := fs.Sub(app.DistFS, "build/web"); err == nil {
		if entries, err := fs.ReadDir(distFS, "."); err == nil && len(entries) > 0 {
			frontendFS = distFS
			logger.Info("serving embedded frontend")
		}
	}

	// Gateway
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
// interface (127.0.0.1, ::1, localhost). An empty host (e.g. ":8640") binds
// all interfaces and is NOT considered loopback.
func isLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return false // binds all interfaces
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
