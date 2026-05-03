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
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// ProxyHandlers serves /api/v1/proxy/{prefix}/* — admin-only.
type ProxyHandlers struct {
	svc *Service
	cl  *CallLogger
	log *slog.Logger
}

// NewProxyHandlers — cl may be nil (call logging disabled), in which
// case proxied calls won't be recorded. The proxy still serves traffic.
func NewProxyHandlers(svc *Service, cl *CallLogger, log *slog.Logger) *ProxyHandlers {
	if log == nil {
		log = slog.Default()
	}
	return &ProxyHandlers{
		svc: svc,
		cl:  cl,
		log: log.With("component", "integration.proxy"),
	}
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

	// Compute the upstream path once so both Director and the call
	// logger see the same value.
	upstreamPath := strings.TrimPrefix(r.URL.Path, stripPrefix)
	if upstreamPath == "" {
		upstreamPath = "/"
	} else if !strings.HasPrefix(upstreamPath, "/") {
		upstreamPath = "/" + upstreamPath
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	origDirector := proxy.Director
	proxy.Director = func(pr *http.Request) {
		origDirector(pr)
		pr.URL.Path = upstreamPath
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

	// Wrap the writer so we can capture status + bytes for the audit
	// row. Without this we'd record status=0 for every call.
	start := time.Now()
	ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
	proxy.ServeHTTP(ww, r)

	if h.cl != nil {
		h.cl.LogOutbound(
			r.Context(),
			intgrID,
			r.Method,
			upstreamPath,
			ww.Status(),
			int(time.Since(start).Milliseconds()),
			int64(ww.BytesWritten()),
			middleware.GetReqID(r.Context()),
		)
	}
}
