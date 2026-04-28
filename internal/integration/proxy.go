package integration

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
)

// ProxyHandlers serves /api/v1/proxy/{prefix}/* — admin-only.
type ProxyHandlers struct {
	svc *Service
	log *slog.Logger
}

func NewProxyHandlers(svc *Service, log *slog.Logger) *ProxyHandlers {
	if log == nil {
		log = slog.Default()
	}
	return &ProxyHandlers{svc: svc, log: log.With("component", "integration.proxy")}
}

// Mount registers the proxy routes. Caller wraps with admin middleware.
func (h *ProxyHandlers) Mount(r chi.Router) {
	r.HandleFunc("/proxy/{prefix}", h.serve)
	r.HandleFunc("/proxy/{prefix}/*", h.serve)
}

func (h *ProxyHandlers) serve(w http.ResponseWriter, r *http.Request) {
	prefix := chi.URLParam(r, "prefix")
	intgr, err := h.svc.GetByPrefix(r.Context(), prefix)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !intgr.Enabled {
		writeError(w, http.StatusServiceUnavailable, errors.New("integration disabled"))
		return
	}
	if intgr.HealthStatus == HealthUnhealthy {
		writeError(w, http.StatusServiceUnavailable, errors.New("integration unhealthy"))
		return
	}

	target, err := url.Parse(intgr.BaseURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("invalid base_url: %w", err))
		return
	}

	stripPrefix := "/api/v1/proxy/" + prefix
	intgrID := intgr.ID
	logger := h.log

	proxy := httputil.NewSingleHostReverseProxy(target)
	origDirector := proxy.Director
	proxy.Director = func(pr *http.Request) {
		origDirector(pr)
		// Rewrite the upstream path: drop our /api/v1/proxy/{prefix}.
		path := strings.TrimPrefix(pr.URL.Path, stripPrefix)
		if path == "" {
			path = "/"
		} else if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		pr.URL.Path = path
		pr.URL.RawPath = ""

		// opendray-supplied headers; strip caller's bearer.
		pr.Header.Set("X-OpenDray-Forwarded-For", r.RemoteAddr)
		pr.Header.Set("X-Integration-ID", intgrID)
		pr.Header.Set("X-OpenDray-API", "v1")
		pr.Header.Del("Authorization")

		pr.Host = target.Host
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		if errors.Is(err, context.Canceled) {
			return
		}
		logger.Warn("proxy error", "id", intgrID, "err", err)
		writeError(w, http.StatusBadGateway, fmt.Errorf("upstream: %w", err))
	}
	proxy.ServeHTTP(w, r)
}
