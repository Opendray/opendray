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
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray-v2/internal/audit"
	"github.com/opendray/opendray-v2/internal/auth"
	"github.com/opendray/opendray-v2/internal/catalog"
	"github.com/opendray/opendray-v2/internal/channel"
	_ "github.com/opendray/opendray-v2/internal/channel/telegram" // register kind=telegram
	"github.com/opendray/opendray-v2/internal/config"
	"github.com/opendray/opendray-v2/internal/eventbus"
	"github.com/opendray/opendray-v2/internal/gateway"
	"github.com/opendray/opendray-v2/internal/integration"
	"github.com/opendray/opendray-v2/internal/session"
	"github.com/opendray/opendray-v2/internal/store"
	"github.com/opendray/opendray-v2/internal/version"
)

type App struct {
	cfg          config.Config
	log          *slog.Logger
	store        *store.Store
	bus          *eventbus.Hub
	sessions     *session.Manager
	channels     *channel.Hub
	integrations *integration.Service
	healthCheck  *integration.HealthChecker
	audit        *audit.Sink
	server       *http.Server
}

// New wires the runtime dependencies but does not start any goroutines.
// Caller is responsible for calling Run or Close.
func New(ctx context.Context, cfg config.Config) (*App, error) {
	log := newLogger(cfg.Log)
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

	var sessionOpts []session.ManagerOption
	if d := cfg.Session.Threshold(); d > 0 {
		sessionOpts = append(sessionOpts, session.WithIdleThreshold(d))
	}
	if d := cfg.Session.Interval(); d > 0 {
		sessionOpts = append(sessionOpts, session.WithIdleInterval(d))
	}
	sessionMgr := session.NewManager(
		st.Pool(),
		bus,
		catalog.NewSessionProvider(cat),
		log,
		sessionOpts...,
	)
	sessionHandlers := session.NewHandlers(sessionMgr, log)

	channelHub := channel.NewHub(st.Pool(), bus, log)
	channelHandlers := channel.NewHandlers(channelHub, log)

	intgrSvc := integration.NewService(st.Pool(), bus, log)
	intgrHandlers := integration.NewHandlers(intgrSvc, log)
	proxyHandlers := integration.NewProxyHandlers(intgrSvc, log)
	eventsHandler := integration.NewEventsHandler(bus, log)
	healthCheck := integration.NewHealthChecker(intgrSvc, bus, log)

	auditSink := audit.NewSink(st.Pool(), bus, log)

	gw := gateway.NewServer(gateway.Deps{
		Logger:    log,
		DB:        st,
		Version:   version.Current(),
		StartedAt: time.Now(),
		V1Routes: func(r chi.Router) {
			// Public: only login. /health stays handled by gateway itself.
			authHandlers.MountPublic(r)

			// Admin-only: integration CRUD + reverse proxy.
			r.Group(func(r chi.Router) {
				r.Use(authSvc.Middleware)
				intgrHandlers.MountAdmin(r)
				proxyHandlers.Mount(r)
			})

			// Dual-auth (admin OR integration API key): all business
			// endpoints. ADR 0006 §1.
			r.Group(func(r chi.Router) {
				r.Use(integration.CombinedMiddleware(authSvc, intgrSvc))
				authHandlers.MountProtected(r)
				sessionHandlers.Mount(r)
				catalogHandlers.Mount(r)
				channelHandlers.Mount(r)
			})

			// Integration-only: event subscription WS.
			r.Group(func(r chi.Router) {
				r.Use(integration.IntegrationOnlyMiddleware(intgrSvc))
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
		cfg:          cfg,
		log:          log,
		store:        st,
		bus:          bus,
		sessions:     sessionMgr,
		channels:     channelHub,
		integrations: intgrSvc,
		healthCheck:  healthCheck,
		audit:        auditSink,
		server:       srv,
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
