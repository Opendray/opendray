package main

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/linivek/ntc/gateway"
	"github.com/linivek/ntc/gateway/mcp"
	"github.com/linivek/ntc/kernel/auth"
	"github.com/linivek/ntc/kernel/hub"
	"github.com/linivek/ntc/kernel/store"
	"github.com/linivek/ntc/app"
	"github.com/linivek/ntc/plugin"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := loadEnv()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Database
	db, err := store.New(ctx, store.Config{
		Host: cfg.dbHost, Port: cfg.dbPort,
		User: cfg.dbUser, Password: cfg.dbPassword, DBName: cfg.dbName,
	})
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
	var jwtAuth *auth.Auth
	if cfg.jwtSecret != "" {
		jwtAuth = auth.New(cfg.jwtSecret, 7*24*time.Hour)
		logger.Info("JWT authentication enabled")
	}

	// Provider Runtime (load before hub)
	hookBus := plugin.NewHookBus(logger)
	providerRuntime := plugin.NewRuntime(db, hookBus, cfg.pluginDir, logger)

	if err := providerRuntime.LoadAll(ctx); err != nil {
		logger.Warn("provider loading had errors", "error", err)
	}
	providerRuntime.StartHealthCheck(ctx, 60*time.Second)
	logger.Info("providers loaded", "count", len(providerRuntime.List()))

	// Session Hub — uses provider runtime to resolve CLI specs
	idleThreshold := 8 * time.Second
	if cfg.idleThresholdSec > 0 {
		idleThreshold = time.Duration(cfg.idleThresholdSec) * time.Second
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
	sessionHub.RecoverOnStartup(ctx, cfg.autoResume)
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
		MCP:  mcpRuntime,
		Auth: jwtAuth, Logger: logger, FrontendFS: frontendFS,
	})

	server := &http.Server{Addr: cfg.listenAddr, Handler: gw.Handler()}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("NTC starting", "addr", cfg.listenAddr)
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

type envConfig struct {
	listenAddr       string
	dbHost           string
	dbPort           string
	dbUser           string
	dbPassword       string
	dbName           string
	jwtSecret        string
	pluginDir        string
	autoResume       bool
	idleThresholdSec int
}

func loadEnv() envConfig {
	cfg := envConfig{
		listenAddr: envOr("LISTEN_ADDR", ":8640"),
		dbHost:     envOr("DB_HOST", ""),
		dbPort:     envOr("DB_PORT", "5432"),
		dbUser:     envOr("DB_USER", "ntc_user"),
		dbPassword: os.Getenv("DB_PASSWORD"),
		dbName:     envOr("DB_NAME", "ntc"),
		jwtSecret:  os.Getenv("JWT_SECRET"),
		pluginDir:  envOr("PLUGIN_DIR", "./plugins"),
		autoResume: os.Getenv("AUTO_RESUME") == "true",
	}
	if cfg.dbHost == "" {
		fmt.Fprintln(os.Stderr, "DB_HOST is required")
		os.Exit(1)
	}
	if v := os.Getenv("IDLE_THRESHOLD_SECONDS"); v != "" {
		n, _ := strconv.Atoi(v)
		cfg.idleThresholdSec = n
	}
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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
