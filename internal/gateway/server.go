// Package gateway is opendray's HTTP layer: chi router + middleware +
// route mounting. It owns no business state — every handler delegates to a
// subsystem passed in via Deps.
//
// Subsystem boundaries (design §6 ordering):
//   - HTTP router is dumb — it routes requests to subsystems.
//   - Subsystems own their state and communicate via the event bus.
//   - Auth is enforced as middleware at the router level *and* defended in
//     depth by each subsystem.
package gateway

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/opendray/opendray-v2/internal/version"
)

// Pinger is satisfied by *store.Store. Defined here so gateway/ does not
// import store/ — accept interfaces, return structs (design §15).
type Pinger interface {
	Ping(ctx context.Context) error
}

type Deps struct {
	Logger    *slog.Logger
	DB        Pinger
	Version   version.Info
	StartedAt time.Time
}

// Server is opendray's HTTP server with mounted v1 routes.
type Server struct {
	deps Deps
	mux  http.Handler
}

func NewServer(deps Deps) *Server {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	if deps.StartedAt.IsZero() {
		deps.StartedAt = time.Now()
	}
	s := &Server{deps: deps}
	s.mux = s.routes()
	return s
}

// Handler returns the http.Handler ready to mount on a net/http server.
func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(slogMiddleware(s.deps.Logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", s.handleHealth)
	})

	return r
}
